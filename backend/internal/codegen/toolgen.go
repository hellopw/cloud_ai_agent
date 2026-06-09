package codegen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"
)

type ToolDSL struct {
	Name        string                `json:"name"`
	Label       string                `json:"label"`
	Description string                `json:"description"`
	Parameters  map[string]ParamField `json:"parameters"`
	Handler     HandlerDef            `json:"handler"`
}

type ParamField struct {
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Default     interface{} `json:"default,omitempty"`
}

type HandlerDef struct {
	Type    string            `json:"type"`
	Method  string            `json:"method,omitempty"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
	Command string            `json:"command,omitempty"`
	Code    string            `json:"code,omitempty"`
}

const toolTemplate = `// Auto-generated tool: {{.Name}}
import type { ExtensionContext, ToolDefinition } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";

export function activate(context: ExtensionContext) {
  context.registerTool({
    name: "{{.Name}}",
    label: "{{.Label}}",
    description: {{.DescriptionJSON}},
    parameters: Type.Object({
{{range $name, $field := .Parameters}}      {{$name}}: {{formatTypeBox $field}},
{{end}}    }),
    async execute(args, { signal }) {
{{.ExecuteBody}}
    },
  } satisfies ToolDefinition);
}
`

func GenerateToolExtension(dslJSON string) (string, error) {
	var dsl ToolDSL
	if err := json.Unmarshal([]byte(dslJSON), &dsl); err != nil {
		return "", fmt.Errorf("invalid DSL: %w", err)
	}

	descJSON, _ := json.Marshal(dsl.Description)

	data := struct {
		Name            string
		Label           string
		DescriptionJSON string
		Parameters      map[string]ParamField
		ExecuteBody     string
	}{
		Name:            dsl.Name,
		Label:           dsl.Label,
		DescriptionJSON: string(descJSON),
		Parameters:      dsl.Parameters,
		ExecuteBody:     generateExecuteBody(dsl.Handler),
	}

	funcMap := template.FuncMap{
		"formatTypeBox": formatTypeBox,
	}

	t, err := template.New("tool").Funcs(funcMap).Parse(toolTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func formatTypeBox(f ParamField) string {
	switch f.Type {
	case "string":
		def := ""
		if f.Default != nil {
			def = fmt.Sprintf(", { default: %q }", f.Default)
		}
		return fmt.Sprintf("Type.String({ description: %q%s })", f.Description, def)
	case "number":
		def := ""
		if f.Default != nil {
			def = fmt.Sprintf(", { default: %v }", f.Default)
		}
		return fmt.Sprintf("Type.Number({ description: %q%s })", f.Description, def)
	case "boolean":
		return fmt.Sprintf("Type.Boolean({ description: %q })", f.Description)
	default:
		return fmt.Sprintf("Type.String({ description: %q })", f.Description)
	}
}

func generateExecuteBody(h HandlerDef) string {
	switch h.Type {
	case "http":
		return generateHTTPBody(h)
	case "shell":
		return generateShellBody(h)
	case "javascript":
		return h.Code
	default:
		return fmt.Sprintf("      throw new Error(%q);", "unsupported handler type: "+h.Type)
	}
}

func generateHTTPBody(h HandlerDef) string {
	var buf bytes.Buffer
	buf.WriteString("      const url = `" + h.URL + "`;\n")
	buf.WriteString("      const headers: Record<string, string> = {};\n")

	for k, v := range h.Headers {
		buf.WriteString(fmt.Sprintf("      headers[%q] = process.env.%s || %q;\n", k, extractEnv(v), v))
	}

	buf.WriteString(fmt.Sprintf("      const resp = await fetch(url, { method: %q, headers, signal, body: %s ? JSON.stringify(%s) : undefined });\n", h.Method, h.Body, h.Body))
	buf.WriteString("      const text = await resp.text();\n")
	buf.WriteString("      return { content: [{ type: \"text\" as const, text }] };\n")
	return buf.String()
}

func generateShellBody(h HandlerDef) string {
	return fmt.Sprintf("      // shell command execution not implemented in wrapper\n      throw new Error(%q);", "shell handler requires container support")
}

func extractEnv(value string) string {
	// Check for {{env.VAR}} pattern
	for i := 0; i < len(value); i++ {
		if i+7 < len(value) && value[i:i+7] == "{{env." {
			end := i + 7
			for end < len(value) && value[end] != '}' {
				end++
			}
			return value[i+7 : end]
		}
	}
	return ""
}
