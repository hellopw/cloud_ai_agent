package proxy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// HandleChatHTTP proxies a chat message to the container and returns SSE stream directly.
// Used as fallback when WebSocket is blocked by corporate proxy.
func HandleChatHTTP(containerHost string, containerPort int) http.HandlerFunc {
	containerURL := fmt.Sprintf("http://%s:%d", containerHost, containerPort)

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var msg struct {
			Message string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if msg.Message == "" {
			http.Error(w, "empty message", http.StatusBadRequest)
			return
		}

		reqBody, _ := json.Marshal(map[string]string{"message": msg.Message})
		resp, err := proxyClient.Post(containerURL+"/chat", "application/json", bytes.NewReader(reqBody))
		if err != nil {
			log.Printf("chat-http: container unreachable: %v", err)
			http.Error(w, fmt.Sprintf("container unreachable: %v", err), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
		w.WriteHeader(http.StatusOK)

		flusher, canFlush := w.(http.Flusher)
		reader := bufio.NewReader(resp.Body)

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			fmt.Fprintf(w, "%s\n\n", line)
			if canFlush {
				flusher.Flush()
			}
		}
	}
}

