import express from "express";
import { Agent } from "@earendil-works/pi-agent-core";
import { streamSimple } from "@earendil-works/pi-ai";
import { readFileSync, existsSync } from "fs";
import { resolve, dirname } from "path";
import { fileURLToPath } from "url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const conversationHistory = [];

function setupAgent() {
  const agent = new Agent({
    streamFn: streamSimple,
    transport: "http",
  });

  const skillsDir = resolve(process.env.SKILLS_DIR || "/app/pi-skills");
  if (existsSync(skillsDir)) console.log(Skills dir:  + skillsDir);

  const promptsDir = resolve(process.env.PROMPTS_DIR || "/app/pi-prompts");
  if (existsSync(promptsDir)) console.log(Prompts dir:  + promptsDir);

  const extensionsDir = resolve(process.env.EXTENSIONS_DIR || "/app/extensions");
  if (existsSync(extensionsDir)) console.log(Extensions dir:  + extensionsDir);

  const memories = process.env.MEMORIES || "";
  if (memories) console.log(Loaded memories ( + memories.length +  chars));

  return agent;
}

function buildContext(memories) {
  let ctx = "You are a helpful AI assistant.";
  if (memories) {
    ctx += "\n\n## Persistent Memories\nThe following memories provide context:\n" + memories + "\nUse these to inform your responses.";
  }
  return ctx;
}

const app = express();
app.use(express.json());

let agent, contextMessage;

app.get("/status", (req, res) => {
  res.json({ status: "running", ready: !!agent, turns: conversationHistory.length });
});

app.get("/health", (req, res) => {
  res.json({ status: "ok" });
});

app.get("/conversation", (req, res) => {
  res.json({ turns: conversationHistory });
});

app.post("/chat", async (req, res) => {
  const { message } = req.body;
  if (!message) return res.status(400).json({ error: "message required" });

  if (!agent) {
    agent = setupAgent();
    contextMessage = buildContext(process.env.MEMORIES || "");
  }

  res.setHeader("Content-Type", "text/event-stream");
  res.setHeader("Cache-Control", "no-cache");
  res.setHeader("Connection", "keep-alive");

  let assistantContent = "";

  function sendEvent(event, data) {
    res.write("event: " + event + "\ndata: " + JSON.stringify(data) + "\n\n");
  }

  try {
    agent.on("event", (event) => {
      if (event.type === "text_delta") assistantContent += event.data?.delta || "";
      sendEvent(event.type, event);
    });

    await agent.prompt(message);
    conversationHistory.push({ role: "user", content: message, timestamp: new Date().toISOString() });
    if (assistantContent) {
      conversationHistory.push({ role: "assistant", content: assistantContent, timestamp: new Date().toISOString() });
    }
    sendEvent("agent_end", { stopReason: "end_turn" });
  } catch (err) {
    sendEvent("error", { message: err.message });
  } finally {
    res.end();
  }
});

app.post("/abort", (req, res) => {
  if (agent) { agent.abort(); res.json({ status: "aborted" }); }
  else { res.json({ status: "no active agent" }); }
});

const port = process.env.PORT || 3000;
app.listen(port, () => {
  console.log(Agent wrapper listening on port  + port);
  console.log(Workspace:  + (process.env.WORKSPACE_DIR || "/workspace"));
});
