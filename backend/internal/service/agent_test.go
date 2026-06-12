package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"cloud_ai_agent/internal/store"
)

func TestWriteToolExtensions_GeneratesManifest(t *testing.T) {
	// Create a temp store with default tools (including MySQL MCP)
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("New store: %v", err)
	}
	defer s.Close()

	if err := s.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	if err := s.SeedDefaultTools(); err != nil {
		t.Fatalf("SeedDefaultTools: %v", err)
	}

	svc := NewAgentService(s)
	tmpDir := t.TempDir()

	// Get the MySQL tool ID
	mysqlTool, err := s.GetTool("default-mysql")
	if err != nil || mysqlTool == nil {
		t.Fatalf("GetTool mysql: %v", err)
	}
	pgTool, err := s.GetTool("default-postgres")
	if err != nil || pgTool == nil {
		t.Fatalf("GetTool postgres: %v", err)
	}

	toolIDs := []string{mysqlTool.ID, pgTool.ID}

	if err := svc.writeToolExtensions(tmpDir, toolIDs); err != nil {
		t.Fatalf("writeToolExtensions: %v", err)
	}

	// Verify .ts files were generated
	for _, name := range []string{"mysql.ts", "postgres.ts"} {
		p := filepath.Join(tmpDir, "extensions", name)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Errorf("missing .ts file: %s", p)
		}
	}

	// Verify extensions.json manifest was generated
	manifestPath := filepath.Join(tmpDir, "extensions", "extensions.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("extensions.json not generated: %v", err)
	}

	var manifest []struct {
		Name        string          `json:"name"`
		Label       string          `json:"label"`
		Description string          `json:"description"`
		Handler     json.RawMessage `json:"handler"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("invalid extensions.json: %v", err)
	}

	if len(manifest) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(manifest))
	}

	// Verify MySQL entry has correct MCP handler
	if manifest[0].Name != "mysql" {
		t.Errorf("expected first entry name=mydql, got %s", manifest[0].Name)
	}
	if manifest[1].Name != "postgres" {
		t.Errorf("expected second entry name=postgres, got %s", manifest[1].Name)
	}

	// Verify handler contains MCP config
	var handler struct {
		Type      string   `json:"type"`
		Transport string   `json:"transport"`
		Command   string   `json:"command"`
		Args      []string `json:"args"`
	}
	if err := json.Unmarshal(manifest[0].Handler, &handler); err != nil {
		t.Fatalf("invalid handler: %v", err)
	}
	if handler.Type != "mcp" {
		t.Errorf("expected handler type=mcp, got %s", handler.Type)
	}
	if handler.Transport != "stdio" {
		t.Errorf("expected transport=stdio, got %s", handler.Transport)
	}
	if handler.Command != "npx" {
		t.Errorf("expected command=npx, got %s", handler.Command)
	}

	t.Logf("extensions.json content: %s", string(data))
}

func TestWriteToolExtensions_EmptyTools(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("New store: %v", err)
	}
	defer s.Close()

	svc := NewAgentService(s)
	tmpDir := t.TempDir()

	if err := svc.writeToolExtensions(tmpDir, nil); err != nil {
		t.Fatalf("writeToolExtensions with nil: %v", err)
	}

	// No extensions.json should be generated when no tools
	manifestPath := filepath.Join(tmpDir, "extensions", "extensions.json")
	if _, err := os.Stat(manifestPath); !os.IsNotExist(err) {
		t.Error("extensions.json should not exist when no tools")
	}
}
