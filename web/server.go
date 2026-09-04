package web

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"
)

// StartServer starts the web dashboard server on the specified port.
func StartServer(port int) error {
	mux := http.NewServeMux()

	// API Endpoints
	mux.HandleFunc("GET /api/fleet", handleFleet)
	mux.HandleFunc("GET /api/fleet/stream", handleFleetSSE)
	mux.HandleFunc("GET /api/pane/{id}/buffer", handlePaneBuffer)
	mux.HandleFunc("GET /api/pane/{id}/raw", handlePaneRaw)
	mux.HandleFunc("POST /api/pane/{id}/kill", handlePaneKill)
	mux.HandleFunc("POST /api/jarvis/response", handleJarvisResponse)
	mux.HandleFunc("GET /api/claude/stats", handleClaudeStats)
	mux.HandleFunc("GET /api/antigravity/quota", handleAntigravityQuota)

	// Review API Endpoints
	mux.HandleFunc("GET /api/reviews", handleReviewList)
	mux.HandleFunc("GET /api/review/{slug}", handleReviewGet)
	mux.HandleFunc("POST /api/review/{slug}/open", handleReviewOpen)

	// WebSocket Endpoint
	mux.HandleFunc("GET /ws/jarvis", handleJarvisWS)

	// Static Assets (Svelte app)
	subFS, err := fs.Sub(FS, "frontend/dist")
	if err != nil {
		return fmt.Errorf("failed to create sub filesystem: %w", err)
	}
	fileServer := http.FileServer(http.FS(subFS))
	
	// Catch-all handler to serve static assets and fallback to index.html (SPA routing)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/review-") {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Web Dashboard server listening on http://localhost%s\n", addr)
	
	// Start server
	return http.ListenAndServe(addr, mux)
}
