package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/noxturne/tmux-ai-orchestrator/internal/prompt"
	"github.com/noxturne/tmux-ai-orchestrator/internal/tmux"
)

// Tab swaps between viewports
type Tab int

const (
	TabFleet Tab = iota
	TabSpawner
)

// SpawnerState manages agent setup menus
type SpawnerState int

const (
	SpawnerStateAgent SpawnerState = iota
	SpawnerStateDir
	SpawnerStateTarget
	SpawnerStateWindow
	SpawnerStateSplitDirection
	SpawnerStateSession
	SpawnerStateMacro
	SpawnerStateExecuting
)

// TreeItem represents a flattened node for navigation in the directory tree
type TreeItem struct {
	IsFolder bool
	Path     string
	Pane     tmux.AgentPane
}

// Msg types for telemetry tick loop
type telemetryTickMsg time.Time

type telemetryResultMsg struct {
	paneID string
	buffer string
	err    error
}

// Msg type sent back when Editor exits
type editorFinishedMsg struct {
	filePath string
	err      error
}

// Msg type sent back when FZF exits
type fzfFinishedMsg struct {
	listPath   string
	outputPath string
	err        error
}

// tickTelemetry schedules the next telemetry query tick in 1 second
func tickTelemetry() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return telemetryTickMsg(t)
	})
}

// openEditorCmd suspends Bubble Tea and spawns Vim (or $EDITOR) connected to the temp file
func openEditorCmd(filePath string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	c := exec.Command(editor, filePath)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{
			filePath: filePath,
			err:      err,
		}
	})
}

// openFzfCmd suspends Bubble Tea and spawns fzf to fuzzy-find directory paths
func openFzfCmd(dirs []string) tea.Cmd {
	listFile, err := os.CreateTemp("", "fzf-list-*.txt")
	if err != nil {
		return func() tea.Msg {
			return fzfFinishedMsg{err: err}
		}
	}
	defer listFile.Close()

	for _, d := range dirs {
		listFile.WriteString(d + "\n")
	}

	outFile, err := os.CreateTemp("", "fzf-out-*.txt")
	if err != nil {
		return func() tea.Msg {
			return fzfFinishedMsg{err: err}
		}
	}
	outFile.Close()

	shellCmd := fmt.Sprintf("fzf --prompt='Select Directory: ' --height=40%% --layout=reverse --border --preview='tree -L 1 -C {1}' < %s > %s", listFile.Name(), outFile.Name())
	c := exec.Command("bash", "-c", shellCmd)
	c.Dir = "/home/noxturne/projects" // Set fzf working directory to projects root so relative preview works!

	return tea.ExecProcess(c, func(err error) tea.Msg {
		return fzfFinishedMsg{
			listPath:   listFile.Name(),
			outputPath: outFile.Name(),
			err:        err,
		}
	})
}

// Model holds the TUI application state
type Model struct {
	ActiveTab    Tab
	SpawnerState SpawnerState

	// Tab 1: Fleet Orchestration state
	FleetPanes          []tmux.AgentPane
	TreeItems           []TreeItem
	SelectedTreeItem    int
	CollapsedPaths      map[string]bool
	TelemetryBuffer     string
	LastTelemetryPaneID string

	// Tab 2: Spawner state
	Agents        []string
	SelectedAgent int

	Dirs           []string
	SelectedDir    int
	CurrentDirPath string

	Targets        []string
	SelectedTarget int

	Windows        []string
	SelectedWindow int

	Sessions        []string
	SelectedSession int

	SplitDirections        []string
	SelectedSplitDirection int

	Macros        []prompt.Macro
	SelectedMacro int

	// Common fields
	ActivePaneID string
	ErrorMsg     string
	IsError      bool

	// Dynamic terminal layout states
	Width  int
	Height int
}

// InitialModel initializes the state variables
func InitialModel() Model {
	agents := []string{"agy-p1", "gemini-p1", "agy-p2", "gemini-p2"}
	macros := []prompt.Macro{
		prompt.NoMacro,
		prompt.Implement,
		prompt.CookIt,
		prompt.WrapItUp,
		prompt.Recon,
	}
	targets := []string{"Pane Split", "New Window"}

	initialDir := "/home/noxturne/projects"
	if _, err := os.Stat(initialDir); err != nil {
		initialDir = "."
	}

	m := Model{
		ActiveTab:              TabFleet,
		SpawnerState:           SpawnerStateAgent,
		CollapsedPaths:         make(map[string]bool),
		Agents:                 agents,
		SelectedAgent:          0,
		CurrentDirPath:         initialDir,
		Dirs:                   nil,
		SelectedDir:            0,
		Targets:                targets,
		SelectedTarget:         0,
		Windows:                nil,
		SelectedWindow:         0,
		Sessions:               nil,
		SelectedSession:        0,
		SplitDirections:        []string{"Horizontal Split (-h)", "Vertical Split (-v)"},
		SelectedSplitDirection: 0,
		Macros:                 macros,
		SelectedMacro:          0,
		ActivePaneID:           "",
		ErrorMsg:               "",
		IsError:                false,
		TelemetryBuffer:        "[Select an active agent to view telemetry]",
		Width:                  80, // Default initialization fallback
		Height:                 24, // Default initialization fallback
	}
	m.populateDirList()
	m.refreshFleet()
	return m
}

// populateWindowList queries active tmux windows and appends an active window fallback
func (m *Model) populateWindowList() {
	wins, err := tmux.ListWindows()
	if err != nil {
		m.Windows = []string{"[ Active Window ]"}
		return
	}
	list := []string{"[ Active Window ]"}
	list = append(list, wins...)
	m.Windows = list
}

// populateSessionList queries active tmux sessions and appends an active session fallback
func (m *Model) populateSessionList() {
	sessions, err := tmux.ListSessions()
	if err != nil {
		m.Sessions = []string{"[ Active Session ]"}
		return
	}
	list := []string{"[ Active Session ]"}
	list = append(list, sessions...)
	m.Sessions = list
}

// refreshFleet queries tmux and rebuilds the folder tree
func (m *Model) refreshFleet() {
	panes, err := tmux.ListAgentPanes()
	if err != nil {
		m.ErrorMsg = fmt.Sprintf("Failed to query fleet: %v", err)
		m.IsError = true
		return
	}
	m.FleetPanes = panes
	m.rebuildTree()
}

// rebuildTree aggregates panes by dir and sorts them to form the interactive tree
func (m *Model) rebuildTree() {
	grouped := make(map[string][]tmux.AgentPane)
	for _, p := range m.FleetPanes {
		grouped[p.Path] = append(grouped[p.Path], p)
	}

	var paths []string
	for k := range grouped {
		paths = append(paths, k)
	}
	sort.Strings(paths)

	var items []TreeItem
	for _, path := range paths {
		items = append(items, TreeItem{
			IsFolder: true,
			Path:     path,
		})

		if !m.CollapsedPaths[path] {
			panes := grouped[path]
			sort.Slice(panes, func(i, j int) bool {
				return panes[i].PaneID < panes[j].PaneID
			})
			for _, p := range panes {
				items = append(items, TreeItem{
					IsFolder: false,
					Pane:     p,
				})
			}
		}
	}

	m.TreeItems = items
	if m.SelectedTreeItem >= len(m.TreeItems) {
		m.SelectedTreeItem = len(m.TreeItems) - 1
	}
	if m.SelectedTreeItem < 0 {
		m.SelectedTreeItem = 0
	}
}

// populateDirList builds browsing options inside the spawner
func (m *Model) populateDirList() {
	var list []string
	list = append(list, "[ Select This Directory ]")

	cleanPath := filepath.Clean(m.CurrentDirPath)
	if cleanPath != "/" {
		list = append(list, "..")
	}

	subdirs, err := tmux.ListSubdirs(cleanPath)
	if err == nil {
		list = append(list, subdirs...)
	}

	m.Dirs = list
}

// Msg type sent back when spawning completes
type spawnResultMsg struct {
	paneID string
	err    error
}

// spawnAgentCmd runs the spawn logic asynchronously as a Bubble Tea Cmd
func (m Model) spawnAgentCmd() tea.Cmd {
	return func() tea.Msg {
		selectedAgent := m.Agents[m.SelectedAgent]
		selectedDir := m.CurrentDirPath
		selectedTarget := tmux.SpawnTarget(m.SelectedTarget)
		selectedMacro := m.Macros[m.SelectedMacro]
		constructedPrompt := prompt.BuildPrompt(selectedMacro)

		var targetWindow string
		var splitDir string
		if selectedTarget == tmux.TargetPane {
			if m.SelectedWindow > 0 && m.SelectedWindow < len(m.Windows) {
				winStr := m.Windows[m.SelectedWindow]
				parts := strings.Split(winStr, " ")
				if len(parts) > 0 {
					targetWindow = parts[0]
				}
			}
			if m.SelectedSplitDirection == 1 {
				splitDir = "-v"
			} else {
				splitDir = "-h"
			}
		} else { // TargetWindow
			if m.SelectedSession > 0 && m.SelectedSession < len(m.Sessions) {
				targetWindow = m.Sessions[m.SelectedSession]
			}
		}

		paneID, err := tmux.SpawnAgent(selectedAgent, constructedPrompt, selectedDir, selectedTarget, targetWindow, splitDir)
		return spawnResultMsg{
			paneID: paneID,
			err:    err,
		}
	}
}

// getPreviewCommand generates raw preview of the tmux invocation
func (m Model) getPreviewCommand() string {
	selectedAgent := m.Agents[m.SelectedAgent]
	selectedDir := m.CurrentDirPath
	selectedTarget := tmux.SpawnTarget(m.SelectedTarget)
	selectedMacro := m.Macros[m.SelectedMacro]

	constructedPrompt := prompt.BuildPrompt(selectedMacro)
	innerCmd, err := tmux.GetSpawnCommand(selectedAgent, constructedPrompt)
	if err != nil {
		return "Error building command preview"
	}

	var tmuxSubCmd string
	var args []string
	if selectedTarget == tmux.TargetWindow {
		tmuxSubCmd = "new-window"
		var targetSession string
		if m.SelectedSession > 0 && m.SelectedSession < len(m.Sessions) {
			targetSession = m.Sessions[m.SelectedSession]
		}
		if targetSession != "" {
			args = append(args, "-t", targetSession+":")
		}
	} else {
		tmuxSubCmd = "split-window"
		var targetWindow string
		var splitDir string
		if m.SelectedWindow > 0 && m.SelectedWindow < len(m.Windows) {
			winStr := m.Windows[m.SelectedWindow]
			parts := strings.Split(winStr, " ")
			if len(parts) > 0 {
				targetWindow = parts[0]
			}
		}
		if m.SelectedSplitDirection == 1 {
			splitDir = "-v"
		} else {
			splitDir = "-h"
		}

		args = append(args, splitDir)
		if targetWindow != "" {
			args = append(args, "-t", targetWindow)
		}
	}

	if selectedDir != "" && selectedDir != "." {
		args = append(args, "-c", selectedDir)
	}

	args = append(args, "-P", "-F", "\"#{pane_id}\"", fmt.Sprintf("\"%s\"", innerCmd))
	return fmt.Sprintf("tmux %s %s", tmuxSubCmd, strings.Join(args, " "))
}

// queryTelemetryCmd fires capture-pane query on highlighted pane
func (m Model) queryTelemetryCmd() tea.Cmd {
	if len(m.TreeItems) == 0 {
		return nil
	}
	item := m.TreeItems[m.SelectedTreeItem]
	if item.IsFolder {
		return nil
	}
	paneID := item.Pane.PaneID
	return func() tea.Msg {
		buf, err := tmux.CapturePaneBuffer(paneID)
		return telemetryResultMsg{
			paneID: paneID,
			buffer: buf,
			err:    err,
		}
	}
}

// Init initializes the Bubble Tea model
func (m Model) Init() tea.Cmd {
	return tickTelemetry()
}

// Update handles state changes on key presses and asynchronous message callbacks
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case telemetryTickMsg:
		var cmd tea.Cmd
		if m.ActiveTab == TabFleet && !m.IsError {
			cmd = m.queryTelemetryCmd()
		}
		return m, tea.Batch(cmd, tickTelemetry())

	case telemetryResultMsg:
		if len(m.TreeItems) > 0 && m.SelectedTreeItem < len(m.TreeItems) {
			item := m.TreeItems[m.SelectedTreeItem]
			if !item.IsFolder && item.Pane.PaneID == msg.paneID {
				if msg.err != nil {
					m.TelemetryBuffer = fmt.Sprintf("Telemetry error: %v", msg.err)
				} else {
					m.TelemetryBuffer = msg.buffer
				}
			}
		}
		return m, nil

	case editorFinishedMsg:
		// Securely delete temporary draft prompt file immediately
		defer os.Remove(msg.filePath)

		if msg.err != nil {
			m.ErrorMsg = fmt.Sprintf("Editor execution failed: %v", msg.err)
			m.IsError = true
			return m, tea.ClearScreen
		}

		contentBytes, err := os.ReadFile(msg.filePath)
		if err != nil {
			m.ErrorMsg = fmt.Sprintf("Failed to read editor draft prompt: %v", err)
			m.IsError = true
			return m, tea.ClearScreen
		}

		promptText := strings.TrimSpace(string(contentBytes))
		if promptText != "" {
			if len(m.TreeItems) > 0 && m.SelectedTreeItem < len(m.TreeItems) {
				item := m.TreeItems[m.SelectedTreeItem]
				if !item.IsFolder {
					err := tmux.InjectPromptViaBuffer(item.Pane.PaneID, promptText)
					if err != nil {
						m.ErrorMsg = fmt.Sprintf("Injection failed: %v", err)
						m.IsError = true
					} else {
						m.refreshFleet()
					}
				}
			}
		}

		// Repaint whole screen once Editor exits
		return m, tea.ClearScreen

	case fzfFinishedMsg:
		defer os.Remove(msg.listPath)
		defer os.Remove(msg.outputPath)

		if msg.err != nil {
			return m, tea.ClearScreen
		}

		contentBytes, err := os.ReadFile(msg.outputPath)
		if err != nil {
			m.ErrorMsg = fmt.Sprintf("Failed to read selected directory: %v", err)
			m.IsError = true
			return m, tea.ClearScreen
		}

		selectedDir := strings.TrimSpace(string(contentBytes))
		if selectedDir != "" {
			cleanRelative := strings.TrimPrefix(selectedDir, "./")
			cleanRelative = strings.TrimPrefix(cleanRelative, ".")

			absoluteDir := "/home/noxturne/projects"
			if cleanRelative != "" {
				absoluteDir = filepath.Join("/home/noxturne/projects", cleanRelative)
			}

			m.CurrentDirPath = filepath.Clean(absoluteDir)
			m.populateDirList()
			m.SelectedDir = 0
			m.SpawnerState = SpawnerStateTarget
		}
		return m, tea.ClearScreen

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "tab":
			if m.IsError {
				m.IsError = false
				return m, nil
			}
			if m.ActiveTab == TabFleet {
				m.ActiveTab = TabSpawner
			} else {
				m.ActiveTab = TabFleet
				m.refreshFleet()
			}
			return m, nil

		case "r", "R":
			if m.ActiveTab == TabFleet && !m.IsError {
				m.refreshFleet()
			}

		case "up", "k":
			if m.IsError {
				return m, nil
			}
			if m.ActiveTab == TabFleet {
				if m.SelectedTreeItem > 0 {
					m.SelectedTreeItem--
					item := m.TreeItems[m.SelectedTreeItem]
					if item.IsFolder {
						m.TelemetryBuffer = "[Select an active agent to view telemetry]"
					} else {
						return m, m.queryTelemetryCmd()
					}
				}
			} else {
				switch m.SpawnerState {
				case SpawnerStateAgent:
					if m.SelectedAgent > 0 {
						m.SelectedAgent--
					}
				case SpawnerStateDir:
					if m.SelectedDir > 0 {
						m.SelectedDir--
					}
				case SpawnerStateTarget:
					if m.SelectedTarget > 0 {
						m.SelectedTarget--
					}
				case SpawnerStateWindow:
					if m.SelectedWindow > 0 {
						m.SelectedWindow--
					}
				case SpawnerStateSplitDirection:
					if m.SelectedSplitDirection > 0 {
						m.SelectedSplitDirection--
					}
				case SpawnerStateSession:
					if m.SelectedSession > 0 {
						m.SelectedSession--
					}
				case SpawnerStateMacro:
					if m.SelectedMacro > 0 {
						m.SelectedMacro--
					}
				}
			}

		case "down", "j":
			if m.IsError {
				return m, nil
			}
			if m.ActiveTab == TabFleet {
				if m.SelectedTreeItem < len(m.TreeItems)-1 {
					m.SelectedTreeItem++
					item := m.TreeItems[m.SelectedTreeItem]
					if item.IsFolder {
						m.TelemetryBuffer = "[Select an active agent to view telemetry]"
					} else {
						return m, m.queryTelemetryCmd()
					}
				}
			} else {
				switch m.SpawnerState {
				case SpawnerStateAgent:
					if m.SelectedAgent < len(m.Agents)-1 {
						m.SelectedAgent++
					}
				case SpawnerStateDir:
					if m.SelectedDir < len(m.Dirs)-1 {
						m.SelectedDir++
					}
				case SpawnerStateTarget:
					if m.SelectedTarget < len(m.Targets)-1 {
						m.SelectedTarget++
					}
				case SpawnerStateWindow:
					if m.SelectedWindow < len(m.Windows)-1 {
						m.SelectedWindow++
					}
				case SpawnerStateSplitDirection:
					if m.SelectedSplitDirection < len(m.SplitDirections)-1 {
						m.SelectedSplitDirection++
					}
				case SpawnerStateSession:
					if m.SelectedSession < len(m.Sessions)-1 {
						m.SelectedSession++
					}
				case SpawnerStateMacro:
					if m.SelectedMacro < len(m.Macros)-1 {
						m.SelectedMacro++
					}
				}
			}

		case "left", "h":
			if m.IsError {
				return m, nil
			}
			if m.ActiveTab == TabFleet {
				if len(m.TreeItems) > 0 {
					item := m.TreeItems[m.SelectedTreeItem]
					if item.IsFolder && !m.CollapsedPaths[item.Path] {
						m.CollapsedPaths[item.Path] = true
						m.rebuildTree()
					} else {
						// Leaf node or collapsed folder: find parent folder
						var targetPath string
						if item.IsFolder {
							targetPath = item.Path
						} else {
							targetPath = item.Pane.Path
						}
						for idx, ti := range m.TreeItems {
							if ti.IsFolder && ti.Path == targetPath {
								m.SelectedTreeItem = idx
								m.TelemetryBuffer = "[Select an active agent to view telemetry]"
								break
							}
						}
					}
				}
			} else {
				switch m.SpawnerState {
				case SpawnerStateDir:
					m.SpawnerState = SpawnerStateAgent
				case SpawnerStateTarget:
					m.SpawnerState = SpawnerStateDir
				case SpawnerStateWindow:
					m.SpawnerState = SpawnerStateTarget
				case SpawnerStateSplitDirection:
					m.SpawnerState = SpawnerStateWindow
				case SpawnerStateSession:
					m.SpawnerState = SpawnerStateTarget
				case SpawnerStateMacro:
					if m.Targets[m.SelectedTarget] == "Pane Split" {
						m.SpawnerState = SpawnerStateSplitDirection
					} else {
						m.SpawnerState = SpawnerStateSession
					}
				case SpawnerStateExecuting:
					m.SpawnerState = SpawnerStateMacro
				}
			}

		case "enter", "right", "l":
			if m.IsError {
				if msg.String() == "enter" {
					m.IsError = false
				}
				return m, nil
			}
			if m.ActiveTab == TabFleet {
				if len(m.TreeItems) > 0 {
					item := m.TreeItems[m.SelectedTreeItem]
					if item.IsFolder {
						if msg.String() == "enter" {
							m.CollapsedPaths[item.Path] = !m.CollapsedPaths[item.Path]
							m.rebuildTree()
						} else { // right or l
							if m.CollapsedPaths[item.Path] {
								m.CollapsedPaths[item.Path] = false
								m.rebuildTree()
							}
						}
					} else {
						err := tmux.TeleportToPane(item.Pane.PaneID)
						if err != nil {
							m.ErrorMsg = fmt.Sprintf("Failed to teleport: %v", err)
							m.IsError = true
						}
					}
				}
			} else {
				switch m.SpawnerState {
				case SpawnerStateAgent:
					initialDir := "/home/noxturne/projects"
					if _, err := os.Stat(initialDir); err != nil {
						initialDir = "."
					}
					m.CurrentDirPath = initialDir
					m.populateDirList()
					m.SelectedDir = 0
					m.SpawnerState = SpawnerStateDir

				case SpawnerStateDir:
					selectedItem := m.Dirs[m.SelectedDir]
					if selectedItem == "[ Select This Directory ]" {
						m.SpawnerState = SpawnerStateTarget
					} else if selectedItem == ".." {
						m.CurrentDirPath = filepath.Dir(filepath.Clean(m.CurrentDirPath))
						m.populateDirList()
						m.SelectedDir = 0
					} else {
						m.CurrentDirPath = selectedItem
						m.populateDirList()
						m.SelectedDir = 0
					}

				case SpawnerStateTarget:
					if m.Targets[m.SelectedTarget] == "Pane Split" {
						m.populateWindowList()
						m.SelectedWindow = 0
						m.SpawnerState = SpawnerStateWindow
					} else {
						m.populateSessionList()
						m.SelectedSession = 0
						m.SpawnerState = SpawnerStateSession
					}

				case SpawnerStateWindow:
					m.SelectedSplitDirection = 0
					m.SpawnerState = SpawnerStateSplitDirection

				case SpawnerStateSplitDirection:
					m.SpawnerState = SpawnerStateMacro

				case SpawnerStateSession:
					m.SpawnerState = SpawnerStateMacro

				case SpawnerStateMacro:
					m.SpawnerState = SpawnerStateExecuting
					return m, m.spawnAgentCmd()

				case SpawnerStateExecuting:
					if m.ActivePaneID != "" {
						err := tmux.TeleportToPane(m.ActivePaneID)
						if err != nil {
							m.ErrorMsg = fmt.Sprintf("Failed to teleport: %v", err)
							m.IsError = true
						} else {
							m.ActiveTab = TabFleet
							m.SpawnerState = SpawnerStateAgent
							m.refreshFleet()
						}
					}
				}
			}

		case "i": // Spatial Macro: Headless Prompt Compose & Injection (Vim Pipeline)
			if m.ActiveTab == TabFleet && !m.IsError && len(m.TreeItems) > 0 {
				item := m.TreeItems[m.SelectedTreeItem]
				if !item.IsFolder {
					tempFile, err := os.CreateTemp("", "mux-ai-prompt-*.md")
					if err != nil {
						m.ErrorMsg = fmt.Sprintf("Failed to generate draft file: %v", err)
						m.IsError = true
						return m, nil
					}
					tempFile.Close() // close file handle immediately, Vim will open it
					return m, openEditorCmd(tempFile.Name())
				}
			}

		case "f", "/":
			if m.ActiveTab == TabSpawner && m.SpawnerState == SpawnerStateDir && !m.IsError {
				dirs, err := tmux.FindAllProjectSubdirs()
				if err != nil {
					m.ErrorMsg = fmt.Sprintf("Failed to list project directories: %v", err)
					m.IsError = true
					return m, nil
				}
				return m, openFzfCmd(dirs)
			}

		case "m":
			if m.ActiveTab == TabFleet && !m.IsError && len(m.TreeItems) > 0 {
				item := m.TreeItems[m.SelectedTreeItem]
				if !item.IsFolder {
					err := tmux.PullPane(item.Pane.PaneID)
					if err != nil {
						m.ErrorMsg = fmt.Sprintf("Pull failed: %v", err)
						m.IsError = true
					} else {
						m.refreshFleet()
					}
				}
			}

		case "e":
			if m.ActiveTab == TabFleet && !m.IsError && len(m.TreeItems) > 0 {
				item := m.TreeItems[m.SelectedTreeItem]
				if !item.IsFolder {
					err := tmux.IsolatePane(item.Pane.PaneID)
					if err != nil {
						m.ErrorMsg = fmt.Sprintf("Isolate failed: %v", err)
						m.IsError = true
					} else {
						m.refreshFleet()
					}
				}
			}

		case "x": // Kill Pane (matching tmux close shortcut)
			if m.ActiveTab == TabFleet && !m.IsError && len(m.TreeItems) > 0 {
				item := m.TreeItems[m.SelectedTreeItem]
				if !item.IsFolder {
					err := tmux.KillPane(item.Pane.PaneID)
					if err != nil {
						m.ErrorMsg = fmt.Sprintf("Kill failed: %v", err)
						m.IsError = true
					} else {
						m.refreshFleet()
					}
				}
			}

		case "esc":
			if m.IsError {
				m.IsError = false
				return m, nil
			}
			if m.ActiveTab == TabSpawner {
				switch m.SpawnerState {
				case SpawnerStateDir:
					m.SpawnerState = SpawnerStateAgent
				case SpawnerStateTarget:
					m.SpawnerState = SpawnerStateDir
				case SpawnerStateSession:
					m.SpawnerState = SpawnerStateTarget
				case SpawnerStateMacro:
					if m.Targets[m.SelectedTarget] == "Pane Split" {
						m.SpawnerState = SpawnerStateSplitDirection
					} else {
						m.SpawnerState = SpawnerStateSession
					}
				case SpawnerStateExecuting:
					m.SpawnerState = SpawnerStateMacro
				}
			}
		}

	case spawnResultMsg:
		m.refreshFleet()
		if msg.err != nil {
			m.ErrorMsg = msg.err.Error()
			m.IsError = true
			m.SpawnerState = SpawnerStateMacro
		} else {
			m.ActivePaneID = msg.paneID
			m.SpawnerState = SpawnerStateExecuting
		}
	}

	return m, nil
}

// Lip Gloss base styles
var (
	// Color Palette (Premium Indigo, Teal, Rose, Violet, Charcoal)
	colorPurple   = lipgloss.Color("#818cf8") // Indigo-400
	colorTeal     = lipgloss.Color("#2dd4bf") // Teal-400
	colorPink     = lipgloss.Color("#f472b6") // Pink-400
	colorAmber    = lipgloss.Color("#fbbf24") // Amber-400
	colorSlate    = lipgloss.Color("#475569") // Slate-600
	colorGray     = lipgloss.Color("#94a3b8") // Slate-400
	colorDarkGray = lipgloss.Color("#334155") // Slate-700
	colorError    = lipgloss.Color("#fb7185") // Rose-400
	colorSuccess  = lipgloss.Color("#34d399") // Emerald-400
	colorMuted    = lipgloss.Color("#64748b") // Slate-500

	borderColor  = colorPurple
	titleColor   = colorPink
	accentColor  = colorTeal
	grayColor    = colorMuted
	errorRed     = colorError
	successGreen = colorSuccess

	// Base panel styles (dimensions are configured dynamically inside View())
	leftPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			PaddingLeft(1).
			PaddingRight(1)

	rightTopStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			PaddingLeft(1).
			PaddingRight(1)

	rightBottomStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				PaddingLeft(1).
				PaddingRight(1)

	rightSpawnerStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				PaddingLeft(1).
				PaddingRight(1)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(titleColor).
			Align(lipgloss.Center)

	dividerStyle = lipgloss.NewStyle().
			Foreground(colorDarkGray).
			Align(lipgloss.Center)

	headerStyle = lipgloss.NewStyle().
			Foreground(colorTeal).
			Bold(true)

	selectedStyle = lipgloss.NewStyle().
			Foreground(colorPurple).
			Bold(true)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(colorTeal).
				Bold(true)

	normalItemStyle = lipgloss.NewStyle().
			Foreground(colorGray)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f1f5f9"))

	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)
)

// getAgentIcon returns an elegant icon representing the AI agent brand/CLI command
func getAgentIcon(command string) string {
	cmd := strings.ToLower(command)
	if strings.Contains(cmd, "agy") {
		return "󰚩 "
	}
	if strings.Contains(cmd, "gemini") {
		return " "
	}
	if strings.Contains(cmd, "claude") {
		return "󰘧 "
	}
	return " "
}

// renderKeyHelp formats interactive keys as high-contrast tags
func renderKeyHelp(key, desc string) string {
	kStyle := lipgloss.NewStyle().Foreground(colorTeal).Bold(true)
	dStyle := lipgloss.NewStyle().Foreground(colorGray)
	return kStyle.Render(key) + " " + dStyle.Render(desc)
}

// truncateStr chops strings longer than maxLen and appends "..."
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// formatDirPath replaces absolute user home folders with ~ for premium clean look
func formatDirPath(path string) string {
	res := filepath.Clean(path)
	if idx := strings.Index(res, "/projects/"); idx != -1 {
		return res[idx:]
	}
	res = strings.Replace(res, "/home/noxturne/.antigravity-personal", "~", 1)
	res = strings.Replace(res, "/home/noxturne", "~", 1)
	return res
}

// wrapStr wraps string lines to keep TUI layouts within boundaries
func wrapStr(s string, limit int) string {
	var result []string
	words := strings.Split(s, " ")
	var currentLine string
	for _, word := range words {
		if len(currentLine)+len(word)+1 > limit {
			if currentLine != "" {
				result = append(result, currentLine)
			}
			currentLine = word
		} else {
			if currentLine == "" {
				currentLine = word
			} else {
				currentLine += " " + word
			}
		}
	}
	if currentLine != "" {
		result = append(result, currentLine)
	}
	return strings.Join(result, "\n")
}

// renderHeader aligns master tab rows horizontally across full terminal width
func renderHeader(activeTab Tab, width int) string {
	var t1, t2 string
	activeTabStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#ffffff")).
		Background(lipgloss.Color("#4f46e5")). // Deep Indigo
		Padding(0, 2)

	inactiveTabStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#94a3b8")).
		Background(lipgloss.Color("#1e293b")). // Slate-800
		Padding(0, 2)

	if activeTab == TabFleet {
		t1 = activeTabStyle.Render("󰚩  AI FLEET RADAR")
		t2 = inactiveTabStyle.Render("  AGENT SPAWNER")
	} else {
		t1 = inactiveTabStyle.Render("󰚩  AI FLEET RADAR")
		t2 = activeTabStyle.Render("  AGENT SPAWNER")
	}

	tabSpacer := lipgloss.NewStyle().
		Background(lipgloss.Color("#0f172a")).
		Foreground(lipgloss.Color("#334155")).
		Render(" ")

	// Calculate remaining space for the bar
	t1Len := lipgloss.Width(t1)
	t2Len := lipgloss.Width(t2)
	barLen := width - t1Len - t2Len - 4
	if barLen < 2 {
		barLen = 2
	}
	bar := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#334155")).
		Render(strings.Repeat("━", barLen))

	return tabSpacer + t1 + tabSpacer + t2 + tabSpacer + tabSpacer + bar
}

// View renders the TUI screen layout dynamically sizing to full screen window bounds
func (m Model) View() string {
	// Guard fallback for tiny terminal layouts
	if m.Width < 50 || m.Height < 10 {
		return "Terminal screen size is too small to render dashboard."
	}

	// Calculate layout bounds dynamically (full width and height of current terminal pane)
	gridHeight := m.Height - 3 // leave buffer room for header (2 rows) and baseline margin
	if gridHeight < 6 {
		gridHeight = 6
	}

	leftWidth := int(float64(m.Width) * 0.4)
	if leftWidth < 22 {
		leftWidth = 22
	}
	rightWidth := m.Width - leftWidth
	if rightWidth < 22 {
		rightWidth = 22
	}

	// Internal content bounds accounting for borders/padding (borders = 2, padding = 2)
	leftInnerWidth := leftWidth - 4
	leftInnerHeight := gridHeight - 2
	rightInnerWidth := rightWidth - 4

	rightTopHeight := int(float64(gridHeight) * 0.7)
	rightBottomHeight := gridHeight - rightTopHeight

	rightTopInnerHeight := rightTopHeight - 2
	rightBottomInnerHeight := rightBottomHeight - 2

	// Set dynamic sizes on style structures
	leftBorderColor := colorSlate
	rightBorderColor := colorSlate
	if m.ActiveTab == TabFleet {
		leftBorderColor = colorPurple
		rightBorderColor = colorTeal
	} else {
		leftBorderColor = colorTeal
		rightBorderColor = colorPurple
	}

	currentLeftPanelStyle := leftPanelStyle.Copy().Width(leftInnerWidth).Height(leftInnerHeight).BorderForeground(leftBorderColor)
	currentRightTopStyle := rightTopStyle.Copy().Width(rightInnerWidth).Height(rightTopInnerHeight).BorderForeground(rightBorderColor)
	currentRightBottomStyle := rightBottomStyle.Copy().Width(rightInnerWidth).Height(rightBottomInnerHeight).BorderForeground(rightBorderColor)
	currentRightSpawnerStyle := rightSpawnerStyle.Copy().Width(rightInnerWidth).Height(leftInnerHeight).BorderForeground(rightBorderColor)

	dividerStr := strings.Repeat("─", m.Width)

	var s strings.Builder

	// Row 1: Header/Tabs Selector
	s.WriteString(renderHeader(m.ActiveTab, m.Width))
	s.WriteString("\n")

	// Row 2: Main Views
	if m.IsError {
		s.WriteString(lipgloss.NewStyle().Foreground(errorRed).Bold(true).Render("  SYSTEM ERROR") + "\n\n")
		s.WriteString(lipgloss.NewStyle().Foreground(errorRed).Width(m.Width).Render(m.ErrorMsg) + "\n\n")
		s.WriteString(dividerStyle.Render(dividerStr))
		s.WriteString("\n")
		s.WriteString(helpStyle.Render("esc / enter: clear error & go back"))

	} else if m.ActiveTab == TabFleet {
		// ==================== TAB 1: FLEET RADAR ====================
		// Left Panel (Radar Tree List)
		maxLeftContentLines := leftInnerHeight - 1
		if maxLeftContentLines < 1 {
			maxLeftContentLines = 1
		}

		var leftContentLines []string
		if len(m.TreeItems) == 0 {
			leftContentLines = append(leftContentLines, "  [No running agents]")
		} else {
			for i, item := range m.TreeItems {
				var renderLine string
				if item.IsFolder {
					collapsedSymbol := " "
					if m.CollapsedPaths[item.Path] {
						collapsedSymbol = " "
					}
					displayPath := formatDirPath(item.Path)
					if i == m.SelectedTreeItem {
						folderStyle := lipgloss.NewStyle().Foreground(colorPurple).Bold(true)
						symbolStyle := lipgloss.NewStyle().Foreground(colorPink).Bold(true)
						renderLine = fmt.Sprintf("❯ %s%s", symbolStyle.Render(collapsedSymbol), folderStyle.Render(displayPath))
					} else {
						folderMutedStyle := lipgloss.NewStyle().Foreground(colorGray).Bold(true)
						symbolStyle := lipgloss.NewStyle().Foreground(colorAmber).Bold(true)
						renderLine = fmt.Sprintf("  %s%s", symbolStyle.Render(collapsedSymbol), folderMutedStyle.Render(displayPath))
					}
					leftContentLines = append(leftContentLines, truncateStr(renderLine, leftInnerWidth+50))
				} else {
					pane := item.Pane
					agentIcon := getAgentIcon(pane.Command)
					if i == m.SelectedTreeItem {
						leafSelectStyle := lipgloss.NewStyle().Foreground(colorTeal).Bold(true)
						paneText := fmt.Sprintf("%s  %s (W: %s)", agentIcon, leafSelectStyle.Render(pane.Command), pane.WindowID)
						renderLine = fmt.Sprintf("  └── ❯ %s", paneText)
					} else {
						paneText := fmt.Sprintf("%s  %s (W: %s)", agentIcon, pane.Command, pane.WindowID)
						renderLine = fmt.Sprintf("  └──   %s", paneText)
					}
					leftContentLines = append(leftContentLines, truncateStr(renderLine, leftInnerWidth+50))
				}
			}
		}

		// Scroll tree view helper: keeps cursor focused within leftInnerHeight viewport limits
		startIndex := 0
		if m.SelectedTreeItem >= maxLeftContentLines {
			startIndex = m.SelectedTreeItem - maxLeftContentLines + 1
		}
		endIndex := startIndex + maxLeftContentLines
		if endIndex > len(leftContentLines) {
			endIndex = len(leftContentLines)
		}

		slicedLeftContent := leftContentLines[startIndex:endIndex]
		for len(slicedLeftContent) < maxLeftContentLines {
			slicedLeftContent = append(slicedLeftContent, "")
		}

		leftLinesSubset := []string{headerStyle.Render(" 󰙅  ACTIVE RADAR FLEET")}
		leftLinesSubset = append(leftLinesSubset, slicedLeftContent...)
		leftView := currentLeftPanelStyle.Render(strings.Join(leftLinesSubset, "\n"))

		// Top Right Panel (Live Telemetry Viewport)
		maxTelemetryContentLines := rightTopInnerHeight - 1
		if maxTelemetryContentLines < 1 {
			maxTelemetryContentLines = 1
		}

		var telemetryContentLines []string
		if m.TelemetryBuffer == "" {
			telemetryContentLines = append(telemetryContentLines, " [Select an active agent to view telemetry]")
		} else {
			rawLines := strings.Split(m.TelemetryBuffer, "\n")
			// Slice off the bottom 8 interface helper lines (input prompt, status bar, and helper descriptions)
			if len(rawLines) > 8 {
				rawLines = rawLines[:len(rawLines)-8]
			} else {
				rawLines = nil
			}
			for _, rl := range rawLines {
				// Truncate to rightInnerWidth-6 to perfectly prevent wrapping inside border and padding boundaries
				telemetryContentLines = append(telemetryContentLines, " "+truncateStr(rl, rightInnerWidth-6))
			}
		}

		// Trim and Pad telemetry content exactly to maxTelemetryContentLines
		if len(telemetryContentLines) > maxTelemetryContentLines {
			telemetryContentLines = telemetryContentLines[len(telemetryContentLines)-maxTelemetryContentLines:]
		}
		for len(telemetryContentLines) < maxTelemetryContentLines {
			telemetryContentLines = append(telemetryContentLines, "")
		}

		telemetryLines := []string{headerStyle.Render(" 󱚞  LIVE AGENT TELEMETRY")}
		telemetryLines = append(telemetryLines, telemetryContentLines...)
		rightTop := currentRightTopStyle.Render(strings.Join(telemetryLines, "\n"))

		// Bottom Right Panel (Action Deck Viewport)
		agentClient := ""
		if len(m.TreeItems) > 0 && m.SelectedTreeItem < len(m.TreeItems) {
			item := m.TreeItems[m.SelectedTreeItem]
			if !item.IsFolder {
				cmdLower := strings.ToLower(item.Pane.Command)
				if strings.Contains(cmdLower, "agy") {
					agentClient = "Antigravity"
				} else if strings.Contains(cmdLower, "gemini") {
					agentClient = "Gemini"
				} else if strings.Contains(cmdLower, "claude") {
					agentClient = "Claude"
				} else {
					agentClient = item.Pane.Command
				}
			}
		}

		maxDeckContentLines := rightBottomInnerHeight - 1
		if maxDeckContentLines < 1 {
			maxDeckContentLines = 1
		}

		var deckContentLines []string
		goalText := "[No active agent selected]"
		if len(m.TreeItems) > 0 && m.SelectedTreeItem < len(m.TreeItems) {
			item := m.TreeItems[m.SelectedTreeItem]
			if !item.IsFolder {
				goalText = item.Pane.ActiveGoal
			} else {
				goalText = "[Directory: " + filepath.Base(item.Path) + "]"
			}
		}

		// 1. Render "Target Goal" label on its own line without the ":"
		deckContentLines = append(deckContentLines, " "+lipgloss.NewStyle().Foreground(colorPurple).Bold(true).Render("  Target Goal"))

		// 2. Render the active plan text on the line below with a premium background color
		planStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1e1e2e")). // elegant dark charcoal text
			Background(colorPurple).               // Indigo background
			PaddingLeft(1).
			PaddingRight(1).
			Bold(true)

		deckContentLines = append(deckContentLines, "  "+planStyle.Render(truncateStr(goalText, rightInnerWidth-6)))

		// Fill in dynamic controls to align footer to panel limits
		if maxDeckContentLines >= 4 {
			deckContentLines = append(deckContentLines, "")
			row1 := []string{
				renderKeyHelp("Enter", "Teleport"),
				renderKeyHelp("m", "Magnet"),
				renderKeyHelp("e", "Isolate"),
			}
			row2 := []string{
				renderKeyHelp("x", "Kill Pane"),
				renderKeyHelp("i", "Compose"),
				renderKeyHelp("r/R", "Refresh"),
				renderKeyHelp("Tab", "Spawner"),
			}
			deckContentLines = append(deckContentLines, " "+strings.Join(row1, "  •  "))
			deckContentLines = append(deckContentLines, " "+strings.Join(row2, "  •  "))
		} else if maxDeckContentLines >= 2 {
			deckContentLines = append(deckContentLines, " "+renderKeyHelp("Enter", "Teleport")+" • "+renderKeyHelp("m", "Magnet")+" • "+renderKeyHelp("e", "Isolate")+" • "+renderKeyHelp("x", "Kill")+" • "+renderKeyHelp("i", "Compose"))
		}

		// Trim and Pad deck content exactly to maxDeckContentLines
		if len(deckContentLines) > maxDeckContentLines {
			deckContentLines = deckContentLines[:maxDeckContentLines]
		}
		for len(deckContentLines) < maxDeckContentLines {
			deckContentLines = append(deckContentLines, "")
		}

		var deckLines []string
		if agentClient != "" {
			deckLines = append(deckLines, headerStyle.Render(" 󰓅  ACTION DECK")+"  "+lipgloss.NewStyle().Foreground(colorTeal).Bold(true).Render("("+agentClient+")"))
		} else {
			deckLines = append(deckLines, headerStyle.Render(" 󰓅  ACTION DECK"))
		}
		deckLines = append(deckLines, deckContentLines...)
		rightBottom := currentRightBottomStyle.Render(strings.Join(deckLines, "\n"))

		// Render dashboard layout side-by-side
		rightPanel := lipgloss.JoinVertical(lipgloss.Left, rightTop, rightBottom)
		dashboard := lipgloss.JoinHorizontal(lipgloss.Top, leftView, rightPanel)
		s.WriteString(dashboard)

	} else {
		// ==================== TAB 2: AGENT SPAWNER ====================
		// Left Spawner Panel (Menu selector)
		var leftLines []string
		switch m.SpawnerState {
		case SpawnerStateAgent:
			leftLines = append(leftLines, headerStyle.Render(" 󰚩  SELECT AGENT")+"\n")
			for i, agent := range m.Agents {
				icon := getAgentIcon(agent)
				if i == m.SelectedAgent {
					leftLines = append(leftLines, selectedItemStyle.Render(fmt.Sprintf(" ❯ %s  %s", icon, agent)))
				} else {
					leftLines = append(leftLines, normalItemStyle.Render(fmt.Sprintf("   %s  %s", icon, agent)))
				}
			}

		case SpawnerStateDir:
			leftLines = append(leftLines, " "+normalStyle.Render(fmt.Sprintf("Agent: %s", selectedItemStyle.Render(m.Agents[m.SelectedAgent]))))
			leftLines = append(leftLines, " "+normalStyle.Render(fmt.Sprintf("Path:  %s\n", selectedItemStyle.Render(truncateStr(formatDirPath(m.CurrentDirPath), leftInnerWidth-8)))))
			leftLines = append(leftLines, headerStyle.Render("   SELECT DIRECTORY")+"\n")

			for i, dir := range m.Dirs {
				var displayDir string
				var icon string
				if dir == "[ Select This Directory ]" {
					displayDir = "[ Select This Directory ]"
					icon = "󰓾 "
				} else if dir == ".." {
					displayDir = ".. (Go Up)"
					icon = " "
				} else {
					displayDir = filepath.Base(dir)
					icon = " "
				}

				if i == m.SelectedDir {
					leftLines = append(leftLines, selectedItemStyle.Render(fmt.Sprintf(" ❯ %s  %s", icon, truncateStr(displayDir, leftInnerWidth-6))))
				} else {
					leftLines = append(leftLines, normalItemStyle.Render(fmt.Sprintf("   %s  %s", icon, truncateStr(displayDir, leftInnerWidth-6))))
				}
			}

		case SpawnerStateTarget:
			leftLines = append(leftLines, " "+normalStyle.Render(fmt.Sprintf("Agent: %s", selectedItemStyle.Render(m.Agents[m.SelectedAgent]))))
			leftLines = append(leftLines, " "+normalStyle.Render(fmt.Sprintf("Dir:   %s\n", selectedItemStyle.Render(truncateStr(formatDirPath(m.CurrentDirPath), leftInnerWidth-8)))))
			leftLines = append(leftLines, headerStyle.Render("   SELECT LAYOUT TARGET")+"\n")

			for i, target := range m.Targets {
				var icon string
				if target == "Pane Split" {
					icon = " "
				} else {
					icon = " "
				}
				if i == m.SelectedTarget {
					leftLines = append(leftLines, selectedItemStyle.Render(fmt.Sprintf(" ❯ %s  %s", icon, target)))
				} else {
					leftLines = append(leftLines, normalItemStyle.Render(fmt.Sprintf("   %s  %s", icon, target)))
				}
			}

		case SpawnerStateWindow:
			leftLines = append(leftLines, " "+normalStyle.Render(fmt.Sprintf("Agent:  %s", selectedItemStyle.Render(m.Agents[m.SelectedAgent]))))
			leftLines = append(leftLines, " "+normalStyle.Render(fmt.Sprintf("Dir:    %s", selectedItemStyle.Render(truncateStr(formatDirPath(m.CurrentDirPath), leftInnerWidth-8)))))
			leftLines = append(leftLines, " "+normalStyle.Render(fmt.Sprintf("Layout: %s\n", selectedItemStyle.Render(m.Targets[m.SelectedTarget]))))
			leftLines = append(leftLines, headerStyle.Render("   SELECT TARGET WINDOW")+"\n")

			headerLen := 5 // number of lines prepended above
			availHeight := leftInnerHeight - headerLen - 1
			if availHeight < 2 {
				availHeight = 2
			}

			var listLines []string
			for i, win := range m.Windows {
				if i == m.SelectedWindow {
					listLines = append(listLines, selectedItemStyle.Render(fmt.Sprintf(" ❯   %s", truncateStr(win, leftInnerWidth-6))))
				} else {
					listLines = append(listLines, normalItemStyle.Render(fmt.Sprintf("     %s", truncateStr(win, leftInnerWidth-6))))
				}
			}

			startIndex := 0
			if m.SelectedWindow >= availHeight {
				startIndex = m.SelectedWindow - availHeight + 1
			}
			endIndex := startIndex + availHeight
			if endIndex > len(listLines) {
				endIndex = len(listLines)
			}
			listSubset := listLines[startIndex:endIndex]
			leftLines = append(leftLines, listSubset...)

		case SpawnerStateSplitDirection:
			leftLines = append(leftLines, " "+normalStyle.Render(fmt.Sprintf("Agent:  %s", selectedItemStyle.Render(m.Agents[m.SelectedAgent]))))
			leftLines = append(leftLines, " "+normalStyle.Render(fmt.Sprintf("Dir:    %s", selectedItemStyle.Render(truncateStr(formatDirPath(m.CurrentDirPath), leftInnerWidth-8)))))
			leftLines = append(leftLines, " "+normalStyle.Render(fmt.Sprintf("Layout: %s", selectedItemStyle.Render(m.Targets[m.SelectedTarget]))))
			var winStr string
			if m.SelectedWindow > 0 && m.SelectedWindow < len(m.Windows) {
				winStr = m.Windows[m.SelectedWindow]
			} else {
				winStr = "[ Active Window ]"
			}
			leftLines = append(leftLines, " "+normalStyle.Render(fmt.Sprintf("Window: %s\n", selectedItemStyle.Render(truncateStr(winStr, leftInnerWidth-9)))))
			leftLines = append(leftLines, headerStyle.Render("   SELECT SPLIT DIRECTION")+"\n")

			for i, dir := range m.SplitDirections {
				var icon string
				if strings.Contains(dir, "Vertical") {
					icon = " "
				} else {
					icon = " "
				}
				if i == m.SelectedSplitDirection {
					leftLines = append(leftLines, selectedItemStyle.Render(fmt.Sprintf(" ❯ %s  %s", icon, dir)))
				} else {
					leftLines = append(leftLines, normalItemStyle.Render(fmt.Sprintf("   %s  %s", icon, dir)))
				}
			}

		case SpawnerStateSession:
			leftLines = append(leftLines, " "+normalStyle.Render(fmt.Sprintf("Agent:  %s", selectedItemStyle.Render(m.Agents[m.SelectedAgent]))))
			leftLines = append(leftLines, " "+normalStyle.Render(fmt.Sprintf("Dir:    %s", selectedItemStyle.Render(truncateStr(formatDirPath(m.CurrentDirPath), leftInnerWidth-8)))))
			leftLines = append(leftLines, " "+normalStyle.Render(fmt.Sprintf("Layout: %s\n", selectedItemStyle.Render(m.Targets[m.SelectedTarget]))))
			leftLines = append(leftLines, headerStyle.Render("   SELECT TARGET SESSION")+"\n")

			headerLen := 5 // number of lines prepended above
			availHeight := leftInnerHeight - headerLen - 1
			if availHeight < 2 {
				availHeight = 2
			}

			var listLines []string
			for i, sess := range m.Sessions {
				if i == m.SelectedSession {
					listLines = append(listLines, selectedItemStyle.Render(fmt.Sprintf(" ❯ %s  %s", " ", truncateStr(sess, leftInnerWidth-6))))
				} else {
					listLines = append(listLines, normalItemStyle.Render(fmt.Sprintf("   %s  %s", " ", truncateStr(sess, leftInnerWidth-6))))
				}
			}

			startIndex := 0
			if m.SelectedSession >= availHeight {
				startIndex = m.SelectedSession - availHeight + 1
			}
			endIndex := startIndex + availHeight
			if endIndex > len(listLines) {
				endIndex = len(listLines)
			}
			listSubset := listLines[startIndex:endIndex]
			leftLines = append(leftLines, listSubset...)

		case SpawnerStateMacro:
			leftLines = append(leftLines, " "+normalStyle.Render(fmt.Sprintf("Agent:  %s", selectedItemStyle.Render(m.Agents[m.SelectedAgent]))))
			leftLines = append(leftLines, " "+normalStyle.Render(fmt.Sprintf("Dir:    %s", selectedItemStyle.Render(truncateStr(formatDirPath(m.CurrentDirPath), leftInnerWidth-9)))))
			leftLines = append(leftLines, " "+normalStyle.Render(fmt.Sprintf("Layout: %s", selectedItemStyle.Render(m.Targets[m.SelectedTarget]))))
			if m.Targets[m.SelectedTarget] == "Pane Split" {
				var winStr string
				if m.SelectedWindow > 0 && m.SelectedWindow < len(m.Windows) {
					winStr = m.Windows[m.SelectedWindow]
				} else {
					winStr = "[ Active Window ]"
				}
				leftLines = append(leftLines, " "+normalStyle.Render(fmt.Sprintf("Window: %s", selectedItemStyle.Render(truncateStr(winStr, leftInnerWidth-9)))))
				var splitStr string
				if m.SelectedSplitDirection == 1 {
					splitStr = "Vertical (-v)"
				} else {
					splitStr = "Horizontal (-h)"
				}
				leftLines = append(leftLines, " "+normalStyle.Render(fmt.Sprintf("Split:  %s\n", selectedItemStyle.Render(splitStr))))
			} else {
				var sessionStr string
				if m.SelectedSession > 0 && m.SelectedSession < len(m.Sessions) {
					sessionStr = m.Sessions[m.SelectedSession]
				} else {
					sessionStr = "[ Active Session ]"
				}
				leftLines = append(leftLines, " "+normalStyle.Render(fmt.Sprintf("Session: %s\n\n", selectedItemStyle.Render(truncateStr(sessionStr, leftInnerWidth-10)))))
			}
			leftLines = append(leftLines, headerStyle.Render(" 󱐋  SELECT MACRO")+"\n")

			macroNames := []string{"Just Spawn (No Macro)", "Implement", "Cook It", "Wrap It Up", "Recon"}
			for i, name := range macroNames {
				var icon string
				switch name {
				case "Just Spawn (No Macro)":
					icon = " "
				case "Implement":
					icon = " "
				case "Cook It":
					icon = "󰠳 "
				case "Wrap It Up":
					icon = "󰏖 "
				case "Recon":
					icon = " "
				}
				if i == m.SelectedMacro {
					leftLines = append(leftLines, selectedItemStyle.Render(fmt.Sprintf(" ❯ %s  %s", icon, name)))
				} else {
					leftLines = append(leftLines, normalItemStyle.Render(fmt.Sprintf("   %s  %s", icon, name)))
				}
			}

		case SpawnerStateExecuting:
			leftLines = append(leftLines, lipgloss.NewStyle().Foreground(colorSuccess).Bold(true).Render("   AGENT SPAWNING...")+"\n")
			leftLines = append(leftLines, " "+fmt.Sprintf("Agent:  %s", normalStyle.Render(m.Agents[m.SelectedAgent])))
			leftLines = append(leftLines, " "+fmt.Sprintf("Dir:    %s", normalStyle.Render(truncateStr(formatDirPath(m.CurrentDirPath), leftInnerWidth-9))))
			leftLines = append(leftLines, " "+fmt.Sprintf("Layout: %s", normalStyle.Render(m.Targets[m.SelectedTarget])))
			if m.Targets[m.SelectedTarget] == "Pane Split" {
				var winStr string
				if m.SelectedWindow > 0 && m.SelectedWindow < len(m.Windows) {
					winStr = m.Windows[m.SelectedWindow]
				} else {
					winStr = "[ Active Window ]"
				}
				leftLines = append(leftLines, " "+fmt.Sprintf("Window: %s", normalStyle.Render(truncateStr(winStr, leftInnerWidth-9))))
				var splitStr string
				if m.SelectedSplitDirection == 1 {
					splitStr = "Vertical (-v)"
				} else {
					splitStr = "Horizontal (-h)"
				}
				leftLines = append(leftLines, " "+fmt.Sprintf("Split:  %s", normalStyle.Render(splitStr)))
			} else {
				var sessionStr string
				if m.SelectedSession > 0 && m.SelectedSession < len(m.Sessions) {
					sessionStr = m.Sessions[m.SelectedSession]
				} else {
					sessionStr = "[ Active Session ]"
				}
				leftLines = append(leftLines, " "+fmt.Sprintf("Session: %s", normalStyle.Render(truncateStr(sessionStr, leftInnerWidth-10))))
			}
			macroNames := []string{"Just Spawn (No Macro)", "Implement", "Cook It", "Wrap It Up", "Recon"}
			leftLines = append(leftLines, " "+fmt.Sprintf("Macro:  %s", normalStyle.Render(macroNames[m.SelectedMacro])))
			leftLines = append(leftLines, " "+fmt.Sprintf("Pane:   %s", selectedItemStyle.Render(m.ActivePaneID)))
		}

		if len(leftLines) > leftInnerHeight {
			leftLines = leftLines[:leftInnerHeight]
		}
		for len(leftLines) < leftInnerHeight {
			leftLines = append(leftLines, "")
		}
		leftView := currentLeftPanelStyle.Render(strings.Join(leftLines, "\n"))

		// Right Spawner Panel (Command Preview Viewport)
		var rightLines []string
		rightLines = append(rightLines, headerStyle.Render("   TMUX SPAWN COMMAND PREVIEW")+"\n")

		var previewContent string
		if m.SpawnerState != SpawnerStateExecuting {
			previewContent = wrapStr(m.getPreviewCommand(), rightInnerWidth-2)
		} else {
			previewContent = "Process successfully launched in target session."
		}

		rawPreviewLines := strings.Split(previewContent, "\n")
		for _, rl := range rawPreviewLines {
			rightLines = append(rightLines, " "+helpStyle.Render(rl))
		}

		var footerHelp string
		if m.SpawnerState == SpawnerStateDir {
			footerHelp = renderKeyHelp("↑/↓", "move") + " • " + renderKeyHelp("Enter", "select") + " • " + renderKeyHelp("f", "fzf search") + " • " + renderKeyHelp("Esc", "back")
		} else if m.SpawnerState == SpawnerStateExecuting {
			footerHelp = renderKeyHelp("Enter", "teleport to agent pane") + " • " + renderKeyHelp("Esc", "back")
		} else {
			footerHelp = renderKeyHelp("↑/↓", "move") + " • " + renderKeyHelp("Enter", "select") + " • " + renderKeyHelp("Esc", "back")
		}

		footerLines := []string{
			"",
			dividerStyle.Render(strings.Repeat("─", rightInnerWidth)),
			" " + footerHelp,
		}

		usedLines := len(rightLines) + len(footerLines)
		paddingCount := leftInnerHeight - usedLines
		var paddedBody []string
		paddedBody = append(paddedBody, rightLines...)
		for i := 0; i < paddingCount; i++ {
			paddedBody = append(paddedBody, "")
		}
		paddedBody = append(paddedBody, footerLines...)

		if len(paddedBody) > leftInnerHeight {
			paddedBody = paddedBody[:leftInnerHeight]
		}
		for len(paddedBody) < leftInnerHeight {
			paddedBody = append(paddedBody, "")
		}
		rightView := currentRightSpawnerStyle.Render(strings.Join(paddedBody, "\n"))

		dashboard := lipgloss.JoinHorizontal(lipgloss.Top, leftView, rightView)
		s.WriteString(dashboard)
	}

	return s.String()
}
