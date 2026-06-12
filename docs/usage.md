# Tmux AI Orchestrator (mux-ai-deck)

A premium TUI dashboard and command deck built in Go using bubbletea and lipgloss to manage, monitor, and spawn containerized and local AI coding agents inside tmux.

## Features

### 1. AI Fleet Radar (Active Radar Fleet)
- **Host-Level Process Scanning**: Bypasses container and sandbox namespace isolation by querying the host process list via `tmux run-shell`.
- **Recursive Process Traversal**: Identifies background-running agent processes (like Node-based `gemini` or `nsjail`-sandboxed `agy` instances) by parsing the process tree and command-line arguments.
- **Live Telemetry Viewport**: Event-driven real-time telemetry streaming using `tmux pipe-pane` and an asynchronous tailer loop, achieving sub-millisecond updates with zero CPU polling overhead.
- **Auto-Scrolling Viewport**: Integrates `github.com/charmbracelet/bubbles/viewport` to render telemetry outputs cleanly with real-time `GotoBottom()` auto-scrolling that mimics a native terminal window.
- **Goal Extraction**: Auto-extracts target objectives from local active agent plans (`.agents/plan/active_plan.md`) for quick overview.
- **Fleet Analytics Status Bar**: Renders a dedicated analytics line at the bottom of the active radar fleet tree displaying real-time metrics (Total Deployed, Doing Task/Busy, and Waiting for Task/Idle).

### 2. Agent Spawner
- **Directory Selector**: Browse local workspaces or use fuzzy search (`fzf`) to select target directory layouts.
- **Layout Target Routing**:
  - **Pane Split**: Choose target window and select vertical (`-v`) or horizontal (`-h`) split.
  - **New Window**: Select target active session and route the new window directly to the chosen tmux session.
- **Command Preview**: Shows real-time compilation of the `tmux` command to be executed.
- **Prompt Macro Injection**: Quick-inject predefined prompts (e.g. `Implement`, `Cook It`, `Wrap It Up`, `Recon`) or compose custom prompts dynamically via `$EDITOR` (Vim/Nano pipeline).

## TUI Keyboard Shortcuts

### Navigation
- `Tab`: Toggle between **AI Fleet Radar** and **Agent Spawner** tabs.
- `↑ / ↓` or `k / j`: Move selection.
- `← / →` or `h / l`: Expand / collapse directory paths (Fleet Radar) or navigate spawner configuration steps (Spawner).
- `Esc`: Clear system error or go back one step in the spawner config.

### Fleet Radar Controls
- `Enter`: Teleport to selected agent pane (switch tmux client focus to the active pane).
- `i`: Compose custom prompt (opens default editor) and inject via stdin buffer into the selected agent pane.
- `m`: Magnet (join-pane) — pull the selected pane into the active window.
- `e`: Isolate (break-pane) — upgrade the selected pane into its own full-screen standalone window.
- `x`: Kill selected agent pane.
- `r` / `R`: Force refresh active fleet list.

### Spawner Execution Controls
- `Enter`: Press on executing screen to teleport focus to the newly spawned agent's pane, reset the spawner, and switch back to the Fleet Radar tab.

## UI Design & Aesthetics
- Styled with a premium **24-bit TrueColor Indigo/Teal** theme.
- Structured tag-based key helpers for navigation feedback.
- Strict use of **Nerd Font** symbols (`󰚩`, ``, `󰓎`, ``, ``, ``, etc.) for visual accents instead of standard emojis.
