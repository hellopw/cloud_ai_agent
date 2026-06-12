package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func New(dbPath string) (*Store, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) DB() *sql.DB { return s.db }
func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS prompts (
		id          TEXT PRIMARY KEY,
		name        TEXT NOT NULL UNIQUE,
		description TEXT DEFAULT '',
		content     TEXT NOT NULL DEFAULT '',
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS skills (
		id          TEXT PRIMARY KEY,
		name        TEXT NOT NULL UNIQUE,
		description TEXT DEFAULT '',
		content     TEXT NOT NULL DEFAULT '',
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS tools (
		id             TEXT PRIMARY KEY,
		name           TEXT NOT NULL UNIQUE,
		label          TEXT NOT NULL DEFAULT '',
		description    TEXT DEFAULT '',
		dsl_definition TEXT NOT NULL DEFAULT '{}',
		created_at     DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at     DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS memories (
		id          TEXT PRIMARY KEY,
		name        TEXT NOT NULL UNIQUE,
		description TEXT DEFAULT '',
		content     TEXT NOT NULL DEFAULT '',
		source      TEXT NOT NULL DEFAULT 'manual',
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS instance_memories (
		instance_id TEXT NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
		memory_id   TEXT NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
		PRIMARY KEY (instance_id, memory_id)
	);

	CREATE TABLE IF NOT EXISTS templates (
		id                 TEXT PRIMARY KEY,
		name               TEXT NOT NULL UNIQUE,
		description        TEXT DEFAULT '',
		agent_type         TEXT DEFAULT 'pi',
		dockerfile_content TEXT DEFAULT '',
		created_at         DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at         DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS template_prompts (
		template_id TEXT NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
		prompt_id   TEXT NOT NULL REFERENCES prompts(id) ON DELETE CASCADE,
		PRIMARY KEY (template_id, prompt_id)
	);

	CREATE TABLE IF NOT EXISTS template_skills (
		template_id TEXT NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
		skill_id    TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
		PRIMARY KEY (template_id, skill_id)
	);

	CREATE TABLE IF NOT EXISTS template_tools (
		template_id TEXT NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
		tool_id     TEXT NOT NULL REFERENCES tools(id) ON DELETE CASCADE,
		PRIMARY KEY (template_id, tool_id)
	);

	CREATE TABLE IF NOT EXISTS agents (
		id           TEXT PRIMARY KEY,
		name         TEXT NOT NULL,
		template_id  TEXT NOT NULL REFERENCES templates(id),
		repo_url     TEXT NOT NULL DEFAULT '',
		git_username TEXT DEFAULT '',
		git_password TEXT DEFAULT '',
		branch       TEXT NOT NULL DEFAULT 'main',
		image_tag    TEXT DEFAULT '',
		status       TEXT NOT NULL DEFAULT 'draft',
		error_msg    TEXT DEFAULT '',
		created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at   DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS instances (
		id           TEXT PRIMARY KEY,
		agent_id     TEXT NOT NULL REFERENCES agents(id),
		team_id      TEXT DEFAULT '',
		container_id TEXT DEFAULT '',
		host_port    INTEGER DEFAULT 0,
		status       TEXT NOT NULL DEFAULT 'starting',
		error_msg    TEXT DEFAULT '',
		created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at   DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS resources (
		id         TEXT PRIMARY KEY,
		name       TEXT NOT NULL,
		type       TEXT NOT NULL DEFAULT 'git',
		config     TEXT NOT NULL DEFAULT '{}',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS provider_configs (
		id         TEXT PRIMARY KEY,
		name       TEXT NOT NULL UNIQUE,
		provider   TEXT NOT NULL DEFAULT 'openai-codex',
		model_id   TEXT NOT NULL DEFAULT '',
		api_key    TEXT NOT NULL DEFAULT '',
		base_url   TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS agent_teams (
		id           TEXT PRIMARY KEY,
		name         TEXT NOT NULL,
		template_id  TEXT NOT NULL REFERENCES templates(id),
		repo_url     TEXT NOT NULL DEFAULT '',
		git_username TEXT DEFAULT '',
		git_password TEXT DEFAULT '',
		branch       TEXT NOT NULL DEFAULT 'main',
		image_tag    TEXT DEFAULT '',
		status       TEXT NOT NULL DEFAULT 'draft',
		error_msg    TEXT DEFAULT '',
		created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at   DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS team_members (
		id                     TEXT PRIMARY KEY,
		team_id                TEXT NOT NULL REFERENCES agent_teams(id) ON DELETE CASCADE,
		name                   TEXT NOT NULL,
		role                   TEXT NOT NULL DEFAULT 'worker',
		agent_template_id      TEXT REFERENCES templates(id),
		provider_config_id     TEXT REFERENCES provider_configs(id),
		system_prompt_override TEXT DEFAULT '',
		sequence               INTEGER DEFAULT 0,
		created_at             DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at             DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS team_member_prompts (
		team_member_id TEXT NOT NULL REFERENCES team_members(id) ON DELETE CASCADE,
		prompt_id      TEXT NOT NULL REFERENCES prompts(id) ON DELETE CASCADE,
		PRIMARY KEY (team_member_id, prompt_id)
	);

	CREATE TABLE IF NOT EXISTS team_member_skills (
		team_member_id TEXT NOT NULL REFERENCES team_members(id) ON DELETE CASCADE,
		skill_id       TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
		PRIMARY KEY (team_member_id, skill_id)
	);

	CREATE TABLE IF NOT EXISTS team_member_tools (
		team_member_id TEXT NOT NULL REFERENCES team_members(id) ON DELETE CASCADE,
		tool_id        TEXT NOT NULL REFERENCES tools(id) ON DELETE CASCADE,
		PRIMARY KEY (team_member_id, tool_id)
	);

	CREATE TABLE IF NOT EXISTS agent_team_prompts (
		team_id   TEXT NOT NULL REFERENCES agent_teams(id) ON DELETE CASCADE,
		prompt_id TEXT NOT NULL REFERENCES prompts(id) ON DELETE CASCADE,
		PRIMARY KEY (team_id, prompt_id)
	);

	CREATE TABLE IF NOT EXISTS agent_team_skills (
		team_id  TEXT NOT NULL REFERENCES agent_teams(id) ON DELETE CASCADE,
		skill_id TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
		PRIMARY KEY (team_id, skill_id)
	);
	`
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}

	// Migration: add columns for existing databases (ignore errors)
	s.db.Exec("ALTER TABLE agents ADD COLUMN git_username TEXT DEFAULT ''")
	s.db.Exec("ALTER TABLE agents ADD COLUMN git_password TEXT DEFAULT ''")
	s.db.Exec("ALTER TABLE instances ADD COLUMN error_msg TEXT DEFAULT ''")
	s.db.Exec("ALTER TABLE instances ADD COLUMN team_id TEXT DEFAULT ''")
	s.db.Exec("ALTER TABLE templates ADD COLUMN agent_type TEXT DEFAULT 'pi'")
	return nil
}

