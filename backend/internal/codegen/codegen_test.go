package codegen

import (
	"strings"
	"testing"
)

func TestGenerateDockerfile(t *testing.T) {
	df, err := GenerateDockerfile(&DockerfileData{
		NodeVersion: "22",
		ExposePort:  3000,
	})
	if err != nil {
		t.Fatalf("GenerateDockerfile: %v", err)
	}
	checks := []string{
		"WORKDIR /app",
		"EXPOSE 3000",
		"server.js",
	}
	for _, c := range checks {
		if !strings.Contains(df, c) {
			t.Errorf("Dockerfile missing: %s", c)
		}
	}
}

func TestGenerateToolExtension(t *testing.T) {
	dsl := `{
		"name": "test_tool",
		"label": "Test Tool",
		"description": "A test tool",
		"parameters": {
			"query": { "type": "string", "description": "Search query" }
		},
		"handler": {
			"type": "http",
			"method": "GET",
			"url": "https://api.example.com?q={{query}}"
		}
	}`
	code, err := GenerateToolExtension(dsl)
	if err != nil {
		t.Fatalf("GenerateToolExtension: %v", err)
	}
	checks := []string{"test_tool", "Test Tool", "A test tool", "Type.String", "activate", "context.registerTool"}
	for _, c := range checks {
		if !strings.Contains(code, c) {
			t.Errorf("Generated code missing: %s", c)
		}
	}
}
