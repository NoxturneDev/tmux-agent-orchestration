package web

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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

type ChatMessage struct {
	Sender    string    `json:"sender"`    // "user" or "jarvis"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

var (
	jarvisMessages    []ChatMessage
	jarvisMessagesMu  sync.RWMutex
	jarvisClientChans = make(map[chan ChatMessage]bool)
	jarvisClientsMu   sync.Mutex
)

var historyFilePath string

func init() {
	historyFilePath = filepath.Join(tmux.ResolveProjectsDir(), "tmux-ai-orchestrator", ".agents", "jarvis_chat_history.json")
	loadHistory()
}

func loadHistory() {
	data, err := os.ReadFile(historyFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		log.Printf("Error reading Jarvis history: %v", err)
		return
	}

	var msgs []ChatMessage
	if err := json.Unmarshal(data, &msgs); err != nil {
		log.Printf("Error parsing Jarvis history: %v", err)
		return
	}

	jarvisMessagesMu.Lock()
	jarvisMessages = msgs
	jarvisMessagesMu.Unlock()
	log.Printf("Loaded %d messages from Jarvis chat history", len(msgs))
}

func saveHistory() {
	jarvisMessagesMu.RLock()
	data, err := json.MarshalIndent(jarvisMessages, "", "  ")
	jarvisMessagesMu.RUnlock()
	if err != nil {
		log.Printf("Error marshaling Jarvis history: %v", err)
		return
	}

	// Ensure parent directory exists
	_ = os.MkdirAll(filepath.Dir(historyFilePath), 0755)

	if err := os.WriteFile(historyFilePath, data, 0644); err != nil {
		log.Printf("Error writing Jarvis history: %v", err)
	}
}

// AddJarvisMessage appends a message to our store and broadcasts it to all WebSocket clients
func AddJarvisMessage(sender, content string) {
	msg := ChatMessage{
		Sender:    sender,
		Content:   content,
		Timestamp: time.Now(),
	}
	jarvisMessagesMu.Lock()
	jarvisMessages = append(jarvisMessages, msg)
	jarvisMessagesMu.Unlock()

	// Persist
	saveHistory()

	// Broadcast
	jarvisClientsMu.Lock()
	defer jarvisClientsMu.Unlock()
	for ch := range jarvisClientChans {
		select {
		case ch <- msg:
		default:
		}
	}
}

// handleJarvisResponse handles POST /api/jarvis/response
func handleJarvisResponse(w http.ResponseWriter, r *http.Request) {
	content := r.FormValue("content")
	if content == "" {
		// Try parsing JSON
		var payload struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err == nil {
			content = payload.Content
		}
	}

	if content == "" {
		http.Error(w, "missing content field", http.StatusBadRequest)
		return
	}

	AddJarvisMessage("jarvis", content)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"success"}`))
}

type wsMessage struct {
	Type    string `json:"type,omitempty"` // "message" or "intervene"
	Content string `json:"content,omitempty"`
}

type wsResponse struct {
	Type    string       `json:"type"` // "status" or "message"
	Status  string       `json:"status,omitempty"`
	PaneID  string       `json:"paneId,omitempty"`
	Message *ChatMessage `json:"message,omitempty"`
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

	// Stream status events (online/offline)
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

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
						_ = conn.WriteJSON(wsResponse{
							Type:   "status",
							Status: "offline",
						})
						writeMu.Unlock()
						lastStatus = "offline"
					}
					continue
				}

				if lastStatus != "online" {
					writeMu.Lock()
					_ = conn.WriteJSON(wsResponse{
						Type:   "status",
						Status: "online",
						PaneID: paneID,
					})
					writeMu.Unlock()
					lastStatus = "online"
				}
			}
		}
	}()

	// Register this client's broadcast channel
	clientChan := make(chan ChatMessage, 50)
	jarvisClientsMu.Lock()
	jarvisClientChans[clientChan] = true
	jarvisClientsMu.Unlock()

	defer func() {
		jarvisClientsMu.Lock()
		delete(jarvisClientChans, clientChan)
		jarvisClientsMu.Unlock()
	}()

	// Goroutine: read from clientChan and write to websocket
	go func() {
		for {
			select {
			case <-stopChan:
				return
			case msg, ok := <-clientChan:
				if !ok {
					return
				}
				writeMu.Lock()
				_ = conn.WriteJSON(wsResponse{
					Type:    "message",
					Message: &msg,
				})
				writeMu.Unlock()
			}
		}
	}()

	// Send current chat history immediately to the client
	jarvisMessagesMu.RLock()
	for _, msg := range jarvisMessages {
		clientChan <- msg
	}
	jarvisMessagesMu.RUnlock()

	// Read messages from client, add to history, broadcast, and inject them into JARVIS pane
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
			_ = conn.WriteJSON(wsResponse{
				Type:   "status",
				Status: "offline",
			})
			writeMu.Unlock()
			continue
		}

		if msg.Type == "intervene" {
			// Send C-c to Jarvis pane
			_ = exec.Command("tmux", "send-keys", "-t", paneID, "C-c").Run()

			// Send C-c to all active worker agent panes
			panes, err := tmux.ListAgentPanes()
			if err == nil {
				for _, p := range panes {
					if p.PaneID != paneID {
						_ = exec.Command("tmux", "send-keys", "-t", p.PaneID, "C-c").Run()
					}
				}
			}

			// Add system intervention notice to logs and save
			AddJarvisMessage("jarvis", "_[Intervention: Response generation halted by user]_")
			TriggerUpdate()
			continue
		}

		// Inject command input into JARVIS pane
		if msg.Content != "" {
			// Save the user's message in the API proxy store
			AddJarvisMessage("user", msg.Content)

			err := tmux.InjectPromptViaBuffer(paneID, msg.Content)
			if err != nil {
				log.Printf("WS error injecting prompt to pane %s: %v", paneID, err)
			}
			TriggerUpdate()
		}
	}
}
