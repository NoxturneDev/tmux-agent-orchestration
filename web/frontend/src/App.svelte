<script>
  import { onMount } from 'svelte';
  import { connectFleet, disconnectFleet, fleet } from './lib/stores/fleet.js';
  import { jarvis } from './lib/stores/jarvis.js';
  import FleetRadar from './lib/components/FleetRadar.svelte';
  import JarvisChat from './lib/components/JarvisChat.svelte';

  onMount(() => {
    connectFleet();
    return () => {
      disconnectFleet();
    };
  });
</script>

<div class="app-container">
  <header class="app-header glass-panel">
    <div class="header-left">
      <div class="logo-dot"></div>
      <h1>AI ORCHESTRATOR</h1>
      <span class="version-tag">v1.0.0</span>
    </div>
    
    <div class="header-right">
      <div class="status-indicator">
        <span class="status-label">Fleet Stream:</span>
        <span class="status-badge" class:status-connected={fleet.status === 'connected'} class:status-offline={fleet.status !== 'connected'}>
          {fleet.status}
        </span>
      </div>
      <div class="status-indicator">
        <span class="status-label">Jarvis WS:</span>
        <span class="status-badge" class:status-connected={jarvis.status === 'online'} class:status-offline={jarvis.status !== 'online'}>
          {jarvis.status}
        </span>
      </div>
      <div class="agent-count-badge">
        {fleet.panes.length} {fleet.panes.length === 1 ? 'Agent' : 'Agents'}
      </div>
    </div>
  </header>

  <main class="app-body">
    <section class="panel-left glass-panel">
      <div class="panel-header">
        <h2>Fleet Radar</h2>
        <span class="panel-subtitle">Live agent panes monitoring</span>
      </div>
      <div class="panel-content">
        <FleetRadar />
      </div>
    </section>

    <section class="panel-right glass-panel">
      <div class="panel-header">
        <h2>Jarvis Workspace Chat</h2>
        <span class="panel-subtitle">Bidirectional supervisor agent console</span>
      </div>
      <div class="panel-content">
        <JarvisChat />
      </div>
    </section>
  </main>
</div>

<style>
  .app-container {
    display: flex;
    flex-direction: column;
    height: 100vh;
    padding: 16px;
    gap: 16px;
    background: radial-gradient(circle at 50% 0%, rgba(0, 229, 255, 0.05) 0%, rgba(7, 10, 19, 0) 70%);
  }

  .app-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 24px;
    height: 64px;
    flex-shrink: 0;
  }

  .header-left {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .logo-dot {
    width: 12px;
    height: 12px;
    background: linear-gradient(135deg, var(--accent-cyan), var(--accent-blue));
    border-radius: 50%;
    box-shadow: 0 0 10px rgba(0, 229, 255, 0.5);
  }

  .header-left h1 {
    font-size: 1.25rem;
    font-weight: 700;
    letter-spacing: 2px;
    background: linear-gradient(135deg, #ffffff, #80e5ff);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
  }

  .version-tag {
    font-family: var(--font-mono);
    font-size: 0.75rem;
    color: var(--text-muted);
    border: 1px solid var(--border-color);
    padding: 2px 6px;
    border-radius: 4px;
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 20px;
  }

  .status-indicator {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 0.85rem;
  }

  .status-label {
    color: var(--text-secondary);
  }

  .status-badge {
    padding: 3px 8px;
    border-radius: 20px;
    font-size: 0.75rem;
    font-weight: 500;
    text-transform: capitalize;
  }

  .status-connected {
    background: rgba(0, 229, 255, 0.1);
    color: var(--accent-cyan);
    border: 1px solid var(--border-color);
  }

  .status-offline {
    background: rgba(244, 67, 54, 0.1);
    color: #ff8a80;
    border: 1px solid rgba(244, 67, 54, 0.3);
  }

  .agent-count-badge {
    background: rgba(0, 229, 255, 0.1);
    color: var(--accent-cyan);
    border: 1px solid var(--border-color);
    padding: 4px 12px;
    border-radius: 6px;
    font-size: 0.85rem;
    font-weight: 600;
  }

  .app-body {
    display: grid;
    grid-template-columns: 3.5fr 2.5fr;
    gap: 16px;
    flex-grow: 1;
    min-height: 0;
  }

  .panel-header {
    padding: 16px 20px;
    border-bottom: 1px solid var(--border-color);
    flex-shrink: 0;
  }

  .panel-header h2 {
    font-size: 1.1rem;
    font-weight: 600;
    color: #ffffff;
  }

  .panel-subtitle {
    font-size: 0.8rem;
    color: var(--text-secondary);
  }

  .panel-left, .panel-right {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
  }

  .panel-content {
    flex-grow: 1;
    overflow-y: auto;
    padding: 20px;
    min-height: 0;
  }

  @media (max-width: 1024px) {
    .app-body {
      grid-template-columns: 1fr;
      overflow-y: auto;
    }
    .panel-left, .panel-right {
      height: 550px;
    }
  }
</style>
