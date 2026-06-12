import express from "express";
import Anthropic from "@anthropic-ai/sdk";
import * as fs from "fs";
import * as path from "path";
import { fileURLToPath } from "url";
import { execSync } from "child_process";
import { listMcpTools, callMcpTool } from "./mcp-client.js";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const workspaceDir = process.env.WORKSPACE_DIR || "/workspace";
fs.mkdirSync(workspaceDir, { recursive: true });

// ---- Built-in tools ----
const builtinTools = [
  {
    name: "bash",
    description: "Execute a shell command in the workspace. Returns stdout, stderr, and exit code.",
    input_schema: {
      type: "object",
      properties: {
        command: { type: "string", description: "The shell command to execute." },
        timeout: { type: "number", description: "Optional timeout in seconds." },
      },
      required: ["command"],
    },
  },
  {
    name: "read_file",
    description: "Read the contents of a file. Returns the file content as text.",
    input_schema: {
      type: "object",
      properties: {
        path: { type: "string", description: "Path to the file, relative to workspace or absolute." },
      },
      required: ["path"],
    },
  },
  {
    name: "write_file",
    description: "Write content to a file. Creates parent directories if needed.",
    input_schema: {
      type: "object",
      properties: {
        path: { type: "string", description: "Path to the file, relative to workspace or absolute." },
        content: { type: "string", description: "Content to write." },
      },
      required: ["path", "content"],
    },
  },
  {
    name: "edit_file",
    description: "Apply a structured patch to edit a file. Replace old_string with new_string.",
    input_schema: {
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
    name: "search_content",
    description: "Search for text patterns in files using grep.",
    input_schema: {
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
    name: "list_files",
    description: "List files and directories in a given path.",
    input_schema: {
      type: "object",
      properties: {
        path: { type: "string", description: "Directory path to list." },
      },
    },
  },
  {
    name: "git",
    description: "Run a git command in the workspace. Common subcommands: status, diff, log, branch, add, commit.",
    input_schema: {
      type: "object",
      properties: {
        subcommand: { type: "string", description: "Git subcommand with arguments (e.g., 'status', 'diff --stat', 'log --oneline -5')." },
      },
      required: ["subcommand"],
    },
  },
];

// ---- MCP extension loading ----
const mcpToolConfigs = {}; // tool_name -> handler config
let allTools = [...builtinTools]; // start with builtins, MCP added progressively

function startMcpDiscovery() {
  const extPath = path.resolve("/app/extensions/extensions.json");
  if (!fs.existsSync(extPath)) {
    console.log("No extensions.json found at", extPath);
    return;
  }
  const extConfigs = JSON.parse(fs.readFileSync(extPath, "utf-8"));
  console.log(`Starting MCP discovery for ${extConfigs.length} extension configs`);

  const mcpConfigs = extConfigs.filter(cfg => cfg.handler && cfg.handler.type === "mcp");
  const seen = new Set(builtinTools.map(t => t.name));

  // Discover each MCP server independently – tools are added as they arrive
  for (const cfg of mcpConfigs) {
    listMcpTools({
      transport: cfg.handler.transport || "stdio",
      command: cfg.handler.command,
      args: cfg.handler.args || [],
      env: cfg.handler.env || {},
      url: cfg.handler.url || "",
    })
      .then(tools => {
        let added = 0;
        for (const t of tools) {
          if (seen.has(t.name)) {
            console.log(`MCP "${cfg.name}": skipping duplicate tool "${t.name}"`);
            continue;
          }
          seen.add(t.name);
          mcpToolConfigs[t.name] = cfg.handler;
          allTools.push({
            name: t.name,
            description: t.description || "",
            input_schema: t.inputSchema || { type: "object", properties: {} },
          });
          added++;
        }
        console.log(`MCP "${cfg.name}": added ${added} tools (total: ${allTools.length})`);
      })
      .catch(err => {
        console.error(`MCP "${cfg.name}": discovery failed - ${err.message || err}`);
      });
  }
}

function getTools() {
  return allTools;
}
async function executeTool(name, input) {
  switch (name) {
    case "bash": {
      try {
        const opts = { cwd: workspaceDir, timeout: (input.timeout || 30) * 1000, encoding: "utf-8" };
        const stdout = execSync(input.command, { ...opts, stdio: ["pipe", "pipe", "pipe"] });
        return String(stdout);
      } catch (e) {
        return `EXIT ${e.status}: ${e.stderr || e.stdout || e.message}`;
      }
    }
    case "read_file": {
      const p = input.path.startsWith("/") ? input.path : path.join(workspaceDir, input.path);
      if (!fs.existsSync(p)) return `Error: file not found: ${p}`;
      return fs.readFileSync(p, "utf-8");
    }
    case "write_file": {
      const p = input.path.startsWith("/") ? input.path : path.join(workspaceDir, input.path);
      fs.mkdirSync(path.dirname(p), { recursive: true });
      fs.writeFileSync(p, input.content, "utf-8");
      return "File written successfully.";
    }
    case "edit_file": {
      const p = input.path.startsWith("/") ? input.path : path.join(workspaceDir, input.path);
      if (!fs.existsSync(p)) return `Error: file not found: ${p}`;
      const content = fs.readFileSync(p, "utf-8");
      if (!content.includes(input.old_string)) return "Error: old_string not found in file.";
      const newContent = content.replace(input.old_string, input.new_string);
      fs.writeFileSync(p, newContent, "utf-8");
      return "File edited successfully.";
    }
    case "search_content": {
      try {
        const searchPath = input.path || workspaceDir;
        let exts = "";
        if (input.fileTypes) {
          exts = input.fileTypes.split(",").map(e => `--include="*${e.trim()}"`).join(" ");
        }
        const cmd = `grep -rn --color=never ${exts} "${input.pattern}" "${searchPath}" 2>/dev/null`;
        const stdout = execSync(cmd, { cwd: workspaceDir, timeout: 30000, encoding: "utf-8" });
        return stdout || "No matches found.";
      } catch (e) {
        if (e.status === 1 && !e.stderr) return "No matches found.";
        return `Error: ${e.message}`;
      }
    }
    case "list_files": {
      try {
        const listPath = input.path || workspaceDir;
        const entries = fs.readdirSync(listPath, { withFileTypes: true });
        return entries.map(f => `${f.isDirectory() ? "d" : "-"} ${f.name} (${f.isFile() ? fs.statSync(path.join(listPath, f.name)).size : 0} bytes)`).join("\n") || "(empty)";
      } catch (e) {
        return `Error: ${e.message}`;
      }
    }
    case "git": {
      try {
        const stdout = execSync(`git ${input.subcommand}`, { cwd: workspaceDir, timeout: 30000, encoding: "utf-8" });
        return stdout || "(no output)";
      } catch (e) {
        return e.stderr || `Git command failed: ${e.message}`;
      }
    }
    default:
      // Route to MCP if registered
      if (mcpToolConfigs[name]) {
        const cfg = mcpToolConfigs[name];
        try {
          const result = await callMcpTool({
            transport: cfg.transport || "stdio",
            command: cfg.command,
            args: cfg.args || [],
            env: cfg.env || {},
            toolName: name,
            toolArgs: input,
          });
          if (result.isError) {
            return `MCP error: ${result.content?.[0]?.text || JSON.stringify(result)}`;
          }
          return result.content?.map(c => c.text).join("\n") || JSON.stringify(result);
        } catch (e) {
          return `MCP tool "${name}" error: ${e.message}`;
        }
      }
      return `Unknown tool: ${name}`;
  }
}

const app = express();
app.use(express.json({ limit: "10mb" }));

// ---- API call logging ----
const _fetch = globalThis.fetch;
globalThis.fetch = async (url, opts) => {
  const start = Date.now();
  const method = opts?.method || "GET";
  const body = opts?.body ? (typeof opts.body === "string" ? opts.body.substring(0, 500) : "[stream]") : "";
  console.log(`[API] --> ${method} ${url} body=${body}`);
  try {
    const resp = await _fetch(url, opts);
    const clone = resp.clone();
    const text = await clone.text();
    const elapsed = Date.now() - start;
    console.log(`[API] <-- ${resp.status} ${elapsed}ms body=${text.substring(0, 2000)}`);
    return resp;
  } catch (e) {
    console.log(`[API] <-- ERROR ${Date.now() - start}ms ${e.message}`);
    throw e;
  }
};

app.get("/status", (req, res) => {
  const provider = process.env.AGENT_PROVIDER || "anthropic";
  const model = process.env.AGENT_MODEL || "claude-sonnet-4-20250514";
  res.json({ status: "running", ready: true, provider, model });
});

app.get("/health", (req, res) => {
  res.json({ status: "ok" });
});

app.post("/chat", async (req, res) => {
  const { message } = req.body;
  if (!message) {
    return res.status(400).json({ error: "message required" });
  }

  const apiKey = process.env.AGENT_API_KEY || "";
  const modelId = process.env.AGENT_MODEL || "claude-sonnet-4-20250514";
  const baseUrl = process.env.AGENT_BASE_URL || undefined;

  const client = new Anthropic({ apiKey, baseURL: baseUrl });

  res.setHeader("Content-Type", "text/event-stream");
  res.setHeader("Cache-Control", "no-cache");
  res.setHeader("Connection", "keep-alive");

  function sendEvent(event, data) {
    res.write(`event: ${event}\ndata: ${JSON.stringify(data)}\n\n`);
  }

  try {
    const messages = [{ role: "user", content: message }];

    let turnCount = 0;
    const maxTurns = 30;

    while (turnCount < maxTurns) {
      turnCount++;

      const stream = client.messages.stream({
        model: modelId,
        max_tokens: 16000,
        system: "You are a helpful AI assistant with access to tools for reading/writing files, running commands, and managing git repositories. Use tools when needed to complete the user's task.",
        messages,
        tools: getTools(),
      });

      let contentBlocks = []; // { index, type, id, name, input }

      for await (const event of stream) {
        switch (event.type) {
          case "message_start":
            sendEvent("message_start", event.message);
            break;
          case "content_block_start":
            contentBlocks.push({
              index: event.index,
              type: event.content_block.type,
              id: event.content_block.id,
              name: event.content_block.name,
              input: "",
            });
            break;
          case "content_block_delta": {
            const block = contentBlocks.find(b => b.index === event.index);
            if (!block) break;
            if (event.delta.type === "thinking_delta") {
              block.input += event.delta.thinking;
              sendEvent("message_update", {
                assistantMessageEvent: {
                  type: "thinking_delta",
                  delta: event.delta.thinking,
                },
              });
            } else if (event.delta.type === "text_delta") {
              block.input += event.delta.text;
              sendEvent("message_update", {
                assistantMessageEvent: {
                  type: "text_delta",
                  delta: event.delta.text,
                },
              });
            } else if (event.delta.type === "input_json_delta") {
              if (event.delta.partial_json) {
                block.input += event.delta.partial_json;
              }
            }
            break;
          }
          case "content_block_stop":
            break;
          case "message_delta":
            sendEvent("message_update", { message_delta: event.delta });
            break;
          case "message_stop":
            break;
        }
      }

      let toolUses = [];
      let assistantContent = [];
      const finalMessage = stream.finalMessage();
      if (finalMessage && finalMessage.content) {
        assistantContent = finalMessage.content;
        toolUses = finalMessage.content.filter(c => c.type === "tool_use");
      }

      // Fallback: reconstruct from accumulated streaming blocks if finalMessage lacks tool_use
      if (toolUses.length === 0) {
        const streamToolUses = contentBlocks.filter(b => b.type === "tool_use" && b.name);
        if (streamToolUses.length > 0) {
          assistantContent = contentBlocks.map(b => {
            if (b.type === "tool_use") {
              let parsedInput = {};
              try { parsedInput = JSON.parse(b.input || "{}"); } catch (_) {}
              return { type: "tool_use", id: b.id, name: b.name, input: parsedInput };
            }
            if (b.type === "text") return { type: "text", text: b.input };
            if (b.type === "thinking") return { type: "thinking", thinking: b.input };
            return null;
          }).filter(Boolean);
          toolUses = assistantContent.filter(c => c.type === "tool_use");
        }
      }

      if (toolUses.length === 0) {
        // No more tools, agent is done
        break;
      }

      // Add assistant message with tool uses
      messages.push({ role: "assistant", content: assistantContent });

      // Execute tools and add results
      const toolResults = [];
      for (const tu of toolUses) {
        sendEvent("tool_call", {
          toolCallId: tu.id,
          toolName: tu.name,
          input: tu.input,
        });

        const result = await executeTool(tu.name, tu.input);
        sendEvent("tool_result", { toolCallId: tu.id, content: result });

        toolResults.push({
          type: "tool_result",
          tool_use_id: tu.id,
          content: result,
        });
      }

      messages.push({ role: "user", content: toolResults });
    }

    sendEvent("agent_end", { stopReason: "end_turn" });
  } catch (err) {
    sendEvent("error", { message: err.message });
  } finally {
    res.end();
  }
});

app.post("/abort", (req, res) => {
  res.json({ status: "no active agent to abort" });
});

const port = process.env.PORT || 3000;
app.listen(port, () => {
  const model = process.env.AGENT_MODEL || "claude-sonnet-4-20250514";
  console.log(`Claude Code wrapper listening on port ${port}`);
  console.log(`Workspace: ${workspaceDir}`);
  console.log(`Model: ${model}`);
  console.log(`Starting with ${builtinTools.length} builtin tools`);
  startMcpDiscovery();
});
