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

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) Close() error {
	return s.db.Close()
}

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

	CREATE TABLE IF NOT EXISTS templates (
		id                 TEXT PRIMARY KEY,
		name               TEXT NOT NULL UNIQUE,
		description        TEXT DEFAULT '',
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
		id          TEXT PRIMARY KEY,
		name        TEXT NOT NULL,
		template_id TEXT NOT NULL REFERENCES templates(id),
		repo_url    TEXT NOT NULL DEFAULT '',
		branch      TEXT NOT NULL DEFAULT 'main',
		image_tag   TEXT DEFAULT '',
		status      TEXT NOT NULL DEFAULT 'draft',
		error_msg   TEXT DEFAULT '',
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS instances (
		id           TEXT PRIMARY KEY,
		agent_id     TEXT NOT NULL REFERENCES agents(id),
		container_id TEXT DEFAULT '',
		host_port    INTEGER DEFAULT 0,
		status       TEXT NOT NULL DEFAULT 'starting',
		created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at   DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err := s.db.Exec(schema)
	return err
}
