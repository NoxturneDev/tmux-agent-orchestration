package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/noxturne/tmux-ai-orchestrator/internal/ui"
)

func main() {
	// Guard: Ensure we are running inside tmux
	if os.Getenv("TMUX") == "" {
		fmt.Fprintln(os.Stderr, "Error: This application must be run inside an active tmux session.")
		os.Exit(1)
	}

	initialModel := ui.InitialModel()
	p := tea.NewProgram(&initialModel, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
