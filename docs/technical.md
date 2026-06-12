# Technical Architecture: Event-Driven Real-Time Telemetry Streaming System

This document outlines the design and technical implementation details of the high-performance, event-driven streaming telemetry system inside the `mux-ai-deck` TUI.

## 1. Telemetry Lifecycle & Asynchronous Tailer Loop

```
+---------------+     tmux.StartPipePane()     +-----------------------------+
| highlighted   | ---------------------------> | Write pane raw ANSI history |
| tree node ID  |                              | & start tmux pipe-pane to   |
+---------------+                              | /tmp/mux-agent-<ID>-str.log |
        |                                      +-----------------------------+
        v                                                     |
+---------------+                                             v
| readNextChunk | <---------------------------------- New output append events
+---------------+
        |  1. File grown (fi.Size() > offset)
        v  2. Read chunk bytes
  streamMsg{chunk, offset}
        |
        +---> Update model buffer & viewport
        |
        +---> Spawn next readNextChunk(offset)
```

### A. Zero-Polling Streaming Mechanics
Rather than polling the terminal output at periodic time intervals (e.g. every 100ms), the orchestrator utilizes `tmux pipe-pane` to stream standard output from the targeted pane to a temporary file:
1. **Streaming Activation:** When a pane becomes active or highlighted in the UI, `TransitionStream(paneID)` is invoked.
2. **Pre-population (CapturePaneRaw):** Before initiating the stream, `tmux capture-pane -e -p` extracts the raw terminal buffer (including ANSI colors) to initialize the log file. This ensures the viewport is pre-populated with past screen history.
3. **Tmux Pipe-Pane Execution:** The command `tmux pipe-pane -t <PaneID> "cat -u > '/tmp/mux-agent-<PaneID>-stream.log'"` is run. Any characters output by the pane are streamed in real time to the file without spawning additional processes per update.
4. **Escape & Option Handling:**
   - Tmux formats treat `%` as specifiers; we escape it as `%%` to prevent padding/expansion side effects.
   - We omit the `-o` toggle argument so the command always starts/overwrites the stream reliably.

### B. Asynchronous Bubble Tea Tailer Loop
Bubble Tea processes telemetry events via a non-blocking background loop:
1. **Background Read (`readNextChunk`):** A Bubble Tea command executes a loop that periodically checks the file size. If the file size has grown larger than the last read `offset`, it reads the new chunk of bytes and returns a `streamMsg`.
2. **Model Processing:** When the main loop receives a `streamMsg` matching the current `ActiveStreamPaneID`, it:
   - Updates the internal file read `offset`.
   - Appends the colorized chunk to `TelemetryBuffer`.
   - Cleans the text via `cleanAnsiAndTabs()` and updates the viewport.
   - Triggers the next `readNextChunk` command with the new `offset`.
3. **Graceful Cancellation:** If the user transitions to another pane or exits, a cancellation channel (`StreamCancelChan`) is closed, causing the active goroutine to terminate and return a `streamFinishedMsg`.

---

## 2. Teleportation and State Management

### A. Lifecycle Cleanup & Signal Handling
To ensure temporary files and dangling tmux pipe-panes do not leak, a comprehensive cleanup routine (`cleanUpTempStreams`) is executed:
* **Startup:** Wipes any stale `/tmp/mux-agent-*-stream.log` files from previous runs.
* **Shutdown:** Intercepts termination signals (`SIGINT`, `SIGTERM`) and regular application exits to stop any active `pipe-pane` streaming tasks and delete the temporary files.

### B. Pointer Receiver Model
To prevent Bubble Tea state copy-loss where mutations (such as updating active selections, tab views, and viewport sizes) are discarded:
* The TUI model methods `Init()`, `Update()`, `View()`, `TransitionStream()`, and `getActivePaneID()` are implemented on pointer receivers (`*Model`).
* The application is initialized as a pointer reference to the initial model (`&initialModel`) inside `main.go`.

---

## 3. Layout Alignment & Dimension Constraints

To prevent visual glitches (such as double borders, line wrapping, or layout distortion) when using Lipgloss styled borders and padding, all inner layouts conform to strict dimension bounds:

### A. Border & Padding Allowances
In Lipgloss, setting a `.Width(W)` or `.Height(H)` defines the **outer** dimensions. The space consumed by the styling elements must be subtracted to calculate the available **inner** space for contents:
* **Horizontal Border + Padding:** The standard rounded border consumes `2` columns (left/right). The left and right paddings consume `2` columns. Therefore, the available inner width for content is `W - 4`.
* **Vertical Border:** The top and bottom borders consume `2` rows. There is no vertical padding. Therefore, the available inner height for content is `H - 2`.

### B. Inner Content Sizing Guidelines
* **Left Fleet Panel / Spawner Menu:** Inner height is `leftInnerHeight - 2`. Content includes `1` header line, so `maxLeftContentLines` must be at most `leftInnerHeight - 3`.
* **Telemetry Viewport:** Outer width is `rightInnerWidth` (where `rightInnerWidth = rightWidth - 4`). Available inner width is `rightInnerWidth - 4 = rightWidth - 8`. Viewport height is `rightTopInnerHeight - 3` to account for `1` header line and `2` border lines.
* **Action Deck:** Inner height is `rightBottomInnerHeight - 2`. Content includes `1` header line, so `maxDeckContentLines` is restricted to `rightBottomInnerHeight - 3`.
* **Spawner Command Preview:** Body must be padded/trimmed to `leftInnerHeight - 2` rows. The horizontal divider length must be restricted to `rightInnerWidth - 4` to prevent overflow-induced wrapping.

### C. Telemetry Stream ANSI & Control Character Filtering
To prevent incoming telemetry stream data from corrupting the parent Bubble Tea TUI render layout (e.g. absolute cursor movements moving the cursor outside the live preview box or clearing the entire physical screen), the `cleanAnsiAndTabs()` function parses the stream buffer:
1. **Frame Extraction (Slicing on Screen Clear/Cursor Home):** ANSI commands for screen clears and cursor home sequences (matching `\x1b\[([0-9;?]*[JH])`) indicate a screen redraw. The parser slices the buffer to only keep content after the last clear/home sequence, discarding stale historical redraw frames.
2. **Carriage Return & Backspace Emulation:**
   - **Carriage Return (`\r`):** Carriage returns are processed line-by-line by splitting on `\r` and retaining only the last non-empty segment of the line. This emulates terminal carriage-return overwrites (e.g., progress indicators and spinners) inside the TUI viewport.
   - **Backspace (`\b`):** Deletes the preceding character in the string buffer.
3. **ANSI Sequence Stripping:**
   - Color code escapes ending with `m` are retained to preserve terminal styling.
   - All non-color escapes (matching `\x1b\[[0-9;?]*[A-Za-ln-z~]`) are stripped. This specific range explicitly excludes the color terminator `m` to prevent greedily matching code digits as non-color characters.
   - VT100 character set modifier escapes (matching `\x1b[^[]`) are stripped.
