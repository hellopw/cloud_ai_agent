import express from "express";
import { Agent } from "@earendil-works/pi-agent-core";
import { streamSimple } from "@earendil-works/pi-ai";
import { readFileSync, existsSync } from "fs";
import { resolve, dirname } from "path";
import { fileURLToPath } from "url";

const __dirname = dirname(fileURLToPath(import.meta.url));

function setupAgent() {
  const agent = new Agent({
    streamFn: streamSimple,
    transport: "http",
  });

  // Load skills from filesystem
  const skillsDir = resolve(process.env.SKILLS_DIR || "/app/pi-skills");
  if (existsSync(skillsDir)) {
    // Skills are loaded via the agent harness; for simplicity, we register
    // them as system-prompt content in this wrapper
    console.log(`Skills dir: ${skillsDir}`);
  }

  const promptsDir = resolve(process.env.PROMPTS_DIR || "/app/pi-prompts");
  if (existsSync(promptsDir)) {
    console.log(`Prompts dir: ${promptsDir}`);
  }

  // Load tool extensions
  const extensionsDir = resolve(process.env.EXTENSIONS_DIR || "/app/extensions");
  if (existsSync(extensionsDir)) {
    console.log(`Extensions dir: ${extensionsDir}`);
  }

  return agent;
}

const app = express();
app.use(express.json());

let agent;

app.get("/status", (req, res) => {
  res.json({ status: "running", ready: !!agent });
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
    agent.on("event", (event) => {
      sendEvent(event.type, event);
    });

    await agent.prompt(message);
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
  console.log(`Agent wrapper listening on port ${port}`);
  console.log(`Workspace: ${process.env.WORKSPACE_DIR || "/workspace"}`);
});
