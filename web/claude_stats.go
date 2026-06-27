package web

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type ClaudeModelUsage struct {
	InputTokens              int64   `json:"inputTokens"`
	OutputTokens             int64   `json:"outputTokens"`
	CacheReadInputTokens     int64   `json:"cacheReadInputTokens"`
	CacheCreationInputTokens int64   `json:"cacheCreationInputTokens"`
	WebSearchRequests        int     `json:"webSearchRequests"`
	CostUSD                  float64 `json:"costUSD"`
}

type ClaudeDailyActivity struct {
	Date          string `json:"date"`
	MessageCount  int    `json:"messageCount"`
	SessionCount  int    `json:"sessionCount"`
	ToolCallCount int    `json:"toolCallCount"`
}

type ClaudeStats struct {
	Version          int                         `json:"version"`
	LastComputedDate string                      `json:"lastComputedDate"`
	DailyActivity    []ClaudeDailyActivity      `json:"dailyActivity"`
	ModelUsage       map[string]ClaudeModelUsage `json:"modelUsage"`
	TotalSessions    int                         `json:"totalSessions"`
	TotalMessages    int                         `json:"totalMessages"`
}

var (
	claudeStatsCached ClaudeStats
	claudeStatsMu     sync.RWMutex
	claudeStatsFile   string
)

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/home/noxturne"
	}
	claudeStatsFile = filepath.Join(home, ".claude", "stats-cache.json")
	
	// Initial load
	loadClaudeStats()
	
	// Start periodic parser (every 60 seconds)
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		for range ticker.C {
			loadClaudeStats()
		}
	}()
}

func loadClaudeStats() {
	data, err := os.ReadFile(claudeStatsFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Error reading Claude stats file: %v", err)
		}
		return
	}

	var stats ClaudeStats
	if err := json.Unmarshal(data, &stats); err != nil {
		log.Printf("Error parsing Claude stats JSON: %v", err)
		return
	}

	claudeStatsMu.Lock()
	claudeStatsCached = stats
	claudeStatsMu.Unlock()
}

func handleClaudeStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	claudeStatsMu.RLock()
	defer claudeStatsMu.RUnlock()
	
	if err := json.NewEncoder(w).Encode(claudeStatsCached); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
