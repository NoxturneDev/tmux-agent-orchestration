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

type ClaudeDailyModelTokens struct {
	Date          string           `json:"date"`
	TokensByModel map[string]int64 `json:"tokensByModel"`
}

type ClaudeStatsFileContent struct {
	Version          int                         `json:"version"`
	LastComputedDate string                      `json:"lastComputedDate"`
	DailyActivity    []ClaudeDailyActivity      `json:"dailyActivity"`
	DailyModelTokens []ClaudeDailyModelTokens    `json:"dailyModelTokens"`
	ModelUsage       map[string]ClaudeModelUsage `json:"modelUsage"`
	TotalSessions    int                         `json:"totalSessions"`
	TotalMessages    int                         `json:"totalMessages"`
}

type ClaudeStats struct {
	Version          int                         `json:"version"`
	LastComputedDate string                      `json:"lastComputedDate"`
	DailyActivity    []ClaudeDailyActivity      `json:"dailyActivity"`
	ModelUsage       map[string]ClaudeModelUsage `json:"modelUsage"`
	TotalSessions    int                         `json:"totalSessions"`
	TotalMessages    int                         `json:"totalMessages"`
	HourlyUsage      map[string]int64            `json:"hourlyUsage"`
	WeeklyUsage      map[string]int64            `json:"weeklyUsage"`
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

	var fileContent ClaudeStatsFileContent
	if err := json.Unmarshal(data, &fileContent); err != nil {
		log.Printf("Error parsing Claude stats JSON: %v", err)
		return
	}

	now := time.Now()

	// Calculate weekly model usage
	weeklyUsage := make(map[string]int64)
	offset := int(now.Weekday()) - 1
	if offset < 0 {
		offset = 6 // Sunday
	}
	monday := now.AddDate(0, 0, -offset)
	mondayStr := monday.Format("2006-01-02")

	for _, entry := range fileContent.DailyModelTokens {
		if entry.Date >= mondayStr {
			for model, tokens := range entry.TokensByModel {
				weeklyUsage[model] += tokens
			}
		}
	}

	// Calculate hourly rolling usage (last 5 hours) from project transcripts
	hourlyUsage := make(map[string]int64)
	threshold := now.Add(-5 * time.Hour)
	projectsDir := filepath.Join(filepath.Dir(claudeStatsFile), "projects")

	_ = filepath.Walk(projectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".jsonl" {
			return nil
		}
		// Performance optimization: skip files not modified in the last 5 hours
		if info.ModTime().Before(threshold) {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		dec := json.NewDecoder(file)
		for dec.More() {
			var line struct {
				Type      string `json:"type"`
				Timestamp string `json:"timestamp"`
				Message   *struct {
					Model string `json:"model"`
					Usage *struct {
						InputTokens  int64 `json:"input_tokens"`
						OutputTokens int64 `json:"output_tokens"`
					} `json:"usage"`
				} `json:"message"`
			}
			if err := dec.Decode(&line); err != nil {
				continue
			}
			if line.Type == "assistant" && line.Timestamp != "" && line.Message != nil && line.Message.Usage != nil {
				logTime, err := time.Parse(time.RFC3339, line.Timestamp)
				if err == nil && logTime.After(threshold) {
					model := line.Message.Model
					tokens := line.Message.Usage.InputTokens + line.Message.Usage.OutputTokens
					hourlyUsage[model] += tokens
				}
			}
		}
		return nil
	})

	claudeStatsMu.Lock()
	claudeStatsCached = ClaudeStats{
		Version:          fileContent.Version,
		LastComputedDate: fileContent.LastComputedDate,
		DailyActivity:    fileContent.DailyActivity,
		ModelUsage:       fileContent.ModelUsage,
		TotalSessions:    fileContent.TotalSessions,
		TotalMessages:    fileContent.TotalMessages,
		HourlyUsage:      hourlyUsage,
		WeeklyUsage:      weeklyUsage,
	}
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
