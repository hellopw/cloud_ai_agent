import { readdirSync, readFileSync, existsSync } from "fs";
import { join } from "path";

/**
 * Parse YAML frontmatter from a markdown string.
 * Returns { frontmatter: Record<string,string>, body: string }
 */
function parseFrontmatter(content) {
  const match = content.match(/^---\s*\n([\s\S]*?)\n---\s*\n([\s\S]*)$/);
  if (!match) {
    return { frontmatter: {}, body: content };
  }
  const frontmatter = {};
  const yamlBlock = match[1];
  for (const line of yamlBlock.split("\n")) {
    const kv = line.match(/^(\w[\w_-]*):\s*(.*)$/);
    if (kv) {
      frontmatter[kv[1]] = kv[2].trim();
    }
  }
  return { frontmatter, body: match[2] };
}

/**
 * Load all .md prompt files from a directory.
 * Returns an array of { name, description, content, text }.
 */
export function loadPrompts(promptsDir) {
  if (!promptsDir || !existsSync(promptsDir)) return [];

  try {
    const files = readdirSync(promptsDir).filter((f) => f.endsWith(".md"));
    return files.map((f) => {
      const raw = readFileSync(join(promptsDir, f), "utf-8");
      const { frontmatter, body } = parseFrontmatter(raw);
      return {
        name: f.replace(/\.md$/, ""),
        description: frontmatter.description || "",
        content: body.trim(),
        text: frontmatter.description
          ? `## ${frontmatter.description}\n\n${body.trim()}`
          : body.trim(),
      };
    });
  } catch {
    return [];
  }
}

/**
 * Load prompts and return as system content blocks array for Anthropic API.
 * Each prompt becomes a separate { type: "text", text: "..." } block.
 * Prepends a base system text if provided.
 */
export function loadPromptsAsSystemBlocks(promptsDir, baseText) {
  const blocks = [];
  if (baseText) {
    blocks.push({ type: "text", text: baseText });
  }
  const prompts = loadPrompts(promptsDir);
  for (const p of prompts) {
    blocks.push({ type: "text", text: p.text });
  }
  return blocks;
}

/**
 * Load prompts and return as a single string.
 * Joins prompt texts with double newlines.
 */
export function loadPromptsAsString(promptsDir, baseText) {
  const parts = [];
  if (baseText) {
    parts.push(baseText);
  }
  const prompts = loadPrompts(promptsDir);
  for (const p of prompts) {
    parts.push(p.text);
  }
  return parts.join("\n\n");
}
