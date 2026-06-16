package web

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/noxturne/tmux-ai-orchestrator/internal/tmux"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for the developer server
	},
}

type wsMessage struct {
	Content string `json:"content"`
}

type wsResponse struct {
	Status string `json:"status"` // "online" or "offline"
	PaneID string `json:"paneId,omitempty"`
	Raw    string `json:"raw,omitempty"`
}

func handleJarvisWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	var writeMu sync.Mutex
	stopChan := make(chan struct{})
	defer close(stopChan)

	// Helper function to find active JARVIS pane
	findJarvisPane := func() (string, bool) {
		panes, err := tmux.ListAgentPanes()
		if err != nil {
			return "", false
		}
		for _, p := range panes {
			if strings.ToLower(p.Command) == "jarvis" {
				return p.PaneID, true
			}
		}
		return "", false
	}

	// Goroutine: Stream Jarvis terminal raw output to browser
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		var lastRaw string
		var lastStatus string

		for {
			select {
			case <-stopChan:
				return
			case <-ticker.C:
				paneID, online := findJarvisPane()
				if !online {
					if lastStatus != "offline" {
						writeMu.Lock()
						_ = conn.WriteJSON(wsResponse{Status: "offline"})
						writeMu.Unlock()
						lastStatus = "offline"
						lastRaw = ""
					}
					continue
				}

				raw, err := tmux.CapturePaneRaw(paneID)
				if err != nil {
					log.Printf("WS error capturing jarvis pane raw: %v", err)
					continue
				}

				if raw != lastRaw || lastStatus != "online" {
					writeMu.Lock()
					_ = conn.WriteJSON(wsResponse{
						Status: "online",
						PaneID: paneID,
						Raw:    raw,
					})
					writeMu.Unlock()
					lastRaw = raw
					lastStatus = "online"
				}
			}
		}
	}()

	// Read messages from client and inject them into JARVIS pane
	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			break // Connection closed
		}

		var msg wsMessage
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			log.Printf("WS error unmarshaling client message: %v", err)
			continue
		}

		paneID, online := findJarvisPane()
		if !online {
			writeMu.Lock()
			_ = conn.WriteJSON(wsResponse{Status: "offline"})
			writeMu.Unlock()
			continue
		}

		// Inject command input into JARVIS pane
		if msg.Content != "" {
			err := tmux.InjectPromptViaBuffer(paneID, msg.Content)
			if err != nil {
				log.Printf("WS error injecting prompt to pane %s: %v", paneID, err)
			}
		}
	}
}
