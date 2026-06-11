import express from "express";
import { ClaudeCodeAgent } from "@anthropic-ai/claude-code-sdk";
import { readFileSync, existsSync, mkdirSync } from "fs";
import { resolve, dirname } from "path";
import { fileURLToPath } from "url";
import { execSync } from "child_process";

const __dirname = dirname(fileURLToPath(import.meta.url));
const workspaceDir = process.env.WORKSPACE_DIR || "/workspace";
mkdirSync(workspaceDir, { recursive: true });

function setupAgent() {
  const apiKey = process.env.AGENT_API_KEY || process.env.ANTHROPIC_API_KEY || "";
  const modelId = process.env.AGENT_MODEL || "claude-sonnet-4-20250514";

  console.log(`Claude Code agent configured with model=${modelId}`);

  const agent = new ClaudeCodeAgent({
    apiKey,
    model: modelId,
    cwd: workspaceDir,
    maxTurns: 50,
  });

  return agent;
}

const app = express();
app.use(express.json({ limit: "10mb" }));

let agent;

app.get("/status", (req, res) => {
  res.json({ status: "running", ready: !!agent, provider: "anthropic", model: process.env.AGENT_MODEL || "claude-sonnet-4-20250514" });
});

app.get("/health", (req, res) => {
  res.json({ status: "ok" });
});

app.post("/chat", async (req, res) => {
  const { message } = req.body;
  if (!message) {
    return res.status(400).json({ error: "message required" });
  }

  if (!agent) {
    agent = setupAgent();
  }

  res.setHeader("Content-Type", "text/event-stream");
  res.setHeader("Cache-Control", "no-cache");
  res.setHeader("Connection", "keep-alive");

  function sendEvent(event, data) {
    res.write(`event: ${event}\ndata: ${JSON.stringify(data)}\n\n`);
  }

  try {
    const stream = agent.stream(message);
    for await (const event of stream) {
      sendEvent(event.type, event);
    }
    sendEvent("agent_end", { stopReason: "end_turn" });
  } catch (err) {
    sendEvent("error", { message: err.message });
  } finally {
    res.end();
  }
});

app.post("/abort", (req, res) => {
  if (agent) {
    agent.abort();
    res.json({ status: "aborted" });
  } else {
    res.json({ status: "no active agent" });
  }
});

const port = process.env.PORT || 3000;
app.listen(port, () => {
  console.log(`Claude Code wrapper listening on port ${port}`);
  console.log(`Workspace: ${workspaceDir}`);
});