let quotaData = $state(null);
let loading = $state(false);
let error = $state(null);

export const quota = {
  get data() { return quotaData; },
  get loading() { return loading; },
  get error() { return error; }
};

export async function fetchQuotaData() {
  loading = true;
  error = null;
  try {
    const res = await fetch('/api/antigravity/quota');
    if (!res.ok) throw new Error('Failed to fetch quota data');
    quotaData = await res.json();
  } catch (e) {
    error = e.message;
    console.error('Fetch quota data error:', e);
  } finally {
    loading = false;
  }
}
