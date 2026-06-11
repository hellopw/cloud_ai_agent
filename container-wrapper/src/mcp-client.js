// Minimal MCP (Model Context Protocol) client for stdio and SSE transports.
// Implements JSON-RPC 2.0 over stdio (child_process) and SSE (fetch + EventSource).

import { spawn } from "child_process";
import { createInterface } from "readline";

let nextId = 1;

// ---- JSON-RPC helpers ----

function createRequest(method, params) {
  return {
    jsonrpc: "2.0",
    id: nextId++,
    method,
    params: params || {},
  };
}

function createNotification(method, params) {
  return {
    jsonrpc: "2.0",
    method,
    params: params || {},
  };
}

// ---- Stdio transport ----

function startStdioServer(command, args = [], env = {}) {
  const proc = spawn(command, args, {
    stdio: ["pipe", "pipe", "pipe"],
    env: { ...process.env, ...env },
  });

  const rl = createInterface({ input: proc.stdout, crlfDelay: Infinity });
  let responseQueue = [];
  let pendingResolve = null;
  let closed = false;

  rl.on("line", (line) => {
    try {
      const msg = JSON.parse(line);
      if (pendingResolve) {
        pendingResolve(msg);
        pendingResolve = null;
      } else {
        responseQueue.push(msg);
      }
    } catch {
      // ignore non-JSON lines (e.g. debug output on stdout)
    }
  });

  proc.on("close", () => { closed = true; });
  proc.stderr.on("data", () => {}); // swallow stderr

  // Send a notification (no id, no response expected)
  async function notify(method, params = {}) {
    if (closed) throw new Error("MCP server process has closed");
    const notif = createNotification(method, params);
    proc.stdin.write(JSON.stringify(notif) + "\n");
  }

  async function send(method, params = {}, timeoutMs = 30000) {
    if (closed) throw new Error("MCP server process has closed");
    const req = createRequest(method, params);
    proc.stdin.write(JSON.stringify(req) + "\n");

    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        pendingResolve = null;
        reject(new Error("MCP request " + method + " timed out after " + timeoutMs + "ms"));
      }, timeoutMs);

      if (responseQueue.length > 0) {
        clearTimeout(timer);
        resolve(responseQueue.shift());
        return;
      }

      pendingResolve = (msg) => {
        clearTimeout(timer);
        resolve(msg);
      };
    });
  }

  async function close() {
    if (!closed) {
      proc.kill();
      setTimeout(() => {
        try { proc.kill("SIGKILL"); } catch (_) {}
      }, 2000);
      closed = true;
    }
  }

  return { send, notify, close };
}

// ---- SSE transport ----

async function startSSEClient(url, timeoutMs = 30000) {
  const initResp = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(createRequest("initialize", {
      protocolVersion: "2024-11-05",
      capabilities: {},
      clientInfo: { name: "cloud-ai-agent", version: "0.1.0" },
    })),
    signal: AbortSignal.timeout(timeoutMs),
  });

  if (!initResp.ok) {
    throw new Error("MCP SSE initialize failed: " + initResp.status);
  }

  const sessionId = initResp.headers.get("mcp-session-id") || "";

  async function notify(method, params = {}) {
    const headers = { "Content-Type": "application/json" };
    if (sessionId) headers["mcp-session-id"] = sessionId;
    // Fire-and-forget notification
    fetch(url, {
      method: "POST",
      headers,
      body: JSON.stringify(createNotification(method, params)),
    }).catch(() => {});
  }

  async function send(method, params = {}, tMs = timeoutMs) {
    const headers = { "Content-Type": "application/json" };
    if (sessionId) headers["mcp-session-id"] = sessionId;

    const resp = await fetch(url, {
      method: "POST",
      headers,
      body: JSON.stringify(createRequest(method, params)),
      signal: AbortSignal.timeout(tMs),
    });

    if (!resp.ok) {
      throw new Error("MCP SSE " + method + " failed: " + resp.status);
    }

    const text = await resp.text();
    try {
      return JSON.parse(text);
    } catch {
      return { result: text };
    }
  }

  return { send, notify, close: () => {} };
}

// ---- Public API ----

export async function callMcpTool(opts) {
  const { transport, command, args = [], env = {}, url, toolName, toolArgs = {}, signal } = opts;

  let client;
  if (transport === "sse") {
    client = await startSSEClient(url);
  } else {
    client = startStdioServer(command, args, env);
  }

  try {
    await client.send("initialize", {
      protocolVersion: "2024-11-05",
      capabilities: {},
      clientInfo: { name: "cloud-ai-agent", version: "0.1.0" },
    });
    await client.notify("notifications/initialized", {});

    const result = await client.send("tools/call", {
      name: toolName,
      arguments: toolArgs,
    });

    await client.close();

    if (result.error) {
      return {
        content: [{ type: "text", text: "MCP error: " + (result.error.message || JSON.stringify(result.error)) }],
        isError: true,
      };
    }

    const content = result.result?.content || result.result;
    if (Array.isArray(content)) {
      return { content };
    }
    return { content: [{ type: "text", text: typeof content === "string" ? content : JSON.stringify(content, null, 2) }] };
  } catch (err) {
    await client.close();
    throw err;
  }
}

export async function listMcpTools(opts) {
  const { transport, command, args = [], env = {}, url, signal } = opts;

  let client;
  if (transport === "sse") {
    client = await startSSEClient(url);
  } else {
    client = startStdioServer(command, args, env);
  }

  try {
    await client.send("initialize", {
      protocolVersion: "2024-11-05",
      capabilities: {},
      clientInfo: { name: "cloud-ai-agent", version: "0.1.0" },
    });
    await client.notify("notifications/initialized", {});

    const result = await client.send("tools/list", {});
    await client.close();

    return result.result?.tools || [];
  } catch (err) {
    await client.close();
    throw err;
  }
}