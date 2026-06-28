let stats = $state(null);
let loading = $state(false);
let error = $state(null);

export const usage = {
  get stats() { return stats; },
  get loading() { return loading; },
  get error() { return error; },
  refresh() { return fetchUsageStats(); }
};

export async function fetchUsageStats() {
  loading = true;
  error = null;
  try {
    const res = await fetch('/api/claude/stats');
    if (!res.ok) throw new Error('Failed to fetch usage stats');
    stats = await res.json();
  } catch (e) {
    error = e.message;
    console.error('Fetch usage stats error:', e);
  } finally {
    loading = false;
  }
}

if (typeof document !== 'undefined') {
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'visible') {
      fetchUsageStats();
    }
  });
}

