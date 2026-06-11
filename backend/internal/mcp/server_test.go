package mcp

import (
	"encoding/json"
	"testing"

	"cloud_ai_agent/internal/model"
	"cloud_ai_agent/internal/store"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func newTestDeps(t *testing.T) *deps {
	t.Helper()
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if err := s.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return &deps{store: s, agentSvc: nil}
}

func callTool(d *deps, handler server.ToolHandlerFunc, args map[string]any) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	result, err := handler(nil, req)
	if err != nil {
		return "ERROR: " + err.Error()
	}
	if len(result.Content) > 0 {
		if tc, ok := result.Content[0].(mcp.TextContent); ok {
			return tc.Text
		}
	}
	data, _ := json.Marshal(result)
	return string(data)
}

func TestAgentTools(t *testing.T) {
	d := newTestDeps(t)

	// Create template first (agent requires template_id)
	tmpl := &model.Template{Name: "test-template"}
	if err := d.store.CreateTemplate(tmpl); err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}

	// Create agent
	createResult := callTool(d, handleCreateAgent(d), map[string]any{
		"name":        "test-agent",
		"template_id": tmpl.ID,
		"repo_url":    "https://github.com/test/repo",
		"branch":      "main",
	})
	if !contains(createResult, "test-agent") {
		t.Errorf("create_agent: expected 'test-agent' in result, got %s", createResult)
	}

	// List agents
	listResult := callTool(d, handleListAgents(d), nil)
	if !contains(listResult, "test-agent") {
		t.Errorf("list_agents: expected 'test-agent' in result, got %s", listResult)
	}

	// Parse agent ID from create result
	var agents []model.Agent
	if err := json.Unmarshal([]byte(listResult), &agents); err != nil {
		// Try parsing with content key
		type toolResult struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		}
		var tr toolResult
		json.Unmarshal([]byte(listResult), &tr)
		if len(tr.Content) > 0 {
			json.Unmarshal([]byte(tr.Content[0].Text), &agents)
		}
	}

	if len(agents) == 0 {
		t.Fatal("no agents created")
	}
	agentID := agents[0].ID

	// Get agent
	getResult := callTool(d, handleGetAgent(d), map[string]any{"id": agentID})
	if !contains(getResult, "test-agent") {
		t.Errorf("get_agent: expected 'test-agent' in result, got %s", getResult)
	}

	// Update agent (draft status allows update)
	updateResult := callTool(d, handleUpdateAgent(d), map[string]any{
		"id":   agentID,
		"name": "updated-agent",
	})
	if !contains(updateResult, "updated-agent") {
		t.Errorf("update_agent: expected 'updated-agent' in result, got %s", updateResult)
	}

	// Delete agent
	deleteResult := callTool(d, handleDeleteAgent(d), map[string]any{"id": agentID})
	if !contains(deleteResult, "deleted") {
		t.Errorf("delete_agent: expected 'deleted' in result, got %s", deleteResult)
	}

	// Verify empty list
	listAfter := callTool(d, handleListAgents(d), nil)
	if !contains(listAfter, "[]") {
		t.Errorf("list_agents after delete: expected empty list, got %s", listAfter)
	}
}

func TestTemplateTools(t *testing.T) {
	d := newTestDeps(t)

	// Create template
	createResult := callTool(d, handleCreateTemplate(d), map[string]any{
		"name":              "test-template",
		"description":       "A test template",
		"dockerfile_content": "FROM alpine",
	})
	if !contains(createResult, "test-template") {
		t.Errorf("create_template: expected 'test-template' in result, got %s", createResult)
	}

	// Parse template ID
	templates := parseTemplates(t, callTool(d, handleListTemplates(d), nil))
	if len(templates) == 0 {
		t.Fatal("no templates created")
	}
	tmplID := templates[0].ID

	// Get template
	getResult := callTool(d, handleGetTemplate(d), map[string]any{"id": tmplID})
	if !contains(getResult, "test-template") {
		t.Errorf("get_template: expected 'test-template' in result, got %s", getResult)
	}

	// Update template
	updateResult := callTool(d, handleUpdateTemplate(d), map[string]any{
		"id":   tmplID,
		"name": "updated-template",
	})
	if !contains(updateResult, "updated-template") {
		t.Errorf("update_template: expected 'updated-template' in result, got %s", updateResult)
	}

	// Delete template
	deleteResult := callTool(d, handleDeleteTemplate(d), map[string]any{"id": tmplID})
	if !contains(deleteResult, "deleted") {
		t.Errorf("delete_template: expected 'deleted' in result, got %s", deleteResult)
	}
}

func TestPromptTools(t *testing.T) {
	d := newTestDeps(t)

	createResult := callTool(d, handleCreatePrompt(d), map[string]any{
		"name":        "test-prompt",
		"description": "A test prompt",
		"content":     "Hello, world!",
	})
	if !contains(createResult, "test-prompt") {
		t.Errorf("create_prompt: expected 'test-prompt' in result, got %s", createResult)
	}

	prompts := parsePrompts(t, callTool(d, handleListPrompts(d), nil))
	if len(prompts) == 0 {
		t.Fatal("no prompts created")
	}
	promptID := prompts[0].ID

	getResult := callTool(d, handleGetPrompt(d), map[string]any{"id": promptID})
	if !contains(getResult, "test-prompt") {
		t.Errorf("get_prompt: expected 'test-prompt' in result, got %s", getResult)
	}

	updateResult := callTool(d, handleUpdatePrompt(d), map[string]any{
		"id":      promptID,
		"content": "Updated content",
	})
	if !contains(updateResult, "Updated content") {
		t.Errorf("update_prompt: expected 'Updated content' in result, got %s", updateResult)
	}

	deleteResult := callTool(d, handleDeletePrompt(d), map[string]any{"id": promptID})
	if !contains(deleteResult, "deleted") {
		t.Errorf("delete_prompt: expected 'deleted' in result, got %s", deleteResult)
	}
}

func TestSkillTools(t *testing.T) {
	d := newTestDeps(t)

	createResult := callTool(d, handleCreateSkill(d), map[string]any{
		"name":        "test-skill",
		"description": "A test skill",
		"content":     "Skill content here",
	})
	if !contains(createResult, "test-skill") {
		t.Errorf("create_skill: expected 'test-skill' in result, got %s", createResult)
	}

	skills := parseSkills(t, callTool(d, handleListSkills(d), nil))
	if len(skills) == 0 {
		t.Fatal("no skills created")
	}
	skillID := skills[0].ID

	getResult := callTool(d, handleGetSkill(d), map[string]any{"id": skillID})
	if !contains(getResult, "test-skill") {
		t.Errorf("get_skill: expected 'test-skill' in result, got %s", getResult)
	}

	deleteResult := callTool(d, handleDeleteSkill(d), map[string]any{"id": skillID})
	if !contains(deleteResult, "deleted") {
		t.Errorf("delete_skill: expected 'deleted' in result, got %s", deleteResult)
	}
}

func TestCustomToolTools(t *testing.T) {
	d := newTestDeps(t)

	createResult := callTool(d, handleCreateCustomTool(d), map[string]any{
		"name":           "test-custom-tool",
		"label":          "Test Tool",
		"description":    "A test custom tool",
		"dsl_definition": `{"type":"http"}`,
	})
	if !contains(createResult, "test-custom-tool") {
		t.Errorf("create_custom_tool: expected 'test-custom-tool' in result, got %s", createResult)
	}

	ctools := parseTools(t, callTool(d, handleListCustomTools(d), nil))
	if len(ctools) == 0 {
		t.Fatal("no custom tools created")
	}
	toolID := ctools[0].ID

	getResult := callTool(d, handleGetCustomTool(d), map[string]any{"id": toolID})
	if !contains(getResult, "test-custom-tool") {
		t.Errorf("get_custom_tool: expected 'test-custom-tool' in result, got %s", getResult)
	}

	deleteResult := callTool(d, handleDeleteCustomTool(d), map[string]any{"id": toolID})
	if !contains(deleteResult, "deleted") {
		t.Errorf("delete_custom_tool: expected 'deleted' in result, got %s", deleteResult)
	}
}

func TestAgentTeamTools(t *testing.T) {
	d := newTestDeps(t)

	tmpl := &model.Template{Name: "team-template"}
	if err := d.store.CreateTemplate(tmpl); err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}

	createResult := callTool(d, handleCreateAgentTeam(d), map[string]any{
		"name":        "test-team",
		"template_id": tmpl.ID,
		"repo_url":    "https://github.com/test/team-repo",
	})
	if !contains(createResult, "test-team") {
		t.Errorf("create_agent_team: expected 'test-team' in result, got %s", createResult)
	}

	teams := parseAgentTeams(t, callTool(d, handleListAgentTeams(d), nil))
	if len(teams) == 0 {
		t.Fatal("no agent teams created")
	}
	teamID := teams[0].ID

	getResult := callTool(d, handleGetAgentTeam(d), map[string]any{"id": teamID})
	if !contains(getResult, "test-team") {
		t.Errorf("get_agent_team: expected 'test-team' in result, got %s", getResult)
	}

	updateResult := callTool(d, handleUpdateAgentTeam(d), map[string]any{
		"id":   teamID,
		"name": "updated-team",
	})
	if !contains(updateResult, "updated-team") {
		t.Errorf("update_agent_team: expected 'updated-team' in result, got %s", updateResult)
	}

	deleteResult := callTool(d, handleDeleteAgentTeam(d), map[string]any{"id": teamID})
	if !contains(deleteResult, "deleted") {
		t.Errorf("delete_agent_team: expected 'deleted' in result, got %s", deleteResult)
	}
}

func TestProviderConfigTools(t *testing.T) {
	d := newTestDeps(t)

	createResult := callTool(d, handleCreateProviderConfig(d), map[string]any{
		"name":     "test-provider",
		"provider": "openai-codex",
		"model_id": "gpt-4",
		"api_key":  "sk-secret-key-123",
		"base_url": "https://api.openai.com",
	})
	if !contains(createResult, "test-provider") {
		t.Errorf("create_provider_config: expected 'test-provider' in result, got %s", createResult)
	}
	// api_key should be masked in output
	if contains(createResult, "sk-secret-key-123") {
		t.Errorf("create_provider_config: api_key should be masked in output")
	}

	configs := parseProviderConfigs(t, callTool(d, handleListProviderConfigs(d), nil))
	if len(configs) == 0 {
		t.Fatal("no provider configs created")
	}
	pcID := configs[0].ID

	getResult := callTool(d, handleGetProviderConfig(d), map[string]any{"id": pcID})
	if contains(getResult, "sk-secret-key-123") {
		t.Errorf("get_provider_config: api_key should be masked in output")
	}

	deleteResult := callTool(d, handleDeleteProviderConfig(d), map[string]any{"id": pcID})
	if !contains(deleteResult, "deleted") {
		t.Errorf("delete_provider_config: expected 'deleted' in result, got %s", deleteResult)
	}
}

func TestResourceTools(t *testing.T) {
	d := newTestDeps(t)

	createResult := callTool(d, handleCreateResource(d), map[string]any{
		"name":   "test-resource",
		"type":   "git",
		"config": `{"url":"https://github.com/test"}`,
	})
	if !contains(createResult, "test-resource") {
		t.Errorf("create_resource: expected 'test-resource' in result, got %s", createResult)
	}

	resources := parseResources(t, callTool(d, handleListResources(d), nil))
	if len(resources) == 0 {
		t.Fatal("no resources created")
	}
	resID := resources[0].ID

	getResult := callTool(d, handleGetResource(d), map[string]any{"id": resID})
	if !contains(getResult, "test-resource") {
		t.Errorf("get_resource: expected 'test-resource' in result, got %s", getResult)
	}

	deleteResult := callTool(d, handleDeleteResource(d), map[string]any{"id": resID})
	if !contains(deleteResult, "deleted") {
		t.Errorf("delete_resource: expected 'deleted' in result, got %s", deleteResult)
	}
}

func TestInstanceTools(t *testing.T) {
	d := newTestDeps(t)

	tmpl := &model.Template{Name: "inst-template"}
	d.store.CreateTemplate(tmpl)

	agent := &model.Agent{Name: "inst-agent", TemplateID: tmpl.ID}
	d.store.CreateAgent(agent)

	// list instances - should start empty
	listResult := callTool(d, handleListInstances(d), nil)
	if !contains(listResult, "[]") {
		t.Errorf("list_instances: expected empty list, got %s", listResult)
	}

	// try get non-existent instance
	getResult := callTool(d, handleGetInstance(d), map[string]any{"id": "nonexistent"})
	if !contains(getResult, "not found") {
		t.Errorf("get_instance: expected 'not found', got %s", getResult)
	}
}

func TestHealthTool(t *testing.T) {
	d := newTestDeps(t)

	result := callTool(d, handleHealth(d), nil)
	if !contains(result, `"status":"ok"`) {
		t.Errorf("health: expected ok status, got %s", result)
	}
}

func TestTemplateBindings(t *testing.T) {
	d := newTestDeps(t)

	p := &model.Prompt{Name: "bind-prompt", Content: "test"}
	d.store.CreatePrompt(p)
	sk := &model.Skill{Name: "bind-skill", Content: "skill"}
	d.store.CreateSkill(sk)
	tool := &model.Tool{Name: "bind-tool"}
	d.store.CreateTool(tool)

	tmpl := &model.Template{Name: "bind-template"}
	d.store.CreateTemplate(tmpl)

	updateResult := callTool(d, handleUpdateTemplateBindings(d), map[string]any{
		"id":             tmpl.ID,
		"prompt_ids":     p.ID,
		"skill_ids":      sk.ID,
		"tool_ids":       tool.ID,
	})
	if !contains(updateResult, "bindings updated") {
		t.Errorf("update_template_bindings: expected success, got %s", updateResult)
	}

	// Verify bindings were applied
	getResult := callTool(d, handleGetTemplate(d), map[string]any{"id": tmpl.ID})
	if !contains(getResult, p.ID) || !contains(getResult, sk.ID) || !contains(getResult, tool.ID) {
		t.Errorf("get_template after bindings: expected prompt/skill/tool IDs, got %s", getResult)
	}
}

func TestBuildAgent(t *testing.T) {
	d := newTestDeps(t)

	tmpl := &model.Template{Name: "build-template"}
	d.store.CreateTemplate(tmpl)
	agent := &model.Agent{Name: "build-agent", TemplateID: tmpl.ID}
	d.store.CreateAgent(agent)

	// build_agent without agent service should return error
	result := callTool(d, handleBuildAgent(d), map[string]any{"id": agent.ID})
	if !contains(result, "not available") {
		t.Errorf("build_agent without service: expected 'not available', got %s", result)
	}

	// get_build_log for agent without build should return error
	logResult := callTool(d, handleGetBuildLog(d), map[string]any{"id": agent.ID})
	if !contains(logResult, "not found") {
		t.Errorf("get_build_log: expected 'not found', got %s", logResult)
	}
}

func TestNewServer(t *testing.T) {
	d := newTestDeps(t)

	mcpServer := NewServer(d.store, d.agentSvc)
	if mcpServer == nil {
		t.Fatal("NewServer returned nil")
	}

	sseServer := NewSSEServer(mcpServer)
	if sseServer == nil {
		t.Fatal("NewSSEServer returned nil")
	}
}

// --- Helpers ---

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func extractJSON(s string) string {
	start := -1
	end := -1
	for i := 0; i < len(s); i++ {
		if s[i] == '[' || s[i] == '{' {
			start = i
			break
		}
	}
	if start == -1 {
		return ""
	}
	depth := 0
	for i := start; i < len(s); i++ {
		if s[i] == '[' || s[i] == '{' {
			depth++
		} else if s[i] == ']' || s[i] == '}' {
			depth--
			if depth == 0 {
				end = i + 1
				break
			}
		}
	}
	if end == -1 {
		return ""
	}
	return s[start:end]
}

func parseTemplates(t *testing.T, result string) []model.Template {
	t.Helper()
	return unmarshalList[model.Template](t, result)
}

func parsePrompts(t *testing.T, result string) []model.Prompt {
	t.Helper()
	return unmarshalList[model.Prompt](t, result)
}

func parseSkills(t *testing.T, result string) []model.Skill {
	t.Helper()
	return unmarshalList[model.Skill](t, result)
}

func parseTools(t *testing.T, result string) []model.Tool {
	t.Helper()
	return unmarshalList[model.Tool](t, result)
}

func parseAgentTeams(t *testing.T, result string) []model.AgentTeam {
	t.Helper()
	return unmarshalList[model.AgentTeam](t, result)
}

func parseProviderConfigs(t *testing.T, result string) []model.ProviderConfig {
	t.Helper()
	return unmarshalList[model.ProviderConfig](t, result)
}

func parseResources(t *testing.T, result string) []model.Resource {
	t.Helper()
	return unmarshalList[model.Resource](t, result)
}

func unmarshalList[T any](t *testing.T, result string) []T {
	t.Helper()
	var items []T
	jsonStr := extractJSON(result)
	if jsonStr == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(jsonStr), &items); err != nil {
		t.Logf("unmarshalList error (%s): %v", jsonStr, err)
		return nil
	}
	return items
}
