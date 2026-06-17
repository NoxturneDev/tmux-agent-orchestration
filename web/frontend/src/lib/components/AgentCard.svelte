<script>
  let { pane, onclick } = $props();

  // Helper to extract last 2 segments of path
  const formatPath = (fullPath) => {
    if (!fullPath) return '';
    const parts = fullPath.split('/');
    if (parts.length <= 2) return fullPath;
    return '.../' + parts.slice(-2).join('/');
  };

  // Helper to get emoji/icon based on agent name/type
  const getAgentIcon = (name) => {
    const lowercase = name.toLowerCase();
    if (lowercase.includes('jarvis')) return '🧠';
    if (lowercase.includes('gemini')) return '♊';
    if (lowercase.includes('claude')) return '⛵';
    if (lowercase.includes('opencode')) return '🔓';
    return '🤖';
  };
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="agent-card glass-panel animate-fade-in" on:click={onclick}>
  <div class="card-header">
    <div class="agent-identity">
      <span class="agent-icon">{getAgentIcon(pane.Command)}</span>
      <span class="agent-name">{pane.Command}</span>
    </div>
    <div class="session-badge">
      {pane.Session}:{pane.WindowID}
    </div>
  </div>

  <div class="card-body">
    <div class="path-info" title={pane.Path}>
      <span class="label">Path:</span>
      <span class="value">{formatPath(pane.Path)}</span>
    </div>

    {#if pane.PlanName}
      <div class="plan-info">
        <span class="label">Plan:</span>
        <span class="value plan-tag">{pane.PlanName}</span>
      </div>
    {/if}

    <div class="goal-info">
      <span class="label">Goal:</span>
      <p class="goal-text" title={pane.ActiveGoal}>
        {pane.ActiveGoal || 'No active goal specified'}
      </p>
    </div>
  </div>

  <div class="card-footer">
    <div class="status-indicator {pane.Status === 'IN PROGRESS' ? 'in-progress' : 'idle'}">
      <span class="status-dot"></span>
      <span class="status-text">{pane.Status || 'IDLE'}</span>
    </div>
    <span class="pane-id-tag">{pane.PaneID}</span>
  </div>
</div>

<style>
  .agent-card {
    padding: 16px;
    display: flex;
    flex-direction: column;
    gap: 12px;
    cursor: pointer;
    background: var(--bg-glass);
    border: 1px solid var(--border-color);
    border-radius: 10px;
    transition: all var(--transition-normal);
  }

  .agent-card:hover {
    transform: translateY(-2px);
    border-color: var(--accent-cyan);
    box-shadow: 0 4px 20px rgba(0, 229, 255, 0.15);
  }

  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-bottom: 1px solid rgba(255, 255, 255, 0.05);
    padding-bottom: 8px;
  }

  .agent-identity {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .agent-icon {
    font-size: 1.2rem;
  }

  .agent-name {
    font-weight: 600;
    color: #ffffff;
    font-size: 0.95rem;
    letter-spacing: 0.5px;
  }

  .session-badge {
    background: rgba(41, 121, 255, 0.15);
    color: var(--accent-blue);
    border: 1px solid rgba(41, 121, 255, 0.3);
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 0.75rem;
    font-family: var(--font-mono);
  }

  .card-body {
    display: flex;
    flex-direction: column;
    gap: 8px;
    flex-grow: 1;
  }

  .label {
    font-size: 0.75rem;
    color: var(--text-muted);
    font-weight: 500;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    margin-right: 4px;
  }

  .value {
    font-size: 0.85rem;
    color: var(--text-primary);
  }

  .path-info, .plan-info {
    display: flex;
    align-items: center;
  }

  .plan-tag {
    background: rgba(213, 0, 249, 0.1);
    color: #f59fff;
    border: 1px solid rgba(213, 0, 249, 0.25);
    padding: 1px 6px;
    border-radius: 3px;
    font-size: 0.75rem;
    font-family: var(--font-mono);
  }

  .goal-info {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .goal-text {
    font-size: 0.85rem;
    color: var(--text-secondary);
    line-height: 1.4;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  .card-footer {
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-top: 1px solid rgba(255, 255, 255, 0.05);
    padding-top: 8px;
    font-size: 0.75rem;
  }

  .status-indicator {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .status-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    transition: all var(--transition-normal);
  }

  .in-progress .status-dot {
    background-color: var(--accent-cyan);
    box-shadow: 0 0 8px var(--accent-cyan);
    animation: pulse 2s infinite ease-in-out;
  }

  .idle .status-dot {
    background-color: #757575;
    box-shadow: 0 0 4px rgba(117, 117, 117, 0.4);
  }

  .status-text {
    font-weight: 500;
    font-size: 0.75rem;
    letter-spacing: 0.5px;
    text-transform: uppercase;
  }

  .in-progress .status-text {
    color: var(--accent-cyan);
  }

  .idle .status-text {
    color: var(--text-muted);
  }

  .pane-id-tag {
    font-family: var(--font-mono);
    color: var(--text-muted);
  }
</style>
