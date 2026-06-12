package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"cloud_ai_agent/internal/model"
	"cloud_ai_agent/internal/store"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func newInProcessClient(t *testing.T) (*deps, *client.Client) {
	t.Helper()
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if err := s.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	d := &deps{store: s, agentSvc: nil}
	mcpSrv := NewServer(s, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c, err := client.NewInProcessClient(mcpSrv)
	if err != nil {
		t.Fatalf("NewInProcessClient: %v", err)
	}
	if err := c.Start(ctx); err != nil {
		t.Fatalf("Start client: %v", err)
	}

	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{Name: "test-client", Version: "1.0.0"}
	_, err = c.Initialize(ctx, initReq)
	if err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	t.Cleanup(func() { c.Close() })

	return d, c
}

func callMCPTool(t *testing.T, c *client.Client, toolName string, args map[string]any) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	callReq := mcp.CallToolRequest{}
	callReq.Params.Name = toolName
	callReq.Params.Arguments = args

	result, err := c.CallTool(ctx, callReq)
	if err != nil {
		return "ERROR: " + err.Error()
	}
	if result.IsError {
		if len(result.Content) > 0 {
			if tc, ok := result.Content[0].(mcp.TextContent); ok {
				return "ERROR: " + tc.Text
			}
		}
		return "ERROR: unknown"
	}
	if len(result.Content) > 0 {
		if tc, ok := result.Content[0].(mcp.TextContent); ok {
			return tc.Text
		}
	}
	data, _ := json.Marshal(result)
	return string(data)
}

func TestIntegration_AgentFullFlow(t *testing.T) {
	d, c := newInProcessClient(t)

	// Create template
	tmpl := &model.Template{Name: "flow-template"}
	d.store.CreateTemplate(tmpl)

	// Step 1: create_agent
	result := callMCPTool(t, c, "create_agent", map[string]any{
		"name":        "flow-agent",
		"template_id": tmpl.ID,
		"repo_url":    "https://github.com/test/repo",
	})
	if !contains(result, "flow-agent") {
		t.Fatalf("create_agent: %s", result)
	}

	// Find agent ID
	listResult := callMCPTool(t, c, "list_agents", nil)
	agents := parseAgentsFromResult(t, listResult)
	if len(agents) == 0 {
		t.Fatal("no agents in list")
	}
	agentID := agents[0].ID

	// Step 2: get_agent
	getResult := callMCPTool(t, c, "get_agent", map[string]any{"id": agentID})
	if !contains(getResult, "flow-agent") {
		t.Fatalf("get_agent: %s", getResult)
	}

	// Step 3: update_agent
	updResult := callMCPTool(t, c, "update_agent", map[string]any{
		"id":   agentID,
		"name": "flow-agent-v2",
	})
	if !contains(updResult, "flow-agent-v2") {
		t.Fatalf("update_agent: %s", updResult)
	}

	// Step 4: build_agent (no service, should return error)
	buildResult := callMCPTool(t, c, "build_agent", map[string]any{"id": agentID})
	if !contains(buildResult, "not available") {
		t.Fatalf("build_agent: %s", buildResult)
	}

	// Step 5: get_build_log
	logResult := callMCPTool(t, c, "get_build_log", map[string]any{"id": agentID})
	if !contains(logResult, "not found") {
		t.Fatalf("get_build_log: %s", logResult)
	}

	// Step 6: start_instance (no service)
	startResult := callMCPTool(t, c, "start_instance", map[string]any{"agent_id": agentID})
	if !contains(startResult, "not available") {
		t.Fatalf("start_instance: %s", startResult)
	}

	// Step 7: delete_agent
	delResult := callMCPTool(t, c, "delete_agent", map[string]any{"id": agentID})
	if !contains(delResult, "deleted") {
		t.Fatalf("delete_agent: %s", delResult)
	}
}

func TestIntegration_PromptCRUD(t *testing.T) {
	_, c := newInProcessClient(t)

	result := callMCPTool(t, c, "create_prompt", map[string]any{
		"name":    "int-prompt",
		"content": "Integration test prompt",
	})
	if !contains(result, "int-prompt") {
		t.Fatalf("create_prompt: %s", result)
	}

	list := callMCPTool(t, c, "list_prompts", nil)
	prompts := parsePromptsFromResult(t, list)
	if len(prompts) == 0 {
		t.Fatal("no prompts")
	}

	get := callMCPTool(t, c, "get_prompt", map[string]any{"id": prompts[0].ID})
	if !contains(get, "int-prompt") {
		t.Fatalf("get_prompt: %s", get)
	}

	del := callMCPTool(t, c, "delete_prompt", map[string]any{"id": prompts[0].ID})
	if !contains(del, "deleted") {
		t.Fatalf("delete_prompt: %s", del)
	}
}

func TestIntegration_TemplateWithBindings(t *testing.T) {
	d, c := newInProcessClient(t)

	// Create prerequisite entities
	p := &model.Prompt{Name: "tp-prompt", Content: "test"}
	d.store.CreatePrompt(p)
	sk := &model.Skill{Name: "tp-skill", Content: "skill"}
	d.store.CreateSkill(sk)

	// Create template
	callMCPTool(t, c, "create_template", map[string]any{
		"name": "tp-template",
	})

	list := callMCPTool(t, c, "list_templates", nil)
	templates := parseTemplatesFromResult(t, list)
	if len(templates) == 0 {
		t.Fatal("no templates")
	}

	// Bind prompts and skills
	result := callMCPTool(t, c, "update_template_bindings", map[string]any{
		"id":         templates[0].ID,
		"prompt_ids": p.ID,
		"skill_ids":  sk.ID,
	})
	if !contains(result, "updated") {
		t.Fatalf("update_template_bindings: %s", result)
	}

	// Verify bindings
	get := callMCPTool(t, c, "get_template", map[string]any{"id": templates[0].ID})
	if !contains(get, p.ID) || !contains(get, sk.ID) {
		t.Fatalf("get_template should contain binding IDs: %s", get)
	}
}

func TestIntegration_ProviderConfigMasking(t *testing.T) {
	_, c := newInProcessClient(t)

	result := callMCPTool(t, c, "create_provider_config", map[string]any{
		"name":     "int-provider",
		"provider": "openai-codex",
		"model_id": "gpt-4",
		"api_key":  "sk-super-secret-key",
	})
	if !contains(result, "int-provider") {
		t.Fatalf("create_provider_config: %s", result)
	}
	// api_key must be masked in output
	if contains(result, "sk-super-secret-key") {
		t.Fatal("api_key not masked in create output")
	}

	list := callMCPTool(t, c, "list_provider_configs", nil)
	if !contains(list, "***") {
		t.Fatalf("list_provider_configs should mask key: %s", list)
	}
	if contains(list, "sk-super-secret-key") {
		t.Fatal("api_key leaked in list output")
	}

	configs := parseProviderConfigsFromResult(t, list)
	get := callMCPTool(t, c, "get_provider_config", map[string]any{"id": configs[0].ID})
	if contains(get, "sk-super-secret-key") {
		t.Fatal("api_key leaked in get output")
	}
	if !contains(get, "***") {
		t.Fatal("api_key should be masked")
	}
}

func TestIntegration_AgentTeamCRUD(t *testing.T) {
	d, c := newInProcessClient(t)

	tmpl := &model.Template{Name: "team-tmpl"}
	d.store.CreateTemplate(tmpl)

	result := callMCPTool(t, c, "create_agent_team", map[string]any{
		"name":        "int-team",
		"template_id": tmpl.ID,
	})
	if !contains(result, "int-team") {
		t.Fatalf("create_agent_team: %s", result)
	}

	list := callMCPTool(t, c, "list_agent_teams", nil)
	teams := parseAgentTeamsFromResult(t, list)
	if len(teams) == 0 {
		t.Fatal("no teams")
	}

	get := callMCPTool(t, c, "get_agent_team", map[string]any{"id": teams[0].ID})
	if !contains(get, "int-team") {
		t.Fatalf("get_agent_team: %s", get)
	}

	upd := callMCPTool(t, c, "update_agent_team", map[string]any{
		"id":   teams[0].ID,
		"name": "int-team-v2",
	})
	if !contains(upd, "int-team-v2") {
		t.Fatalf("update_agent_team: %s", upd)
	}

	del := callMCPTool(t, c, "delete_agent_team", map[string]any{"id": teams[0].ID})
	if !contains(del, "deleted") {
		t.Fatalf("delete_agent_team: %s", del)
	}
}

func TestIntegration_Health(t *testing.T) {
	_, c := newInProcessClient(t)

	result := callMCPTool(t, c, "health", nil)
	if !contains(result, `"status":"ok"`) {
		t.Fatalf("health: %s", result)
	}
}

func TestIntegration_ResourceCRUD(t *testing.T) {
	_, c := newInProcessClient(t)

	result := callMCPTool(t, c, "create_resource", map[string]any{
		"name": "int-resource",
		"type": "git",
	})
	if !contains(result, "int-resource") {
		t.Fatalf("create_resource: %s", result)
	}

	list := callMCPTool(t, c, "list_resources", nil)
	resources := parseResourcesFromResult(t, list)
	if len(resources) == 0 {
		t.Fatal("no resources")
	}

	del := callMCPTool(t, c, "delete_resource", map[string]any{"id": resources[0].ID})
	if !contains(del, "deleted") {
		t.Fatalf("delete_resource: %s", del)
	}
}

func TestIntegration_SkillAndCustomToolCRUD(t *testing.T) {
	_, c := newInProcessClient(t)

	// Skill
	skResult := callMCPTool(t, c, "create_skill", map[string]any{
		"name":    "int-skill",
		"content": "Integration skill content",
	})
	if !contains(skResult, "int-skill") {
		t.Fatalf("create_skill: %s", skResult)
	}
	skills := parseSkillsFromResult(t, callMCPTool(t, c, "list_skills", nil))
	callMCPTool(t, c, "delete_skill", map[string]any{"id": skills[0].ID})

	// Custom tool
	toolResult := callMCPTool(t, c, "create_custom_tool", map[string]any{
		"name":           "int-custom-tool",
		"label":          "Int Tool",
		"dsl_definition": `{}`,
	})
	if !contains(toolResult, "int-custom-tool") {
		t.Fatalf("create_custom_tool: %s", toolResult)
	}
	tools := parseToolsFromResult(t, callMCPTool(t, c, "list_custom_tools", nil))
	callMCPTool(t, c, "delete_custom_tool", map[string]any{"id": tools[0].ID})
}

func TestIntegration_InstanceLifecycle(t *testing.T) {
	d, c := newInProcessClient(t)

	// list should be empty initially
	result := callMCPTool(t, c, "list_instances", nil)
	if !contains(result, "[]") {
		t.Fatalf("list_instances should be empty: %s", result)
	}

	// get non-existent
	getResult := callMCPTool(t, c, "get_instance", map[string]any{"id": "does-not-exist"})
	if !contains(getResult, "not found") {
		t.Fatalf("get non-existent instance: %s", getResult)
	}

	// start_instance needs agent service (not available in test)
	tmpl := &model.Template{Name: "inst-tmpl"}
	d.store.CreateTemplate(tmpl)
	agent := &model.Agent{Name: "inst-agent", TemplateID: tmpl.ID}
	d.store.CreateAgent(agent)

	startResult := callMCPTool(t, c, "start_instance", map[string]any{"agent_id": agent.ID})
	if !contains(startResult, "not available") {
		t.Fatalf("start_instance should fail: %s", startResult)
	}
}

func parseAgentsFromResult(t *testing.T, result string) []model.Agent {
	t.Helper()
	return unmarshalList[model.Agent](t, result)
}

func parsePromptsFromResult(t *testing.T, result string) []model.Prompt {
	t.Helper()
	return unmarshalList[model.Prompt](t, result)
}

func parseTemplatesFromResult(t *testing.T, result string) []model.Template {
	t.Helper()
	return unmarshalList[model.Template](t, result)
}

func parseProviderConfigsFromResult(t *testing.T, result string) []model.ProviderConfig {
	t.Helper()
	return unmarshalList[model.ProviderConfig](t, result)
}

func parseAgentTeamsFromResult(t *testing.T, result string) []model.AgentTeam {
	t.Helper()
	return unmarshalList[model.AgentTeam](t, result)
}

func parseResourcesFromResult(t *testing.T, result string) []model.Resource {
	t.Helper()
	return unmarshalList[model.Resource](t, result)
}

func parseSkillsFromResult(t *testing.T, result string) []model.Skill {
	t.Helper()
	return unmarshalList[model.Skill](t, result)
}

func parseToolsFromResult(t *testing.T, result string) []model.Tool {
	t.Helper()
	return unmarshalList[model.Tool](t, result)
}
