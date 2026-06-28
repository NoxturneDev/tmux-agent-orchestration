<script>
  import { onMount } from 'svelte';
  import { quota, fetchQuotaData } from '../stores/quota.svelte.js';

  let countdowns = $state({});

  onMount(() => {
    fetchQuotaData();
    const apiInterval = setInterval(fetchQuotaData, 30000); // refresh API every 30s
    
    // Local ticking countdown (every 1 second)
    const tickInterval = setInterval(() => {
      let updatedCountdowns = { ...countdowns };
      let ticked = false;
      
      if (quota.data && quota.data.accounts) {
        quota.data.accounts.forEach(acc => {
          if (acc.quota && acc.quota.models) {
            acc.quota.models.forEach(m => {
              const key = `${acc.accountName}-${m.modelId}`;
              if (updatedCountdowns[key] !== undefined) {
                updatedCountdowns[key] = Math.max(0, updatedCountdowns[key] - 1000);
                ticked = true;
              }
            });
          }
        });
      }
      if (ticked) {
        countdowns = updatedCountdowns;
      }
    }, 1000);

    return () => {
      clearInterval(apiInterval);
      clearInterval(tickInterval);
    };
  });

  // Sync local countdowns with fresh API loads
  $effect(() => {
    if (quota.data && quota.data.accounts) {
      let freshCountdowns = {};
      quota.data.accounts.forEach(acc => {
        if (acc.quota && acc.quota.models) {
          acc.quota.models.forEach(m => {
            const key = `${acc.accountName}-${m.modelId}`;
            freshCountdowns[key] = m.timeUntilResetMs || 0;
          });
        }
      });
      countdowns = freshCountdowns;
    }
  });

  const getPercentageText = (fraction) => {
    if (fraction === undefined || fraction === null) return '0%';
    return Math.round(fraction * 100) + '%';
  };

  const getProgressColorClass = (fraction) => {
    if (fraction === undefined || fraction === null) return 'progress-red';
    const pct = fraction * 100;
    if (pct >= 75) return 'progress-green';
    if (pct >= 50) return 'progress-yellow';
    if (pct >= 25) return 'progress-orange';
    return 'progress-red';
  };

  const getRemainingLabel = (model) => {
    if (model.isExhausted) return 'Exhausted';
    return getPercentageText(model.remainingPercentage) + ' remaining';
  };

  const formatTimeUntilReset = (ms) => {
    if (ms === undefined || ms === null || ms <= 0) return 'N/A';
    const hours = Math.floor(ms / (1000 * 60 * 60));
    const minutes = Math.floor((ms % (1000 * 60 * 60)) / (1000 * 60));
    const seconds = Math.floor((ms % (1000 * 60)) / 1000);
    
    if (hours > 0) return `${hours}h ${minutes}m`;
    if (minutes > 0) return `${minutes}m ${seconds}s`;
    return `${seconds}s`;
  };
</script>

<div class="quota-container">
  {#if quota.loading && !quota.data}
    <div class="loading-state">
      <div class="spinner"></div>
      <span>Loading quota configurations...</span>
    </div>
  {:else if quota.error}
    <div class="error-state">
      <span class="error-icon">⚠️</span>
      <span class="error-msg">{quota.error}</span>
      <button class="retry-btn" onclick={fetchQuotaData}>Retry</button>
    </div>
  {:else if quota.data && quota.data.accounts}
    <div class="accounts-grid">
      {#each quota.data.accounts as acc}
        <div class="account-card glass-panel">
          <div class="account-header">
            <div>
              <span class="account-title">{acc.accountName.toUpperCase()}</span>
              {#if acc.quota && acc.quota.email}
                <span class="account-email font-mono">{acc.quota.email}</span>
              {/if}
            </div>
            <span class="status-badge" class:offline={acc.error} class:online={!acc.error}>
              {acc.error ? 'OFFLINE' : 'CONNECTED'}
            </span>
          </div>

          <div class="account-body">
            {#if acc.error}
              <div class="account-error-box">
                <span class="warn-icon">🔑</span>
                <p class="error-text">
                  Unable to fetch quota for this workspace profile:
                  <span class="error-detail">{acc.error}</span>
                </p>
                <div class="terminal-cmd">
                  <span class="cmd-label">Run this inside tmux shell to login:</span>
                  <code class="font-mono">HOME={acc.homeDir} antigravity-usage login</code>
                </div>
              </div>
            {:else if acc.quota}
              <!-- Prompt Credits monthly overall limit -->
              {#if acc.quota.promptCredits}
                {@const credits = acc.quota.promptCredits}
                <div class="credits-section">
                  <div class="section-label">MONTHLY PROMPT CREDITS</div>
                  <div class="progress-details">
                    <span class="credits-usage font-mono">
                      {credits.available.toLocaleString()} / {credits.monthly.toLocaleString()} remaining
                    </span>
                    <span class="pct font-mono">{getPercentageText(credits.remainingPercentage)}</span>
                  </div>
                  <div class="progress-track">
                    <div 
                      class="progress-bar {getProgressColorClass(credits.remainingPercentage)}" 
                      style="width: {getPercentageText(credits.remainingPercentage)}"
                    ></div>
                  </div>
                </div>
              {/if}

              <!-- Models limits -->
              <div class="models-section">
                <div class="section-label">MODEL LIMITS & THROTTLES</div>
                <div class="models-list">
                  {#each acc.quota.models as m}
                    {#if !m.isAutocompleteOnly}
                      {@const countdownKey = `${acc.accountName}-${m.modelId}`}
                      <div class="model-row">
                        <div class="model-info">
                          <span class="model-name font-mono">{m.label}</span>
                          <span class="reset-time font-mono" class:low-quota={m.remainingPercentage < 0.15 || m.isExhausted}>
                            Resets: {formatTimeUntilReset(countdowns[countdownKey])}
                          </span>
                        </div>
                        <div class="model-progress-group">
                          <div class="progress-details">
                            <span class="pct font-mono" class:low-quota={m.remainingPercentage < 0.15 || m.isExhausted}>
                              {getRemainingLabel(m)}
                            </span>
                          </div>
                          <div class="progress-track compact">
                            <div 
                              class="progress-bar {getProgressColorClass(m.remainingPercentage)}" 
                              style="width: {getPercentageText(m.remainingPercentage)}"
                            ></div>
                          </div>
                        </div>
                      </div>
                    {/if}
                  {/each}
                </div>
              </div>
            {/if}
          </div>
        </div>
      {/each}
    </div>
  {:else}
    <div class="loading-state">
      <span>No quota information available.</span>
    </div>
  {/if}
</div>

<style>
  .quota-container {
    display: flex;
    flex-direction: column;
    gap: 20px;
    height: 100%;
  }

  .loading-state, .error-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 16px;
    padding: 60px 0;
    color: var(--text-secondary);
  }

  .spinner {
    width: 32px;
    height: 32px;
    border: 3px solid rgba(0, 229, 255, 0.1);
    border-top-color: var(--accent-cyan);
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .error-icon {
    font-size: 2rem;
  }

  .retry-btn {
    background: var(--accent-cyan);
    color: #070a13;
    border: none;
    padding: 8px 20px;
    border-radius: 6px;
    font-weight: 600;
    cursor: pointer;
    transition: opacity 0.2s;
  }

  .retry-btn:hover {
    opacity: 0.9;
  }

  .accounts-grid {
    display: grid;
    grid-template-columns: 1fr;
    gap: 20px;
  }

  @media (min-width: 1200px) {
    .accounts-grid {
      grid-template-columns: 1fr 1fr;
    }
  }

  .account-card {
    border: 1px solid var(--border-color);
    border-radius: 8px;
    padding: 20px;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .account-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    border-bottom: 1px solid var(--border-color);
    padding-bottom: 12px;
  }

  .account-title {
    font-size: 0.9rem;
    font-weight: 700;
    color: #ffffff;
    letter-spacing: 0.05em;
    display: block;
  }

  .account-email {
    font-size: 0.8rem;
    color: var(--accent-cyan);
    margin-top: 4px;
    display: block;
  }

  .status-badge {
    font-size: 0.65rem;
    font-weight: 700;
    padding: 4px 8px;
    border-radius: 4px;
    letter-spacing: 0.05em;
  }

  .status-badge.online {
    background: rgba(16, 185, 129, 0.1);
    color: #10b981;
    border: 1px solid rgba(16, 185, 129, 0.2);
  }

  .status-badge.offline {
    background: rgba(239, 68, 68, 0.1);
    color: #ef4444;
    border: 1px solid rgba(239, 68, 68, 0.2);
  }

  .account-body {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .account-error-box {
    background: rgba(239, 68, 68, 0.05);
    border: 1px solid rgba(239, 68, 68, 0.1);
    border-radius: 6px;
    padding: 16px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .warn-icon {
    font-size: 1.5rem;
  }

  .error-text {
    font-size: 0.85rem;
    color: var(--text-secondary);
    line-height: 1.4;
  }

  .error-detail {
    display: block;
    color: #ef4444;
    font-weight: 500;
    margin-top: 4px;
  }

  .terminal-cmd {
    background: #000000;
    border-radius: 4px;
    padding: 12px;
    border: 1px solid var(--border-color);
  }

  .cmd-label {
    display: block;
    font-size: 0.7rem;
    color: var(--text-muted, #728aa1);
    margin-bottom: 6px;
  }

  .terminal-cmd code {
    color: #ffffff;
    font-size: 0.8rem;
    word-break: break-all;
  }

  .section-label {
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--text-secondary);
    letter-spacing: 0.05em;
    margin-bottom: 8px;
  }

  .progress-details {
    display: flex;
    justify-content: space-between;
    font-size: 0.8rem;
    color: #ffffff;
    margin-bottom: 6px;
  }

  .pct {
    font-weight: 600;
  }

  .progress-track {
    height: 8px;
    background: rgba(255, 255, 255, 0.05);
    border-radius: 4px;
    overflow: hidden;
  }

  .progress-track.compact {
    height: 6px;
  }

  .progress-bar {
    height: 100%;
    border-radius: 4px;
    transition: width 0.3s ease;
  }

  .progress-green {
    background: #10b981;
    box-shadow: 0 0 8px rgba(16, 185, 129, 0.3);
  }

  .progress-yellow {
    background: #fbbf24;
    box-shadow: 0 0 8px rgba(251, 191, 36, 0.3);
  }

  .progress-orange {
    background: #f97316;
    box-shadow: 0 0 8px rgba(249, 115, 22, 0.3);
  }

  .progress-red {
    background: #ef4444;
    box-shadow: 0 0 8px rgba(239, 68, 68, 0.3);
  }

  .models-list {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .model-row {
    background: rgba(255, 255, 255, 0.02);
    border: 1px solid rgba(255, 255, 255, 0.03);
    border-radius: 6px;
    padding: 12px;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .model-info {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .model-name {
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--accent-cyan);
  }

  .reset-time {
    font-size: 0.7rem;
    color: var(--text-muted, #728aa1);
  }

  .low-quota {
    color: #ef4444;
    font-weight: 700;
  }
</style>
