package codegen

import (
	"bytes"
	"text/template"
)

type TeamDockerfileData struct {
	AgentDirs      []string
	TeamPromptsDir string
	TeamSkillsDir  string
	McpClient      string
	LLMLogger      string
	ExposePort     int
	CustomContent  string
}

func GenerateTeamDockerfile(data *TeamDockerfileData) (string, error) {
	if data.McpClient == "" {
		data.McpClient = "mcp-client.js"
	}
	if data.LLMLogger == "" {
		data.LLMLogger = "llm-logger.js"
	}
	if data.ExposePort == 0 {
		data.ExposePort = 3000
	}

	tmpl := data.CustomContent
	if tmpl == "" {
		var err error
		tmpl, err = getTemplate("team")
		if err != nil {
			return "", err
		}
	}

	t, err := template.New("TeamDockerfile").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
