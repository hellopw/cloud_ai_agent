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
	dockerfile, err := codegen.GenerateDockerfile(&codegen.DockerfileData{
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

	// Copy wrapper server.js to build context root (Dockerfile expects it there)
	wrapperSrc := filepath.Join(svc.projectRoot, "container-wrapper", "src", "server.js")
	wrapperData, err := os.ReadFile(wrapperSrc)
	if err != nil {
		svc.store.UpdateAgentStatus(agentID, "failed", "", "read wrapper: "+err.Error())
		return err
	}
	if err := os.WriteFile(filepath.Join(buildDir, "server.js"), wrapperData, 0644); err != nil {
		svc.store.UpdateAgentStatus(agentID, "failed", "", err.Error())
		return err
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
	for _, tid := range toolIDs {
		tool, err := svc.store.GetTool(tid)
		if err != nil { continue }
		tsCode, err := codegen.GenerateToolExtension(tool.DSLDefinition)
		if err != nil { continue }
		if err := os.WriteFile(filepath.Join(extDir, tool.Name+".ts"), []byte(tsCode), 0644); err != nil { return err }
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

func (svc *AgentService) StartInstance(ctx context.Context, agentID string) (*model.Instance, error) {
	agent, err := svc.store.GetAgent(agentID)
	if err != nil || agent == nil { return nil, fmt.Errorf("agent not found") }
	if agent.Status != "ready" { return nil, fmt.Errorf("agent not ready: %s", agent.Status) }

	instance := &model.Instance{AgentID: agentID}
	if err := svc.store.CreateInstance(instance); err != nil { return nil, err }

	port := 3001 + len(instance.ID)%1000
	containerName := fmt.Sprintf("cloud-agent-%s", instance.ID[:12])

	go func() {
		ctx := context.Background()
		env := map[string]string{
			"AGENT_PROVIDER": os.Getenv("AGENT_PROVIDER"),
			"AGENT_MODEL":    os.Getenv("AGENT_MODEL"),
			"AGENT_API_KEY":  os.Getenv("AGENT_API_KEY"),
			"AGENT_BASE_URL": os.Getenv("AGENT_BASE_URL"),
		}
		repoAbs, _ := filepath.Abs(filepath.Join(svc.projectRoot, "builds", agentID, "repo"))
		_, err := svc.docker.RunContainer(ctx, agent.ImageTag, containerName,
			repoAbs, port, env)
		if err != nil {
			svc.store.UpdateInstanceStatus(instance.ID, "error", "", 0)
			return
		}
		svc.store.UpdateInstanceStatus(instance.ID, "running", containerName, port)
	}()

	return instance, nil
}
