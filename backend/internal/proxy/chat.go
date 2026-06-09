package proxy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type ChatMessage struct {
	Type    string `json:"type"`
	Message string `json:"message,omitempty"`
}

type SSEEvent struct {
	Event string `json:"event"`
	Data  json.RawMessage `json:"data"`
}

func HandleChat(containerHost string, containerPort int) http.HandlerFunc {
	containerURL := fmt.Sprintf("http://%s:%d", containerHost, containerPort)

	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("ws upgrade: %v", err)
			return
		}
		defer conn.Close()

		for {
			_, msgBytes, err := conn.ReadMessage()
			if err != nil {
				break
			}

			var msg ChatMessage
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				continue
			}

			if msg.Type == "chat" {
				go proxyChatToContainer(conn, containerURL, msg.Message)
			}
		}
	}
}

func proxyChatToContainer(conn *websocket.Conn, containerURL, message string) {
	reqBody, _ := json.Marshal(map[string]string{"message": message})

	resp, err := http.Post(containerURL+"/chat", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		sendWSEvent(conn, "error", map[string]string{"message": fmt.Sprintf("container unreachable: %v", err)})
		return
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	var mu sync.Mutex

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				sendWSEvent(conn, "error", map[string]string{"message": err.Error()})
			}
			break
		}
		line = strings.TrimSpace(line)
		if line == "" { continue }

		if strings.HasPrefix(line, "event: ") {
			eventType := strings.TrimPrefix(line, "event: ")
			dataLine, err := reader.ReadString('\n')
			if err != nil { break }
			dataLine = strings.TrimSpace(dataLine)
			if strings.HasPrefix(dataLine, "data: ") {
				dataJSON := strings.TrimPrefix(dataLine, "data: ")
				mu.Lock()
				conn.WriteJSON(map[string]interface{}{
					"type": eventType,
					"data": json.RawMessage(dataJSON),
				})
				mu.Unlock()
			}
		}
	}
}

func sendWSEvent(conn *websocket.Conn, eventType string, data interface{}) {
	conn.WriteJSON(map[string]interface{}{
		"type": eventType,
		"data": data,
	})
}
