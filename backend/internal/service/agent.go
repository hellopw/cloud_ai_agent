package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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
	if err != nil || agent == nil { return fmt.Errorf("agent not found") }
	tmpl, err := svc.store.GetTemplate(agent.TemplateID)
	if err != nil || tmpl == nil { return fmt.Errorf("template not found") }

	svc.store.UpdateAgentStatus(agentID, "building", "", "")

	buildDir := filepath.Join(svc.projectRoot, "builds", agentID)
	os.RemoveAll(buildDir)
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		svc.store.UpdateAgentStatus(agentID, "failed", "", err.Error()); return err
	}

	logPath := filepath.Join(buildDir, "build.log")
	logFile, err := os.Create(logPath)
	if err != nil { svc.store.UpdateAgentStatus(agentID, "failed", "", err.Error()); return err }
	defer logFile.Close()
	logWriter := io.MultiWriter(logFile, os.Stdout)

	repoDir := filepath.Join(buildDir, "repo")
	if err := svc.git.Clone(ctx, agent.RepoURL, agent.Branch, repoDir, agent.GitUsername, agent.GitPassword, logWriter); err != nil {
		svc.store.UpdateAgentStatus(agentID, "failed", "", "git clone: "+err.Error()); return err
	}

	dockerfile, err := codegen.GenerateDockerfile(&codegen.DockerfileData{CustomContent: tmpl.DockerfileContent})
	if err != nil { svc.store.UpdateAgentStatus(agentID, "failed", "", err.Error()); return err }
	if err := os.WriteFile(filepath.Join(buildDir, "Dockerfile"), []byte(dockerfile), 0644); err != nil {
		svc.store.UpdateAgentStatus(agentID, "failed", "", err.Error()); return err
	}

	wrapperSrc := filepath.Join(svc.projectRoot, "container-wrapper", "src", "server.js")
	wrapperData, err := os.ReadFile(wrapperSrc)
	if err != nil { svc.store.UpdateAgentStatus(agentID, "failed", "", "read wrapper: "+err.Error()); return err }
	if err := os.WriteFile(filepath.Join(buildDir, "server.js"), wrapperData, 0644); err != nil {
		svc.store.UpdateAgentStatus(agentID, "failed", "", err.Error()); return err
	}

	if err := svc.writeToolExtensions(buildDir, tmpl.ToolIDs); err != nil {
		svc.store.UpdateAgentStatus(agentID, "failed", "", err.Error()); return err
	}
	if err := svc.writePromptsSkills(buildDir, tmpl.PromptIDs, tmpl.SkillIDs); err != nil {
		svc.store.UpdateAgentStatus(agentID, "failed", "", err.Error()); return err
	}

	imageTag := fmt.Sprintf("cloud-agent/%s:latest", agentID)
	if err := svc.docker.BuildImage(ctx, buildDir, imageTag, logWriter); err != nil {
		svc.store.UpdateAgentStatus(agentID, "failed", "", "docker build: "+err.Error()); return err
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
		os.WriteFile(filepath.Join(extDir, tool.Name+".ts"), []byte(tsCode), 0644)
	}
	return nil
}

func (svc *AgentService) writePromptsSkills(buildDir string, promptIDs, skillIDs []string) error {
	promptsDir := filepath.Join(buildDir, "pi-prompts"); os.MkdirAll(promptsDir, 0755)
	if len(promptIDs) > 0 {
		for _, pid := range promptIDs {
			p, err := svc.store.GetPrompt(pid)
			if err != nil || p == nil { continue }
			content := fmt.Sprintf("---\ndescription: %s\n---\n\n%s", p.Description, p.Content)
			os.WriteFile(filepath.Join(promptsDir, p.Name+".md"), []byte(content), 0644)
		}
	}
	skillsDir := filepath.Join(buildDir, "pi-skills"); os.MkdirAll(skillsDir, 0755)
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

func (svc *AgentService) StartInstance(ctx context.Context, agentID string, memoryIDs []string) (*model.Instance, error) {
	agent, err := svc.store.GetAgent(agentID)
	if err != nil || agent == nil { return nil, fmt.Errorf("agent not found") }
	if agent.Status != "ready" { return nil, fmt.Errorf("agent not ready: %s", agent.Status) }

	instance := &model.Instance{AgentID: agentID, MemoryIDs: memoryIDs}
	if err := svc.store.CreateInstance(instance); err != nil { return nil, err }

	port := 3000 + (len(instance.ID)%9000 + 1)
	containerName := fmt.Sprintf("cloud-agent-%s", instance.ID[:12])

	memoriesEnv := ""
	if len(memoryIDs) > 0 {
		mems, err := svc.store.GetMemoriesByIDs(memoryIDs)
		if err == nil {
			for _, m := range mems {
				memoriesEnv += fmt.Sprintf("## %s\n\n%s\n\n---\n\n", m.Name, m.Content)
			}
		}
	}

	go func() {
		ctx := context.Background()
		env := map[string]string{
			"AGENT_PROVIDER": os.Getenv("AGENT_PROVIDER"),
			"AGENT_MODEL":    os.Getenv("AGENT_MODEL"),
			"AGENT_API_KEY":  os.Getenv("AGENT_API_KEY"),
			"AGENT_BASE_URL": os.Getenv("AGENT_BASE_URL"),
			"MEMORIES":       memoriesEnv,
		}
		repoAbs, _ := filepath.Abs(filepath.Join(svc.projectRoot, "builds", agentID, "repo"))
		_, err := svc.docker.RunContainer(ctx, agent.ImageTag, containerName, repoAbs, port, env)
		if err != nil {
			svc.store.UpdateInstanceStatus(instance.ID, "error", "", 0)
			return
		}
		svc.store.UpdateInstanceStatus(instance.ID, "running", containerName, port)
	}()

	return instance, nil
}

func (svc *AgentService) StopInstance(ctx context.Context, instanceID string) error {
	inst, err := svc.store.GetInstance(instanceID)
	if err != nil || inst == nil { return fmt.Errorf("instance not found") }

	convSummary := svc.fetchConversation(inst.HostPort)

	if inst.ContainerID != "" {
		svc.docker.StopContainer(ctx, inst.ContainerID)
	}

	svc.store.UpdateInstanceStatus(instanceID, "stopped", "", 0)

	if convSummary != "" {
		memory := &model.Memory{
			Name:        fmt.Sprintf("Session %s", inst.ID[:8]),
			Description: fmt.Sprintf("Auto-extracted from instance %s", inst.ID[:8]),
			Content:     truncate(convSummary, 4000),
			Source:      "auto",
		}
		svc.store.CreateMemory(memory)
		svc.store.LinkInstanceMemory(instanceID, memory.ID)
	}

	return nil
}

func (svc *AgentService) fetchConversation(port int) string {
	if port <= 0 { return "" }
	url := fmt.Sprintf("http://localhost:%d/conversation", port)
	resp, err := http.Get(url)
	if err != nil { return "" }
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil { return "" }
	var result struct {
		Turns []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"turns"`
	}
	if err := json.Unmarshal(body, &result); err != nil { return "" }
	var sb strings.Builder
	for _, t := range result.Turns {
		sb.WriteString(fmt.Sprintf("%s: %s\n", t.Role, t.Content))
	}
	return sb.String()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen { return s }
	return s[:maxLen] + "..."
}
