package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"cloud_ai_agent/internal/model"
	"cloud_ai_agent/internal/codegen"
	"github.com/google/uuid"
)

// --- Prompts ---

func (h *Handler) listPrompts(w http.ResponseWriter, r *http.Request) {
	prompts, err := h.store.ListPrompts()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, prompts)
}

func (h *Handler) getPrompt(w http.ResponseWriter, r *http.Request, id string) {
	p, err := h.store.GetPrompt(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if p == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *Handler) createPrompt(w http.ResponseWriter, r *http.Request) {
	var p model.Prompt
	if err := decodeJSON(r, &p); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.CreatePrompt(&p); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (h *Handler) updatePrompt(w http.ResponseWriter, r *http.Request, id string) {
	existing, err := h.store.GetPrompt(id)
	if err != nil || existing == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var p model.Prompt
	if err := decodeJSON(r, &p); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	p.ID = existing.ID
	if err := h.store.UpdatePrompt(&p); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *Handler) deletePrompt(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.store.DeletePrompt(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// --- Skills ---

func (h *Handler) listSkills(w http.ResponseWriter, r *http.Request) {
	skills, err := h.store.ListSkills()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, skills)
}

func (h *Handler) getSkill(w http.ResponseWriter, r *http.Request, id string) {
	s, err := h.store.GetSkill(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if s == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func (h *Handler) createSkill(w http.ResponseWriter, r *http.Request) {
	var s model.Skill
	if err := decodeJSON(r, &s); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.CreateSkill(&s); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, s)
}

func (h *Handler) updateSkill(w http.ResponseWriter, r *http.Request, id string) {
	existing, err := h.store.GetSkill(id)
	if err != nil || existing == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var s model.Skill
	if err := decodeJSON(r, &s); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.ID = existing.ID
	if err := h.store.UpdateSkill(&s); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func (h *Handler) deleteSkill(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.store.DeleteSkill(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// --- Tools ---

func (h *Handler) listTools(w http.ResponseWriter, r *http.Request) {
	tools, err := h.store.ListTools()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tools)
}

func (h *Handler) getTool(w http.ResponseWriter, r *http.Request, id string) {
	t, err := h.store.GetTool(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if t == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) createTool(w http.ResponseWriter, r *http.Request) {
	var t model.Tool
	if err := decodeJSON(r, &t); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.CreateTool(&t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (h *Handler) updateTool(w http.ResponseWriter, r *http.Request, id string) {
	existing, err := h.store.GetTool(id)
	if err != nil || existing == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var t model.Tool
	if err := decodeJSON(r, &t); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	t.ID = existing.ID
	if err := h.store.UpdateTool(&t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) deleteTool(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.store.DeleteTool(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// --- Templates ---

type bindRequest struct {
	PromptIDs []string `json:"prompt_ids"`
	SkillIDs  []string `json:"skill_ids"`
	ToolIDs   []string `json:"tool_ids"`
}

func (h *Handler) listTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := h.store.ListTemplates()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for i := range templates {
		fillEffectiveDockerfile(&templates[i])
	}
	writeJSON(w, http.StatusOK, templates)
}

func (h *Handler) getTemplate(w http.ResponseWriter, r *http.Request, id string) {
	t, err := h.store.GetTemplate(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if t == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	fillEffectiveDockerfile(t)
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) createTemplate(w http.ResponseWriter, r *http.Request) {
	var t model.Template
	if err := decodeJSON(r, &t); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.CreateTemplate(&t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (h *Handler) updateTemplate(w http.ResponseWriter, r *http.Request, id string) {
	existing, err := h.store.GetTemplate(id)
	if err != nil || existing == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var t model.Template
	if err := decodeJSON(r, &t); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	t.ID = existing.ID
	if err := h.store.UpdateTemplate(&t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) deleteTemplate(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.store.DeleteTemplate(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func (h *Handler) updateTemplateBindings(w http.ResponseWriter, r *http.Request, id string) {
	var req bindRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.UpdateTemplateBindings(id, req.PromptIDs, req.SkillIDs, req.ToolIDs); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "bindings updated"})
}

// --- Agents ---

func (h *Handler) listAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := h.store.ListAgents()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, agents)
}

func (h *Handler) getAgent(w http.ResponseWriter, r *http.Request, id string) {
	a, err := h.store.GetAgent(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if a == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (h *Handler) createAgent(w http.ResponseWriter, r *http.Request) {
	var a model.Agent
	if err := decodeJSON(r, &a); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.CreateAgent(&a); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

func (h *Handler) updateAgent(w http.ResponseWriter, r *http.Request, id string) {
	existing, err := h.store.GetAgent(id)
	if err != nil || existing == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if existing.Status != "draft" && existing.Status != "failed" {
		writeError(w, http.StatusConflict, "can only edit draft or failed agents")
		return
	}
	var a model.Agent
	if err := decodeJSON(r, &a); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	a.ID = existing.ID
	a.Status = existing.Status
	a.ImageTag = existing.ImageTag
	a.ErrorMsg = existing.ErrorMsg
	a.CreatedAt = existing.CreatedAt
	// Preserve credentials if not re-submitted (empty string means keep existing)
	if a.GitUsername == "" {
		a.GitUsername = existing.GitUsername
	}
	if a.GitPassword == "" {
		a.GitPassword = existing.GitPassword
	}
	if err := h.store.UpdateAgent(&a); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (h *Handler) deleteAgent(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.store.DeleteAgent(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func (h *Handler) getBuildLog(w http.ResponseWriter, r *http.Request, id string) {
	logPath := filepath.Join("..", "builds", id, "build.log")
	if os.Getenv("PROJECT_ROOT") != "" {
		logPath = filepath.Join(os.Getenv("PROJECT_ROOT"), "builds", id, "build.log")
	}
	// Fallback: try current directory (binary runs from project root on deployed server)
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		logPath = filepath.Join(".", "builds", id, "build.log")
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		writeError(w, http.StatusNotFound, "build log not found")
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(data)
}

func (h *Handler) buildAgent(w http.ResponseWriter, r *http.Request, id string) {
	if h.agentSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "agent service not initialized")
		return
	}
	go h.agentSvc.BuildAgent(context.Background(), id)
	writeJSON(w, http.StatusAccepted, map[string]string{"message": "build started"})
}

// --- Instances ---

func (h *Handler) listInstances(w http.ResponseWriter, r *http.Request) {
	instances, err := h.store.ListInstances()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, instances)
}

func (h *Handler) getInstance(w http.ResponseWriter, r *http.Request, id string) {
	i, err := h.store.GetInstance(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if i == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, i)
}

func (h *Handler) startInstance(w http.ResponseWriter, r *http.Request, id string) {
	if h.agentSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "agent service not initialized")
		return
	}
	var req struct {
		ProviderConfigID string `json:"provider_config_id"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	instance, err := h.agentSvc.StartInstance(r.Context(), id, req.ProviderConfigID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, instance)
}

func (h *Handler) deleteInstance(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.store.DeleteInstance(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

type InstanceConfig struct {
	Prompts []model.Prompt `json:"prompts"`
	Skills  []model.Skill  `json:"skills"`
	Tools   []model.Tool   `json:"tools"`
}

func (h *Handler) getInstanceConfig(w http.ResponseWriter, r *http.Request, id string) {
	inst, err := h.store.GetInstance(id)
	if err != nil || inst == nil {
		writeError(w, http.StatusNotFound, "instance not found")
		return
	}

	cfg := InstanceConfig{
		Prompts: []model.Prompt{},
		Skills:  []model.Skill{},
		Tools:   []model.Tool{},
	}

	if inst.AgentID != "" {
		agent, err := h.store.GetAgent(inst.AgentID)
		if err != nil || agent == nil {
			writeError(w, http.StatusNotFound, "agent not found")
			return
		}
		tmpl, err := h.store.GetTemplate(agent.TemplateID)
		if err != nil || tmpl == nil {
			writeError(w, http.StatusNotFound, "template not found")
			return
		}
		for _, pid := range tmpl.PromptIDs {
			if p, _ := h.store.GetPrompt(pid); p != nil {
				cfg.Prompts = append(cfg.Prompts, *p)
			}
		}
		for _, sid := range tmpl.SkillIDs {
			if s, _ := h.store.GetSkill(sid); s != nil {
				cfg.Skills = append(cfg.Skills, *s)
			}
		}
		for _, tid := range tmpl.ToolIDs {
			if t, _ := h.store.GetTool(tid); t != nil {
				cfg.Tools = append(cfg.Tools, *t)
			}
		}
	} else if inst.TeamID != "" {
		team, err := h.store.GetAgentTeam(inst.TeamID)
		if err != nil || team == nil {
			writeError(w, http.StatusNotFound, "team not found")
			return
		}
		for _, pid := range team.PromptIDs {
			if p, _ := h.store.GetPrompt(pid); p != nil {
				cfg.Prompts = append(cfg.Prompts, *p)
			}
		}
		for _, sid := range team.SkillIDs {
			if s, _ := h.store.GetSkill(sid); s != nil {
				cfg.Skills = append(cfg.Skills, *s)
			}
		}
		for _, m := range team.Members {
			if m.AgentTemplateID != "" {
				tmpl, _ := h.store.GetTemplate(m.AgentTemplateID)
				if tmpl != nil {
					for _, tid := range tmpl.ToolIDs {
						if t, _ := h.store.GetTool(tid); t != nil {
							cfg.Tools = append(cfg.Tools, *t)
						}
					}
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, cfg)
}

// --- Resources ---

func (h *Handler) listResources(w http.ResponseWriter, r *http.Request) {
	resources, err := h.store.ListResources()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resources)
}

func (h *Handler) getResource(w http.ResponseWriter, r *http.Request, id string) {
	res, err := h.store.GetResource(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if res == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *Handler) createResource(w http.ResponseWriter, r *http.Request) {
	var res model.Resource
	if err := decodeJSON(r, &res); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.CreateResource(&res); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, res)
}

func (h *Handler) updateResource(w http.ResponseWriter, r *http.Request, id string) {
	existing, err := h.store.GetResource(id)
	if err != nil || existing == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var res model.Resource
	if err := decodeJSON(r, &res); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	res.ID = existing.ID
	if err := h.store.UpdateResource(&res); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *Handler) deleteResource(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.store.DeleteResource(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// --- Memories ---

func (h *Handler) listMemories(w http.ResponseWriter, r *http.Request) {
	memories, err := h.store.ListMemories()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, memories)
}

func (h *Handler) getMemory(w http.ResponseWriter, r *http.Request, id string) {
	m, err := h.store.GetMemory(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if m == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (h *Handler) createMemory(w http.ResponseWriter, r *http.Request) {
	var m model.Memory
	if err := decodeJSON(r, &m); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.CreateMemory(&m); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

func (h *Handler) updateMemory(w http.ResponseWriter, r *http.Request, id string) {
	existing, err := h.store.GetMemory(id)
	if err != nil || existing == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var m model.Memory
	if err := decodeJSON(r, &m); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	m.ID = existing.ID
	if err := h.store.UpdateMemory(&m); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (h *Handler) deleteMemory(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.store.DeleteMemory(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// --- Agent Teams ---

func (h *Handler) listAgentTeams(w http.ResponseWriter, r *http.Request) {
	teams, err := h.store.ListAgentTeams()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, teams)
}

func (h *Handler) getAgentTeam(w http.ResponseWriter, r *http.Request, id string) {
	t, err := h.store.GetAgentTeam(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if t == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) createAgentTeam(w http.ResponseWriter, r *http.Request) {
	var t model.AgentTeam
	if err := decodeJSON(r, &t); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.CreateAgentTeam(&t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (h *Handler) updateAgentTeam(w http.ResponseWriter, r *http.Request, id string) {
	existing, err := h.store.GetAgentTeam(id)
	if err != nil || existing == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if existing.Status != "draft" && existing.Status != "failed" {
		writeError(w, http.StatusConflict, "can only edit draft or failed teams")
		return
	}
	var t model.AgentTeam
	if err := decodeJSON(r, &t); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	t.ID = existing.ID
	t.Status = existing.Status
	t.ImageTag = existing.ImageTag
	t.ErrorMsg = existing.ErrorMsg
	t.CreatedAt = existing.CreatedAt
	if t.GitUsername == "" {
		t.GitUsername = existing.GitUsername
	}
	if t.GitPassword == "" {
		t.GitPassword = existing.GitPassword
	}
	if err := h.store.UpdateAgentTeam(&t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) deleteAgentTeam(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.store.DeleteAgentTeam(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func (h *Handler) getTeamBuildLog(w http.ResponseWriter, r *http.Request, id string) {
	logPath := filepath.Join("..", "builds", id, "build.log")
	if os.Getenv("PROJECT_ROOT") != "" {
		logPath = filepath.Join(os.Getenv("PROJECT_ROOT"), "builds", id, "build.log")
	}
	// Fallback: try current directory (binary runs from project root on deployed server)
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		logPath = filepath.Join(".", "builds", id, "build.log")
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		writeError(w, http.StatusNotFound, "build log not found")
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(data)
}

func (h *Handler) buildAgentTeam(w http.ResponseWriter, r *http.Request, id string) {
	if h.agentSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "agent service not initialized")
		return
	}
	go h.agentSvc.BuildAgentTeam(context.Background(), id)
	writeJSON(w, http.StatusAccepted, map[string]string{"message": "build started"})
}

func (h *Handler) startTeamInstance(w http.ResponseWriter, r *http.Request, id string) {
	if h.agentSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "agent service not initialized")
		return
	}
	instance, err := h.agentSvc.StartTeamInstance(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, instance)
}

// --- Health ---

// --- Provider Configs ---

func (h *Handler) listProviderConfigs(w http.ResponseWriter, r *http.Request) {
	configs, err := h.store.ListProviderConfigs()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, configs)
}

func (h *Handler) getProviderConfig(w http.ResponseWriter, r *http.Request, id string) {
	pc, err := h.store.GetProviderConfig(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if pc == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, pc)
}

func (h *Handler) createProviderConfig(w http.ResponseWriter, r *http.Request) {
	var pc model.ProviderConfig
	if err := decodeJSON(r, &pc); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.CreateProviderConfig(&pc); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, pc)
}

func (h *Handler) updateProviderConfig(w http.ResponseWriter, r *http.Request, id string) {
	existing, err := h.store.GetProviderConfig(id)
	if err != nil || existing == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var pc model.ProviderConfig
	if err := decodeJSON(r, &pc); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	pc.ID = existing.ID
	// Preserve api_key if not re-submitted
	if pc.APIKey == "" {
		pc.APIKey = existing.APIKey
	}
	if err := h.store.UpdateProviderConfig(&pc); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, pc)
}

func (h *Handler) deleteProviderConfig(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.store.DeleteProviderConfig(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Chat Messages ---

func (h *Handler) listChatMessages(w http.ResponseWriter, r *http.Request, instanceID string) {
	msgs, err := h.store.ListChatMessages(instanceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if msgs == nil {
		msgs = []model.ChatMessage{}
	}
	writeJSON(w, http.StatusOK, msgs)
}

func (h *Handler) createChatMessage(w http.ResponseWriter, r *http.Request, instanceID string) {
	var msg model.ChatMessage
	if err := decodeJSON(r, &msg); err != nil {
		log.Printf("[chat-msg] decode error for instance %s: %v", instanceID, err)
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	log.Printf("[chat-msg] saving role=%s content_len=%d for instance %s", msg.Role, len(msg.Content), instanceID)
	msg.ID = uuid.New().String()
	msg.InstanceID = instanceID
	if err := h.store.CreateChatMessage(&msg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, msg)
}

// --- LLM Logs ---

var safeFilenameRe = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)

func (h *Handler) getLLMLogDir(instanceID string) (string, error) {
	inst, err := h.store.GetInstance(instanceID)
	if err != nil || inst == nil {
		return "", fmt.Errorf("instance not found")
	}
	projectRoot := h.agentSvc.ProjectRoot()
	targetID := inst.AgentID
	if targetID == "" {
		targetID = inst.TeamID
	}
	return filepath.Join(projectRoot, "builds", targetID, "llm-logs", instanceID), nil
}

func (h *Handler) listLLMLogs(w http.ResponseWriter, r *http.Request, instanceID string) {
	logDir, err := h.getLLMLogDir(instanceID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	entries, err := os.ReadDir(logDir)
	if err != nil {
		writeError(w, http.StatusNotFound, "llm logs not found")
		return
	}

	type fileInfo struct {
		Name string `json:"name"`
		Size int64  `json:"size"`
	}

	var files []fileInfo
	// Also scan subdirectories (for team mode: leader/, worker1/, etc.)
	for _, entry := range entries {
		if entry.IsDir() {
			subEntries, err := os.ReadDir(filepath.Join(logDir, entry.Name()))
			if err != nil {
				continue
			}
			for _, sub := range subEntries {
				if !sub.IsDir() {
					info, _ := sub.Info()
					files = append(files, fileInfo{
						Name: entry.Name() + "/" + sub.Name(),
						Size: info.Size(),
					})
				}
			}
		} else {
			info, _ := entry.Info()
			files = append(files, fileInfo{
				Name: entry.Name(),
				Size: info.Size(),
			})
		}
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})

	if r.Method == "GET" {
		writeJSON(w, http.StatusOK, map[string]interface{}{"files": files})
	} else {
		methodNotAllowed(w)
	}
}

func (h *Handler) getLLMLogFile(w http.ResponseWriter, r *http.Request, instanceID, filename string) {
	if r.Method != "GET" {
		methodNotAllowed(w)
		return
	}

	logDir, err := h.getLLMLogDir(instanceID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// Allow subdirectory (e.g. "leader/request-0001.jsonl")
	cleaned := filepath.Clean(filename)
	parts := strings.SplitN(cleaned, "/", 2)
	for _, part := range parts {
		if !safeFilenameRe.MatchString(part) {
			writeError(w, http.StatusBadRequest, "invalid filename")
			return
		}
	}

	fullPath := filepath.Join(logDir, cleaned)
	// Security: verify path is within logDir
	absPath, _ := filepath.Abs(fullPath)
	absLogDir, _ := filepath.Abs(logDir)
	if !strings.HasPrefix(absPath, absLogDir) {
		writeError(w, http.StatusForbidden, "path traversal denied")
		return
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}

	// Determine content type by extension
	contentType := "text/plain; charset=utf-8"
	if strings.HasSuffix(filename, ".json") {
		contentType = "application/json; charset=utf-8"
	} else if strings.HasSuffix(filename, ".jsonl") {
		contentType = "application/jsonl; charset=utf-8"
	}

	w.Header().Set("Content-Type", contentType)
	w.Write(data)
}

func fillEffectiveDockerfile(t *model.Template) {
	if t.DockerfileContent != "" {
		t.EffectiveDockerfile = t.DockerfileContent
		return
	}
	df, err := codegen.GenerateDockerfileByAgentType(t.AgentType, &codegen.DockerfileData{})
	if err == nil && df != "" {
		t.EffectiveDockerfile = df
	}
}
