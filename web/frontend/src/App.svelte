<script>
  import { onMount } from 'svelte';
  import { connectFleet, disconnectFleet, fleet } from './lib/stores/fleet.svelte.js';
  import { jarvis } from './lib/stores/jarvis.svelte.js';
  import FleetRadar from './lib/components/FleetRadar.svelte';
  import JarvisChat from './lib/components/JarvisChat.svelte';
  import ClaudeUsage from './lib/components/ClaudeUsage.svelte';
  import AntigravityQuota from './lib/components/AntigravityQuota.svelte';

  let activeTab = $state('chat'); // 'chat', 'radar', 'usage', or 'quota'
  let activeLeftTab = $state('radar'); // 'radar', 'usage', or 'quota'

  onMount(() => {
    connectFleet();
    return () => {
      disconnectFleet();
    };
  });
</script>

<div class="app-container">


  <main class="app-body">
    <section class="panel-left glass-panel" class:mobile-hidden={activeTab === 'chat'}>
      <div class="panel-header tabbed-header">
        <div class="tabs-container">
          <button class="panel-tab-btn" class:active={activeLeftTab === 'radar'} onclick={() => { activeLeftTab = 'radar'; activeTab = 'radar'; }}>
            Fleet Radar
          </button>
          <button class="panel-tab-btn" class:active={activeLeftTab === 'usage'} onclick={() => { activeLeftTab = 'usage'; activeTab = 'usage'; }}>
            Claude Usage
          </button>
          <button class="panel-tab-btn" class:active={activeLeftTab === 'quota'} onclick={() => { activeLeftTab = 'quota'; activeTab = 'quota'; }}>
            Antigravity Quota
          </button>
        </div>
        <span class="panel-subtitle">
          {#if activeLeftTab === 'radar'}
            Live agent panes monitoring
          {:else if activeLeftTab === 'usage'}
            Claude Code token and cost statistics
          {:else}
            Multi-profile Antigravity credits & API remaining quotas
          {/if}
        </span>
      </div>
      <div class="panel-content">
        {#if activeLeftTab === 'radar'}
          <FleetRadar />
        {:else if activeLeftTab === 'usage'}
          <ClaudeUsage />
        {:else}
          <AntigravityQuota />
        {/if}
      </div>
    </section>

    <section class="panel-right glass-panel" class:mobile-hidden={activeTab !== 'chat'}>
      <div class="panel-header">
        <h2>Jarvis Workspace Chat</h2>
        <span class="panel-subtitle">Bidirectional supervisor agent console</span>
      </div>
      <div class="panel-content">
        <JarvisChat />
      </div>
    </section>
  </main>

  <!-- Mobile Tab Bar navigation -->
  <div class="mobile-tab-bar">
    <button class="tab-btn" class:active={activeTab === 'radar'} onclick={() => { activeTab = 'radar'; activeLeftTab = 'radar'; }}>
      <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="tab-svg">
        <circle cx="12" cy="12" r="10"/>
        <path d="M12 12m-6 0a6 6 0 1 0 12 0a6 6 0 1 0 -12 0"/>
        <path d="M12 12m-2 0a2 2 0 1 0 4 0a2 2 0 1 0 -4 0"/>
        <line x1="12" y1="2" x2="12" y2="12"/>
      </svg>
      <span>Fleet Radar</span>
    </button>
    <button class="tab-btn" class:active={activeTab === 'usage'} onclick={() => { activeTab = 'usage'; activeLeftTab = 'usage'; }}>
      <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="tab-svg">
        <line x1="18" y1="20" x2="18" y2="10"/>
        <line x1="12" y1="20" x2="12" y2="4"/>
        <line x1="6" y1="20" x2="6" y2="14"/>
      </svg>
      <span>Usage</span>
    </button>
    <button class="tab-btn" class:active={activeTab === 'quota'} onclick={() => { activeTab = 'quota'; activeLeftTab = 'quota'; }}>
      <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="tab-svg">
        <path d="M12 2v20M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/>
      </svg>
      <span>Quota</span>
    </button>
    <button class="tab-btn" class:active={activeTab === 'chat'} onclick={() => activeTab = 'chat'}>
      <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="tab-svg">
        <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
      </svg>
      <span>Jarvis Chat</span>
    </button>
  </div>
</div>

<style>
  .app-container {
    display: flex;
    flex-direction: column;
    height: 100%;
    padding: 16px;
    gap: 16px;
    background: radial-gradient(circle at 50% 0%, rgba(0, 229, 255, 0.05) 0%, rgba(7, 10, 19, 0) 70%);
  }

  /* App Header Styles Removed */

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

  @media (max-width: 768px) {
    .panel-left.mobile-hidden, .panel-right.mobile-hidden {
      display: none !important;
    }
    .app-body {
      grid-template-columns: 1fr !important;
      height: calc(100% - 60px) !important; /* Tab bar is 60px */
      overflow: hidden !important;
      gap: 0 !important;
    }
    .panel-left, .panel-right {
      height: 100% !important;
    }
    .mobile-tab-bar {
      display: flex !important;
    }
  }

  .mobile-tab-bar {
    display: none;
    justify-content: space-around;
    align-items: center;
    height: 60px;
    background: rgba(13, 18, 34, 0.95);
    border-top: 1px solid var(--border-color);
    flex-shrink: 0;
    margin: -16px;
    margin-top: 16px;
    padding: 0 16px;
  }

  .tab-btn {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 4px;
    background: none;
    border: none;
    color: var(--text-secondary);
    font-size: 0.75rem;
    font-weight: 500;
    cursor: pointer;
    transition: color var(--transition-fast);
    flex: 1;
    height: 100%;
  }

  .tab-btn.active {
    color: var(--accent-cyan);
    text-shadow: 0 0 10px rgba(0, 229, 255, 0.3);
  }

  .tab-svg {
    width: 20px;
    height: 20px;
    stroke: currentColor;
    transition: transform var(--transition-fast);
  }

  .tab-btn.active .tab-svg {
    transform: scale(1.1);
  }

  .tabbed-header {
    padding: 12px 20px 8px 20px !important;
  }

  .tabs-container {
    display: flex;
    gap: 20px;
    margin-bottom: 4px;
  }

  .panel-tab-btn {
    background: none;
    border: none;
    padding: 4px 0;
    font-size: 1.1rem;
    font-weight: 600;
    color: var(--text-secondary);
    cursor: pointer;
    position: relative;
    transition: color var(--transition-fast);
  }

  .panel-tab-btn.active {
    color: #ffffff;
  }

  .panel-tab-btn.active::after {
    content: '';
    position: absolute;
    bottom: -9px;
    left: 0;
    right: 0;
    height: 2px;
    background: var(--accent-cyan);
    box-shadow: 0 0 8px var(--accent-cyan);
  }
</style>
