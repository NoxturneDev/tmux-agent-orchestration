package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"
)

type AntigravityPromptCredits struct {
	Available           float64 `json:"available"`
	Monthly             float64 `json:"monthly"`
	UsedPercentage      float64 `json:"usedPercentage"`
	RemainingPercentage float64 `json:"remainingPercentage"`
}

type AntigravityModelQuota struct {
	Label               string  `json:"label"`
	ModelID             string  `json:"modelId"`
	RemainingPercentage float64 `json:"remainingPercentage"`
	IsExhausted         bool    `json:"isExhausted"`
	ResetTime           string  `json:"resetTime"`
	TimeUntilResetMs    int64   `json:"timeUntilResetMs"`
	IsAutocompleteOnly  bool    `json:"isAutocompleteOnly"`
}

type AntigravityQuotaSnapshot struct {
	Timestamp     string                    `json:"timestamp"`
	Method        string                    `json:"method"`
	Email         string                    `json:"email"`
	Models        []AntigravityModelQuota   `json:"models"`
	PromptCredits *AntigravityPromptCredits `json:"promptCredits,omitempty"`
}

// Wrapper struct matching the actual API response format
type AntigravityAccountAPIResponse struct {
	Email    string                    `json:"email"`
	IsActive bool                      `json:"isActive"`
	Status   string                    `json:"status"`
	Snapshot *AntigravityQuotaSnapshot `json:"snapshot,omitempty"`
}

type AccountQuotaResponse struct {
	AccountName string                    `json:"accountName"`
	HomeDir     string                    `json:"homeDir"`
	Error       string                    `json:"error,omitempty"`
	Quota       *AntigravityQuotaSnapshot `json:"quota,omitempty"`
}

type CombinedQuotaResponse struct {
	Accounts []AccountQuotaResponse `json:"accounts"`
}

var (
	quotaCached CombinedQuotaResponse
	quotaMu     sync.RWMutex
)

func init() {
	// Initial load
	loadAllQuotas()

	// Start periodic parser (every 60 seconds)
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		for range ticker.C {
			loadAllQuotas()
		}
	}()
}

func loadAllQuotas() {
	accounts := []struct {
		name string
		home string
	}{
		{"personal (agy-p1)", "/home/noxturne/.antigravity-personal"},
		{"work (agy-p2)", "/home/noxturne/.antigravity-work"},
	}

	var responses []AccountQuotaResponse

	for _, acc := range accounts {
		quota, err := fetchAccountQuota(acc.home)
		resp := AccountQuotaResponse{
			AccountName: acc.name,
			HomeDir:     acc.home,
		}
		if err != nil {
			resp.Error = err.Error()
		} else {
			resp.Quota = quota
		}
		responses = append(responses, resp)
	}

	quotaMu.Lock()
	quotaCached = CombinedQuotaResponse{Accounts: responses}
	quotaMu.Unlock()
}

func fetchAccountQuota(homeDir string) (*AntigravityQuotaSnapshot, error) {
	cmd := exec.Command("/home/noxturne/.nvm/versions/node/v22.21.1/bin/antigravity-usage", "quota", "--json", "--all")
	cmd.Env = append(os.Environ(),
		"HOME="+homeDir,
		"PATH=/home/noxturne/.nvm/versions/node/v22.21.1/bin:/usr/local/bin:/usr/bin:/bin",
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// If command failed, check if there's stderr output to show the user
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		return nil, &runError{Msg: errMsg}
	}

	var apiResponses []AntigravityAccountAPIResponse
	if err := json.Unmarshal(stdout.Bytes(), &apiResponses); err != nil {
		return nil, err
	}

	if len(apiResponses) == 0 {
		return nil, &runError{Msg: "No quota data returned"}
	}

	// Return the snapshot from the first account
	if apiResponses[0].Snapshot == nil {
		return nil, &runError{Msg: "No snapshot data in response"}
	}

	return apiResponses[0].Snapshot, nil
}

type runError struct {
	Msg string
}

func (e *runError) Error() string {
	return e.Msg
}

func handleAntigravityQuota(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	quotaMu.RLock()
	defer quotaMu.RUnlock()

	if err := json.NewEncoder(w).Encode(quotaCached); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
