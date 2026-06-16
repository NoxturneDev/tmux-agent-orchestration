<script>
  import { fleet } from '../stores/fleet.js';
  import AgentCard from './AgentCard.svelte';
  import AgentDrawer from './AgentDrawer.svelte';

  let selectedPane = $state(null);
  let searchQuery = $state('');

  // Svelte 5 derived state using getters
  const filteredPanes = $derived.by(() => {
    const panes = fleet.panes;
    if (!searchQuery.trim()) return panes;
    const query = searchQuery.toLowerCase();
    return panes.filter(p => 
      p.Command.toLowerCase().includes(query) ||
      p.Session.toLowerCase().includes(query) ||
      p.Path.toLowerCase().includes(query) ||
      (p.PlanName && p.PlanName.toLowerCase().includes(query))
    );
  });
</script>

<div class="fleet-radar-container">
  <div class="radar-controls">
    <div class="search-input-wrapper">
      <span class="search-icon">🔍</span>
      <input 
        type="text" 
        placeholder="Filter by agent name, session, path..." 
        bind:value={searchQuery}
        class="search-input"
      />
      {#if searchQuery}
        <button class="clear-btn" on:click={() => searchQuery = ''}>&times;</button>
      {/if}
    </div>
    <div class="status-meta">
      <span class="pulse-radar"></span>
      <span class="meta-txt">{filteredPanes.length} / {fleet.panes.length} Agents</span>
    </div>
  </div>

  {#if fleet.panes.length === 0}
    <div class="empty-state">
      <div class="radar-dish">📡</div>
      <h3>No Active Agents Found</h3>
      <p>Ensure agents are spawned inside your tmux workspace (e.g. via TUI or antigravity-cli).</p>
    </div>
  {:else if filteredPanes.length === 0}
    <div class="empty-state">
      <h3>No Agents Match Search</h3>
      <p>Try resetting your filter query parameters.</p>
    </div>
  {:else}
    <div class="agent-grid">
      {#each filteredPanes as pane (pane.PaneID)}
        <AgentCard {pane} onclick={() => selectedPane = pane} />
      {/each}
    </div>
  {/if}

  {#if selectedPane}
    <!-- Ensure pane detail references are reactive by matching against the updated stream -->
    <AgentDrawer 
      pane={fleet.panes.find(p => p.PaneID === selectedPane.PaneID)} 
      onclose={() => selectedPane = null} 
    />
  {/if}
</div>

<style>
  .fleet-radar-container {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
    gap: 16px;
  }

  .radar-controls {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 16px;
    flex-shrink: 0;
  }

  .search-input-wrapper {
    position: relative;
    flex-grow: 1;
    max-width: 450px;
    display: flex;
    align-items: center;
  }

  .search-icon {
    position: absolute;
    left: 14px;
    color: var(--text-muted);
    font-size: 0.95rem;
  }

  .search-input {
    width: 100%;
    background: rgba(255, 255, 255, 0.03);
    border: 1px solid var(--border-color);
    border-radius: 8px;
    padding: 10px 16px 10px 38px;
    color: #ffffff;
    font-size: 0.9rem;
    outline: none;
    transition: all var(--transition-fast);
  }

  .search-input:focus {
    border-color: var(--accent-cyan);
    box-shadow: 0 0 10px rgba(0, 229, 255, 0.1);
    background: rgba(255, 255, 255, 0.05);
  }

  .clear-btn {
    position: absolute;
    right: 12px;
    background: none;
    border: none;
    color: var(--text-muted);
    font-size: 1.25rem;
    cursor: pointer;
  }

  .clear-btn:hover {
    color: #ffffff;
  }

  .status-meta {
    display: flex;
    align-items: center;
    gap: 10px;
    background: rgba(255, 255, 255, 0.02);
    padding: 8px 16px;
    border-radius: 6px;
    border: 1px solid var(--border-color);
  }

  .meta-txt {
    font-size: 0.85rem;
    color: var(--text-secondary);
    font-weight: 500;
  }

  .pulse-radar {
    width: 10px;
    height: 10px;
    background: var(--accent-cyan);
    border-radius: 50%;
    box-shadow: 0 0 8px var(--accent-cyan);
    position: relative;
  }

  .pulse-radar::after {
    content: '';
    position: absolute;
    top: -5px;
    left: -5px;
    right: -5px;
    bottom: -5px;
    border: 1px solid var(--accent-cyan);
    border-radius: 50%;
    animation: pulse 2s infinite ease-out;
    opacity: 0;
  }

  .agent-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: 16px;
    overflow-y: auto;
    flex-grow: 1;
    padding-bottom: 20px;
    min-height: 0;
  }

  .empty-state {
    flex-grow: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    text-align: center;
    padding: 40px;
    background: rgba(255, 255, 255, 0.01);
    border-radius: 12px;
    border: 1px dashed var(--border-color);
    color: var(--text-secondary);
    gap: 12px;
  }

  .radar-dish {
    font-size: 3rem;
    animation: rotate-radar 6s infinite linear;
  }

  @keyframes rotate-radar {
    0% { transform: rotate(0deg); }
    50% { transform: rotate(15deg); }
    100% { transform: rotate(0deg); }
  }

  .empty-state h3 {
    color: #ffffff;
    font-weight: 600;
  }

  .empty-state p {
    font-size: 0.9rem;
    max-width: 380px;
    line-height: 1.5;
  }
</style>
