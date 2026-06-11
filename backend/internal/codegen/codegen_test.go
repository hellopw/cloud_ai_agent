package codegen

import (
	"strings"
	"testing"
)

func TestGenerateDockerfile(t *testing.T) {
	df, err := GenerateDockerfile(&DockerfileData{NodeVersion: "22", ExposePort: 3000})
	if err != nil { t.Fatalf("GenerateDockerfile: %v", err) }
	for _, c := range []string{"FROM private-registry.sohucs.com/domeos-pub/node:22.14.0-alpine3.21", "WORKDIR /app", "EXPOSE 3000"} {
		if !strings.Contains(df, c) { t.Errorf("missing: %s", c) }
	}
}

func TestGenerateToolExtension(t *testing.T) {
	dsl := "{\"name\":\"test_tool\",\"handler\":{\"type\":\"http\",\"method\":\"GET\",\"url\":\"https://api.example.com\"}}"
	code, err := GenerateToolExtension(dsl)
	if err != nil { t.Fatalf("GenerateToolExtension: %v", err) }
	for _, c := range []string{"test_tool", "activate", "context.registerTool"} {
		if !strings.Contains(code, c) { t.Errorf("missing: %s", c) }
	}
}

func TestGenerateMCPExtension_Stdio(t *testing.T) {
	dsl := "{\"name\":\"mcp_weather\",\"handler\":{\"type\":\"mcp\",\"transport\":\"stdio\",\"command\":\"npx\",\"args\":[\"-y\",\"@mcptools/server\"],\"env\":{},\"tool_name\":\"get_weather\"}}"
	code, err := GenerateToolExtension(dsl)
	if err != nil { t.Fatalf("MCP stdio: %v", err) }
	for _, c := range []string{"mcp_weather", "callMcpTool", "./mcp-client.js", "get_weather"} {
		if !strings.Contains(code, c) { t.Errorf("MCP stdio missing: %s", c) }
	}
}

func TestGenerateMCPExtension_Discover(t *testing.T) {
	dsl := "{\"name\":\"mcp_discover\",\"handler\":{\"type\":\"mcp\",\"transport\":\"stdio\",\"command\":\"npx\",\"args\":[\"-y\",\"@mcptools/server\"],\"env\":{}}}"
	code, err := GenerateToolExtension(dsl)
	if err != nil { t.Fatalf("MCP discover: %v", err) }
	for _, c := range []string{"mcp_discover", "listMcpTools", "./mcp-client.js"} {
		if !strings.Contains(code, c) { t.Errorf("MCP discover missing: %s", c) }
	}
}

func TestGenerateMCPExtension_SSE(t *testing.T) {
	dsl := "{\"name\":\"mcp_sse\",\"handler\":{\"type\":\"mcp\",\"transport\":\"sse\",\"url\":\"https://mcp.example.com/sse\",\"tool_name\":\"search\"}}"
	code, err := GenerateToolExtension(dsl)
	if err != nil { t.Fatalf("MCP SSE: %v", err) }
	for _, c := range []string{"mcp_sse", "callMcpTool", "./mcp-client.js", "search"} {
		if !strings.Contains(code, c) { t.Errorf("MCP SSE missing: %s", c) }
	}
}