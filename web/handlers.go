package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/noxturne/tmux-ai-orchestrator/internal/tmux"
)

// Broker handles SSE client connections and event broadcasting
type Broker struct {
	notifier       chan []tmux.AgentPane
	newClients     chan chan []tmux.AgentPane
	closingClients chan chan []tmux.AgentPane
	clients        map[chan []tmux.AgentPane]bool
}

var (
	broker  *Broker
	sseOnce sync.Once
)

func NewBroker() *Broker {
	return &Broker{
		notifier:       make(chan []tmux.AgentPane, 10),
		newClients:     make(chan chan []tmux.AgentPane),
		closingClients: make(chan chan []tmux.AgentPane),
		clients:        make(map[chan []tmux.AgentPane]bool),
	}
}

func (b *Broker) Start() {
	go func() {
		for {
			select {
			case s := <-b.newClients:
				b.clients[s] = true
			case s := <-b.closingClients:
				delete(b.clients, s)
				close(s)
			case event := <-b.notifier:
				for clientChan := range b.clients {
					select {
					case clientChan <- event:
					default:
					}
				}
			}
		}
	}()
}

type AgentMonitor struct {
	mu          sync.Mutex
	runningPIDs map[int]bool
}

var monitor = &AgentMonitor{
	runningPIDs: make(map[int]bool),
}

func (m *AgentMonitor) UpdatePIDs(panes []tmux.AgentPane) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runningPIDs = make(map[int]bool)
	for _, p := range panes {
		if p.PID > 0 {
			m.runningPIDs[p.PID] = true
		}
	}
}

func (m *AgentMonitor) CheckActive() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.runningPIDs) == 0 {
		return false
	}

	changed := false
	for pid := range m.runningPIDs {
		if !isPIDAlive(pid) {
			delete(m.runningPIDs, pid)
			changed = true
		}
	}
	return changed
}

func isPIDAlive(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func InitSSE() {
	sseOnce.Do(func() {
		broker = NewBroker()
		broker.Start()
		go monitorAgents()
	})
}

func monitorAgents() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if monitor.CheckActive() {
			log.Printf("SSE: AI finished running, triggering update")
			TriggerUpdate()
		}
	}
}

func TriggerUpdate() {
	panes, err := tmux.ListAgentPanes()
	if err != nil {
		log.Printf("SSE: error listing panes: %v", err)
		return
	}
	monitor.UpdatePIDs(panes)
	broker.notifier <- panes
}

// handleFleet handles GET /api/fleet
func handleFleet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	panes, err := tmux.ListAgentPanes()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(panes)
}

// handleFleetSSE handles GET /api/fleet/stream (SSE)
func handleFleetSSE(w http.ResponseWriter, r *http.Request) {
	InitSSE()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	clientChan := make(chan []tmux.AgentPane, 10)
	broker.newClients <- clientChan
	defer func() {
		broker.closingClients <- clientChan
	}()

	// Send initial event
	panes, err := tmux.ListAgentPanes()
	if err != nil {
		log.Printf("SSE error fetching agent panes: %v", err)
	} else {
		monitor.UpdatePIDs(panes)
		data, _ := json.Marshal(panes)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case panes, ok := <-clientChan:
			if !ok {
				return
			}
			data, err := json.Marshal(panes)
			if err != nil {
				log.Printf("SSE JSON marshal error: %v", err)
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

// handlePaneBuffer handles GET /api/pane/{id}/buffer
func handlePaneBuffer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, `{"error": "missing pane id"}`, http.StatusBadRequest)
		return
	}

	content, err := tmux.CapturePaneBuffer(id)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	resp := map[string]string{
		"paneID": id,
		"buffer": content,
	}
	json.NewEncoder(w).Encode(resp)
}

// handlePaneRaw handles GET /api/pane/{id}/raw
func handlePaneRaw(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, `{"error": "missing pane id"}`, http.StatusBadRequest)
		return
	}

	content, err := tmux.CapturePaneRaw(id)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	resp := map[string]string{
		"paneID": id,
		"raw":    content,
	}
	json.NewEncoder(w).Encode(resp)
}

// handlePaneKill handles POST /api/pane/{id}/kill
func handlePaneKill(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, `{"error": "missing pane id"}`, http.StatusBadRequest)
		return
	}

	err := tmux.KillPane(id)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	// Trigger SSE update so frontend updates its list immediately
	TriggerUpdate()

	resp := map[string]string{
		"status":  "ok",
		"message": fmt.Sprintf("Pane %s killed successfully", id),
	}
	json.NewEncoder(w).Encode(resp)
}

