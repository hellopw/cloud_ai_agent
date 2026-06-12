package proxy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var proxyClient = &http.Client{
	Transport: &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return nil, nil
		},
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	},
	Timeout: 300 * time.Second,
}

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
		log.Printf("ws: connection established from %s for %s:%d", r.RemoteAddr, containerHost, containerPort)

		for {
			_, msgBytes, err := conn.ReadMessage()
			if err != nil {
				log.Printf("ws: read error (client likely disconnected): %v", err)
				break
			}

			log.Printf("ws: raw message received: %s", string(msgBytes)[:min(len(msgBytes), 200)])
			var msg ChatMessage
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				log.Printf("ws: unmarshal error: %v", err)
				continue
			}

			log.Printf("ws: parsed message type=%s message_len=%d", msg.Type, len(msg.Message))

			if msg.Type == "chat" {
				log.Printf("ws: received chat message, proxying to %s/chat", containerURL)
				go proxyChatToContainer(conn, containerURL, msg.Message)
			}
		}
	}
}

func proxyChatToContainer(conn *websocket.Conn, containerURL, message string) {
	reqBody, _ := json.Marshal(map[string]string{"message": message})

	log.Printf("ws: posting to %s/chat", containerURL)
	resp, err := proxyClient.Post(containerURL+"/chat", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		log.Printf("ws: post error: %v", err)
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
			// Skip empty lines between event and data
			var dataLine string
			for {
				dataLine, err = reader.ReadString('\n')
				if err != nil { break }
				dataLine = strings.TrimSpace(dataLine)
				if dataLine != "" { break }
			}
			if strings.HasPrefix(dataLine, "data: ") {
						dataJSON := strings.TrimPrefix(dataLine, "data: ")
												log.Printf("ws: forwarding event type=%s data=%s", eventType, dataJSON)
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

// HandleProxy forwards a simple HTTP GET to a container endpoint
func HandleProxy(containerHost string, containerPort int, path string) http.HandlerFunc {
	url := fmt.Sprintf("http://%s:%d%s", containerHost, containerPort, path)
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := proxyClient.Get(url)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		for k, vs := range resp.Header {
			for _, v := range vs {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}
}

