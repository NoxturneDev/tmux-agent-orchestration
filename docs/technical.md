# Technical Architecture: Colorized Viewport Polling Telemetry System

This document outlines the design and technical implementation details of the real-time snappy colorized viewport polling telemetry system inside the `mux-ai-deck` TUI.

## 1. Telemetry Lifecyle & Event Loop

```
+-------------+        queryTelemetryCmd()        +-------------------------+
| telemetry-  | -------------------------------> | tmux capture-pane -e -p |
| TickMsg     |                                  +-------------------------+
+-------------+                                               |
       ^                                                      v
       |                                           telemetryResultMsg
       |           100ms Periodic Ticks                       |
       +------------------------------------------------------+
```

### A. Asynchronous Periodic Polling
Telemetry queries are driven in the background using non-blocking Bubble Tea commands:
1. **Periodic Telemetry Ticks:** Every 100 milliseconds, a `telemetryTickMsg` is processed inside the TUI's main `Update()` event loop.
2. **Background Command Spawn:** When a tick is received and the user is on `TabFleet` with no active system error, `queryTelemetryCmd()` is invoked to return a `tea.Cmd`.
3. **Non-Blocking Execution:** The returned command function runs in a separate background goroutine. It queries the target tmux pane's terminal buffer using the `-e` flag to keep raw ANSI escapes intact.
4. **Message Dispatch:** Upon receiving the raw buffer string, it returns a `telemetryResultMsg{paneID, buffer, err}` back to the main thread.

### B. Viewport Rendering & Auto-Scrolling
When `telemetryResultMsg` is received:
* **Target Integrity Check:** The TUI verifies that the pane ID in the message matches the currently highlighted item's pane ID in the tree selection list to prevent rendering lag/races from stale queries.
* **ANSI Stripping (Filtered):** Clean text is formatted using `cleanAnsiAndTabs()` (which cleans only cursor movement controls and carriage returns, keeping standard text foreground/background color codes intact).
* **Viewport Integration:** Standard `github.com/charmbracelet/bubbles/viewport` is populated with the colorized buffer via `m.Viewport.SetContent(cleanText)`.
* **Real-Time Scrolling:** Calls `m.Viewport.GotoBottom()` on every new tick update to emulate a native terminal emulator screen auto-scroll.

---

## 2. Teleportation and State Management

### A. Pointer Receiver Model
To prevent Bubble Tea state copy-loss where mutations (such as updating active selections, tab views, and viewport sizes) are discarded:
* The TUI model methods `Init()`, `Update()`, `View()`, `queryTelemetryCmd()`, and `getActivePaneID()` are implemented on pointer receivers (`*Model`).
* The application is initialized as a pointer reference to the initial model (`&initialModel`) inside `main.go`.

### B. Active Telemetry Toggling
To keep TUI memory and layout footprint clean:
* **Directory Highlight:** Selecting a directory folder automatically clears the telemetry buffer (`TelemetryBuffer = ""`) and viewport text, displaying a placeholder `[Select an active agent to view telemetry]` until a pane is selected.
* **Pane Highlight:** Moving selection focus to a pane immediately fires `queryTelemetryCmd()` for instant terminal pre-population, bypassing the next 100ms tick delay.
* **Tab Switcher:** Transitioning to `TabSpawner` clears the viewport, and switching back to `TabFleet` instantly queries the active pane.
