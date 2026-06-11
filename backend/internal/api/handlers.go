package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"cloud_ai_agent/internal/model"
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
	if a.GitUsername == "" { a.GitUsername = existing.GitUsername }
	if a.GitPassword == "" { a.GitPassword = existing.GitPassword }
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
	if t.GitUsername == "" { t.GitUsername = existing.GitUsername }
	if t.GitPassword == "" { t.GitPassword = existing.GitPassword }
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
