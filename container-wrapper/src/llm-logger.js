import { mkdirSync, appendFileSync, writeFileSync, readdirSync, readFileSync, statSync, existsSync } from "fs";
import { join, basename } from "path";

// Sensitive keys to redact from logs
const SENSITIVE_KEYS = new Set([
  "api_key", "apiKey", "api-key",
  "x-api-key", "x-api-key", "X-Api-Key",
  "authorization", "Authorization",
]);

const REDACTED = "***REDACTED***";

export function sanitize(obj) {
  if (obj === null || obj === undefined) return obj;
  if (typeof obj === "string") {
    // Redact URL query params that look like API keys
    return obj.replace(/([?&](api_key|apiKey|key|token|secret)=)[^&\s]+/gi, "$1" + REDACTED);
  }
  if (Array.isArray(obj)) return obj.map(sanitize);
  if (typeof obj === "object") {
    const cleaned = {};
    for (const [key, value] of Object.entries(obj)) {
      if (SENSITIVE_KEYS.has(key)) {
        cleaned[key] = REDACTED;
      } else if (key.toLowerCase().includes("api_key") || key.toLowerCase() === "apikey") {
        cleaned[key] = REDACTED;
      } else {
        cleaned[key] = sanitize(value);
      }
    }
    return cleaned;
  }
  return obj;
}

export function createLLMLogger(logsDir, instanceId) {
  let seq = 0;
  let sessionDir;

  function ensureDir() {
    if (!sessionDir) {
      sessionDir = join(logsDir, instanceId);
      mkdirSync(sessionDir, { recursive: true });
    }
  }

  function newSession(subDir) {
    sessionDir = join(logsDir, instanceId, subDir || "");
    mkdirSync(sessionDir, { recursive: true });
    seq = 0;
  }

  function nextSeq() {
    seq++;
    return String(seq).padStart(4, "0");
  }

  function logRequest(seqStr, data) {
    ensureDir();
    const path = join(sessionDir, `request-${seqStr}.jsonl`);
    const sanitized = sanitize(data);
    const line = JSON.stringify(sanitized) + "\n";
    writeFileSync(path, line);
  }

  function appendResponseLine(seqStr, event) {
    ensureDir();
    const path = join(sessionDir, `response-${seqStr}.jsonl`);
    const eventWithTs = { ts: new Date().toISOString(), ...sanitize(event) };
    appendFileSync(path, JSON.stringify(eventWithTs) + "\n");
  }

  function writeSnapshot(filename, data) {
    ensureDir();
    const path = join(sessionDir, filename);
    writeFileSync(path, JSON.stringify(sanitize(data), null, 2));
  }

  function writeToolsSnapshot(tools) {
    const summary = tools.map((t) => ({
      name: t.name,
      description: t.description,
      parameters: t.parameters || t.input_schema,
    }));
    writeSnapshot("tools.json", summary);
  }

  function writePromptsSnapshot(promptsDir) {
    if (!promptsDir || !existsSync(promptsDir)) return;
    try {
      const files = readdirSync(promptsDir).filter((f) => f.endsWith(".md"));
      const prompts = files.map((f) => {
        const content = readFileSync(join(promptsDir, f), "utf-8");
        return { file: f, content };
      });
      writeSnapshot("prompts.json", { dir: promptsDir, files: prompts });
    } catch (e) {
      // prompts dir is optional, silently skip
    }
  }

  function writeSkillsSnapshot(skillsDir) {
    if (!skillsDir || !existsSync(skillsDir)) return;
    try {
      const files = readdirSync(skillsDir).filter((f) => f.endsWith(".md"));
      const skills = files.map((f) => {
        const content = readFileSync(join(skillsDir, f), "utf-8");
        return { file: f, content };
      });
      writeSnapshot("skills.json", { dir: skillsDir, files: skills });
    } catch (e) {
      // skills dir is optional, silently skip
    }
  }

  // For team mode: pass subDir for per-agent logs
  function subLogger(name) {
    const sub = join(sessionDir || join(logsDir, instanceId), name);
    return createLLMLogger(logsDir, join(instanceId, name));
  }

  return {
    newSession,
    nextSeq,
    logRequest,
    appendResponseLine,
    writeToolsSnapshot,
    writePromptsSnapshot,
    writeSkillsSnapshot,
    subLogger,
  };
}
