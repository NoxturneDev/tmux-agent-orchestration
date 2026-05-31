package tmux

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// SpawnTarget specifies the target tmux container layout for the agent
type SpawnTarget int

const (
	TargetPane SpawnTarget = iota
	TargetWindow
)

// AgentConfig holds the template command for spawning an agent
type AgentConfig struct {
	Name    string
	Command string // e.g. "mkdir -p ~/.antigravity-personal && HOME=$HOME/.antigravity-personal agy"
}

// Predefined list of agents as per aliases in ~/.zshrc
var Agents = []AgentConfig{
	{Name: "agy-p1", Command: "mkdir -p ~/.antigravity-personal && HOME=$HOME/.antigravity-personal agy"},
	{Name: "gemini-p1", Command: "mkdir -p ~/.gemini-personal && HOME=$HOME/.gemini-personal gemini"},
	{Name: "agy-p2", Command: "mkdir -p ~/.antigravity-work && HOME=$HOME/.antigravity-work agy"},
	{Name: "gemini-p2", Command: "mkdir -p ~/.gemini-work && HOME=$HOME/.gemini-work gemini"},
}

// AgentPane holds parsed host tmux pane metadata for Mission Control (AI Fleet Radar)
type AgentPane struct {
	PaneID     string // e.g., %4
	Session    string // e.g., "ziad-laravel"
	Command    string // e.g., "agy-p1"
	Path       string // e.g., "/home/user/workspace/ziad-laravel"
	ActiveGoal string // Extracted objective from active_plan.md
	WindowID   string // e.g., "2" (cleansed from tmux @2 representation)
}

// EscapeShellSingleQuote escapes single quotes for use inside a single-quoted shell string.
func EscapeShellSingleQuote(s string) string {
	return strings.ReplaceAll(s, "'", "'\\''")
}

// GetSpawnCommand returns the compiled shell execution command about to be run, for spawner preview
func GetSpawnCommand(agentName, prompt string) (string, error) {
	var targetCmd string
	for _, agent := range Agents {
		if agent.Name == agentName {
			targetCmd = agent.Command
			break
		}
	}
	if targetCmd == "" {
		return "", fmt.Errorf("unknown agent: %s", agentName)
	}

	escapedPrompt := EscapeShellSingleQuote(prompt)
	return fmt.Sprintf("%s -i '%s'", targetCmd, escapedPrompt), nil
}

// SpawnAgent splits the window or creates a new window and runs the agent with the given prompt in the specified dir
func SpawnAgent(agentName, prompt, dir string, target SpawnTarget, targetWindow string, splitDir string) (string, error) {
	fullShellCmd, err := GetSpawnCommand(agentName, prompt)
	if err != nil {
		return "", err
	}

	// Build tmux command argument list
	var tmuxSubCmd string
	var args []string

	if target == TargetWindow {
		tmuxSubCmd = "new-window"
		if targetWindow != "" {
			args = append(args, "-t", targetWindow+":")
		}
	} else {
		tmuxSubCmd = "split-window"
		if splitDir == "-v" {
			args = append(args, "-v") // vertical pane split
		} else {
			args = append(args, "-h") // horizontal pane split
		}
		if targetWindow != "" {
			args = append(args, "-t", targetWindow)
		}
	}

	if dir != "" && dir != "." {
		args = append(args, "-c", dir)
	}

	args = append(args, "-P", "-F", "#{pane_id}", fullShellCmd)

	// Combine to run: tmux split-window/new-window [args...]
	cmd := exec.Command("tmux", append([]string{tmuxSubCmd}, args...)...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to spawn agent: %v (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}

	paneID := strings.TrimSpace(stdout.String())
	if paneID == "" {
		return "", fmt.Errorf("tmux did not return a pane ID")
	}

	// Tag the pane natively in tmux as an AI agent pane
	_ = TagAgentPane(paneID, agentName)

	return paneID, nil
}

// TagAgentPane tags a tmux pane with custom options to identify it as an AI agent
func TagAgentPane(paneID, agentName string) error {
	cmd1 := exec.Command("tmux", "set-option", "-p", "-t", paneID, "@is_agent", "1")
	_ = cmd1.Run()
	cmd2 := exec.Command("tmux", "set-option", "-p", "-t", paneID, "@agent_name", agentName)
	return cmd2.Run()
}

// ListWindows returns all active tmux windows in the format "session_name:window_index (window_name)"
func ListWindows() ([]string, error) {
	cmd := exec.Command("tmux", "list-windows", "-a", "-F", "#{session_name}:#{window_index} (#{window_name})")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to query tmux windows: %v", err)
	}

	lines := strings.Split(stdout.String(), "\n")
	var windows []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			windows = append(windows, line)
		}
	}
	return windows, nil
}


// KillPane kills the specified tmux pane layout grid
func KillPane(paneID string) error {
	if paneID == "" {
		return fmt.Errorf("empty pane ID")
	}
	cmd := exec.Command("tmux", "kill-pane", "-t", paneID)
	return cmd.Run()
}

// PullPane pulls the targeted pane into the active window layout (join-pane)
func PullPane(paneID string) error {
	if paneID == "" {
		return fmt.Errorf("empty pane ID")
	}
	cmd := exec.Command("tmux", "join-pane", "-s", paneID, "-t", "!")
	return cmd.Run()
}

// IsolatePane upgrades the pane into its own full-screen standalone window (break-pane)
func IsolatePane(paneID string) error {
	if paneID == "" {
		return fmt.Errorf("empty pane ID")
	}
	cmd := exec.Command("tmux", "break-pane", "-s", paneID)
	return cmd.Run()
}

// CapturePaneBuffer captures the last 20 lines of the target pane's terminal buffer
func CapturePaneBuffer(paneID string) (string, error) {
	if paneID == "" {
		return "", fmt.Errorf("empty pane ID")
	}
	cmd := exec.Command("tmux", "capture-pane", "-p", "-t", paneID, "-S", "-20")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("capture failed: %v", err)
	}
	return stdout.String(), nil
}

// InjectPromptViaBuffer loads content to tmux clipboard via stdin, pastes it into the pane, and sends Enter
func InjectPromptViaBuffer(paneID, content string) error {
	if paneID == "" {
		return fmt.Errorf("empty pane ID")
	}

	// 1. Use load-buffer - with stdin to load prompt securely without shell-escaping limits
	setBufCmd := exec.Command("tmux", "load-buffer", "-")
	setBufCmd.Stdin = strings.NewReader(content)
	if err := setBufCmd.Run(); err != nil {
		return fmt.Errorf("failed to load tmux buffer: %v", err)
	}

	// 2. Paste clipboard buffer into target pane
	pasteCmd := exec.Command("tmux", "paste-buffer", "-p", "-t", paneID)
	if err := pasteCmd.Run(); err != nil {
		return fmt.Errorf("failed to paste buffer: %v", err)
	}

	// 3. Send Enter key to trigger submission
	sendEnterCmd := exec.Command("tmux", "send-keys", "-t", paneID, "Enter")
	if err := sendEnterCmd.Run(); err != nil {
		return fmt.Errorf("failed to submit prompt: %v", err)
	}

	return nil
}

// getPPIDAndComm parses /proc/<pid>/stat to extract parent PID and command name
func getPPIDAndComm(pid int) (int, string, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0, "", err
	}
	s := string(data)
	lastParen := strings.LastIndex(s, ")")
	if lastParen == -1 || lastParen+2 >= len(s) {
		return 0, "", fmt.Errorf("invalid stat format")
	}

	firstParen := strings.Index(s, "(")
	comm := ""
	if firstParen != -1 && firstParen < lastParen {
		comm = s[firstParen+1 : lastParen]
	}

	rest := s[lastParen+2:]
	parts := strings.Fields(rest)
	if len(parts) < 2 {
		return 0, "", fmt.Errorf("invalid stat format")
	}

	var ppid int
	_, err = fmt.Sscanf(parts[1], "%d", &ppid)
	if err != nil {
		return 0, "", err
	}
	return ppid, comm, nil
}

// buildPidTree scans host processes via tmux run-shell ps (or falls back to /proc) to construct parent-child PID mappings and cmdline lists
func buildPidTree() (map[int][]int, map[int]string, map[int]string) {
	parentToChildren := make(map[int][]int)
	pidToComm := make(map[int]string)
	pidToArgs := make(map[int]string)

	// 1. Try host-level ps via tmux run-shell (resolves container/sandbox namespace isolation)
	cmd := exec.Command("tmux", "run-shell", "ps -eo pid,ppid,comm,args")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err == nil {
		lines := strings.Split(stdout.String(), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.Fields(line)
			if len(parts) < 3 {
				continue
			}
			pid, err1 := strconv.Atoi(parts[0])
			ppid, err2 := strconv.Atoi(parts[1])
			if err1 != nil || err2 != nil {
				continue
			}
			comm := parts[2]
			var args string
			idx := strings.Index(line, comm)
			if idx != -1 {
				args = strings.TrimSpace(line[idx+len(comm):])
			}
			parentToChildren[ppid] = append(parentToChildren[ppid], pid)
			pidToComm[pid] = comm
			pidToArgs[pid] = args
		}
		if len(pidToComm) > 0 {
			return parentToChildren, pidToComm, pidToArgs
		}
	}

	// 2. Fallback: local /proc scanning
	files, err := os.ReadDir("/proc")
	if err != nil {
		return parentToChildren, pidToComm, pidToArgs
	}

	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(f.Name())
		if err != nil {
			continue
		}

		ppid, comm, err := getPPIDAndComm(pid)
		if err == nil {
			parentToChildren[ppid] = append(parentToChildren[ppid], pid)
			pidToComm[pid] = comm
			cmdlineBytes, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
			if err == nil {
				cmdline := string(cmdlineBytes)
				cmdline = strings.ReplaceAll(cmdline, "\x00", " ")
				pidToArgs[pid] = cmdline
			}
		}
	}

	return parentToChildren, pidToComm, pidToArgs
}

// isAgentProcess checks recursively if a pane PID or any descendant process is an AI agent
func isAgentProcess(panePID int, parentToChildren map[int][]int, pidToComm map[int]string, pidToArgs map[int]string) (bool, string) {
	queue := []int{panePID}
	visited := make(map[int]bool)

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if visited[curr] {
			continue
		}
		visited[curr] = true

		comm := pidToComm[curr]
		args := pidToArgs[curr]
		if strings.Contains(comm, "agy") || strings.Contains(comm, "gemini") || strings.Contains(comm, "claude") {
			return true, comm
		}
		if strings.Contains(args, "agy") || strings.Contains(args, "gemini") || strings.Contains(args, "claude") {
			if strings.Contains(args, "agy") {
				return true, "agy"
			}
			if strings.Contains(args, "gemini") {
				return true, "gemini"
			}
			if strings.Contains(args, "claude") {
				return true, "claude"
			}
			return true, comm
		}

		if children, ok := parentToChildren[curr]; ok {
			queue = append(queue, children...)
		}
	}

	return false, ""
}

// ListAgentPanes queries host tmux for all running AI agent panes and silent-extracts their plans
func ListAgentPanes() ([]AgentPane, error) {
	cmd := exec.Command("tmux", "list-panes", "-a", "-F", "#{pane_id}|#{session_name}|#{window_id}|#{pane_current_command}|#{pane_current_path}|#{pane_pid}|#{@is_agent}|#{@agent_name}")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to query tmux panes: %v", err)
	}

	lines := strings.Split(stdout.String(), "\n")
	var panes []AgentPane

	// Build the PID tree once for this list scan
	parentToChildren, pidToComm, pidToArgs := buildPidTree()

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 5 {
			continue
		}
		paneID := parts[0]
		session := parts[1]
		windowID := strings.TrimPrefix(parts[2], "@") // clean prefix @ symbol
		command := parts[3]
		path := parts[4]

		isAgent := false
		displayCommand := command

		// 1. Check native tmux tagging first
		if len(parts) >= 8 && parts[6] == "1" {
			isAgent = true
			if parts[7] != "" {
				displayCommand = parts[7]
			}
		}

		// 2. Process tree traversal if not natively tagged
		if !isAgent && len(parts) >= 6 {
			panePID, err := strconv.Atoi(parts[5])
			if err == nil && panePID > 0 {
				if ok, matchedCmd := isAgentProcess(panePID, parentToChildren, pidToComm, pidToArgs); ok {
					isAgent = true
					displayCommand = matchedCmd
				}
			}
		}

		// 3. Fallback check (command name matching)
		if !isAgent {
			if strings.Contains(command, "agy") || strings.Contains(command, "gemini") || strings.Contains(command, "claude") {
				isAgent = true
			}
		}

		if !isAgent {
			continue
		}

		// Retrieve active goal objective from local active_plan.md
		planPath := filepath.Join(path, ".agents", "plan", "active_plan.md")
		activeGoal := extractActiveGoal(planPath)

		panes = append(panes, AgentPane{
			PaneID:     paneID,
			Session:    session,
			Command:    displayCommand,
			Path:       path,
			ActiveGoal: activeGoal,
			WindowID:   windowID,
		})
	}

	return panes, nil
}

// TeleportToPane shifts host terminal focus to the selected pane
func TeleportToPane(paneID string) error {
	if paneID == "" {
		return fmt.Errorf("empty pane ID")
	}
	cmd := exec.Command("tmux", "switch-client", "-t", paneID)
	return cmd.Run()
}

// ScanProjectsDir lists directories under /home/noxturne/projects and always appends "." as first choice
func ScanProjectsDir() []string {
	projectsPath := "/home/noxturne/projects"
	dirs := []string{"."}

	entries, err := os.ReadDir(projectsPath)
	if err != nil {
		return dirs
	}

	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			dirs = append(dirs, filepath.Join(projectsPath, entry.Name()))
		}
	}
	return dirs
}

// ListSubdirs returns absolute paths of direct subdirectories within the given path folder
func ListSubdirs(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, filepath.Join(path, entry.Name()))
		}
	}
	return dirs, nil
}

// extractActiveGoal scans active_plan.md for the first heading or non-empty line
func extractActiveGoal(planPath string) string {
	info, err := os.Stat(planPath)
	if err != nil || info.IsDir() {
		return "[No active plan - Idle]"
	}

	file, err := os.Open(planPath)
	if err != nil {
		return "[No active plan - Idle]"
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// Find first header
		if strings.HasPrefix(line, "#") {
			cleaned := strings.TrimLeft(line, "# ")
			cleaned = strings.TrimSpace(cleaned)
			if cleaned != "" {
				return cleaned
			}
			continue
		}
		return line
	}
	return "[No active plan - Idle]"
}

// FindAllProjectSubdirs lists all directories under /home/noxturne/projects up to 3 levels deep relatively
func FindAllProjectSubdirs() ([]string, error) {
	cmd := exec.Command("find", "-L", ".", "-maxdepth", "3", "-type", "d", "-not", "-path", "*/.*")
	cmd.Dir = "/home/noxturne/projects"
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(stdout.String(), "\n")
	var dirs []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			dirs = append(dirs, line)
		}
	}
	sort.Strings(dirs)
	return dirs, nil
}

// ListSessions returns all active tmux sessions
func ListSessions() ([]string, error) {
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to query tmux sessions: %v", err)
	}

	lines := strings.Split(stdout.String(), "\n")
	var sessions []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			sessions = append(sessions, line)
		}
	}
	return sessions, nil
}

