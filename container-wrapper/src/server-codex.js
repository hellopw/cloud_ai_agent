import express from "express";
import OpenAI from "openai";
import { readFileSync, existsSync, mkdirSync } from "fs";
import { resolve, dirname } from "path";
import { fileURLToPath } from "url";
import { execSync } from "child_process";
import { listMcpTools, callMcpTool } from "./mcp-client.js";
import { createLLMLogger } from "./llm-logger.js";

const __dirname = dirname(fileURLToPath(import.meta.url));
const workspaceDir = process.env.WORKSPACE_DIR || "/workspace";
mkdirSync(workspaceDir, { recursive: true });

// LLM logger setup
const logsDir = process.env.LOGS_DIR || "/logs";
const instanceId = process.env.INSTANCE_ID || "unknown";
const logger = createLLMLogger(logsDir, instanceId);

// ---- Built-in tools ----
const builtinTools = [
  {
    type: "function",
    name: "bash",
    description: "Execute a shell command in the workspace. Returns stdout, stderr, and exit code.",
    parameters: {
      type: "object",
      properties: {
        command: { type: "string", description: "The shell command to execute." },
        timeout: { type: "number", description: "Optional timeout in seconds." },
      },
      required: ["command"],
    },
  },
  {
    type: "function",
    name: "read_file",
    description: "Read the contents of a file. Returns the file content as text.",
    parameters: {
      type: "object",
      properties: {
        path: { type: "string", description: "Path to the file, relative to workspace or absolute." },
      },
      required: ["path"],
    },
  },
  {
    type: "function",
    name: "write_file",
    description: "Write content to a file. Creates parent directories if needed.",
    parameters: {
      type: "object",
      properties: {
        path: { type: "string", description: "Path to the file, relative to workspace or absolute." },
        content: { type: "string", description: "Content to write." },
      },
      required: ["path", "content"],
    },
  },
  {
    type: "function",
    name: "edit_file",
    description: "Apply a structured patch to edit a file. Replace old_string with new_string.",
    parameters: {
      type: "object",
      properties: {
        path: { type: "string", description: "Path to the file to edit." },
        old_string: { type: "string", description: "Exact string to find and replace." },
        new_string: { type: "string", description: "Replacement string." },
      },
      required: ["path", "old_string", "new_string"],
    },
  },
  {
    type: "function",
    name: "search_content",
    description: "Search for text patterns in files using grep.",
    parameters: {
      type: "object",
      properties: {
        pattern: { type: "string", description: "The search pattern (regex supported)." },
        path: { type: "string", description: "Directory or file to search in." },
        fileTypes: { type: "string", description: "Optional comma-separated file extensions (e.g., '.js,.ts,.go')." },
      },
      required: ["pattern"],
    },
  },
  {
    type: "function",
    name: "list_files",
    description: "List files and directories in a given path.",
    parameters: {
      type: "object",
      properties: {
        path: { type: "string", description: "Directory path to list." },
      },
    },
  },
  {
    type: "function",
    name: "git",
    description: "Run a git command in the workspace. Common subcommands: status, diff, log, branch, add, commit.",
    parameters: {
      type: "object",
      properties: {
        subcommand: { type: "string", description: "Git subcommand with arguments (e.g., 'status', 'diff --stat', 'log --oneline -5')." },
      },
      required: ["subcommand"],
    },
  },
];

function mcpToolToCodexFormat(t) {
  return {
    type: "function",
    name: t.name,
    description: t.description || "",
    parameters: t.inputSchema || { type: "object", properties: {} },
  };
}

// ---- MCP extension loading ----
const mcpToolConfigs = {}; // tool_name -> handler config
const mcpCodexTools = []; // discovered MCP tools in Codex format
let mcpDiscoveryPromise = null;

function startMcpDiscovery() {
  const extPath = resolve("/app/extensions/extensions.json");
  if (!existsSync(extPath)) {
    console.log("No extensions.json found at", extPath);
    mcpDiscoveryPromise = Promise.resolve();
    return;
  }
  const extConfigs = JSON.parse(readFileSync(extPath, "utf-8"));
  console.log(`Starting MCP discovery for ${extConfigs.length} extension configs`);

  const mcpConfigs = extConfigs.filter(cfg => cfg.handler && cfg.handler.type === "mcp");
  const seen = new Set(builtinTools.map(t => t.name));

  const discoveryTasks = mcpConfigs.map(cfg => {
    const h = cfg.handler;
    return listMcpTools({
      transport: h.transport || "stdio",
      command: h.command,
      args: h.args || [],
      env: h.env || {},
      url: h.url || "",
    })
      .then(tools => {
        let added = 0;
        for (const t of tools) {
          if (seen.has(t.name)) {
            console.log(`MCP "${cfg.name}": skipping duplicate tool "${t.name}"`);
            continue;
          }
          seen.add(t.name);
          mcpToolConfigs[t.name] = h;
          mcpCodexTools.push(mcpToolToCodexFormat(t));
          added++;
        }
        console.log(`MCP "${cfg.name}": added ${added} tools (total MCP: ${mcpCodexTools.length})`);
      })
      .catch(err => {
        console.error(`MCP "${cfg.name}": discovery failed - ${err.message || err}`);
      });
  });

  mcpDiscoveryPromise = Promise.allSettled(discoveryTasks);
}

async function ensureMcpReady() {
  if (mcpDiscoveryPromise) await mcpDiscoveryPromise;
}

function getTools() {
  return [...builtinTools, ...mcpCodexTools];
}

// ---- Tool execution ----
async function executeTool(name, args) {
  // MCP tools
  if (mcpToolConfigs[name]) {
    const cfg = mcpToolConfigs[name];
    try {
      const result = await callMcpTool({
        transport: cfg.transport || "stdio",
        command: cfg.command,
        args: cfg.args || [],
        env: cfg.env || {},
        toolName: name,
        toolArgs: args,
      });
      if (result.isError) {
        return `Error: ${result.content?.[0]?.text || JSON.stringify(result)}`;
      }
      return result.content?.map(c => c.text).join("\n") || JSON.stringify(result);
    } catch (e) {
      return `MCP tool "${name}" error: ${e.message}`;
    }
  }

  // Built-in tools
  switch (name) {
    case "bash": {
      try {
        const opts = { cwd: workspaceDir, timeout: (args.timeout || 30) * 1000, encoding: "utf-8" };
        const stdout = execSync(args.command, { ...opts, stdio: ["pipe", "pipe", "pipe"] });
        return String(stdout);
      } catch (e) {
        return `EXIT ${e.status}: ${e.stderr || e.stdout || e.message}`;
      }
    }
    case "read_file": {
      const p = (args.path || "").startsWith("/") ? args.path : resolve(workspaceDir, args.path);
      if (!existsSync(p)) return `Error: file not found: ${p}`;
      return readFileSync(p, "utf-8");
    }
    case "write_file": {
      const p = (args.path || "").startsWith("/") ? args.path : resolve(workspaceDir, args.path);
      mkdirSync(dirname(p), { recursive: true });
      const { writeFileSync } = await import("fs");
      writeFileSync(p, args.content, "utf-8");
      return "File written successfully.";
    }
    case "edit_file": {
      const p = (args.path || "").startsWith("/") ? args.path : resolve(workspaceDir, args.path);
      if (!existsSync(p)) return `Error: file not found: ${p}`;
      const content = readFileSync(p, "utf-8");
      if (!content.includes(args.old_string)) return "Error: old_string not found in file.";
      const newContent = content.replace(args.old_string, args.new_string);
      const { writeFileSync } = await import("fs");
      writeFileSync(p, newContent, "utf-8");
      return "File edited successfully.";
    }
    case "search_content": {
      try {
        const searchPath = args.path || workspaceDir;
        let exts = "";
        if (args.fileTypes) {
          exts = args.fileTypes.split(",").map(e => `--include="*${e.trim()}"`).join(" ");
        }
        const cmd = `grep -rn --color=never ${exts} "${args.pattern}" "${searchPath}" 2>/dev/null`;
        const stdout = execSync(cmd, { cwd: workspaceDir, timeout: 30000, encoding: "utf-8" });
        return stdout || "No matches found.";
      } catch (e) {
        if (e.status === 1 && !e.stderr) return "No matches found.";
        return `Error: ${e.message}`;
      }
    }
    case "list_files": {
      try {
        const listPath = args.path || workspaceDir;
        const { readdirSync, statSync } = await import("fs");
        const entries = readdirSync(listPath, { withFileTypes: true });
        return entries.map(f => `${f.isDirectory() ? "d" : "-"} ${f.name} (${f.isFile() ? statSync(resolve(listPath, f.name)).size : 0} bytes)`).join("\n") || "(empty)";
      } catch (e) {
        return `Error: ${e.message}`;
      }
    }
    case "git": {
      try {
        const stdout = execSync(`git ${args.subcommand}`, { cwd: workspaceDir, timeout: 30000, encoding: "utf-8" });
        return stdout || "(no output)";
      } catch (e) {
        return e.stderr || `Git command failed: ${e.message}`;
      }
    }
    default:
      return `Unknown tool: ${name}`;
  }
}

function setupAgent() {
  const apiKey = process.env.AGENT_API_KEY || process.env.OPENAI_API_KEY || "";
  const baseURL = process.env.AGENT_BASE_URL || "";
  const modelId = process.env.AGENT_MODEL || "gpt-5-codex";

  console.log(`Codex agent configured with model=${modelId}`);

  const client = new OpenAI({
    apiKey,
    baseURL: baseURL || undefined,
  });

  return { client, modelId };
}

const app = express();
app.use(express.json({ limit: "10mb" }));

let agentState;
let activeSeq = null;

app.get("/status", (req, res) => {
  res.json({ status: "running", ready: !!agentState, provider: "openai", model: process.env.AGENT_MODEL || "gpt-5-codex" });
});

app.get("/health", (req, res) => {
  res.json({ status: "ok" });
});

app.post("/chat", async (req, res) => {
  const { message } = req.body;
  if (!message) {
    return res.status(400).json({ error: "message required" });
  }

  if (!agentState) {
    agentState = setupAgent();
  }

  // Init LLM logging session
  logger.newSession();
  logger.writeToolsSnapshot(getTools());
  const skillsDir = process.env.SKILLS_DIR || "/app/pi-skills";
  const promptsDir = process.env.PROMPTS_DIR || "/app/pi-prompts";
  logger.writeSkillsSnapshot(skillsDir);
  logger.writePromptsSnapshot(promptsDir);
  activeSeq = logger.nextSeq();

  res.setHeader("Content-Type", "text/event-stream");
  res.setHeader("Cache-Control", "no-cache");
  res.setHeader("Connection", "keep-alive");

  function sendEvent(event, data) {
    res.write(`event: ${event}\ndata: ${JSON.stringify(data)}\n\n`);
  }

  try {
    await ensureMcpReady();
    const tools = getTools();
    let input = [{ role: "user", content: message }];
    let turnCount = 0;
    const maxTurns = 30;

    while (turnCount < maxTurns) {
      turnCount++;

      const stream = await agentState.client.responses.create({
        model: agentState.modelId,
        input: input,
        tools: tools,
        stream: true,
      });

      let finalResponse = null;

      for await (const event of stream) {
        if (event.type === "response.completed") {
          finalResponse = event.response;
        }
        sendEvent("response", event);
      }

      if (!finalResponse) break;

      const output = finalResponse.output || [];
      const functionCalls = output.filter(o => o.type === "function_call");
      if (functionCalls.length === 0) break;

      // Build next input with tool results
      const newInput = [...output.filter(o => o.type !== "function_call")];

      for (const fc of functionCalls) {
        sendEvent("tool_call", {
          toolCallId: fc.call_id,
          toolName: fc.name,
          input: fc.arguments,
        });

        const args = typeof fc.arguments === "string" ? JSON.parse(fc.arguments) : fc.arguments;
        const result = await executeTool(fc.name, args);

        sendEvent("tool_result", { toolCallId: fc.call_id, content: result });

        newInput.push({
          type: "function_call_output",
          call_id: fc.call_id,
          output: result,
        });
      }

      input = newInput;
    }

    sendEvent("agent_end", { stopReason: "end_turn" });
  } catch (err) {
    logger.appendResponseLine(activeSeq, { type: "fatal_error", message: err.message, stack: err.stack });
    sendEvent("error", { message: err.message });
  } finally {
    activeSeq = null;
    res.end();
  }
});

app.post("/abort", (req, res) => {
  res.json({ status: "aborted" });
});

// Write initial snapshots on startup
const startupSkillsDir = process.env.SKILLS_DIR || "/app/pi-skills";
const startupPromptsDir = process.env.PROMPTS_DIR || "/app/pi-prompts";
if (startupSkillsDir || startupPromptsDir) {
  logger.newSession();
  logger.writeSkillsSnapshot(startupSkillsDir);
  logger.writePromptsSnapshot(startupPromptsDir);
}

const port = process.env.PORT || 3000;
app.listen(port, () => {
  console.log(`Codex wrapper listening on port ${port}`);
  console.log(`Workspace: ${workspaceDir}`);
  console.log(`Starting with ${builtinTools.length} builtin tools`);
  console.log(`Logs: ${logsDir}/${instanceId}`);
  startMcpDiscovery();
});
