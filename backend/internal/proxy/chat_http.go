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
			log.Printf("chat-http: bad request, decode error: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if msg.Message == "" {
			log.Printf("chat-http: bad request, empty message")
			http.Error(w, "empty message", http.StatusBadRequest)
			return
		}

		log.Printf("chat-http: proxying to %s/chat message_len=%d", containerURL, len(msg.Message))
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

		var pendingLines []string
		for {
			line, err := reader.ReadString('\n')
			if err != nil && line == "" {
				break
			}
			line = strings.TrimSpace(line)

			if line == "" {
				// End of SSE event — flush accumulated lines as one event
				if len(pendingLines) > 0 {
					for _, l := range pendingLines {
						fmt.Fprintf(w, "%s\n", l)
					}
					fmt.Fprintf(w, "\n")
					if canFlush {
						flusher.Flush()
					}
					pendingLines = pendingLines[:0]
				}
				continue
			}

			pendingLines = append(pendingLines, line)
		}
		// Flush any remaining lines
		if len(pendingLines) > 0 {
			for _, l := range pendingLines {
				fmt.Fprintf(w, "%s\n", l)
			}
			fmt.Fprintf(w, "\n")
			if canFlush {
				flusher.Flush()
			}
		}
	}
}
