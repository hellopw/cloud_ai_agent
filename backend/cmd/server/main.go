package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud_ai_agent/internal/api"
	"cloud_ai_agent/internal/service"
	"cloud_ai_agent/internal/store"
)

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" { return v }
	return def
}

func main() {
	dbPath := envOrDefault("DB_PATH", "data/cloud_ai_agent.db")

	s, err := store.New(dbPath)
	if err != nil { log.Fatalf("Failed to open database: %v", err) }
	defer s.Close()

	if err := s.Migrate(); err != nil { log.Fatalf("Failed to run migrations: %v", err) }
	log.Println("Database migrated successfully")

	h := api.NewHandler(s)
	agentSvc := service.NewAgentService(s)
	h.WithAgentService(agentSvc)

	mux := api.NewRouter(h)
	port := envOrDefault("PORT", "8080")
	srv := &http.Server{Addr: ":" + port, Handler: mux}

	go func() {
		log.Printf("Server starting on :%s", port)
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
	if err := srv.Shutdown(ctx); err != nil { log.Fatalf("Server forced shutdown: %v", err) }
}
