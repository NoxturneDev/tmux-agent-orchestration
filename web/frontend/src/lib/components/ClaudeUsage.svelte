<script>
  import { onMount } from 'svelte';
  import { usage, fetchUsageStats } from '../stores/usage.svelte.js';

  onMount(() => {
    fetchUsageStats();
    const interval = setInterval(fetchUsageStats, 30000); // refresh every 30s
    return () => clearInterval(interval);
  });

  // Helper to format large numbers nicely
  const formatNum = (num) => {
    if (num === undefined || num === null) return '0';
    if (num >= 1_000_000) return (num / 1_000_000).toFixed(1) + 'M';
    if (num >= 1_000) return (num / 1_000).toFixed(1) + 'k';
    return num.toLocaleString();
  };

  // Helper to format currency
  const formatUSD = (val) => {
    if (val === undefined || val === null) return '$0.00';
    return '$' + val.toFixed(2);
  };

  // Aggregated computed values from stats
  let aggregates = $derived.by(() => {
    if (!usage.stats || !usage.stats.modelUsage) {
      return {
        totalInput: 0,
        totalOutput: 0,
        totalCacheRead: 0,
        totalCacheCreation: 0,
        totalCost: 0,
        cacheEfficiency: 0
      };
    }

    let totalInput = 0;
    let totalOutput = 0;
    let totalCacheRead = 0;
    let totalCacheCreation = 0;
    let totalCost = 0;

    Object.values(usage.stats.modelUsage).forEach((m) => {
      totalInput += m.inputTokens || 0;
      totalOutput += m.outputTokens || 0;
      totalCacheRead += m.cacheReadInputTokens || 0;
      totalCacheCreation += m.cacheCreationInputTokens || 0;
      totalCost += m.costUSD || 0;
    });

    const cacheTotal = totalCacheRead + totalCacheCreation;
    const cacheEfficiency = cacheTotal > 0 ? (totalCacheRead / cacheTotal) * 100 : 0;

    return {
      totalInput,
      totalOutput,
      totalCacheRead,
      totalCacheCreation,
      totalCost,
      cacheEfficiency
    };
  });

  // Safe daily activity data sorted by date
  let dailyData = $derived.by(() => {
    if (!usage.stats || !usage.stats.dailyActivity) return [];
    return [...usage.stats.dailyActivity]
      .sort((a, b) => a.date.localeCompare(b.date))
      .slice(-10); // Show last 10 days for legibility
  });

  let chartHeight = 150;
  let chartWidth = 500;
  let maxVal = $derived.by(() => {
    if (dailyData.length === 0) return 1;
    return Math.max(...dailyData.map((d) => d.messageCount), 1);
  });

  // Determine rate limits based on standard Claude Code quotas
  const getRateLimitData = (model) => {
    let limit5h = 4_000_000;
    let limitWeekly = 20_000_000;

    if (model.includes('haiku')) {
      limit5h = 10_000_000;
      limitWeekly = 50_000_000;
    } else if (model.includes('opus')) {
      limit5h = 2_000_000;
      limitWeekly = 10_000_000;
    } else if (model.includes('sonnet')) {
      limit5h = 4_000_000;
      limitWeekly = 20_000_000;
    }

    const hourly = usage.stats?.hourlyUsage?.[model] || 0;
    const weekly = usage.stats?.weeklyUsage?.[model] || 0;

    return {
      hourly,
      limit5h,
      hourlyPct: Math.min((hourly / limit5h) * 100, 100),
      weekly,
      limitWeekly,
      weeklyPct: Math.min((weekly / limitWeekly) * 100, 100)
    };
  };

  let lastRefreshed = $state('');
  $effect(() => {
    if (usage.stats) {
      lastRefreshed = new Date().toLocaleTimeString();
    }
  });
</script>

<div class="usage-container">
  {#if usage.loading && !usage.stats}
    <div class="loading-state">
      <div class="spinner"></div>
      <span>Loading usage statistics...</span>
    </div>
  {:else if usage.error}
    <div class="error-state">
      <span class="error-icon">⚠️</span>
      <span class="error-msg">{usage.error}</span>
      <button class="retry-btn" onclick={fetchUsageStats}>Retry</button>
    </div>
  {:else if usage.stats}
    <!-- High-level Metric Cards Grid -->
    <div class="metrics-grid">
      <div class="metric-card glass-panel">
        <span class="metric-title">TOTAL SESSIONS</span>
        <span class="metric-val">{formatNum(usage.stats.totalSessions)}</span>
        <span class="metric-desc">Claude Code workspaces</span>
      </div>

      <div class="metric-card glass-panel">
        <span class="metric-title">TOTAL MESSAGES</span>
        <span class="metric-val">{formatNum(usage.stats.totalMessages)}</span>
        <span class="metric-desc">Commands & responses</span>
      </div>

      <div class="metric-card glass-panel">
        <span class="metric-title">TOTAL TOKENS</span>
        <span class="metric-val">{formatNum(aggregates.totalInput + aggregates.totalOutput)}</span>
        <div class="token-split">
          <span class="in">In: {formatNum(aggregates.totalInput)}</span>
          <span class="out">Out: {formatNum(aggregates.totalOutput)}</span>
        </div>
      </div>

      <div class="metric-card glass-panel">
        <span class="metric-title">CACHE EFFICIENCY</span>
        <span class="metric-val text-cyan">{aggregates.cacheEfficiency.toFixed(1)}%</span>
        <span class="metric-desc">Saved: {formatNum(aggregates.totalCacheRead)} read tokens</span>
      </div>
    </div>

    <!-- Rate Limits Panel -->
    <div class="section-card glass-panel">
      <div class="section-header flex-header">
        <h3>Claude Code Rolling Rate Limits</h3>
        <span class="refresh-time font-mono">Refreshed: {lastRefreshed}</span>
      </div>
      <div class="rate-limits-grid">
        {#each Object.keys(usage.stats.modelUsage) as model}
          {@const limits = getRateLimitData(model)}
          <div class="limit-model-card">
            <span class="model-name font-mono">{model}</span>
            
            <!-- 5-hour limit -->
            <div class="limit-bar-group">
              <div class="limit-details">
                <span class="limit-label">5-Hour Window Usage</span>
                <span class="limit-values font-mono">
                  {formatNum(limits.hourly)} / {formatNum(limits.limit5h)}
                </span>
              </div>
              <div class="limit-track">
                <div 
                  class="limit-bar" 
                  class:warning={limits.hourlyPct > 80} 
                  style="width: {limits.hourlyPct}%"
                ></div>
              </div>
            </div>

            <!-- Weekly limit -->
            <div class="limit-bar-group">
              <div class="limit-details">
                <span class="limit-label">Weekly Usage (Mon-Sun)</span>
                <span class="limit-values font-mono">
                  {formatNum(limits.weekly)} / {formatNum(limits.limitWeekly)}
                </span>
              </div>
              <div class="limit-track">
                <div 
                  class="limit-bar" 
                  class:warning={limits.weeklyPct > 80} 
                  style="width: {limits.weeklyPct}%"
                ></div>
              </div>
            </div>
          </div>
        {/each}
      </div>
    </div>

    <!-- Daily Activity SVG Chart -->
    {#if dailyData.length > 0}
      <div class="section-card glass-panel">
        <div class="section-header">
          <h3>Daily Messages Activity</h3>
        </div>
        <div class="chart-wrapper">
          <svg viewBox="0 0 {chartWidth} {chartHeight + 40}" class="activity-chart" preserveAspectRatio="xMidYMid meet">
            <!-- Grid lines -->
            <line x1="0" y1={chartHeight / 2} x2={chartWidth} y2={chartHeight / 2} stroke="rgba(255,255,255,0.05)" stroke-dasharray="4" />
            <line x1="0" y1={chartHeight} x2={chartWidth} y2={chartHeight} stroke="rgba(255,255,255,0.1)" />

            <!-- Bars -->
            {#each dailyData as day, i}
              {@const x = (chartWidth / dailyData.length) * i + 10}
              {@const barW = (chartWidth / dailyData.length) - 20}
              {@const barH = (day.messageCount / maxVal) * (chartHeight - 30)}
              {@const y = chartHeight - barH}

              <g class="chart-bar-group">
                <!-- Message Count Bar -->
                <rect
                  {x}
                  {y}
                  width={barW}
                  height={barH}
                  rx="4"
                  fill="url(#barGradient)"
                  class="chart-bar"
                />

                <!-- Text label above bar -->
                <text
                  x={x + barW / 2}
                  y={y - 8}
                  text-anchor="middle"
                  fill="#ffffff"
                  font-size="9"
                  font-weight="600"
                  class="bar-value"
                >
                  {day.messageCount}
                </text>

                <!-- Date label below axis -->
                <text
                  x={x + barW / 2}
                  y={chartHeight + 20}
                  text-anchor="middle"
                  fill="var(--text-secondary)"
                  font-size="9"
                  class="bar-date"
                >
                  {day.date.substring(5)}
                </text>
              </g>
            {/each}

            <!-- Gradients -->
            <defs>
              <linearGradient id="barGradient" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stop-color="var(--accent-cyan)" />
                <stop offset="100%" stop-color="rgba(0, 229, 255, 0.2)" />
              </linearGradient>
            </defs>
          </svg>
        </div>
      </div>
    {/if}

    <!-- Model Usage Table -->
    <div class="section-card glass-panel">
      <div class="section-header">
        <h3>Per-Model Token Breakdown</h3>
      </div>
      <div class="table-responsive">
        <table class="usage-table">
          <thead>
            <tr>
              <th>Model Name</th>
              <th class="text-right">Input Tokens</th>
              <th class="text-right">Output Tokens</th>
              <th class="text-right">Cached Read</th>
              <th class="text-right">Cost (USD)</th>
            </tr>
          </thead>
          <tbody>
            {#each Object.entries(usage.stats.modelUsage) as [model, m]}
              <tr>
                <td class="model-name font-mono">{model}</td>
                <td class="text-right font-mono">{formatNum(m.inputTokens)}</td>
                <td class="text-right font-mono">{formatNum(m.outputTokens)}</td>
                <td class="text-right font-mono text-cyan">{formatNum(m.cacheReadInputTokens)}</td>
                <td class="text-right font-mono">{formatUSD(m.costUSD)}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>
  {:else}
    <div class="loading-state">
      <span>No stats available. Start Claude Code to populate statistics.</span>
    </div>
  {/if}
</div>

<style>
  .usage-container {
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

  .metrics-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
    gap: 16px;
  }

  .metric-card {
    display: flex;
    flex-direction: column;
    padding: 16px 20px;
    border-radius: 8px;
    border: 1px solid var(--border-color);
  }

  .metric-title {
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--text-secondary);
    letter-spacing: 0.05em;
  }

  .metric-val {
    font-size: 1.8rem;
    font-weight: 700;
    color: #ffffff;
    margin: 8px 0 4px 0;
    line-height: 1.1;
  }

  .metric-desc {
    font-size: 0.75rem;
    color: var(--text-muted, #728aa1);
  }

  .token-split {
    display: flex;
    gap: 12px;
    font-size: 0.75rem;
    font-family: monospace;
    margin-top: 4px;
  }

  .token-split .in {
    color: var(--accent-indigo, #6366f1);
  }

  .token-split .out {
    color: var(--accent-cyan);
  }

  .section-card {
    display: flex;
    flex-direction: column;
    border-radius: 8px;
    border: 1px solid var(--border-color);
    padding: 20px;
  }

  .section-header {
    margin-bottom: 16px;
  }

  .section-header h3 {
    font-size: 1rem;
    font-weight: 600;
    color: #ffffff;
  }

  .flex-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .refresh-time {
    font-size: 0.75rem;
    color: var(--text-muted, #728aa1);
  }

  .rate-limits-grid {
    display: grid;
    grid-template-columns: 1fr;
    gap: 16px;
  }

  @media (min-width: 768px) {
    .rate-limits-grid {
      grid-template-columns: 1fr 1fr;
    }
  }

  .limit-model-card {
    background: rgba(255, 255, 255, 0.02);
    border: 1px solid rgba(255, 255, 255, 0.03);
    border-radius: 6px;
    padding: 16px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .limit-bar-group {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .limit-details {
    display: flex;
    justify-content: space-between;
    font-size: 0.75rem;
  }

  .limit-label {
    color: var(--text-secondary);
  }

  .limit-values {
    color: #ffffff;
    font-weight: 600;
  }

  .limit-track {
    height: 6px;
    background: rgba(255, 255, 255, 0.05);
    border-radius: 3px;
    overflow: hidden;
  }

  .limit-bar {
    height: 100%;
    background: var(--accent-cyan);
    border-radius: 3px;
    transition: width 0.3s ease;
  }

  .limit-bar.warning {
    background: #ef4444;
    box-shadow: 0 0 8px rgba(239, 68, 68, 0.3);
  }

  .chart-wrapper {
    background: rgba(0, 0, 0, 0.2);
    border-radius: 8px;
    padding: 16px;
    min-height: 180px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .activity-chart {
    width: 100%;
    height: 100%;
    overflow: visible;
  }

  .chart-bar {
    transition: opacity 0.2s;
  }

  .chart-bar:hover {
    opacity: 0.8;
  }

  .bar-value {
    opacity: 0;
    transition: opacity 0.2s;
  }

  .chart-bar-group:hover .bar-value {
    opacity: 1;
  }

  .table-responsive {
    overflow-x: auto;
  }

  .usage-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.85rem;
    text-align: left;
  }

  .usage-table th, .usage-table td {
    padding: 12px 16px;
    border-bottom: 1px solid var(--border-color);
  }

  .usage-table th {
    font-weight: 600;
    color: var(--text-secondary);
    letter-spacing: 0.05em;
    font-size: 0.75rem;
    text-transform: uppercase;
  }

  .usage-table td {
    color: #ffffff;
  }

  .model-name {
    color: var(--accent-cyan) !important;
    font-weight: 600;
  }

  .text-right {
    text-align: right;
  }

  .text-cyan {
    color: var(--accent-cyan) !important;
  }
</style>
