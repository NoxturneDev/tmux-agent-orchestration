package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var slugRe = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

func isValidSlug(slug string) bool {
	return slugRe.MatchString(slug)
}

type CombinedReview struct {
	Meta      json.RawMessage `json:"meta"`
	Diff      json.RawMessage `json:"diff"`
	SummaryMd string          `json:"summary_md"`
}

type ReviewListItem struct {
	Slug string          `json:"slug"`
	Meta json.RawMessage `json:"meta"`
}

type MetaLite struct {
	GitRoot string `json:"git_root"`
}

type OpenRequest struct {
	File string `json:"file"`
	Line int    `json:"line"`
}

func reviewRoot() string {
	root := os.Getenv("ANTIGRAVITY_REVIEW_ROOT")
	if root == "" {
		root = "/mnt/workspace/projects/my-notes/reviews"
	}
	return root
}

func isSafePath(baseDir, path string) (string, bool) {
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", false
	}
	joined := filepath.Join(absBase, path)
	absJoined, err := filepath.Abs(joined)
	if err != nil {
		return "", false
	}
	if !strings.HasPrefix(absJoined, absBase) {
		return "", false
	}
	return absJoined, true
}

func handleReviewGet(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if !isValidSlug(slug) {
		http.Error(w, `{"error": "invalid slug"}`, http.StatusBadRequest)
		return
	}

	dir := filepath.Join(reviewRoot(), slug)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		http.Error(w, `{"error": "review not found"}`, http.StatusNotFound)
		return
	}

	metaBytes, err := os.ReadFile(filepath.Join(dir, "meta.json"))
	if err != nil {
		http.Error(w, `{"error": "failed to read meta.json"}`, http.StatusInternalServerError)
		return
	}

	diffBytes, err := os.ReadFile(filepath.Join(dir, "diff.json"))
	if err != nil {
		http.Error(w, `{"error": "failed to read diff.json"}`, http.StatusInternalServerError)
		return
	}

	summaryBytes, err := os.ReadFile(filepath.Join(dir, "summary.md"))
	if err != nil {
		summaryBytes = []byte("")
	}

	res := CombinedReview{
		Meta:      json.RawMessage(metaBytes),
		Diff:      json.RawMessage(diffBytes),
		SummaryMd: string(summaryBytes),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func handleReviewList(w http.ResponseWriter, r *http.Request) {
	root := reviewRoot()
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("[]"))
			return
		}
		http.Error(w, `{"error": "failed to read reviews directory"}`, http.StatusInternalServerError)
		return
	}

	var list []ReviewListItem
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		slug := entry.Name()
		if !isValidSlug(slug) {
			continue
		}

		metaPath := filepath.Join(root, slug, "meta.json")
		metaBytes, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}

		list = append(list, ReviewListItem{
			Slug: slug,
			Meta: json.RawMessage(metaBytes),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func handleReviewOpen(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if !isValidSlug(slug) {
		http.Error(w, `{"error": "invalid slug"}`, http.StatusBadRequest)
		return
	}

	var req OpenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "failed to decode request body"}`, http.StatusBadRequest)
		return
	}

	dir := filepath.Join(reviewRoot(), slug)
	metaBytes, err := os.ReadFile(filepath.Join(dir, "meta.json"))
	if err != nil {
		http.Error(w, `{"error": "failed to read meta.json"}`, http.StatusInternalServerError)
		return
	}

	var meta MetaLite
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		http.Error(w, `{"error": "failed to parse meta.json"}`, http.StatusInternalServerError)
		return
	}

	if meta.GitRoot == "" {
		http.Error(w, `{"error": "git_root missing from meta.json"}`, http.StatusInternalServerError)
		return
	}

	absFile, ok := isSafePath(meta.GitRoot, req.File)
	if !ok {
		http.Error(w, `{"error": "invalid file path/path traversal detected"}`, http.StatusForbidden)
		return
	}

	tmuxSession := os.Getenv("ANTIGRAVITY_TMUX_SESSION")

	nvimCmd := fmt.Sprintf("nvim +%d %q", req.Line, absFile)
	var cmd *exec.Cmd
	if tmuxSession != "" {
		cmd = exec.Command("tmux", "popup", "-t", tmuxSession, "-E", "-w", "90%", "-h", "90%", nvimCmd)
	} else {
		cmd = exec.Command("tmux", "run-shell", fmt.Sprintf("tmux popup -E -w 90%% -h 90%% %q", nvimCmd))
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "failed to launch tmux popup: %s, output: %s"}`, err.Error(), string(output)), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "ok"}`))
}
