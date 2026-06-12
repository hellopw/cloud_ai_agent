import express from "express";
import { Agent } from "@earendil-works/pi-agent-core";
import { getModel, streamSimple } from "@earendil-works/pi-ai";
import { NodeExecutionEnv } from "@earendil-works/pi-agent-core/node";
import { readFileSync, existsSync, mkdirSync } from "fs";
import { resolve, dirname, join } from "path";
import { fileURLToPath } from "url";
import { listMcpTools, callMcpTool } from "./mcp-client.js";
import { createLLMLogger } from "./llm-logger.js";

const __dirname = dirname(fileURLToPath(import.meta.url));
const workspaceDir = process.env.WORKSPACE_DIR || "/workspace";
mkdirSync(workspaceDir, { recursive: true });

const env = new NodeExecutionEnv({ cwd: workspaceDir });

// LLM logger setup
const logsDir = process.env.LOGS_DIR || "/logs";
const instanceId = process.env.INSTANCE_ID || "unknown";
const rootLogger = createLLMLogger(logsDir, instanceId);

// ---- Built-in tools (shared by all agents) ----

function buildBuiltinTools() {
  return [
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
}

// ---- MCP extension loading per member ----

const mcpToolsCache = {};   // memberName -> tools array (populated progressively)
const mcpConfigCache = {};  // memberName -> { toolName: handlerConfig }

function startMcpDiscoveryForMember(memberName) {
  const extPath = resolve(`/app/agents/${memberName}/extensions/extensions.json`);
  if (!existsSync(extPath)) {
    console.log(`No extensions.json for member "${memberName}" at ${extPath}`);
    mcpToolsCache[memberName] = [];
    mcpConfigCache[memberName] = {};
    return Promise.resolve();
  }
  const extConfigs = JSON.parse(readFileSync(extPath, "utf-8"));
  console.log(`[${memberName}] Starting MCP discovery for ${extConfigs.length} extension configs`);

  const mcpConfigs = extConfigs.filter(cfg => cfg.handler && cfg.handler.type === "mcp");
  const seen = new Set();
  const memberTools = [];
  const memberConfigs = {};
  mcpToolsCache[memberName] = memberTools;
  mcpConfigCache[memberName] = memberConfigs;

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
            console.log(`[${memberName}] MCP "${cfg.name}": skipping duplicate tool "${t.name}"`);
            continue;
          }
          seen.add(t.name);
          memberConfigs[t.name] = h;
          memberTools.push({
            name: t.name,
            description: t.description || "",
            parameters: t.inputSchema || { type: "object", properties: {} },
            execute: async (args, { signal }) => {
              const result = await callMcpTool({
                transport: h.transport || "stdio",
                command: h.command,
                args: h.args || [],
                env: h.env || {},
                toolName: t.name,
                toolArgs: args,
                signal,
              });
              return result;
            },
          });
          added++;
        }
        console.log(`[${memberName}] MCP "${cfg.name}": added ${added} tools (total: ${memberTools.length})`);
      })
      .catch(err => {
        console.error(`[${memberName}] MCP "${cfg.name}": discovery failed - ${err.message || err}`);
      });
  });

  return Promise.allSettled(discoveryTasks);
}

// ---- Agent setup per member ----

function setupAgent(memberConfig, additionalTools = []) {
  const { provider, model_id, api_key, base_url, system_prompt_override } = memberConfig;

  let model;
  try {
    model = getModel(provider || "openai-codex", model_id || "gpt-5.1-codex-max");
    if (base_url) {
      model = { ...model, baseUrl: base_url };
    }
  } catch (e) {
    console.error(`Failed to get model ${provider}/${model_id}:`, e.message);
    model = {
      id: model_id || "gpt-5.1-codex-max",
      name: model_id || "gpt-5.1-codex-max",
      api: "openai-codex-responses",
      provider: provider || "openai-codex",
      baseUrl: base_url || "",
      reasoning: false,
      input: [],
      cost: { input: 0, output: 0, cacheRead: 0, cacheWrite: 0 },
      contextWindow: 200000,
      maxTokens: 32000,
    };
  }

  const allTools = [...buildBuiltinTools(), ...additionalTools];

  const agent = new Agent({
    streamFn: streamSimple,
    transport: "http",
    getApiKey: async (providerName) => {
      if (api_key) return api_key;
      return undefined;
    },
    initialState: {
      model,
      tools: allTools,
    },
  });

  return { agent, model, tools: allTools };
}

// ---- Per-agent active sequence tracking for fetch logging ----
const activeSeqs = {}; // agentName -> current seq string

// ---- Fetch monkey-patch with per-agent logging ----
const _fetch = globalThis.fetch;
globalThis.fetch = async (url, opts) => {
  const start = Date.now();
  const method = opts?.method || "GET";
  const activeSeq = activeSeqs[leaderName] || Object.values(activeSeqs).find(s => s);

  if (activeSeq) {
    const bodyStr = opts?.body;
    let parsedBody = "";
    try { parsedBody = bodyStr ? JSON.parse(typeof bodyStr === "string" ? bodyStr : "[stream]") : ""; } catch { parsedBody = typeof bodyStr === "string" ? bodyStr.substring(0, 2000) : "[stream]"; }
    for (const [name, seq] of Object.entries(activeSeqs)) {
      if (seq) rootLogger.logRequest(seq, { method, url, body: parsedBody, agent: name });
    }
  }

  console.log(`[API] --> ${method} ${url} body=${typeof opts?.body === "string" ? opts.body.substring(0, 500) : "[stream]"}`);

  try {
    const resp = await _fetch(url, opts);
    const clone = resp.clone();
    const text = await clone.text();
    const elapsed = Date.now() - start;

    if (activeSeq) {
      const isStream = resp.headers.get("content-type")?.includes("text/event-stream");
      if (isStream) {
        const lines = text.split("\n").filter(Boolean);
        for (const line of lines) {
          if (line.startsWith("data: ")) {
            try { const data = JSON.parse(line.slice(6)); for (const seq of Object.values(activeSeqs).filter(Boolean)) { rootLogger.appendResponseLine(seq, data); } } catch { /* raw */ }
          }
        }
      }
      for (const seq of Object.values(activeSeqs).filter(Boolean)) {
        rootLogger.appendResponseLine(seq, { type: "http_response", status: resp.status, elapsed_ms: elapsed, content_type: resp.headers.get("content-type") });
      }
    }

    console.log(`[API] <-- ${resp.status} ${elapsed}ms body=${text.substring(0, 2000)}`);
    return resp;
  } catch (e) {
    for (const seq of Object.values(activeSeqs).filter(Boolean)) {
      rootLogger.appendResponseLine(seq, { type: "fetch_error", message: e.message });
    }
    console.log(`[API] <-- ERROR ${Date.now() - start}ms ${e.message}`);
    throw e;
  }
};

// ---- Load team manifest ----

let manifest;
const manifestPath = resolve("/app/team-manifest.json");
try {
  manifest = JSON.parse(readFileSync(manifestPath, "utf-8"));
  console.log(`Loaded team manifest: ${manifest.team_name} with ${manifest.members.length} members`);
} catch (e) {
  console.error(`Failed to load team manifest from ${manifestPath}:`, e.message);
  process.exit(1);
}

// ---- Initialize all agents ----

const agents = {};       // name -> Agent instance
const memberConfigs = {}; // name -> member config
const memberModels = {};  // name -> model info
const memberTools = {};   // name -> tools array
let leaderName = null;

for (const member of manifest.members) {
  const memberConfig = {
    provider: member.provider || process.env.AGENT_PROVIDER || "openai-codex",
    model_id: member.model_id || process.env.AGENT_MODEL || "gpt-5.1-codex-max",
    api_key: member.api_key || process.env.AGENT_API_KEY || process.env.OPENAI_API_KEY || "",
    base_url: member.base_url || process.env.AGENT_BASE_URL || "",
    system_prompt_override: member.system_prompt_override || "",
  };

  memberConfigs[member.name] = memberConfig;

  console.log(`Setting up agent: ${member.name} (${member.role}) with ${memberConfig.provider}/${memberConfig.model_id}`);
}

// Leader is set up lazily with delegate_task tool (needs workers initialized first)
// Workers are set up eagerly

function getDelegateTool() {
  const workerNames = manifest.members
    .filter((m) => m.role === "worker")
    .map((m) => m.name);

  return {
    name: "delegate_task",
    description: `Delegate a task to a specialized worker agent. Available workers: ${workerNames.join(", ")}. Each worker has its own tools, skills, and context. Use this for complex multi-step tasks that require specialized expertise.`,
    parameters: {
      type: "object",
      properties: {
        worker: {
          type: "string",
          description: `Name of the worker agent to delegate to. One of: ${workerNames.join(", ")}`,
        },
        task: {
          type: "string",
          description: "Full task description and instructions for the worker agent.",
        },
      },
      required: ["worker", "task"],
    },
    execute: async ({ worker, task }) => {
      if (!workerNames.includes(worker)) {
        return {
          content: [{ type: "text", text: `Error: Unknown worker "${worker}". Available: ${workerNames.join(", ")}` }],
          isError: true,
        };
      }

      try {
        const result = await delegateToWorker(worker, task);
        return {
          content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
        };
      } catch (err) {
        return {
          content: [{ type: "text", text: `Error delegating to worker "${worker}": ${err.message}` }],
          isError: true,
        };
      }
    },
  };
}

// ---- Worker delegation (internal HTTP call) ----

async function delegateToWorker(workerName, task) {
  return new Promise((resolve, reject) => {
    const workerAgent = agents[workerName];
    if (!workerAgent) {
      return reject(new Error(`Worker agent "${workerName}" is not initialized`));
    }

    const messages = [];
    let completed = false;

    const sub = workerAgent.subscribe((event) => {
      messages.push(event);
      if (event.type === "agent_end" || event.type === "error") {
        completed = true;
        sub.unsubscribe();
      }
    });

    workerAgent.prompt(task)
      .then(() => {
        if (!completed) {
          messages.push({ type: "agent_end", stopReason: "end_turn" });
        }
      })
      .catch((err) => {
        messages.push({ type: "error", message: err.message });
      })
      .finally(() => {
        // Wait a tick for events to flush
        setTimeout(() => {
          sub.unsubscribe();
          const textMessages = messages
            .filter((m) => m.type === "text_delta" || m.type === "tool_call" || m.type === "tool_result")
            .map((m) => {
              if (m.type === "text_delta") return { type: "text", text: m.text || m.delta || "" };
              if (m.type === "tool_call") return { type: "tool_call", name: m.name, args: m.arguments };
              if (m.type === "tool_result") return { type: "tool_result", content: m.content };
              return m;
            });
          resolve({
            worker: workerName,
            task,
            messages: textMessages,
          });
        }, 100);
      });
  });
}

// ---- Express app ----

const app = express();
app.use(express.json({ limit: "10mb" }));

// Lazy leader init
let leaderAgent = null;
function getLeaderAgent() {
  if (!leaderAgent) {
    for (const member of manifest.members) {
      if (member.role === "leader") {
        leaderName = member.name;
        break;
      }
    }
    if (!leaderName) {
      leaderName = manifest.members[0]?.name;
    }

    const mcpTools = mcpToolsCache[leaderName] || [];
    const result = setupAgent(memberConfigs[leaderName], [getDelegateTool(), ...mcpTools]);
    leaderAgent = result.agent;
    agents[leaderName] = result.agent;
    memberModels[leaderName] = result.model;
    memberTools[leaderName] = result.tools;
    console.log(`Leader agent "${leaderName}" initialized with ${mcpTools.length} MCP tools`);
  }
  return leaderAgent;
}

  const discoveryPromises = manifest.members.map(m => startMcpDiscoveryForMember(m.name));
  Promise.allSettled(discoveryPromises).then(() => {
    // Initialize workers after MCP discovery completes
    for (const member of manifest.members) {
      if (member.role !== "leader") {
        const mcpTools = mcpToolsCache[member.name] || [];
        const result = setupAgent(memberConfigs[member.name], mcpTools);
        agents[member.name] = result.agent;
        memberModels[member.name] = result.model;
        memberTools[member.name] = result.tools;
        console.log(`Worker agent "${member.name}" initialized with ${mcpTools.length} MCP tools`);

        // Write per-agent snapshots on startup
        rootLogger.newSession(member.name);
        rootLogger.writeToolsSnapshot(result.tools);
        const agentSkillsDir = join("/app/agents", member.name, "pi-skills");
        const agentPromptsDir = join("/app/agents", member.name, "pi-prompts");
        rootLogger.writeSkillsSnapshot(agentSkillsDir);
        rootLogger.writePromptsSnapshot(agentPromptsDir);
      }
    }
  });
// Also write team-level snapshots
rootLogger.newSession("team");
const teamSkillsDir = process.env.SKILLS_DIR || "/app/pi-skills";
const teamPromptsDir = process.env.PROMPTS_DIR || "/app/pi-prompts";
rootLogger.writeSkillsSnapshot(teamSkillsDir);
rootLogger.writePromptsSnapshot(teamPromptsDir);

app.get("/status", (req, res) => {
  const statuses = manifest.members.map((m) => ({
    name: m.name,
    role: m.role,
    ready: !!agents[m.name],
  }));
  res.json({
    status: "running",
    team_name: manifest.team_name,
    members: statuses,
  });
});

app.get("/health", (req, res) => {
  res.json({ status: "ok" });
});

app.post("/chat", async (req, res) => {
  const { message } = req.body;
  if (!message) {
    return res.status(400).json({ error: "message required" });
  }

  const leader = getLeaderAgent();

  // Init LLM logging session for leader
  rootLogger.newSession(leaderName);
  if (memberTools[leaderName]) rootLogger.writeToolsSnapshot(memberTools[leaderName]);
  const lSkillsDir = join("/app/agents", leaderName, "pi-skills");
  const lPromptsDir = join("/app/agents", leaderName, "pi-prompts");
  rootLogger.writeSkillsSnapshot(lSkillsDir);
  rootLogger.writePromptsSnapshot(lPromptsDir);
  activeSeqs[leaderName] = rootLogger.nextSeq();

  res.setHeader("Content-Type", "text/event-stream");
  res.setHeader("Cache-Control", "no-cache");
  res.setHeader("Connection", "keep-alive");

  function sendEvent(event, data) {
    res.write(`event: ${event}\ndata: ${JSON.stringify(data)}\n\n`);
  }

  try {
    leader.subscribe((event) => {
      sendEvent(event.type, event);
    });

    await leader.prompt(message);
    sendEvent("agent_end", { stopReason: "end_turn" });
    if (activeSeqs[leaderName]) rootLogger.appendResponseLine(activeSeqs[leaderName], { type: "agent_end", stopReason: "end_turn" });
  } catch (err) {
    sendEvent("error", { message: err.message });
    if (activeSeqs[leaderName]) rootLogger.appendResponseLine(activeSeqs[leaderName], { type: "fatal_error", message: err.message, stack: err.stack });
  } finally {
    activeSeqs[leaderName] = null;
    res.end();
  }
});

app.post("/abort", (req, res) => {
  if (leaderAgent) {
    leaderAgent.abort();
    res.json({ status: "aborted" });
  } else {
    res.json({ status: "no active agent" });
  }
});

// Internal endpoint for worker execution
app.post("/internal/agents/:name/chat", async (req, res) => {
  const { name } = req.params;
  const { message } = req.body;

  if (!message) {
    return res.status(400).json({ error: "message required" });
  }

  const workerAgent = agents[name];
  if (!workerAgent) {
    return res.status(404).json({ error: `worker "${name}" not found` });
  }

  // Track worker execution in its own log subdirectory
  rootLogger.newSession(name);
  if (memberTools[name]) rootLogger.writeToolsSnapshot(memberTools[name]);
  activeSeqs[name] = rootLogger.nextSeq();

  try {
    const result = await delegateToWorker(name, message);
    if (activeSeqs[name]) rootLogger.appendResponseLine(activeSeqs[name], { type: "worker_end", result });
    res.json(result);
  } catch (err) {
    if (activeSeqs[name]) rootLogger.appendResponseLine(activeSeqs[name], { type: "worker_error", message: err.message });
    res.status(500).json({ error: err.message });
  } finally {
    activeSeqs[name] = null;
  }
});

// Internal status for workers
app.get("/internal/agents/:name/status", (req, res) => {
  const { name } = req.params;
  const workerAgent = agents[name];
  res.json({
    name,
    ready: !!workerAgent,
  });
});

const port = process.env.PORT || 3000;
app.listen(port, () => {
  console.log(`Team agent wrapper listening on port ${port}`);
  console.log(`Workspace: ${workspaceDir}`);
  console.log(`Team: ${manifest.team_name}`);
  console.log(`Logs: ${logsDir}/${instanceId}`);
});