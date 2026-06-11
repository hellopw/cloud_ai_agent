package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"cloud_ai_agent/internal/model"
	"cloud_ai_agent/internal/service"
	"cloud_ai_agent/internal/store"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type deps struct {
	store    *store.Store
	agentSvc *service.AgentService
}

func NewServer(s *store.Store, agentSvc *service.AgentService) *server.MCPServer {
	d := &deps{store: s, agentSvc: agentSvc}

	mcpServer := server.NewMCPServer(
		"cloud-ai-agent",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithLogging(),
	)

	registerAgentTools(mcpServer, d)
	registerInstanceTools(mcpServer, d)
	registerTemplateTools(mcpServer, d)
	registerPromptTools(mcpServer, d)
	registerSkillTools(mcpServer, d)
	registerCustomToolTools(mcpServer, d)
	registerAgentTeamTools(mcpServer, d)
	registerProviderConfigTools(mcpServer, d)
	registerResourceTools(mcpServer, d)
	registerHealthTool(mcpServer, d)

	return mcpServer
}

func NewSSEServer(mcpServer *server.MCPServer) *server.SSEServer {
	return server.NewSSEServer(mcpServer)
}

func formatJSON(v interface{}) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%+v", v)
	}
	return string(data)
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(text),
		},
	}
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			mcp.NewTextContent(msg),
		},
	}
}

// --- Agents ---

func registerAgentTools(s *server.MCPServer, d *deps) {
	s.AddTool(mcp.NewTool("list_agents",
		mcp.WithDescription("List all agents"),
	), handleListAgents(d))

	s.AddTool(mcp.NewTool("get_agent",
		mcp.WithDescription("Get a single agent by ID"),
		mcp.WithString("id", mcp.Description("Agent ID"), mcp.Required()),
	), handleGetAgent(d))

	s.AddTool(mcp.NewTool("create_agent",
		mcp.WithDescription("Create a new agent"),
		mcp.WithString("name", mcp.Description("Agent name"), mcp.Required()),
		mcp.WithString("template_id", mcp.Description("Template ID"), mcp.Required()),
		mcp.WithString("repo_url", mcp.Description("Git repository URL")),
		mcp.WithString("branch", mcp.Description("Git branch"), mcp.DefaultString("main")),
		mcp.WithString("git_username", mcp.Description("Git username for private repos")),
		mcp.WithString("git_password", mcp.Description("Git password/token for private repos")),
	), handleCreateAgent(d))

	s.AddTool(mcp.NewTool("update_agent",
		mcp.WithDescription("Update an agent (only draft/failed status)"),
		mcp.WithString("id", mcp.Description("Agent ID"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Agent name")),
		mcp.WithString("template_id", mcp.Description("Template ID")),
		mcp.WithString("repo_url", mcp.Description("Git repository URL")),
		mcp.WithString("branch", mcp.Description("Git branch")),
		mcp.WithString("git_username", mcp.Description("Git username for private repos")),
		mcp.WithString("git_password", mcp.Description("Git password/token for private repos")),
	), handleUpdateAgent(d))

	s.AddTool(mcp.NewTool("delete_agent",
		mcp.WithDescription("Delete an agent"),
		mcp.WithString("id", mcp.Description("Agent ID"), mcp.Required()),
	), handleDeleteAgent(d))

	s.AddTool(mcp.NewTool("build_agent",
		mcp.WithDescription("Build agent Docker image asynchronously"),
		mcp.WithString("id", mcp.Description("Agent ID"), mcp.Required()),
	), handleBuildAgent(d))

	s.AddTool(mcp.NewTool("get_build_log",
		mcp.WithDescription("Get agent build log"),
		mcp.WithString("id", mcp.Description("Agent ID"), mcp.Required()),
	), handleGetBuildLog(d))
}

func handleListAgents(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		agents, err := d.store.ListAgents()
		if err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult(formatJSON(agents)), nil
	}
}

func handleGetAgent(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		a, err := d.store.GetAgent(id)
		if err != nil {
			return errorResult(err.Error()), nil
		}
		if a == nil {
			return errorResult("agent not found"), nil
		}
		return textResult(formatJSON(a)), nil
	}
}

func handleCreateAgent(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := request.RequireString("name")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		templateID, err := request.RequireString("template_id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		a := &model.Agent{
			Name:        name,
			TemplateID:  templateID,
			RepoURL:     request.GetString("repo_url", ""),
			Branch:      request.GetString("branch", "main"),
			GitUsername: request.GetString("git_username", ""),
			GitPassword: request.GetString("git_password", ""),
		}
		if err := d.store.CreateAgent(a); err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult(formatJSON(a)), nil
	}
}

func handleUpdateAgent(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		existing, err := d.store.GetAgent(id)
		if err != nil || existing == nil {
			return errorResult("agent not found"), nil
		}
		if existing.Status != "draft" && existing.Status != "failed" {
			return errorResult("can only edit draft or failed agents"), nil
		}
		a := *existing
		if v := request.GetString("name", ""); v != "" {
			a.Name = v
		}
		if v := request.GetString("template_id", ""); v != "" {
			a.TemplateID = v
		}
		if v := request.GetString("repo_url", ""); v != "" {
			a.RepoURL = v
		}
		if v := request.GetString("branch", ""); v != "" {
			a.Branch = v
		}
		if v := request.GetString("git_username", ""); v != "" {
			a.GitUsername = v
		}
		if v := request.GetString("git_password", ""); v != "" {
			a.GitPassword = v
		}
		if err := d.store.UpdateAgent(&a); err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult(formatJSON(a)), nil
	}
}

func handleDeleteAgent(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		if err := d.store.DeleteAgent(id); err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult("agent deleted"), nil
	}
}

func handleBuildAgent(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		if d.agentSvc == nil {
			return errorResult("agent service not available"), nil
		}
		go d.agentSvc.BuildAgent(context.Background(), id)
		return textResult("build started"), nil
	}
}

func handleGetBuildLog(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		logPath := filepath.Join("..", "builds", id, "build.log")
		if root := os.Getenv("PROJECT_ROOT"); root != "" {
			logPath = filepath.Join(root, "builds", id, "build.log")
		}
		data, err := os.ReadFile(logPath)
		if err != nil {
			return errorResult("build log not found"), nil
		}
		return textResult(string(data)), nil
	}
}

// --- Instances ---

func registerInstanceTools(s *server.MCPServer, d *deps) {
	s.AddTool(mcp.NewTool("list_instances",
		mcp.WithDescription("List all instances"),
	), handleListInstances(d))

	s.AddTool(mcp.NewTool("get_instance",
		mcp.WithDescription("Get a single instance by ID"),
		mcp.WithString("id", mcp.Description("Instance ID"), mcp.Required()),
	), handleGetInstance(d))

	s.AddTool(mcp.NewTool("start_instance",
		mcp.WithDescription("Start an agent container instance"),
		mcp.WithString("agent_id", mcp.Description("Agent ID"), mcp.Required()),
	), handleStartInstance(d))

	s.AddTool(mcp.NewTool("delete_instance",
		mcp.WithDescription("Delete an instance"),
		mcp.WithString("id", mcp.Description("Instance ID"), mcp.Required()),
	), handleDeleteInstance(d))
}

func handleListInstances(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		instances, err := d.store.ListInstances()
		if err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult(formatJSON(instances)), nil
	}
}

func handleGetInstance(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		i, err := d.store.GetInstance(id)
		if err != nil {
			return errorResult(err.Error()), nil
		}
		if i == nil {
			return errorResult("instance not found"), nil
		}
		return textResult(formatJSON(i)), nil
	}
}

func handleStartInstance(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		agentID, err := request.RequireString("agent_id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		if d.agentSvc == nil {
			return errorResult("agent service not available"), nil
		}
		instance, err := d.agentSvc.StartInstance(ctx, agentID)
		if err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult(formatJSON(instance)), nil
	}
}

func handleDeleteInstance(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		if err := d.store.DeleteInstance(id); err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult("instance deleted"), nil
	}
}

// --- Templates ---

func registerTemplateTools(s *server.MCPServer, d *deps) {
	s.AddTool(mcp.NewTool("list_templates",
		mcp.WithDescription("List all templates"),
	), handleListTemplates(d))

	s.AddTool(mcp.NewTool("get_template",
		mcp.WithDescription("Get a template by ID including associated prompts/skills/tools"),
		mcp.WithString("id", mcp.Description("Template ID"), mcp.Required()),
	), handleGetTemplate(d))

	s.AddTool(mcp.NewTool("create_template",
		mcp.WithDescription("Create a new template"),
		mcp.WithString("name", mcp.Description("Template name"), mcp.Required()),
		mcp.WithString("description", mcp.Description("Template description")),
		mcp.WithString("dockerfile_content", mcp.Description("Custom Dockerfile content")),
	), handleCreateTemplate(d))

	s.AddTool(mcp.NewTool("update_template",
		mcp.WithDescription("Update a template"),
		mcp.WithString("id", mcp.Description("Template ID"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Template name")),
		mcp.WithString("description", mcp.Description("Template description")),
		mcp.WithString("dockerfile_content", mcp.Description("Custom Dockerfile content")),
	), handleUpdateTemplate(d))

	s.AddTool(mcp.NewTool("delete_template",
		mcp.WithDescription("Delete a template"),
		mcp.WithString("id", mcp.Description("Template ID"), mcp.Required()),
	), handleDeleteTemplate(d))

	s.AddTool(mcp.NewTool("update_template_bindings",
		mcp.WithDescription("Update template prompt/skill/tool bindings"),
		mcp.WithString("id", mcp.Description("Template ID"), mcp.Required()),
		mcp.WithString("prompt_ids", mcp.Description("Comma-separated prompt IDs")),
		mcp.WithString("skill_ids", mcp.Description("Comma-separated skill IDs")),
		mcp.WithString("tool_ids", mcp.Description("Comma-separated tool IDs")),
	), handleUpdateTemplateBindings(d))
}

func handleListTemplates(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		templates, err := d.store.ListTemplates()
		if err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult(formatJSON(templates)), nil
	}
}

func handleGetTemplate(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		t, err := d.store.GetTemplate(id)
		if err != nil {
			return errorResult(err.Error()), nil
		}
		if t == nil {
			return errorResult("template not found"), nil
		}
		return textResult(formatJSON(t)), nil
	}
}

func handleCreateTemplate(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := request.RequireString("name")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		t := &model.Template{
			Name:              name,
			Description:       request.GetString("description", ""),
			DockerfileContent: request.GetString("dockerfile_content", ""),
		}
		if err := d.store.CreateTemplate(t); err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult(formatJSON(t)), nil
	}
}

func handleUpdateTemplate(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		existing, err := d.store.GetTemplate(id)
		if err != nil || existing == nil {
			return errorResult("template not found"), nil
		}
		t := *existing
		if v := request.GetString("name", ""); v != "" {
			t.Name = v
		}
		if v := request.GetString("description", ""); v != "" {
			t.Description = v
		}
		if v := request.GetString("dockerfile_content", ""); v != "" {
			t.DockerfileContent = v
		}
		if err := d.store.UpdateTemplate(&t); err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult(formatJSON(t)), nil
	}
}

func handleDeleteTemplate(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		if err := d.store.DeleteTemplate(id); err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult("template deleted"), nil
	}
}

func handleUpdateTemplateBindings(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		promptIDs := parseStringSlice(request.GetString("prompt_ids", ""))
		skillIDs := parseStringSlice(request.GetString("skill_ids", ""))
		toolIDs := parseStringSlice(request.GetString("tool_ids", ""))
		if err := d.store.UpdateTemplateBindings(id, promptIDs, skillIDs, toolIDs); err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult("bindings updated"), nil
	}
}

// --- Prompts ---

func registerPromptTools(s *server.MCPServer, d *deps) {
	s.AddTool(mcp.NewTool("list_prompts",
		mcp.WithDescription("List all prompts"),
	), handleListPrompts(d))

	s.AddTool(mcp.NewTool("get_prompt",
		mcp.WithDescription("Get a prompt by ID"),
		mcp.WithString("id", mcp.Description("Prompt ID"), mcp.Required()),
	), handleGetPrompt(d))

	s.AddTool(mcp.NewTool("create_prompt",
		mcp.WithDescription("Create a new prompt"),
		mcp.WithString("name", mcp.Description("Prompt name"), mcp.Required()),
		mcp.WithString("description", mcp.Description("Prompt description")),
		mcp.WithString("content", mcp.Description("Prompt content"), mcp.Required()),
	), handleCreatePrompt(d))

	s.AddTool(mcp.NewTool("update_prompt",
		mcp.WithDescription("Update a prompt"),
		mcp.WithString("id", mcp.Description("Prompt ID"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Prompt name")),
		mcp.WithString("description", mcp.Description("Prompt description")),
		mcp.WithString("content", mcp.Description("Prompt content")),
	), handleUpdatePrompt(d))

	s.AddTool(mcp.NewTool("delete_prompt",
		mcp.WithDescription("Delete a prompt"),
		mcp.WithString("id", mcp.Description("Prompt ID"), mcp.Required()),
	), handleDeletePrompt(d))
}

func handleListPrompts(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		prompts, err := d.store.ListPrompts()
		if err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult(formatJSON(prompts)), nil
	}
}

func handleGetPrompt(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		p, err := d.store.GetPrompt(id)
		if err != nil {
			return errorResult(err.Error()), nil
		}
		if p == nil {
			return errorResult("prompt not found"), nil
		}
		return textResult(formatJSON(p)), nil
	}
}

func handleCreatePrompt(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := request.RequireString("name")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		content, err := request.RequireString("content")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		p := &model.Prompt{
			Name:        name,
			Description: request.GetString("description", ""),
			Content:     content,
		}
		if err := d.store.CreatePrompt(p); err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult(formatJSON(p)), nil
	}
}

func handleUpdatePrompt(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		existing, err := d.store.GetPrompt(id)
		if err != nil || existing == nil {
			return errorResult("prompt not found"), nil
		}
		p := *existing
		if v := request.GetString("name", ""); v != "" {
			p.Name = v
		}
		if v := request.GetString("description", ""); v != "" {
			p.Description = v
		}
		if v := request.GetString("content", ""); v != "" {
			p.Content = v
		}
		if err := d.store.UpdatePrompt(&p); err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult(formatJSON(p)), nil
	}
}

func handleDeletePrompt(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		if err := d.store.DeletePrompt(id); err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult("prompt deleted"), nil
	}
}

// --- Skills ---

func registerSkillTools(s *server.MCPServer, d *deps) {
	s.AddTool(mcp.NewTool("list_skills",
		mcp.WithDescription("List all skills"),
	), handleListSkills(d))

	s.AddTool(mcp.NewTool("get_skill",
		mcp.WithDescription("Get a skill by ID"),
		mcp.WithString("id", mcp.Description("Skill ID"), mcp.Required()),
	), handleGetSkill(d))

	s.AddTool(mcp.NewTool("create_skill",
		mcp.WithDescription("Create a new skill"),
		mcp.WithString("name", mcp.Description("Skill name"), mcp.Required()),
		mcp.WithString("description", mcp.Description("Skill description")),
		mcp.WithString("content", mcp.Description("Skill content"), mcp.Required()),
	), handleCreateSkill(d))

	s.AddTool(mcp.NewTool("update_skill",
		mcp.WithDescription("Update a skill"),
		mcp.WithString("id", mcp.Description("Skill ID"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Skill name")),
		mcp.WithString("description", mcp.Description("Skill description")),
		mcp.WithString("content", mcp.Description("Skill content")),
	), handleUpdateSkill(d))

	s.AddTool(mcp.NewTool("delete_skill",
		mcp.WithDescription("Delete a skill"),
		mcp.WithString("id", mcp.Description("Skill ID"), mcp.Required()),
	), handleDeleteSkill(d))
}

func handleListSkills(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		skills, err := d.store.ListSkills()
		if err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult(formatJSON(skills)), nil
	}
}

func handleGetSkill(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		sk, err := d.store.GetSkill(id)
		if err != nil {
			return errorResult(err.Error()), nil
		}
		if sk == nil {
			return errorResult("skill not found"), nil
		}
		return textResult(formatJSON(sk)), nil
	}
}

func handleCreateSkill(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := request.RequireString("name")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		content, err := request.RequireString("content")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		sk := &model.Skill{
			Name:        name,
			Description: request.GetString("description", ""),
			Content:     content,
		}
		if err := d.store.CreateSkill(sk); err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult(formatJSON(sk)), nil
	}
}

func handleUpdateSkill(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		existing, err := d.store.GetSkill(id)
		if err != nil || existing == nil {
			return errorResult("skill not found"), nil
		}
		sk := *existing
		if v := request.GetString("name", ""); v != "" {
			sk.Name = v
		}
		if v := request.GetString("description", ""); v != "" {
			sk.Description = v
		}
		if v := request.GetString("content", ""); v != "" {
			sk.Content = v
		}
		if err := d.store.UpdateSkill(&sk); err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult(formatJSON(sk)), nil
	}
}

func handleDeleteSkill(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		if err := d.store.DeleteSkill(id); err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult("skill deleted"), nil
	}
}

// --- Custom Tools ---

func registerCustomToolTools(s *server.MCPServer, d *deps) {
	s.AddTool(mcp.NewTool("list_custom_tools",
		mcp.WithDescription("List all custom tools"),
	), handleListCustomTools(d))

	s.AddTool(mcp.NewTool("get_custom_tool",
		mcp.WithDescription("Get a custom tool by ID"),
		mcp.WithString("id", mcp.Description("Tool ID"), mcp.Required()),
	), handleGetCustomTool(d))

	s.AddTool(mcp.NewTool("create_custom_tool",
		mcp.WithDescription("Create a new custom tool"),
		mcp.WithString("name", mcp.Description("Tool name"), mcp.Required()),
		mcp.WithString("label", mcp.Description("Tool label")),
		mcp.WithString("description", mcp.Description("Tool description")),
		mcp.WithString("dsl_definition", mcp.Description("DSL definition JSON")),
	), handleCreateCustomTool(d))

	s.AddTool(mcp.NewTool("update_custom_tool",
		mcp.WithDescription("Update a custom tool"),
		mcp.WithString("id", mcp.Description("Tool ID"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Tool name")),
		mcp.WithString("label", mcp.Description("Tool label")),
		mcp.WithString("description", mcp.Description("Tool description")),
		mcp.WithString("dsl_definition", mcp.Description("DSL definition JSON")),
	), handleUpdateCustomTool(d))

	s.AddTool(mcp.NewTool("delete_custom_tool",
		mcp.WithDescription("Delete a custom tool"),
		mcp.WithString("id", mcp.Description("Tool ID"), mcp.Required()),
	), handleDeleteCustomTool(d))
}

func handleListCustomTools(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		tools, err := d.store.ListTools()
		if err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult(formatJSON(tools)), nil
	}
}

func handleGetCustomTool(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		t, err := d.store.GetTool(id)
		if err != nil {
			return errorResult(err.Error()), nil
		}
		if t == nil {
			return errorResult("tool not found"), nil
		}
		return textResult(formatJSON(t)), nil
	}
}

func handleCreateCustomTool(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := request.RequireString("name")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		t := &model.Tool{
			Name:          name,
			Label:         request.GetString("label", ""),
			Description:   request.GetString("description", ""),
			DSLDefinition: request.GetString("dsl_definition", "{}"),
		}
		if err := d.store.CreateTool(t); err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult(formatJSON(t)), nil
	}
}

func handleUpdateCustomTool(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		existing, err := d.store.GetTool(id)
		if err != nil || existing == nil {
			return errorResult("tool not found"), nil
		}
		t := *existing
		if v := request.GetString("name", ""); v != "" {
			t.Name = v
		}
		if v := request.GetString("label", ""); v != "" {
			t.Label = v
		}
		if v := request.GetString("description", ""); v != "" {
			t.Description = v
		}
		if v := request.GetString("dsl_definition", ""); v != "" {
			t.DSLDefinition = v
		}
		if err := d.store.UpdateTool(&t); err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult(formatJSON(t)), nil
	}
}

func handleDeleteCustomTool(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		if err := d.store.DeleteTool(id); err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult("tool deleted"), nil
	}
}

// --- Agent Teams ---

func registerAgentTeamTools(s *server.MCPServer, d *deps) {
	s.AddTool(mcp.NewTool("list_agent_teams",
		mcp.WithDescription("List all agent teams"),
	), handleListAgentTeams(d))

	s.AddTool(mcp.NewTool("get_agent_team",
		mcp.WithDescription("Get an agent team by ID including members"),
		mcp.WithString("id", mcp.Description("Agent Team ID"), mcp.Required()),
	), handleGetAgentTeam(d))

	s.AddTool(mcp.NewTool("create_agent_team",
		mcp.WithDescription("Create a new agent team"),
		mcp.WithString("name", mcp.Description("Team name"), mcp.Required()),
		mcp.WithString("template_id", mcp.Description("Template ID"), mcp.Required()),
		mcp.WithString("repo_url", mcp.Description("Git repository URL")),
		mcp.WithString("branch", mcp.Description("Git branch"), mcp.DefaultString("main")),
		mcp.WithString("git_username", mcp.Description("Git username for private repos")),
		mcp.WithString("git_password", mcp.Description("Git password/token for private repos")),
	), handleCreateAgentTeam(d))

	s.AddTool(mcp.NewTool("update_agent_team",
		mcp.WithDescription("Update an agent team"),
		mcp.WithString("id", mcp.Description("Agent Team ID"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Team name")),
		mcp.WithString("template_id", mcp.Description("Template ID")),
		mcp.WithString("repo_url", mcp.Description("Git repository URL")),
		mcp.WithString("branch", mcp.Description("Git branch")),
		mcp.WithString("git_username", mcp.Description("Git username for private repos")),
		mcp.WithString("git_password", mcp.Description("Git password/token for private repos")),
	), handleUpdateAgentTeam(d))

	s.AddTool(mcp.NewTool("delete_agent_team",
		mcp.WithDescription("Delete an agent team"),
		mcp.WithString("id", mcp.Description("Agent Team ID"), mcp.Required()),
	), handleDeleteAgentTeam(d))

	s.AddTool(mcp.NewTool("build_agent_team",
		mcp.WithDescription("Build agent team Docker image asynchronously"),
		mcp.WithString("id", mcp.Description("Agent Team ID"), mcp.Required()),
	), handleBuildAgentTeam(d))

	s.AddTool(mcp.NewTool("get_team_build_log",
		mcp.WithDescription("Get agent team build log"),
		mcp.WithString("id", mcp.Description("Agent Team ID"), mcp.Required()),
	), handleGetTeamBuildLog(d))

	s.AddTool(mcp.NewTool("start_team_instance",
		mcp.WithDescription("Start an agent team instance"),
		mcp.WithString("team_id", mcp.Description("Agent Team ID"), mcp.Required()),
	), handleStartTeamInstance(d))
}

func handleListAgentTeams(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		teams, err := d.store.ListAgentTeams()
		if err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult(formatJSON(teams)), nil
	}
}

func handleGetAgentTeam(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		t, err := d.store.GetAgentTeam(id)
		if err != nil {
			return errorResult(err.Error()), nil
		}
		if t == nil {
			return errorResult("agent team not found"), nil
		}
		return textResult(formatJSON(t)), nil
	}
}

func handleCreateAgentTeam(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := request.RequireString("name")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		templateID, err := request.RequireString("template_id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		t := &model.AgentTeam{
			Name:        name,
			TemplateID:  templateID,
			RepoURL:     request.GetString("repo_url", ""),
			Branch:      request.GetString("branch", "main"),
			GitUsername: request.GetString("git_username", ""),
			GitPassword: request.GetString("git_password", ""),
		}
		if err := d.store.CreateAgentTeam(t); err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult(formatJSON(t)), nil
	}
}

func handleUpdateAgentTeam(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		existing, err := d.store.GetAgentTeam(id)
		if err != nil || existing == nil {
			return errorResult("agent team not found"), nil
		}
		if existing.Status != "draft" && existing.Status != "failed" {
			return errorResult("can only edit draft or failed teams"), nil
		}
		t := *existing
		if v := request.GetString("name", ""); v != "" {
			t.Name = v
		}
		if v := request.GetString("template_id", ""); v != "" {
			t.TemplateID = v
		}
		if v := request.GetString("repo_url", ""); v != "" {
			t.RepoURL = v
		}
		if v := request.GetString("branch", ""); v != "" {
			t.Branch = v
		}
		if v := request.GetString("git_username", ""); v != "" {
			t.GitUsername = v
		}
		if v := request.GetString("git_password", ""); v != "" {
			t.GitPassword = v
		}
		if err := d.store.UpdateAgentTeam(&t); err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult(formatJSON(t)), nil
	}
}

func handleDeleteAgentTeam(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		if err := d.store.DeleteAgentTeam(id); err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult("agent team deleted"), nil
	}
}

func handleBuildAgentTeam(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		if d.agentSvc == nil {
			return errorResult("agent service not available"), nil
		}
		go d.agentSvc.BuildAgentTeam(context.Background(), id)
		return textResult("build started"), nil
	}
}

func handleGetTeamBuildLog(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		logPath := filepath.Join("..", "builds", id, "build.log")
		if root := os.Getenv("PROJECT_ROOT"); root != "" {
			logPath = filepath.Join(root, "builds", id, "build.log")
		}
		data, err := os.ReadFile(logPath)
		if err != nil {
			return errorResult("build log not found"), nil
		}
		return textResult(string(data)), nil
	}
}

func handleStartTeamInstance(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		teamID, err := request.RequireString("team_id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		if d.agentSvc == nil {
			return errorResult("agent service not available"), nil
		}
		instance, err := d.agentSvc.StartTeamInstance(ctx, teamID)
		if err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult(formatJSON(instance)), nil
	}
}

// --- Provider Configs ---

func registerProviderConfigTools(s *server.MCPServer, d *deps) {
	s.AddTool(mcp.NewTool("list_provider_configs",
		mcp.WithDescription("List all provider configs (api_key masked)"),
	), handleListProviderConfigs(d))

	s.AddTool(mcp.NewTool("get_provider_config",
		mcp.WithDescription("Get a provider config by ID (api_key masked)"),
		mcp.WithString("id", mcp.Description("Provider Config ID"), mcp.Required()),
	), handleGetProviderConfig(d))

	s.AddTool(mcp.NewTool("create_provider_config",
		mcp.WithDescription("Create a new provider config"),
		mcp.WithString("name", mcp.Description("Config name"), mcp.Required()),
		mcp.WithString("provider", mcp.Description("Provider type"), mcp.Required()),
		mcp.WithString("model_id", mcp.Description("Model ID"), mcp.Required()),
		mcp.WithString("api_key", mcp.Description("API key"), mcp.Required()),
		mcp.WithString("base_url", mcp.Description("Base URL")),
	), handleCreateProviderConfig(d))

	s.AddTool(mcp.NewTool("update_provider_config",
		mcp.WithDescription("Update a provider config"),
		mcp.WithString("id", mcp.Description("Provider Config ID"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Config name")),
		mcp.WithString("provider", mcp.Description("Provider type")),
		mcp.WithString("model_id", mcp.Description("Model ID")),
		mcp.WithString("api_key", mcp.Description("API key")),
		mcp.WithString("base_url", mcp.Description("Base URL")),
	), handleUpdateProviderConfig(d))

	s.AddTool(mcp.NewTool("delete_provider_config",
		mcp.WithDescription("Delete a provider config"),
		mcp.WithString("id", mcp.Description("Provider Config ID"), mcp.Required()),
	), handleDeleteProviderConfig(d))
}

func maskProviderConfigs(configs []model.ProviderConfig) []model.ProviderConfig {
	result := make([]model.ProviderConfig, len(configs))
	for i, pc := range configs {
		result[i] = pc
		if pc.APIKey != "" {
			result[i].APIKey = "***"
		}
	}
	return result
}

func maskProviderConfig(pc *model.ProviderConfig) *model.ProviderConfig {
	if pc == nil {
		return nil
	}
	masked := *pc
	if masked.APIKey != "" {
		masked.APIKey = "***"
	}
	return &masked
}

func handleListProviderConfigs(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		configs, err := d.store.ListProviderConfigs()
		if err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult(formatJSON(maskProviderConfigs(configs))), nil
	}
}

func handleGetProviderConfig(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		pc, err := d.store.GetProviderConfig(id)
		if err != nil {
			return errorResult(err.Error()), nil
		}
		if pc == nil {
			return errorResult("provider config not found"), nil
		}
		return textResult(formatJSON(maskProviderConfig(pc))), nil
	}
}

func handleCreateProviderConfig(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := request.RequireString("name")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		provider, err := request.RequireString("provider")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		modelID, err := request.RequireString("model_id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		apiKey, err := request.RequireString("api_key")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		pc := &model.ProviderConfig{
			Name:     name,
			Provider: provider,
			ModelID:  modelID,
			APIKey:   apiKey,
			BaseURL:  request.GetString("base_url", ""),
		}
		if err := d.store.CreateProviderConfig(pc); err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult(formatJSON(maskProviderConfig(pc))), nil
	}
}

func handleUpdateProviderConfig(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		existing, err := d.store.GetProviderConfig(id)
		if err != nil || existing == nil {
			return errorResult("provider config not found"), nil
		}
		pc := *existing
		if v := request.GetString("name", ""); v != "" {
			pc.Name = v
		}
		if v := request.GetString("provider", ""); v != "" {
			pc.Provider = v
		}
		if v := request.GetString("model_id", ""); v != "" {
			pc.ModelID = v
		}
		if v := request.GetString("api_key", ""); v != "" {
			pc.APIKey = v
		}
		if v := request.GetString("base_url", ""); v != "" {
			pc.BaseURL = v
		}
		if err := d.store.UpdateProviderConfig(&pc); err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult(formatJSON(maskProviderConfig(&pc))), nil
	}
}

func handleDeleteProviderConfig(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		if err := d.store.DeleteProviderConfig(id); err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult("provider config deleted"), nil
	}
}

// --- Resources ---

func registerResourceTools(s *server.MCPServer, d *deps) {
	s.AddTool(mcp.NewTool("list_resources",
		mcp.WithDescription("List all resources"),
	), handleListResources(d))

	s.AddTool(mcp.NewTool("get_resource",
		mcp.WithDescription("Get a resource by ID"),
		mcp.WithString("id", mcp.Description("Resource ID"), mcp.Required()),
	), handleGetResource(d))

	s.AddTool(mcp.NewTool("create_resource",
		mcp.WithDescription("Create a new resource"),
		mcp.WithString("name", mcp.Description("Resource name"), mcp.Required()),
		mcp.WithString("type", mcp.Description("Resource type"), mcp.DefaultString("git")),
		mcp.WithString("config", mcp.Description("Resource config JSON")),
	), handleCreateResource(d))

	s.AddTool(mcp.NewTool("update_resource",
		mcp.WithDescription("Update a resource"),
		mcp.WithString("id", mcp.Description("Resource ID"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Resource name")),
		mcp.WithString("type", mcp.Description("Resource type")),
		mcp.WithString("config", mcp.Description("Resource config JSON")),
	), handleUpdateResource(d))

	s.AddTool(mcp.NewTool("delete_resource",
		mcp.WithDescription("Delete a resource"),
		mcp.WithString("id", mcp.Description("Resource ID"), mcp.Required()),
	), handleDeleteResource(d))
}

func handleListResources(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		resources, err := d.store.ListResources()
		if err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult(formatJSON(resources)), nil
	}
}

func handleGetResource(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		res, err := d.store.GetResource(id)
		if err != nil {
			return errorResult(err.Error()), nil
		}
		if res == nil {
			return errorResult("resource not found"), nil
		}
		return textResult(formatJSON(res)), nil
	}
}

func handleCreateResource(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := request.RequireString("name")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		res := &model.Resource{
			Name:   name,
			Type:   request.GetString("type", "git"),
			Config: request.GetString("config", "{}"),
		}
		if err := d.store.CreateResource(res); err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult(formatJSON(res)), nil
	}
}

func handleUpdateResource(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		existing, err := d.store.GetResource(id)
		if err != nil || existing == nil {
			return errorResult("resource not found"), nil
		}
		res := *existing
		if v := request.GetString("name", ""); v != "" {
			res.Name = v
		}
		if v := request.GetString("type", ""); v != "" {
			res.Type = v
		}
		if v := request.GetString("config", ""); v != "" {
			res.Config = v
		}
		if err := d.store.UpdateResource(&res); err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult(formatJSON(res)), nil
	}
}

func handleDeleteResource(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return errorResult(err.Error()), nil
		}
		if err := d.store.DeleteResource(id); err != nil {
			return errorResult(err.Error()), nil
		}
		return textResult("resource deleted"), nil
	}
}

// --- Health ---

func registerHealthTool(s *server.MCPServer, d *deps) {
	s.AddTool(mcp.NewTool("health",
		mcp.WithDescription("Check service health status"),
	), handleHealth(d))
}

func handleHealth(d *deps) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return textResult(`{"status":"ok"}`), nil
	}
}

// --- Helpers ---

func parseStringSlice(s string) []string {
	if s == "" {
		return nil
	}
	var result []string
	for _, part := range splitAndTrim(s, ",") {
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func splitAndTrim(s, sep string) []string {
	parts := make([]string, 0)
	current := ""
	for _, r := range s {
		if string(r) == sep {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(r)
		}
	}
	parts = append(parts, current)
	for i, p := range parts {
		// trim spaces
		start, end := 0, len(p)
		for start < end && (p[start] == ' ' || p[start] == '\t') {
			start++
		}
		for end > start && (p[end-1] == ' ' || p[end-1] == '\t') {
			end--
		}
		parts[i] = p[start:end]
	}
	return parts
}
