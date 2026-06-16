package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/noxturne/tmux-ai-orchestrator/internal/tmux"
)

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
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// Immediate push on connection
	sendEvent := func() {
		panes, err := tmux.ListAgentPanes()
		if err != nil {
			log.Printf("SSE error fetching agent panes: %v", err)
			return
		}
		data, err := json.Marshal(panes)
		if err != nil {
			log.Printf("SSE JSON marshal error: %v", err)
			return
		}
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	sendEvent()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			sendEvent()
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
