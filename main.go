package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/noxturne/tmux-ai-orchestrator/internal/ui"
	"github.com/noxturne/tmux-ai-orchestrator/web"
)

func cleanUpTempStreams() {
	// 1. Find and remove all temp stream files from /tmp
	files, err := filepath.Glob("/tmp/mux-agent-*-stream.log")
	if err == nil {
		for _, f := range files {
			_ = os.Remove(f)
		}
	}

	// 2. Query all active tmux panes and stop any dangling pipe-panes
	cmd := exec.Command("tmux", "list-panes", "-a", "-F", "#{pane_id}")
	output, err := cmd.Output()
	if err == nil {
		panes := strings.Split(string(output), "\n")
		for _, p := range panes {
			p = strings.TrimSpace(p)
			if p != "" {
				_ = exec.Command("tmux", "pipe-pane", "-t", p).Run()
			}
		}
	}
}

func main() {
	// Guard: Ensure we are running inside tmux
	if os.Getenv("TMUX") == "" {
		fmt.Fprintln(os.Stderr, "Error: This application must be run inside an active tmux session.")
		os.Exit(1)
	}

	// Handle 'web' subcommand
	if len(os.Args) > 1 && os.Args[1] == "web" {
		port := 8080
		for _, arg := range os.Args[2:] {
			if strings.HasPrefix(arg, "--port=") {
				pStr := strings.TrimPrefix(arg, "--port=")
				fmt.Sscanf(pStr, "%d", &port)
			}
		}
		
		// Run web dashboard server
		if err := web.StartServer(port); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting web server: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Clean up any stale streams from previous crashes or runs
	cleanUpTempStreams()

	// Register signal channel to cleanly intercept interrupt/termination signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cleanUpTempStreams()
		os.Exit(0)
	}()

	// Defer cleanup for regular exit pathways
	defer cleanUpTempStreams()

	initialModel := ui.InitialModel()
	p := tea.NewProgram(&initialModel, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
