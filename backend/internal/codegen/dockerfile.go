package codegen

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// TemplatesDir is the directory containing Dockerfile template files.
// Set before calling Generate functions, or it will auto-detect relative to cwd.
var TemplatesDir = ""

func init() {
	if TemplatesDir == "" {
		if _, err := os.Stat("../container-wrapper/dockerfiles"); err == nil {
			TemplatesDir = "../container-wrapper/dockerfiles"
		}
	}
}

type DockerfileData struct {
	NodeVersion   string
	SkillsDir     string
	PromptsDir    string
	ExtensionsDir string
	WrapperScript string
	McpClient     string
	ExposePort    int
	CustomContent string
}

// loadTemplate reads a Dockerfile template from the configured TemplatesDir.
func loadTemplate(name string) (string, error) {
	if TemplatesDir == "" {
		return "", fmt.Errorf("codegen: TemplatesDir not configured")
	}
	path := filepath.Join(TemplatesDir, name+".dockerfile")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("codegen: read template %s: %w", path, err)
	}
	return string(data), nil
}

// defaultDockerfiles holds built-in fallback templates keyed by name.
var defaultDockerfiles = map[string]string{
	"pi":          defaultDockerfilePI,
	"claude-code": defaultDockerfileClaudeCode,
	"codex":       defaultDockerfileCodex,
	"team":        defaultDockerfileTeam,
}

// getTemplate loads a template by name. Tries external file first, then
// falls back to the compiled-in default.
func getTemplate(name string) (string, error) {
	if tmpl, err := loadTemplate(name); err == nil {
		return tmpl, nil
	}
	if fallback, ok := defaultDockerfiles[name]; ok {
		return fallback, nil
	}
	return "", fmt.Errorf("codegen: template %q not found in %s and no built-in fallback", name, TemplatesDir)
}

// generateFromTemplate renders a Dockerfile from the given template string and data.
func generateFromTemplate(tmpl string, data *DockerfileData) (string, error) {
	t, err := template.New("Dockerfile").Parse(tmpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// GenerateDockerfile generates a Dockerfile for the PI agent type.
func GenerateDockerfile(data *DockerfileData) (string, error) {
	if data.NodeVersion == "" {
		data.NodeVersion = "22"
	}
	if data.WrapperScript == "" {
		data.WrapperScript = "server.js"
	}
	if data.McpClient == "" {
		data.McpClient = "mcp-client.js"
	}
	if data.SkillsDir == "" {
		data.SkillsDir = "pi-skills"
	}
	if data.PromptsDir == "" {
		data.PromptsDir = "pi-prompts"
	}
	if data.ExtensionsDir == "" {
		data.ExtensionsDir = "extensions"
	}
	if data.ExposePort == 0 {
		data.ExposePort = 3000
	}

	tmpl := data.CustomContent
	if tmpl == "" {
		var err error
		tmpl, err = getTemplate("pi")
		if err != nil {
			return "", err
		}
	}

	t, err := template.New("Dockerfile").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
