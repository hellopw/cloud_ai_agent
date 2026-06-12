package service

import (
	"context"
	"io"
	"fmt"
	"os"
	"path/filepath"

	"cloud_ai_agent/internal/codegen"
	dockersvc "cloud_ai_agent/internal/docker"
	gitsvc "cloud_ai_agent/internal/git"
	"cloud_ai_agent/internal/model"
	"cloud_ai_agent/internal/store"
	"encoding/json"
	"hash/fnv"
	"slices"
)

type AgentService struct {
	store       *store.Store
	docker      *dockersvc.Service
	git         *gitsvc.Service
	projectRoot string
}

func NewAgentService(s *store.Store) *AgentService {
	// PROJECT_ROOT env var overrides auto-detection
	projectRoot := os.Getenv("PROJECT_ROOT")
	if projectRoot == "" {
		projectRoot = ".."
		if _, err := os.Stat(filepath.Join(projectRoot, "container-wrapper", "src", "server.js")); os.IsNotExist(err) {
			projectRoot = "."
		}
	}
	// Configure codegen templates dir relative to project root
	codegen.TemplatesDir = filepath.Join(projectRoot, "container-wrapper", "dockerfiles")

	return &AgentService{
		store:       s,
		docker:      dockersvc.NewService(filepath.Join(projectRoot, "builds")),
		git:         gitsvc.NewService(),
		projectRoot: projectRoot,
	}
}

func (svc *AgentService) BuildAgent(ctx context.Context, agentID string) error {
	agent, err := svc.store.GetAgent(agentID)
	if err != nil || agent == nil {
		return fmt.Errorf("agent not found")
	}

	tmpl, err := svc.store.GetTemplate(agent.TemplateID)
	if err != nil || tmpl == nil {
		return fmt.Errorf("template not found")
	}

	svc.store.UpdateAgentStatus(agentID, "building", "", "")

	buildDir := filepath.Join(svc.projectRoot, "builds", agentID)
	os.RemoveAll(buildDir)
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		svc.store.UpdateAgentStatus(agentID, "failed", "", err.Error())
		return err
	}

	// Open build log file
	logPath := filepath.Join(buildDir, "build.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		svc.store.UpdateAgentStatus(agentID, "failed", "", err.Error())
		return err
	}
	defer logFile.Close()
	logWriter := io.MultiWriter(logFile, os.Stdout)

	// Clone code repository
	repoDir := filepath.Join(buildDir, "repo")
	if err := svc.git.Clone(ctx, agent.RepoURL, agent.Branch, repoDir, agent.GitUsername, agent.GitPassword, logWriter); err != nil {
		svc.store.UpdateAgentStatus(agentID, "failed", "", "git clone: "+err.Error())
		return err
	}

	// Generate Dockerfile
	dockerfile, err := codegen.GenerateDockerfileByAgentType(tmpl.AgentType, &codegen.DockerfileData{
		CustomContent: tmpl.DockerfileContent,
	})
	if err != nil {
		svc.store.UpdateAgentStatus(agentID, "failed", "", err.Error())
		return err
	}
	if err := os.WriteFile(filepath.Join(buildDir, "Dockerfile"), []byte(dockerfile), 0644); err != nil {
		svc.store.UpdateAgentStatus(agentID, "failed", "", err.Error())
		return err
	}

	// Copy wrapper based on agent type
	wrapperName := "server.js"
	switch tmpl.AgentType {
	case codegen.AgentTypeClaudeCode:
		wrapperName = "server-claude-code.js"
	case codegen.AgentTypeCodex:
		wrapperName = "server-codex.js"
	}
	wrapperSrc := filepath.Join(svc.projectRoot, "container-wrapper", "src", wrapperName)
	wrapperData, err := os.ReadFile(wrapperSrc)
	if err != nil {
		svc.store.UpdateAgentStatus(agentID, "failed", "", "read wrapper: "+err.Error())
		return err
	}
	if err := os.WriteFile(filepath.Join(buildDir, "server.js"), wrapperData, 0644); err != nil {
		svc.store.UpdateAgentStatus(agentID, "failed", "", err.Error())
		return err
	}
	// Copy MCP client to build context root
	mcpClientSrc := filepath.Join(svc.projectRoot, "container-wrapper", "src", "mcp-client.js")
	if mcpData, err := os.ReadFile(mcpClientSrc); err == nil {
		if err := os.WriteFile(filepath.Join(buildDir, "mcp-client.js"), mcpData, 0644); err != nil {
			svc.store.UpdateAgentStatus(agentID, "failed", "", err.Error())
			return err
		}
	}

	// Generate tool extensions
	if err := svc.writeToolExtensions(buildDir, tmpl.ToolIDs); err != nil {
		svc.store.UpdateAgentStatus(agentID, "failed", "", err.Error())
		return err
	}

	// Write prompts and skills
	if err := svc.writePromptsSkills(buildDir, tmpl.PromptIDs, tmpl.SkillIDs); err != nil {
		svc.store.UpdateAgentStatus(agentID, "failed", "", err.Error())
		return err
	}

	// Build Docker image
	imageTag := fmt.Sprintf("cloud-agent/%s:latest", agentID)
	if err := svc.docker.BuildImage(ctx, buildDir, imageTag, logWriter); err != nil {
		svc.store.UpdateAgentStatus(agentID, "failed", "", "docker build: "+err.Error())
		return err
	}

	svc.store.UpdateAgentStatus(agentID, "ready", imageTag, "")
	return nil
}

func (svc *AgentService) writeToolExtensions(buildDir string, toolIDs []string) error {
	extDir := filepath.Join(buildDir, "extensions")
	if err := os.MkdirAll(extDir, 0755); err != nil { return err }
	if len(toolIDs) == 0 { return nil }

	type extEntry struct {
		Name        string          `json:"name"`
		Label       string          `json:"label"`
		Description string          `json:"description"`
		Parameters  json.RawMessage `json:"parameters"`
		Handler     json.RawMessage `json:"handler"`
	}
	var manifest []extEntry

	for _, tid := range toolIDs {
		tool, err := svc.store.GetTool(tid)
		if err != nil { continue }
		tsCode, err := codegen.GenerateToolExtension(tool.DSLDefinition)
		if err != nil { continue }
		if err := os.WriteFile(filepath.Join(extDir, tool.Name+".ts"), []byte(tsCode), 0644); err != nil { return err }

		// Parse DSL to build manifest entry
		var dsl struct {
			Name        string          `json:"name"`
			Label       string          `json:"label"`
			Description string          `json:"description"`
			Parameters  json.RawMessage `json:"parameters"`
			Handler     json.RawMessage `json:"handler"`
		}
		if json.Unmarshal([]byte(tool.DSLDefinition), &dsl) == nil {
			var h struct{ Type string `json:"type"` }
			if json.Unmarshal(dsl.Handler, &h) == nil && h.Type == "mcp" {
				manifest = append(manifest, extEntry{
					Name:        dsl.Name,
					Label:       dsl.Label,
					Description: dsl.Description,
					Parameters:  dsl.Parameters,
					Handler:     dsl.Handler,
				})
			}
		}
	}

	if len(manifest) > 0 {
		manifestJSON, _ := json.Marshal(manifest)
		if err := os.WriteFile(filepath.Join(extDir, "extensions.json"), manifestJSON, 0644); err != nil {
			return err
		}
	}
	return nil
}

func (svc *AgentService) writePromptsSkills(buildDir string, promptIDs, skillIDs []string) error {
	promptsDir := filepath.Join(buildDir, "pi-prompts")
	os.MkdirAll(promptsDir, 0755)
	if len(promptIDs) > 0 {
		for _, pid := range promptIDs {
			p, err := svc.store.GetPrompt(pid)
			if err != nil || p == nil { continue }
			content := fmt.Sprintf("---\ndescription: %s\n---\n\n%s", p.Description, p.Content)
			os.WriteFile(filepath.Join(promptsDir, p.Name+".md"), []byte(content), 0644)
		}
	}
	skillsDir := filepath.Join(buildDir, "pi-skills")
	os.MkdirAll(skillsDir, 0755)
	if len(skillIDs) > 0 {
		for _, sid := range skillIDs {
			s, err := svc.store.GetSkill(sid)
			if err != nil || s == nil { continue }
			content := fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\n%s", s.Name, s.Description, s.Content)
			os.WriteFile(filepath.Join(skillsDir, s.Name+".md"), []byte(content), 0644)
		}
	}
	return nil
}

func (svc *AgentService) StartInstance(ctx context.Context, agentID, providerConfigID string) (*model.Instance, error) {
	agent, err := svc.store.GetAgent(agentID)
	if err != nil || agent == nil { return nil, fmt.Errorf("agent not found") }
	if agent.Status != "ready" { return nil, fmt.Errorf("agent not ready: %s", agent.Status) }

	instance := &model.Instance{AgentID: agentID}
	if err := svc.store.CreateInstance(instance); err != nil { return nil, err }

	port := svc.findFreePort(instance.ID)
	containerName := fmt.Sprintf("cloud-agent-%s", instance.ID[:12])

	// Create log directory for this instance
	logDir := filepath.Join(svc.projectRoot, "builds", agentID, "llm-logs", instance.ID)
	os.MkdirAll(logDir, 0777)

	go func() {
		ctx := context.Background()
		env := map[string]string{
			"AGENT_PROVIDER": os.Getenv("AGENT_PROVIDER"),
			"AGENT_MODEL":    os.Getenv("AGENT_MODEL"),
			"AGENT_API_KEY":  os.Getenv("AGENT_API_KEY"),
			"AGENT_BASE_URL": os.Getenv("AGENT_BASE_URL"),
			"LOGS_DIR":       "/logs",
			"INSTANCE_ID":    instance.ID,
		}
		// Provider config from request overrides env vars
		if providerConfigID != "" {
			pc, err := svc.store.GetProviderConfig(providerConfigID)
			if err == nil && pc != nil {
				env["AGENT_PROVIDER"] = pc.Provider
				env["AGENT_MODEL"] = pc.ModelID
				if pc.APIKey != "" {
					env["AGENT_API_KEY"] = pc.APIKey
				}
				if pc.BaseURL != "" {
					env["AGENT_BASE_URL"] = pc.BaseURL
				}
			}
		}
		repoAbs, _ := filepath.Abs(filepath.Join(svc.projectRoot, "builds", agentID, "repo"))
		logAbs, _ := filepath.Abs(filepath.Join(svc.projectRoot, "builds", agentID, "llm-logs"))
		_, err := svc.docker.RunContainer(ctx, agent.ImageTag, containerName,
			repoAbs, logAbs, port, env)
		if err != nil {
			svc.store.UpdateInstanceStatus(instance.ID, "error", "", 0, err.Error())
			return
		}
		svc.store.UpdateInstanceStatus(instance.ID, "running", containerName, port, "")
	}()

	return instance, nil
}

func (svc *AgentService) BuildAgentTeam(ctx context.Context, teamID string) error {
	team, err := svc.store.GetAgentTeam(teamID)
	if err != nil || team == nil {
		return fmt.Errorf("team not found")
	}

	svc.store.UpdateAgentTeamStatus(teamID, "building", "", "")

	buildDir := filepath.Join(svc.projectRoot, "builds", teamID)
	os.RemoveAll(buildDir)
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		svc.store.UpdateAgentTeamStatus(teamID, "failed", "", err.Error())
		return err
	}

	logPath := filepath.Join(buildDir, "build.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		svc.store.UpdateAgentTeamStatus(teamID, "failed", "", err.Error())
		return err
	}
	defer logFile.Close()
	logWriter := io.MultiWriter(logFile, os.Stdout)

	// Clone code repository
	repoDir := filepath.Join(buildDir, "repo")
	if err := svc.git.Clone(ctx, team.RepoURL, team.Branch, repoDir, team.GitUsername, team.GitPassword, logWriter); err != nil {
		svc.store.UpdateAgentTeamStatus(teamID, "failed", "", "git clone: "+err.Error())
		return err
	}

	// Generate team Dockerfile
	agentDirs := make([]string, 0, len(team.Members))
	for _, m := range team.Members {
		agentDirs = append(agentDirs, filepath.Join("agents", m.Name))
	}

	var teamPromptsDir, teamSkillsDir string
	if len(team.PromptIDs) > 0 {
		teamPromptsDir = "team-prompts"
	}
	if len(team.SkillIDs) > 0 {
		teamSkillsDir = "team-skills"
	}

	dockerfile, err := codegen.GenerateTeamDockerfile(&codegen.TeamDockerfileData{
		AgentDirs:      agentDirs,
		TeamPromptsDir: teamPromptsDir,
		TeamSkillsDir:  teamSkillsDir,
	})
	if err != nil {
		svc.store.UpdateAgentTeamStatus(teamID, "failed", "", err.Error())
		return err
	}
	if err := os.WriteFile(filepath.Join(buildDir, "Dockerfile"), []byte(dockerfile), 0644); err != nil {
		svc.store.UpdateAgentTeamStatus(teamID, "failed", "", err.Error())
		return err
	}

	// Copy team-server.js to build context
	wrapperSrc := filepath.Join(svc.projectRoot, "container-wrapper", "src", "team-server.js")
	if _, statErr := os.Stat(wrapperSrc); os.IsNotExist(statErr) {
		svc.store.UpdateAgentTeamStatus(teamID, "failed", "", "team-server.js not found")
		return fmt.Errorf("team-server.js not found")
	}
	wrapperData, err := os.ReadFile(wrapperSrc)
	if err != nil {
		svc.store.UpdateAgentTeamStatus(teamID, "failed", "", "read team-server.js: "+err.Error())
		return err
	}
	if err := os.WriteFile(filepath.Join(buildDir, "team-server.js"), wrapperData, 0644); err != nil {
		svc.store.UpdateAgentTeamStatus(teamID, "failed", "", err.Error())
		return err
	}

	// Copy MCP client
	mcpClientSrc := filepath.Join(svc.projectRoot, "container-wrapper", "src", "mcp-client.js")
	if mcpData, err := os.ReadFile(mcpClientSrc); err == nil {
		os.WriteFile(filepath.Join(buildDir, "mcp-client.js"), mcpData, 0644)
	}

	// Build manifest with member configs
	type manifestMember struct {
		Name                 string `json:"name"`
		Role                 string `json:"role"`
		Provider             string `json:"provider"`
		ModelID              string `json:"model_id"`
		APIKey               string `json:"api_key"`
		BaseURL              string `json:"base_url"`
		SystemPromptOverride string `json:"system_prompt_override,omitempty"`
	}
	type manifest struct {
		TeamName       string           `json:"team_name"`
		Members        []manifestMember `json:"members"`
		TeamPromptsDir string           `json:"team_prompts_dir,omitempty"`
		TeamSkillsDir  string           `json:"team_skills_dir,omitempty"`
	}

	mf := manifest{TeamName: team.Name, TeamPromptsDir: teamPromptsDir, TeamSkillsDir: teamSkillsDir}

	for _, m := range team.Members {
		memberDir := filepath.Join(buildDir, "agents", m.Name)
		os.MkdirAll(memberDir, 0755)

		tmpl, err := svc.store.GetTemplate(m.AgentTemplateID)
		if err != nil || tmpl == nil {
			svc.store.UpdateAgentTeamStatus(teamID, "failed", "", "member "+m.Name+" template not found")
			return fmt.Errorf("member %s template not found", m.Name)
		}

		memberPromptIDs := make([]string, 0)
		memberSkillIDs := make([]string, 0)
		seenPrompts := make(map[string]bool)
		seenSkills := make(map[string]bool)

		for _, pid := range team.PromptIDs {
			if !seenPrompts[pid] { memberPromptIDs = append(memberPromptIDs, pid); seenPrompts[pid] = true }
		}
		for _, pid := range m.PromptIDs {
			if !seenPrompts[pid] { memberPromptIDs = append(memberPromptIDs, pid); seenPrompts[pid] = true }
		}
		for _, sid := range team.SkillIDs {
			if !seenSkills[sid] { memberSkillIDs = append(memberSkillIDs, sid); seenSkills[sid] = true }
		}
		for _, sid := range m.SkillIDs {
			if !seenSkills[sid] { memberSkillIDs = append(memberSkillIDs, sid); seenSkills[sid] = true }
		}

		extDir := filepath.Join(memberDir, "extensions")
		os.MkdirAll(extDir, 0755)

		type memberExtEntry struct {
			Name        string          `json:"name"`
			Label       string          `json:"label"`
			Description string          `json:"description"`
			Parameters  json.RawMessage `json:"parameters"`
			Handler     json.RawMessage `json:"handler"`
		}
		var memberExtManifest []memberExtEntry

		for _, tid := range tmpl.ToolIDs {
			tool, err := svc.store.GetTool(tid)
			if err != nil { continue }
			tsCode, err := codegen.GenerateToolExtension(tool.DSLDefinition)
			if err != nil { continue }
			os.WriteFile(filepath.Join(extDir, tool.Name+".ts"), []byte(tsCode), 0644)
			var dsl struct {
				Name        string          `json:"name"`
				Label       string          `json:"label"`
				Description string          `json:"description"`
				Parameters  json.RawMessage `json:"parameters"`
				Handler     json.RawMessage `json:"handler"`
			}
			if json.Unmarshal([]byte(tool.DSLDefinition), &dsl) == nil {
				var h struct{ Type string `json:"type"` }
				if json.Unmarshal(dsl.Handler, &h) == nil && h.Type == "mcp" {
					memberExtManifest = append(memberExtManifest, memberExtEntry{
						Name: dsl.Name, Label: dsl.Label, Description: dsl.Description,
						Parameters: dsl.Parameters, Handler: dsl.Handler,
					})
				}
			}
		}
		for _, tid := range m.ToolIDs {
			tool, err := svc.store.GetTool(tid)
			if err != nil { continue }
			tsCode, err := codegen.GenerateToolExtension(tool.DSLDefinition)
			if err != nil { continue }
			os.WriteFile(filepath.Join(extDir, tool.Name+".ts"), []byte(tsCode), 0644)
			var dsl struct {
				Name        string          `json:"name"`
				Label       string          `json:"label"`
				Description string          `json:"description"`
				Parameters  json.RawMessage `json:"parameters"`
				Handler     json.RawMessage `json:"handler"`
			}
			if json.Unmarshal([]byte(tool.DSLDefinition), &dsl) == nil {
				var h struct{ Type string `json:"type"` }
				if json.Unmarshal(dsl.Handler, &h) == nil && h.Type == "mcp" {
					memberExtManifest = append(memberExtManifest, memberExtEntry{
						Name: dsl.Name, Label: dsl.Label, Description: dsl.Description,
						Parameters: dsl.Parameters, Handler: dsl.Handler,
					})
				}
			}
		}

		if len(memberExtManifest) > 0 {
			manifestJSON, _ := json.Marshal(memberExtManifest)
			os.WriteFile(filepath.Join(extDir, "extensions.json"), manifestJSON, 0644)
		}

		svc.writePromptsSkills(memberDir, memberPromptIDs, memberSkillIDs)

		provider := ""
		modelID := ""
		apiKey := ""
		baseURL := ""
		if m.ProviderConfigID != "" {
			pc, err := svc.store.GetProviderConfig(m.ProviderConfigID)
			if err == nil && pc != nil {
				provider = pc.Provider
				modelID = pc.ModelID
				apiKey = pc.APIKey
				baseURL = pc.BaseURL
			}
		}

		mf.Members = append(mf.Members, manifestMember{
			Name:                 m.Name,
			Role:                 m.Role,
			Provider:             provider,
			ModelID:              modelID,
			APIKey:               apiKey,
			BaseURL:              baseURL,
			SystemPromptOverride: m.SystemPromptOverride,
		})
	}

	// Write team-level prompts/skills
	if teamPromptsDir != "" {
		teampDir := filepath.Join(buildDir, teamPromptsDir)
		os.MkdirAll(teampDir, 0755)
		svc.writePromptsSkills(teampDir, team.PromptIDs, nil)
	}
	if teamSkillsDir != "" {
		teamsDir := filepath.Join(buildDir, teamSkillsDir)
		os.MkdirAll(teamsDir, 0755)
		svc.writePromptsSkills(teamsDir, nil, team.SkillIDs)
	}

	manifestData, _ := json.MarshalIndent(mf, "", "  ")
	os.WriteFile(filepath.Join(buildDir, "team-manifest.json"), manifestData, 0644)

	imageTag := fmt.Sprintf("cloud-agent-team/%s:latest", teamID)
	if err := svc.docker.BuildImage(ctx, buildDir, imageTag, logWriter); err != nil {
		svc.store.UpdateAgentTeamStatus(teamID, "failed", "", "docker build: "+err.Error())
		return err
	}

	svc.store.UpdateAgentTeamStatus(teamID, "ready", imageTag, "")
	return nil
}

// findFreePort computes a port from instance ID hash and resolves conflicts.
// Port range: 3001-12000.
func (svc *AgentService) findFreePort(instanceID string) int {
	h := fnv.New32a()
	h.Write([]byte(instanceID))
	base := 3001 + int(h.Sum32()%9000)

	instances, err := svc.store.ListInstances()
	if err != nil {
		return base
	}
	used := make([]int, 0)
	for _, inst := range instances {
		if inst.HostPort > 0 && (inst.Status == "running" || inst.Status == "starting") {
			used = append(used, inst.HostPort)
		}
	}
	for port := base; port <= 12000; port++ {
		if !slices.Contains(used, port) {
			return port
		}
	}
	// fallback: scan from 3001 upward
	for port := 3001; port <= 12000; port++ {
		if !slices.Contains(used, port) {
			return port
		}
	}
	return 3001
}

func (svc *AgentService) StartTeamInstance(ctx context.Context, teamID string) (*model.Instance, error) {
	team, err := svc.store.GetAgentTeam(teamID)
	if err != nil || team == nil { return nil, fmt.Errorf("team not found") }
	if team.Status != "ready" { return nil, fmt.Errorf("team not ready: %s", team.Status) }

	instance := &model.Instance{AgentID: "", TeamID: teamID}
	if err := svc.store.CreateInstance(instance); err != nil { return nil, err }

	port := svc.findFreePort(instance.ID)
	containerName := fmt.Sprintf("cloud-agent-team-%s", instance.ID[:12])

	// Create log directory for this instance
	logDir := filepath.Join(svc.projectRoot, "builds", teamID, "llm-logs", instance.ID)
	os.MkdirAll(logDir, 0777)

	go func() {
		ctx := context.Background()
		env := map[string]string{
			"AGENT_PROVIDER": os.Getenv("AGENT_PROVIDER"),
			"AGENT_MODEL":    os.Getenv("AGENT_MODEL"),
			"AGENT_API_KEY":  os.Getenv("AGENT_API_KEY"),
			"AGENT_BASE_URL": os.Getenv("AGENT_BASE_URL"),
			"LOGS_DIR":       "/logs",
			"INSTANCE_ID":    instance.ID,
		}
		repoAbs, _ := filepath.Abs(filepath.Join(svc.projectRoot, "builds", teamID, "repo"))
		logAbs, _ := filepath.Abs(filepath.Join(svc.projectRoot, "builds", teamID, "llm-logs"))
		_, err := svc.docker.RunContainer(ctx, team.ImageTag, containerName,
			repoAbs, logAbs, port, env)
		if err != nil {
			svc.store.UpdateInstanceStatus(instance.ID, "error", "", 0, err.Error())
			return
		}
		svc.store.UpdateInstanceStatus(instance.ID, "running", containerName, port, "")
	}()

	return instance, nil
}

