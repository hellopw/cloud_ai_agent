import express from "express";
import { Agent } from "@earendil-works/pi-agent-core";
import { getModel, streamSimple } from "@earendil-works/pi-ai";
import { NodeExecutionEnv } from "@earendil-works/pi-agent-core/node";
import { readFileSync, existsSync, mkdirSync } from "fs";
import { resolve, dirname, join } from "path";
import { fileURLToPath } from "url";
import { execSync } from "child_process";
import { createLLMLogger } from "./llm-logger.js";

const __dirname = dirname(fileURLToPath(import.meta.url));
const workspaceDir = process.env.WORKSPACE_DIR || "/workspace";
mkdirSync(workspaceDir, { recursive: true });

const env = new NodeExecutionEnv({ cwd: workspaceDir });

// LLM logger setup
const logsDir = process.env.LOGS_DIR || "/logs";
const instanceId = process.env.INSTANCE_ID || "unknown";
const logger = createLLMLogger(logsDir, instanceId);

// ---- Built-in tools ----

const tools = [
  {
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
    execute: async ({ command, timeout }, { signal }) => {
      const result = await env.exec(command, {
        cwd: workspaceDir,
        timeout,
        abortSignal: signal,
      });
      if (result.ok) {
        return {
          content: [
            { type: "text", text: result.value.stdout || "(no output)" },
            ...(result.value.stderr ? [{ type: "text", text: "STDERR:\n" + result.value.stderr }] : []),
          ],
          details: { exitCode: result.value.exitCode },
          isError: result.value.exitCode !== 0,
        };
      }
      return {
        content: [{ type: "text", text: "Error: " + result.error.message }],
        isError: true,
      };
    },
  },
  {
    name: "read_file",
    description: "Read the contents of a file. Returns the file content as text.",
    parameters: {
      type: "object",
      properties: {
        path: { type: "string", description: "Path to the file, relative to workspace or absolute." },
      },
      required: ["path"],
    },
    execute: async ({ path }, { signal }) => {
      const result = await env.readTextFile(path, signal);
      if (result.ok) {
        return { content: [{ type: "text", text: result.value }] };
      }
      return { content: [{ type: "text", text: "Error: " + result.error.message }], isError: true };
    },
  },
  {
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
    execute: async ({ path, content }, { signal }) => {
      const result = await env.writeFile(path, content, signal);
      if (result.ok) {
        return { content: [{ type: "text", text: "File written successfully." }] };
      }
      return { content: [{ type: "text", text: "Error: " + result.error.message }], isError: true };
    },
  },
  {
    name: "edit_file",
    description: "Apply a structured patch to edit a file. Patch format: find and replace blocks with line numbers.",
    parameters: {
      type: "object",
      properties: {
        path: { type: "string", description: "Path to the file to edit." },
        old_string: { type: "string", description: "Exact string to find and replace." },
        new_string: { type: "string", description: "Replacement string." },
      },
      required: ["path", "old_string", "new_string"],
    },
    execute: async ({ path, old_string, new_string }, { signal }) => {
      const readResult = await env.readTextFile(path, signal);
      if (!readResult.ok) {
        return { content: [{ type: "text", text: "Error reading file: " + readResult.error.message }], isError: true };
      }
      const content = readResult.value;
      if (!content.includes(old_string)) {
        return { content: [{ type: "text", text: "Error: old_string not found in file." }], isError: true };
      }
      const newContent = content.replace(old_string, new_string);
      const writeResult = await env.writeFile(path, newContent, signal);
      if (writeResult.ok) {
        return { content: [{ type: "text", text: "File edited successfully." }] };
      }
      return { content: [{ type: "text", text: "Error writing file: " + writeResult.error.message }], isError: true };
    },
  },
  {
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
    execute: async ({ pattern, path, fileTypes }) => {
      const searchPath = path || workspaceDir;
      let cmd = `grep -rn --color=never "${pattern}" "${searchPath}" 2>/dev/null`;
      if (fileTypes) {
        const exts = fileTypes.split(",").map((e) => e.trim()).filter(Boolean);
        if (exts.length > 0) {
          const includes = exts.map((e) => `--include="*${e}"`).join(" ");
          cmd = `grep -rn --color=never ${includes} "${pattern}" "${searchPath}" 2>/dev/null`;
        }
      }
      const result = await env.exec(cmd, { cwd: workspaceDir, timeout: 30 });
      if (result.ok) {
        return { content: [{ type: "text", text: result.value.stdout || "No matches found." }] };
      }
      return { content: [{ type: "text", text: "No matches found." }] };
    },
  },
  {
    name: "list_files",
    description: "List files and directories in a given path.",
    parameters: {
      type: "object",
      properties: {
        path: { type: "string", description: "Directory path to list." },
      },
      required: [],
    },
    execute: async ({ path }) => {
      const listPath = path || workspaceDir;
      const result = await env.listDir(listPath);
      if (result.ok) {
        const lines = result.value.map(
          (f) => `${f.isDirectory ? "d" : "-"} ${f.name} (${f.size || 0} bytes)`
        );
        return { content: [{ type: "text", text: lines.join("\n") || "(empty)" }] };
      }
      return { content: [{ type: "text", text: "Error: " + result.error.message }], isError: true };
    },
  },
  {
    name: "git",
    description: "Run a git command in the workspace. Common subcommands: status, diff, log, branch, add, commit.",
    parameters: {
      type: "object",
      properties: {
        subcommand: { type: "string", description: "Git subcommand with arguments (e.g., 'status', 'diff --stat', 'log --oneline -5')." },
      },
      required: ["subcommand"],
    },
    execute: async ({ subcommand }) => {
      const result = await env.exec(`git ${subcommand}`, { cwd: workspaceDir, timeout: 30 });
      if (result.ok) {
        return { content: [{ type: "text", text: result.value.stdout || "(no output)" }] };
      }
      return { content: [{ type: "text", text: result.value.stderr || "Git command failed." }], isError: true };
    },
  },
];

function setupAgent() {
  const provider = process.env.AGENT_PROVIDER || "openai-codex";
  const modelId = process.env.AGENT_MODEL || "gpt-5.1-codex-max";
  const apiKey = process.env.AGENT_API_KEY || process.env.OPENAI_API_KEY || "";
  const baseUrl = process.env.AGENT_BASE_URL || "";

  let model;
  try {
    model = getModel(provider, modelId);
    if (model && baseUrl) {
      model = { ...model, baseUrl };
    }
  } catch (e) {
    console.error(`Failed to get model ${provider}/${modelId}:`, e.message);
    model = null;
  }

  if (!model) {
    console.error(`Falling back to default model for ${provider}/${modelId}`);
    const api = baseUrl ? "anthropic-messages" : "openai-codex-responses";
    model = {
      id: modelId,
      name: modelId,
      api,
      provider: provider,
      baseUrl: baseUrl,
      reasoning: false,
      input: [],
      cost: { input: 0, output: 0, cacheRead: 0, cacheWrite: 0 },
      contextWindow: 200000,
      maxTokens: 32000,
    };
  }

  console.log(`Agent configured with provider=${provider} model=${modelId}`);
  console.log(`Model: ${JSON.stringify(model)}`);

  const agent = new Agent({
    streamFn: streamSimple,
    transport: "http",
    getApiKey: async (providerName) => {
      if (apiKey) return apiKey;
      return undefined;
    },
    initialState: {
      model,
      tools,
    },
  });

  return { agent, model };
}

const app = express();
app.use(express.json({ limit: "10mb" }));

// ---- API call logging (fetch monkey-patch + file logging) ----
let activeSeq = null;

const _fetch = globalThis.fetch;
globalThis.fetch = async (url, opts) => {
  const start = Date.now();
  const method = opts?.method || "GET";

  // Log request to file if we're in an active logging session
  if (activeSeq) {
    const bodyStr = opts?.body;
    let parsedBody = "";
    try {
      parsedBody = bodyStr ? JSON.parse(typeof bodyStr === "string" ? bodyStr : "[stream]") : "";
    } catch {
      parsedBody = typeof bodyStr === "string" ? bodyStr.substring(0, 2000) : "[stream]";
    }
    logger.logRequest(activeSeq, {
      method,
      url,
      body: parsedBody,
    });
  }

  console.log(`[API] --> ${method} ${url} body=${typeof opts?.body === "string" ? opts.body.substring(0, 500) : "[stream]"}`);

  try {
    const resp = await _fetch(url, opts);
    const clone = resp.clone();
    const elapsed = Date.now() - start;

    // Log response to file (read full SSE text for PI agent)
    if (activeSeq) {
      const text = await clone.text();
      const isStream = resp.headers.get("content-type")?.includes("text/event-stream");
      if (isStream) {
        const lines = text.split("\n").filter(Boolean);
        for (const line of lines) {
          if (line.startsWith("data: ")) {
            try {
              const data = JSON.parse(line.slice(6));
              logger.appendResponseLine(activeSeq, data);
            } catch {
              logger.appendResponseLine(activeSeq, { raw: line });
            }
          }
        }
      }
      logger.appendResponseLine(activeSeq, {
        type: "http_response",
        status: resp.status,
        elapsed_ms: elapsed,
        content_type: resp.headers.get("content-type"),
      });
    }

    console.log(`[API] <-- ${resp.status} ${elapsed}ms body=${text.substring(0, 2000)}`);
    return resp;
  } catch (e) {
    if (activeSeq) {
      logger.appendResponseLine(activeSeq, { type: "fetch_error", message: e.message });
    }
    console.log(`[API] <-- ERROR ${Date.now() - start}ms ${e.message}`);
    throw e;
  }
};

let agentState;

app.get("/status", (req, res) => {
  res.json({ status: "running", ready: !!agentState, provider: process.env.AGENT_PROVIDER || "openai-codex", model: process.env.AGENT_MODEL || "gpt-5.1-codex-max" });
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

  // Init logging session and write snapshots
  logger.newSession();
  logger.writeToolsSnapshot(tools);
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
    agentState.agent.subscribe((event) => {
      sendEvent(event.type, event);
    });

    await agentState.agent.prompt(message);
    sendEvent("agent_end", { stopReason: "end_turn" });
    logger.appendResponseLine(activeSeq, { type: "agent_end", stopReason: "end_turn" });
  } catch (err) {
    sendEvent("error", { message: err.message });
    logger.appendResponseLine(activeSeq, { type: "fatal_error", message: err.message, stack: err.stack });
  } finally {
    activeSeq = null;
    res.end();
  }
});

app.post("/abort", (req, res) => {
  if (agentState) {
    agentState.agent.abort();
    res.json({ status: "aborted" });
  } else {
    res.json({ status: "no active agent" });
  }
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
  console.log(`Agent wrapper listening on port ${port}`);
  console.log(`Workspace: ${workspaceDir}`);
  console.log(`Logs: ${logsDir}/${instanceId}`);
});
