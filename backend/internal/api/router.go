package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"cloud_ai_agent/internal/proxy"
	"cloud_ai_agent/internal/service"
	"cloud_ai_agent/internal/store"
)

type Handler struct {
	store    *store.Store
	agentSvc *service.AgentService
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) WithAgentService(svc *service.AgentService) {
	h.agentSvc = svc
}

func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/prompts", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET": h.listPrompts(w, r)
		case "POST": h.createPrompt(w, r)
		default: methodNotAllowed(w)
		}
	})
	mux.HandleFunc("/api/prompts/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/prompts/")
		switch r.Method {
		case "GET": h.getPrompt(w, r, id)
		case "PUT": h.updatePrompt(w, r, id)
		case "DELETE": h.deletePrompt(w, r, id)
		default: methodNotAllowed(w)
		}
	})

	mux.HandleFunc("/api/skills", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET": h.listSkills(w, r)
		case "POST": h.createSkill(w, r)
		default: methodNotAllowed(w)
		}
	})
	mux.HandleFunc("/api/skills/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/skills/")
		switch r.Method {
		case "GET": h.getSkill(w, r, id)
		case "PUT": h.updateSkill(w, r, id)
		case "DELETE": h.deleteSkill(w, r, id)
		default: methodNotAllowed(w)
		}
	})

	mux.HandleFunc("/api/tools", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET": h.listTools(w, r)
		case "POST": h.createTool(w, r)
		default: methodNotAllowed(w)
		}
	})
	mux.HandleFunc("/api/tools/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/tools/")
		switch r.Method {
		case "GET": h.getTool(w, r, id)
		case "PUT": h.updateTool(w, r, id)
		case "DELETE": h.deleteTool(w, r, id)
		default: methodNotAllowed(w)
		}
	})

	mux.HandleFunc("/api/templates", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET": h.listTemplates(w, r)
		case "POST": h.createTemplate(w, r)
		default: methodNotAllowed(w)
		}
	})
	mux.HandleFunc("/api/templates/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/templates/")
		if strings.HasSuffix(path, "/bind") {
			id := strings.TrimSuffix(path, "/bind")
			if r.Method == "PUT" { h.updateTemplateBindings(w, r, id) } else { methodNotAllowed(w) }
			return
		}
		switch r.Method {
		case "GET": h.getTemplate(w, r, path)
		case "PUT": h.updateTemplate(w, r, path)
		case "DELETE": h.deleteTemplate(w, r, path)
		default: methodNotAllowed(w)
		}
	})

	mux.HandleFunc("/api/agents", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET": h.listAgents(w, r)
		case "POST": h.createAgent(w, r)
		default: methodNotAllowed(w)
		}
	})
	mux.HandleFunc("/api/agents/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/agents/")
		if strings.HasSuffix(path, "/build") {
			id := strings.TrimSuffix(path, "/build")
			if r.Method == "POST" { h.buildAgent(w, r, id) } else { methodNotAllowed(w) }
			return
		}
		if strings.HasSuffix(path, "/start") {
			id := strings.TrimSuffix(path, "/start")
			if r.Method == "POST" { h.startInstance(w, r, id) } else { methodNotAllowed(w) }
			return
		}
		switch r.Method {
		case "GET": h.getAgent(w, r, path)
		case "DELETE": h.deleteAgent(w, r, path)
		default: methodNotAllowed(w)
		}
	})

	mux.HandleFunc("/api/instances", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET": h.listInstances(w, r)
		default: methodNotAllowed(w)
		}
	})
	mux.HandleFunc("/api/instances/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/instances/")
		if strings.HasSuffix(path, "/chat") {
			id := strings.TrimSuffix(path, "/chat")
			proxy.HandleChat("localhost", h.getInstancePort(id))(w, r)
			return
		}
		switch r.Method {
		case "GET": h.getInstance(w, r, path)
		case "DELETE": h.deleteInstance(w, r, path)
		default: methodNotAllowed(w)
		}
	})

	mux.HandleFunc("/api/health", h.health)

	return corsMiddleware(mux)
}

func (h *Handler) getInstancePort(id string) int {
	inst, err := h.store.GetInstance(id)
	if err != nil || inst == nil { return 3001 }
	return inst.HostPort
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" { w.WriteHeader(http.StatusNoContent); return }
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func methodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}
