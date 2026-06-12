import express from "express";
import OpenAI from "openai";
import { existsSync, mkdirSync } from "fs";
import { resolve, dirname } from "path";
import { fileURLToPath } from "url";
import { createLLMLogger } from "./llm-logger.js";

const __dirname = dirname(fileURLToPath(import.meta.url));
const workspaceDir = process.env.WORKSPACE_DIR || "/workspace";
mkdirSync(workspaceDir, { recursive: true });

// LLM logger setup
const logsDir = process.env.LOGS_DIR || "/logs";
const instanceId = process.env.INSTANCE_ID || "unknown";
const logger = createLLMLogger(logsDir, instanceId);

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

  // Init logging session and write snapshots
  logger.newSession();
  const skillsDir = process.env.SKILLS_DIR || "/app/skills";
  const promptsDir = process.env.PROMPTS_DIR || "/app/prompts";
  logger.writeSkillsSnapshot(skillsDir);
  logger.writePromptsSnapshot(promptsDir);

  res.setHeader("Content-Type", "text/event-stream");
  res.setHeader("Cache-Control", "no-cache");
  res.setHeader("Connection", "keep-alive");

  function sendEvent(event, data) {
    res.write(`event: ${event}\ndata: ${JSON.stringify(data)}\n\n`);
  }

  const baseURL = process.env.AGENT_BASE_URL || "";

  try {
    const seq = logger.nextSeq();
    logger.logRequest(seq, {
      provider: "openai",
      model: agentState.modelId,
      baseUrl: baseURL || "https://api.openai.com",
      input: message,
      stream: true,
    });

    const stream = await agentState.client.responses.create({
      model: agentState.modelId,
      input: message,
      stream: true,
    });

    for await (const event of stream) {
      sendEvent("response", event);
      logger.appendResponseLine(seq, event);
    }
    sendEvent("agent_end", { stopReason: "end_turn" });
    logger.appendResponseLine(seq, { type: "agent_end", stopReason: "end_turn" });
  } catch (err) {
    sendEvent("error", { message: err.message });
    logger.appendResponseLine("error", { type: "fatal_error", message: err.message, stack: err.stack });
  } finally {
    res.end();
  }
});

app.post("/abort", (req, res) => {
  res.json({ status: "aborted" });
});

// Write initial snapshots on startup
const startupSkillsDir = process.env.SKILLS_DIR || "/app/skills";
const startupPromptsDir = process.env.PROMPTS_DIR || "/app/prompts";
if (startupSkillsDir || startupPromptsDir) {
  logger.newSession();
  logger.writeSkillsSnapshot(startupSkillsDir);
  logger.writePromptsSnapshot(startupPromptsDir);
}

const port = process.env.PORT || 3000;
app.listen(port, () => {
  console.log(`Codex wrapper listening on port ${port}`);
  console.log(`Workspace: ${workspaceDir}`);
  console.log(`Logs: ${logsDir}/${instanceId}`);
});
