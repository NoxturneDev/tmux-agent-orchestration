<script>
  import { ansiToHtml } from '../utils/ansi.js';

  let { pane, onclose } = $props();

  let rawLogs = $state('Loading terminal output...');
  let logInterval = null;
  let terminalViewport = $state(null);

  const fetchLogs = async () => {
    if (!pane) return;
    try {
      const res = await fetch(`/api/pane/${pane.PaneID}/raw`);
      if (!res.ok) throw new Error('API request failed');
      const data = await res.json();
      rawLogs = data.raw || 'No logs available.';
      
      // Auto-scroll terminal viewport to bottom when logs load
      if (terminalViewport) {
        setTimeout(() => {
          terminalViewport.scrollTop = terminalViewport.scrollHeight;
        }, 50);
      }
    } catch (e) {
      console.error('Failed to fetch raw terminal buffer:', e);
      rawLogs = `Error loading terminal logs: ${e.message}`;
    }
  };

  // React to pane parameter changes
  $effect(() => {
    if (pane) {
      rawLogs = 'Loading terminal output...';
      fetchLogs();
      logInterval = setInterval(fetchLogs, 1000);
    }
    return () => {
      if (logInterval) {
        clearInterval(logInterval);
        logInterval = null;
      }
    };
  });
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
{#if pane}
  <div class="drawer-overlay" on:click={onclose}>
    <div class="drawer-content glass-panel" on:click|stopPropagation>
      <header class="drawer-header">
        <div class="header-title">
          <span class="agent-tag">Agent Console</span>
          <h2>{pane.Command}</h2>
        </div>
        <button class="close-btn" on:click={onclose}>&times;</button>
      </header>

      <section class="drawer-meta-section">
        <div class="meta-grid">
          <div class="meta-item">
            <span class="label">Pane ID</span>
            <span class="value font-mono">{pane.PaneID}</span>
          </div>
          <div class="meta-item">
            <span class="label">Session</span>
            <span class="value font-mono">{pane.Session}</span>
          </div>
          <div class="meta-item">
            <span class="label">Window ID</span>
            <span class="value font-mono">{pane.WindowID}</span>
          </div>
          <div class="meta-item">
            <span class="label">Active Plan</span>
            <span class="value font-mono text-purple">{pane.PlanName || 'N/A'}</span>
          </div>
        </div>
        
        <div class="meta-item full-width">
          <span class="label">Working Directory</span>
          <span class="value font-mono dir-text">{pane.Path}</span>
        </div>

        <div class="meta-item full-width">
          <span class="label">Current Goal</span>
          <p class="goal-text">{pane.ActiveGoal || 'No active goal specified.'}</p>
        </div>
      </section>

      <section class="drawer-terminal-section">
        <div class="terminal-header">
          <span class="terminal-title">TERMINAL BUFFER (LAST 200 LINES)</span>
          <span class="live-pill">LIVE STREAMING</span>
        </div>
        <div class="terminal-viewport" bind:this={terminalViewport}>
          <pre class="terminal-output">{@html ansiToHtml(rawLogs)}</pre>
        </div>
      </section>

      <footer class="drawer-footer">
        <button class="btn btn-secondary" on:click={onclose}>Close Drawer</button>
        <button class="btn btn-primary" disabled title="Wired in V2">Isolate Pane</button>
      </footer>
    </div>
  </div>
{/if}

<style>
  .drawer-overlay {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    background: rgba(0, 0, 0, 0.6);
    backdrop-filter: blur(4px);
    z-index: 1000;
    display: flex;
    justify-content: flex-end;
    animation: fadeIn var(--transition-fast) forwards;
  }

  .drawer-content {
    width: 600px;
    height: 100%;
    background: var(--bg-secondary);
    border-left: 1px solid var(--border-color);
    box-shadow: -10px 0 30px rgba(0, 0, 0, 0.5);
    display: flex;
    flex-direction: column;
    animation: slideInRight var(--transition-normal) forwards;
    border-radius: 0;
  }

  @media (max-width: 768px) {
    .drawer-content {
      width: 100%;
    }
  }

  .drawer-header {
    padding: 24px;
    border-bottom: 1px solid var(--border-color);
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .header-title h2 {
    font-size: 1.5rem;
    font-weight: 700;
    color: #ffffff;
    margin-top: 4px;
  }

  .agent-tag {
    font-size: 0.75rem;
    color: var(--accent-cyan);
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 1px;
  }

  .close-btn {
    background: none;
    border: none;
    color: var(--text-muted);
    font-size: 2rem;
    cursor: pointer;
    transition: color var(--transition-fast);
  }

  .close-btn:hover {
    color: #ffffff;
  }

  .drawer-meta-section {
    padding: 20px 24px;
    border-bottom: 1px solid var(--border-color);
    display: flex;
    flex-direction: column;
    gap: 16px;
    background: rgba(255, 255, 255, 0.01);
  }

  .meta-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 16px;
  }

  .meta-item {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .meta-item.full-width {
    grid-column: span 2;
  }

  .label {
    font-size: 0.7rem;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.5px;
    font-weight: 600;
  }

  .value {
    font-size: 0.9rem;
    color: var(--text-primary);
  }

  .font-mono {
    font-family: var(--font-mono);
  }

  .text-purple {
    color: #e57373;
    background: rgba(229, 115, 115, 0.1);
    padding: 1px 6px;
    border-radius: 4px;
    width: fit-content;
  }

  .dir-text {
    word-break: break-all;
    font-size: 0.8rem;
    color: var(--text-secondary);
  }

  .goal-text {
    font-size: 0.85rem;
    color: var(--text-secondary);
    line-height: 1.4;
  }

  .drawer-terminal-section {
    flex-grow: 1;
    padding: 24px;
    display: flex;
    flex-direction: column;
    gap: 10px;
    min-height: 0;
  }

  .terminal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .terminal-title {
    font-size: 0.75rem;
    color: var(--text-secondary);
    font-weight: 600;
    letter-spacing: 0.5px;
  }

  .live-pill {
    background: rgba(0, 229, 255, 0.1);
    color: var(--accent-cyan);
    border: 1px solid var(--border-color);
    font-size: 0.65rem;
    font-weight: 700;
    padding: 2px 6px;
    border-radius: 20px;
    letter-spacing: 0.5px;
  }

  .terminal-viewport {
    flex-grow: 1;
    background: var(--terminal-bg);
    border: 1px solid var(--terminal-border);
    border-radius: 6px;
    padding: 16px;
    overflow-y: auto;
    font-family: var(--font-mono);
    font-size: 0.8rem;
    line-height: 1.5;
    white-space: pre-wrap;
    min-height: 0;
  }

  .terminal-output {
    margin: 0;
    color: #e0e6ed;
    word-break: break-all;
  }

  .drawer-footer {
    padding: 20px 24px;
    border-top: 1px solid var(--border-color);
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }

  .btn {
    padding: 10px 18px;
    border-radius: 6px;
    font-size: 0.9rem;
    font-weight: 600;
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .btn-secondary {
    background: none;
    border: 1px solid var(--border-color);
    color: var(--text-primary);
  }

  .btn-secondary:hover {
    border-color: var(--border-hover);
    background: rgba(255, 255, 255, 0.02);
  }

  .btn-primary {
    background: linear-gradient(135deg, var(--accent-cyan), var(--accent-blue));
    border: none;
    color: #000;
  }

  .btn-primary:hover:not(:disabled) {
    box-shadow: var(--shadow-glow);
    transform: translateY(-1px);
  }

  .btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
