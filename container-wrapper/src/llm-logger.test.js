import { describe, it, before, after, beforeEach } from "node:test";
import assert from "node:assert";
import { rmSync, existsSync, mkdirSync, readFileSync, writeFileSync } from "fs";
import { join } from "path";
import { createLLMLogger, sanitize } from "./llm-logger.js";

const TEST_ROOT = join(import.meta.dirname || ".", "..", "test-tmp");

function cleanUp() {
  if (existsSync(TEST_ROOT)) {
    rmSync(TEST_ROOT, { recursive: true, force: true });
  }
}

function readLines(filepath) {
  const content = readFileSync(filepath, "utf-8").trim();
  if (!content) return [];
  return content.split("\n").map((line) => JSON.parse(line));
}

describe("sanitize", () => {
  it("redacts api_key field", () => {
    const result = sanitize({ api_key: "sk-1234567890abcdef" });
    assert.strictEqual(result.api_key, "***REDACTED***");
  });

  it("redacts apiKey (camelCase)", () => {
    const result = sanitize({ apiKey: "sk-secret" });
    assert.strictEqual(result.apiKey, "***REDACTED***");
  });

  it("redacts x-api-key header", () => {
    const result = sanitize({ "x-api-key": "secret-token" });
    assert.strictEqual(result["x-api-key"], "***REDACTED***");
  });

  it("redacts authorization header", () => {
    const result = sanitize({ authorization: "Bearer sk-abc123" });
    assert.strictEqual(result.authorization, "***REDACTED***");
  });

  it("redacts Authorization (capitalized)", () => {
    const result = sanitize({ Authorization: "Bearer sk-abc123" });
    assert.strictEqual(result.Authorization, "***REDACTED***");
  });

  it("redacts nested sensitive keys", () => {
    const result = sanitize({ config: { api_key: "nested-secret" } });
    assert.strictEqual(result.config.api_key, "***REDACTED***");
  });

  it("redacts api_key in array of objects", () => {
    const result = sanitize([{ api_key: "a" }, { api_key: "b" }]);
    assert.strictEqual(result[0].api_key, "***REDACTED***");
    assert.strictEqual(result[1].api_key, "***REDACTED***");
  });

  it("preserves non-sensitive fields", () => {
    const result = sanitize({ model: "claude-sonnet", max_tokens: 16000 });
    assert.strictEqual(result.model, "claude-sonnet");
    assert.strictEqual(result.max_tokens, 16000);
  });

  it("redacts api_key in URL query string", () => {
    const result = sanitize("https://api.example.com?api_key=mysecret&foo=bar");
    assert(result.includes("***REDACTED***"));
    assert(!result.includes("mysecret"));
  });

  it("handles null gracefully", () => {
    assert.strictEqual(sanitize(null), null);
  });

  it("handles undefined gracefully", () => {
    assert.strictEqual(sanitize(undefined), undefined);
  });

  it("handles non-object types", () => {
    assert.strictEqual(sanitize(42), 42);
    assert.strictEqual(sanitize(true), true);
    assert.strictEqual(sanitize("hello"), "hello");
  });
});

describe("createLLMLogger", () => {
  const INSTANCE_ID = "inst-test-001";
  let logger;

  before(() => {
    cleanUp();
    mkdirSync(TEST_ROOT, { recursive: true });
    logger = createLLMLogger(TEST_ROOT, INSTANCE_ID);
  });

  after(() => cleanUp());

  beforeEach(() => {
    const dir = join(TEST_ROOT, INSTANCE_ID);
    if (existsSync(dir)) rmSync(dir, { recursive: true, force: true });
    logger.newSession();
  });

  it("newSession creates directory", () => {
    const dir = join(TEST_ROOT, INSTANCE_ID);
    assert(existsSync(dir));
  });

  it("nextSeq resets after newSession", () => {
    const s1 = logger.nextSeq();
    const s2 = logger.nextSeq();
    assert.strictEqual(s1, "0001");
    assert.strictEqual(s2, "0002");

    logger.newSession();
    const s3 = logger.nextSeq();
    assert.strictEqual(s3, "0001");
  });

  it("nextSeq pads to 4 digits", () => {
    logger.newSession();
    for (let i = 0; i < 99; i++) logger.nextSeq();
    assert.strictEqual(logger.nextSeq(), "0100");
  });

  it("logRequest writes a file with one JSONL line", () => {
    const seq = logger.nextSeq();
    logger.logRequest(seq, { model: "claude-sonnet", messages: [{ role: "user", content: "hello" }] });

    const file = join(TEST_ROOT, INSTANCE_ID, `request-${seq}.jsonl`);
    assert(existsSync(file));
    const lines = readLines(file);
    assert.strictEqual(lines.length, 1);
    assert.strictEqual(lines[0].model, "claude-sonnet");
    assert.deepStrictEqual(lines[0].messages, [{ role: "user", content: "hello" }]);
  });

  it("logRequest sanitizes sensitive data", () => {
    const seq = logger.nextSeq();
    logger.logRequest(seq, { api_key: "sk-secret", model: "claude" });

    const file = join(TEST_ROOT, INSTANCE_ID, `request-${seq}.jsonl`);
    const lines = readLines(file);
    assert.strictEqual(lines[0].api_key, "***REDACTED***");
    assert.strictEqual(lines[0].model, "claude");
  });

  it("appendResponseLine creates file and appends multiple lines", () => {
    const seq = logger.nextSeq();
    logger.appendResponseLine(seq, { type: "text_delta", delta: "Hello" });
    logger.appendResponseLine(seq, { type: "text_delta", delta: " world" });
    logger.appendResponseLine(seq, { type: "message_stop", stop_reason: "end_turn" });

    const file = join(TEST_ROOT, INSTANCE_ID, `response-${seq}.jsonl`);
    assert(existsSync(file));

    const lines = readLines(file);
    assert.strictEqual(lines.length, 3);
    assert.strictEqual(lines[0].type, "text_delta");
    assert.strictEqual(lines[0].delta, "Hello");
    assert.strictEqual(lines[1].type, "text_delta");
    assert.strictEqual(lines[1].delta, " world");
    assert.strictEqual(lines[2].type, "message_stop");
    assert.strictEqual(lines[2].stop_reason, "end_turn");
  });

  it("appendResponseLine adds timestamp automatically", () => {
    const seq = logger.nextSeq();
    logger.appendResponseLine(seq, { type: "test" });

    const file = join(TEST_ROOT, INSTANCE_ID, `response-${seq}.jsonl`);
    const lines = readLines(file);
    assert(lines[0].ts);
    assert(lines[0].ts.endsWith("Z") || lines[0].ts.includes("+"));
  });

  it("writeToolsSnapshot creates tools.json", () => {
    const tools = [
      { name: "bash", description: "Run commands", parameters: { type: "object" } },
      { name: "read_file", description: "Read file", input_schema: { type: "object" } },
    ];
    logger.writeToolsSnapshot(tools);

    const file = join(TEST_ROOT, INSTANCE_ID, "tools.json");
    assert(existsSync(file));
    const data = JSON.parse(readFileSync(file, "utf-8"));
    assert.strictEqual(data.length, 2);
    assert.strictEqual(data[0].name, "bash");
    assert.strictEqual(data[1].name, "read_file");
  });

  it("writePromptsSnapshot writes prompts.json when dir exists", () => {
    const promptsDir = join(TEST_ROOT, "test-prompts");
    mkdirSync(promptsDir, { recursive: true });
    writeFileSync(join(promptsDir, "code-review.md"), "# Code Review\nReview code carefully.");

    logger.writePromptsSnapshot(promptsDir);

    const file = join(TEST_ROOT, INSTANCE_ID, "prompts.json");
    assert(existsSync(file));
    const data = JSON.parse(readFileSync(file, "utf-8"));
    assert(data.files.length > 0);
    assert.strictEqual(data.files[0].file, "code-review.md");
    assert(data.files[0].content.includes("Code Review"));
  });

  it("writePromptsSnapshot does nothing when dir is null or missing", () => {
    logger.writePromptsSnapshot(null);
    logger.writePromptsSnapshot("/nonexistent/dir");

    const file = join(TEST_ROOT, INSTANCE_ID, "prompts.json");
    assert(!existsSync(file));
  });

  it("writeSkillsSnapshot writes skills.json when dir exists", () => {
    const skillsDir = join(TEST_ROOT, "test-skills");
    mkdirSync(skillsDir, { recursive: true });
    writeFileSync(join(skillsDir, "debugger.md"), "# Debugger\nDebug code.");

    logger.writeSkillsSnapshot(skillsDir);

    const file = join(TEST_ROOT, INSTANCE_ID, "skills.json");
    assert(existsSync(file));
    const data = JSON.parse(readFileSync(file, "utf-8"));
    assert(data.files.length > 0);
    assert.strictEqual(data.files[0].file, "debugger.md");
  });

  it("handles empty tools array", () => {
    logger.writeToolsSnapshot([]);
    const file = join(TEST_ROOT, INSTANCE_ID, "tools.json");
    const data = JSON.parse(readFileSync(file, "utf-8"));
    assert.strictEqual(data.length, 0);
  });

  it("logRequest handles null data", () => {
    const seq = logger.nextSeq();
    logger.logRequest(seq, null);
    const file = join(TEST_ROOT, INSTANCE_ID, `request-${seq}.jsonl`);
    assert(existsSync(file));
    const lines = readLines(file);
    assert.strictEqual(lines[0], null);
  });

  it("auto-creates parent directories", () => {
    const deepLogger = createLLMLogger(join(TEST_ROOT, "a", "b", "c"), "deep-inst");
    deepLogger.newSession();
    const dir = join(TEST_ROOT, "a", "b", "c", "deep-inst");
    assert(existsSync(dir));
  });
});
