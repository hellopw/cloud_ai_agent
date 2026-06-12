package store

import (
	"strings"
	"time"
)

// SeedExampleAgents creates example prompts, skills, tools, and templates
// for three agent types: git-code, database-query, and excel.
func (s *Store) SeedExampleAgents() error {
	now := time.Now()

	// ── Tools ────────────────────────────────────────────────
	tools := []struct {
		name        string
		label       string
		description string
		dsl         string
	}{
		{
			name:        "excel",
			label:       "Excel MCP",
			description: "Read, write, and manipulate Excel (.xlsx) files via MCP server. Set EXCEL_FILES_DIR environment variable to the directory containing Excel files.",
			dsl: `{
  "name": "excel",
  "label": "Excel MCP",
  "description": "Read, write, and manipulate Excel (.xlsx) files via MCP server",
  "parameters": {},
  "handler": {
    "type": "mcp",
    "transport": "stdio",
    "command": "npx",
    "args": ["-y", "@earendil-works/mcp-server-excel"],
    "env": {
      "EXCEL_FILES_DIR": "/workspace"
    }
  }
}`,
		},
	}

	for _, t := range tools {
		var exists int
		if err := s.db.QueryRow("SELECT COUNT(*) FROM tools WHERE name = ?", t.name).Scan(&exists); err != nil {
			return err
		}
		if exists > 0 {
			continue
		}
		id := "example-" + t.name
		_, err := s.db.Exec(
			"INSERT OR IGNORE INTO tools (id, name, label, description, dsl_definition, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			id, t.name, t.label, t.description, t.dsl, now, now,
		)
		if err != nil {
			return err
		}
	}

	// Collect tool IDs for binding
	toolIDs := map[string]string{}
	for _, name := range []string{"mysql", "postgres", "excel"} {
		var id string
		s.db.QueryRow("SELECT id FROM tools WHERE name = ?", name).Scan(&id)
		toolIDs[name] = id
	}

	// ── Prompts ──────────────────────────────────────────────
	prompts := []struct {
		name        string
		description string
		content     string
	}{
		{
			name:        "git-code-agent",
			description: "System prompt for a Git code modification agent",
			content: `You are an expert software engineer agent that modifies code in git repositories.

## Core Capabilities
- Read, understand, and modify source code across multiple languages
- Use git commands to branch, commit, and push changes
- Follow project conventions and coding standards found in the repository
- Write clear, concise commit messages

## Workflow
1. When given a task, first explore the repository structure using list_files and search_content
2. Read relevant files to understand the codebase
3. Make precise, minimal edits using edit_file or write_file
4. Verify changes by reading the modified files
5. Stage and commit changes with descriptive messages using git
6. Push changes to the remote repository when asked

## Guidelines
- Always check git status and git diff before committing
- Create feature branches for non-trivial changes
- Never force push or modify protected branches
- Prefer edit_file over write_file for existing files to make surgical changes
- Add comments only when the WHY is non-obvious`,
		},
		{
			name:        "database-query-agent",
			description: "System prompt for a database query agent",
			content: `You are a database query agent that helps users explore and query databases.

## Core Capabilities
- Connect to MySQL and PostgreSQL databases via MCP tools
- Write and execute SQL queries
- Explore database schemas (tables, columns, indexes)
- Analyze query results and provide insights

## Workflow
1. When given a query task, first discover available tools by listing MCP tools
2. Explore the database schema (show tables, describe tables)
3. Write SQL queries to answer the user's question
4. Present results in a clear, readable format (markdown tables)
5. For complex queries, explain the query logic step by step

## Guidelines
- Always start by understanding the schema before writing queries
- Use LIMIT clauses for exploratory queries to avoid large result sets
- Never execute DROP, TRUNCATE, or DELETE without explicit user confirmation
- Explain query performance implications for large tables
- Suggest indexes when you notice slow query patterns`,
		},
		{
			name:        "excel-agent",
			description: "System prompt for an Excel file manipulation agent",
			content: `You are an Excel file manipulation agent that reads, creates, and modifies Excel spreadsheets.

## Core Capabilities
- Read data from existing Excel (.xlsx) files
- Create new workbooks and worksheets
- Write formulas and apply formatting
- Merge data from multiple sources into Excel reports
- Convert between data formats (CSV, JSON, Excel)

## Workflow
1. When given a task, first discover available Excel MCP tools
2. List or read the target Excel file to understand its structure
3. Perform the requested operations (read, write, modify, create)
4. Verify the result by reading back the modified file
5. Summarize what was done

## Guidelines
- Always preserve existing data unless explicitly asked to overwrite
- Use meaningful sheet names
- Apply number formatting for currency, percentages, and dates
- Create summary sheets when working with large datasets
- Backup critical files before making large changes`,
		},
	}

	promptIDs := map[string]string{}
	for _, p := range prompts {
		var exists int
		if err := s.db.QueryRow("SELECT COUNT(*) FROM prompts WHERE name = ?", p.name).Scan(&exists); err != nil {
			return err
		}
		if exists > 0 {
			var id string
			s.db.QueryRow("SELECT id FROM prompts WHERE name = ?", p.name).Scan(&id)
			promptIDs[p.name] = id
			continue
		}
		id := "example-" + p.name
		_, err := s.db.Exec(
			"INSERT OR IGNORE INTO prompts (id, name, description, content, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
			id, p.name, p.description, p.content, now, now,
		)
		if err != nil {
			return err
		}
		promptIDs[p.name] = id
	}

	// ── Skills ───────────────────────────────────────────────
	skills := []struct {
		name        string
		description string
		content     string
	}{
		{
			name:        "code-review-and-fix",
			description: "Review code changes and apply fixes based on review feedback",
			content: `## Code Review and Fix

When asked to review and fix code:

1. Run git diff to see uncommitted changes, or git log to find recent commits
2. For each change, check for:
   - Logic errors and edge cases
   - Security vulnerabilities (injection, XSS, etc.)
   - Performance issues (N+1 queries, unnecessary allocations)
   - Style and convention violations
3. If issues are found, apply fixes using edit_file
4. Commit fixes with messages like "fix: <description of what was fixed>"
5. Summarize all changes made`,
		},
		{
			name:        "schema-explorer",
			description: "Explore and document database schemas",
			content: `## Schema Explorer

When asked to explore a database:

1. List all tables using the appropriate database MCP tool
2. For each table of interest, get the column definitions and types
3. Identify primary keys, foreign keys, and indexes
4. Generate an ER diagram description in markdown
5. Suggest sample queries for common use cases
6. Document any missing indexes or schema improvements`,
		},
		{
			name:        "excel-report-builder",
			description: "Build formatted Excel reports from data",
			content: `## Excel Report Builder

When asked to create an Excel report:

1. Determine the data source (existing file, database query, or raw data)
2. Plan the report structure (sheets, columns, formatting)
3. Create the workbook with appropriate sheet names
4. Write headers with bold formatting
5. Populate data rows
6. Add formulas for totals, averages, or other aggregations
7. Apply number formatting (currency, percentage, date)
8. Adjust column widths to fit content
9. Verify the output by reading back the file`,
		},
	}

	skillIDs := map[string]string{}
	for _, sk := range skills {
		var exists int
		if err := s.db.QueryRow("SELECT COUNT(*) FROM skills WHERE name = ?", sk.name).Scan(&exists); err != nil {
			return err
		}
		if exists > 0 {
			var id string
			s.db.QueryRow("SELECT id FROM skills WHERE name = ?", sk.name).Scan(&id)
			skillIDs[sk.name] = id
			continue
		}
		id := "example-" + sk.name
		_, err := s.db.Exec(
			"INSERT OR IGNORE INTO skills (id, name, description, content, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
			id, sk.name, sk.description, sk.content, now, now,
		)
		if err != nil {
			return err
		}
		skillIDs[sk.name] = id
	}

	// ── Templates ────────────────────────────────────────────
	type templateDef struct {
		name        string
		description string
		agentType   string
		promptIDs   []string
		skillIDs    []string
		toolIDs     []string
	}

	templates := []templateDef{
		{
			name:        "Git Code Agent",
			description: "An agent that reads and modifies code in git repositories, creates branches, and commits changes.",
			agentType:   "pi",
			promptIDs:   []string{promptIDs["git-code-agent"]},
			skillIDs:    []string{skillIDs["code-review-and-fix"]},
			toolIDs:     []string{},
		},
		{
			name:        "Database Query Agent",
			description: "An agent that connects to MySQL/PostgreSQL databases, explores schemas, and executes SQL queries.",
			agentType:   "pi",
			promptIDs:   []string{promptIDs["database-query-agent"]},
			skillIDs:    []string{skillIDs["schema-explorer"]},
			toolIDs:     []string{toolIDs["mysql"], toolIDs["postgres"]},
		},
		{
			name:        "Excel Agent",
			description: "An agent that reads, creates, and modifies Excel spreadsheets with formulas and formatting.",
			agentType:   "pi",
			promptIDs:   []string{promptIDs["excel-agent"]},
			skillIDs:    []string{skillIDs["excel-report-builder"]},
			toolIDs:     []string{toolIDs["excel"]},
		},
	}

	for _, t := range templates {
		var exists int
		if err := s.db.QueryRow("SELECT COUNT(*) FROM templates WHERE name = ?", t.name).Scan(&exists); err != nil {
			return err
		}
		if exists > 0 {
			continue
		}
		id := "example-tmpl-" + strings.ReplaceAll(strings.ToLower(t.name), " ", "-")
		_, err := s.db.Exec(
			"INSERT OR IGNORE INTO templates (id, name, description, agent_type, dockerfile_content, created_at, updated_at) VALUES (?, ?, ?, ?, '', ?, ?)",
			id, t.name, t.description, t.agentType, now, now,
		)
		if err != nil {
			return err
		}
		// Bind prompts
		for _, pid := range t.promptIDs {
			if pid != "" {
				s.db.Exec("INSERT OR IGNORE INTO template_prompts (template_id, prompt_id) VALUES (?, ?)", id, pid)
			}
		}
		// Bind skills
		for _, sid := range t.skillIDs {
			if sid != "" {
				s.db.Exec("INSERT OR IGNORE INTO template_skills (template_id, skill_id) VALUES (?, ?)", id, sid)
			}
		}
		// Bind tools
		for _, tid := range t.toolIDs {
			if tid != "" {
				s.db.Exec("INSERT OR IGNORE INTO template_tools (template_id, tool_id) VALUES (?, ?)", id, tid)
			}
		}
	}

	return nil
}
