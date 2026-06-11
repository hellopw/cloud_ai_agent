import express from "express";
import OpenAI from "openai";
import { readFileSync, existsSync, mkdirSync } from "fs";
import { resolve, dirname } from "path";
import { fileURLToPath } from "url";
import { execSync } from "child_process";

const __dirname = dirname(fileURLToPath(import.meta.url));
const workspaceDir = process.env.WORKSPACE_DIR || "/workspace";
mkdirSync(workspaceDir, { recursive: true });

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

  res.setHeader("Content-Type", "text/event-stream");
  res.setHeader("Cache-Control", "no-cache");
  res.setHeader("Connection", "keep-alive");

  function sendEvent(event, data) {
    res.write(`event: ${event}\ndata: ${JSON.stringify(data)}\n\n`);
  }

  try {
    const stream = await agentState.client.responses.create({
      model: agentState.modelId,
      input: message,
      stream: true,
    });

    for await (const event of stream) {
      sendEvent("response", event);
    }
    sendEvent("agent_end", { stopReason: "end_turn" });
  } catch (err) {
    sendEvent("error", { message: err.message });
  } finally {
    res.end();
  }
});

app.post("/abort", (req, res) => {
  res.json({ status: "aborted" });
});

const port = process.env.PORT || 3000;
app.listen(port, () => {
  console.log(`Codex wrapper listening on port ${port}`);
  console.log(`Workspace: ${workspaceDir}`);
});