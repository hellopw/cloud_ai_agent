package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"cloud_ai_agent/internal/api"
	"cloud_ai_agent/internal/mcp"
	"cloud_ai_agent/internal/service"
	"cloud_ai_agent/internal/store"
)

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	dbPath := envOrDefault("DB_PATH", "data/cloud_ai_agent.db")
	frontendDir := envOrDefault("FRONTEND_DIR", "frontend")

	s, err := store.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer s.Close()

	if err := s.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Database migrated successfully")

	if err := s.SeedDefaultTools(); err != nil {
		log.Printf("Warning: seed default tools: %v", err)
	}

	if err := s.SeedExampleAgents(); err != nil {
		log.Printf("Warning: seed example agents: %v", err)
	}

	h := api.NewHandler(s)
	agentSvc := service.NewAgentService(s)
	h.WithAgentService(agentSvc)

	// Initialize MCP server on SSE transport
	mcpSrv := mcp.NewServer(s, agentSvc)
	mcpSSEServer := mcp.NewSSEServer(mcpSrv)

	apiMux := api.NewRouter(h)
	fs := http.FileServer(http.Dir(frontendDir))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/mcp") {
			http.StripPrefix("/api/mcp", mcpSSEServer).ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api") {
			apiMux.ServeHTTP(w, r)
			return
		}
		// Check if file exists, otherwise serve index.html for SPA routing
		filePath := filepath.Join(frontendDir, r.URL.Path)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			http.ServeFile(w, r, filepath.Join(frontendDir, "index.html"))
			return
		}
		fs.ServeHTTP(w, r)
	})

	port := envOrDefault("PORT", "8080")
	srv := &http.Server{Addr: ":" + port, Handler: handler}

	go func() {
		log.Printf("Server starting on :%s", port)
		log.Printf("Frontend dir: %s", frontendDir)
		log.Printf("DB path: %s", dbPath)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Failed to server shutdown: %v", err)
	}
}
