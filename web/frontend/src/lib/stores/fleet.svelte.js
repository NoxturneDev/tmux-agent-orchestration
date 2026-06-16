let panes = $state([]);
let status = $state('disconnected');

export const fleet = {
  get panes() { return panes; },
  get status() { return status; }
};

let eventSource = null;
let reconnectTimeout = null;
let reconnectDelay = 1000;

export function connectFleet() {
  if (eventSource) return;

  status = 'connecting';
  eventSource = new EventSource('/api/fleet/stream');

  eventSource.onopen = () => {
    status = 'connected';
    reconnectDelay = 1000;
  };

  eventSource.onmessage = (event) => {
    try {
      panes = JSON.parse(event.data) || [];
    } catch (e) {
      console.error('Failed to parse fleet data:', e);
    }
  };

  eventSource.onerror = (err) => {
    console.error('Fleet stream error, reconnecting:', err);
    status = 'reconnecting';
    disconnectFleet();

    reconnectTimeout = setTimeout(() => {
      reconnectDelay = Math.min(reconnectDelay * 2, 30000);
      connectFleet();
    }, reconnectDelay);
  };
}

export function disconnectFleet() {
  if (eventSource) {
    eventSource.close();
    eventSource = null;
  }
  if (reconnectTimeout) {
    clearTimeout(reconnectTimeout);
    reconnectTimeout = null;
  }
}
