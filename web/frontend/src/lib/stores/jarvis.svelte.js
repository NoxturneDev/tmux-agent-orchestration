let status = $state('disconnected');

let cachedMessages = [];
if (typeof localStorage !== 'undefined') {
  try {
    const cached = JSON.parse(localStorage.getItem('jarvis_chat_history') || '[]');
    cachedMessages = cached.map(m => ({
      sender: m.sender,
      content: m.content,
      timestamp: new Date(m.timestamp)
    }));
  } catch (e) {}
}

let messages = $state(cachedMessages);
let paneId = $state('');
let isThinking = $state(false);

export const jarvis = {
  get status() { return status; },
  get messages() { return messages; },
  get paneId() { return paneId; },
  get isThinking() { return isThinking; }
};

let ws = null;
let reconnectTimeout = null;
let reconnectDelay = 1000;

export function connectJarvis() {
  if (messages.length === 0 && typeof localStorage !== 'undefined') {
    try {
      const cached = JSON.parse(localStorage.getItem('jarvis_chat_history') || '[]');
      messages = cached.map(m => ({
        sender: m.sender,
        content: m.content,
        timestamp: new Date(m.timestamp)
      }));
    } catch (e) {}
  }

  if (ws) return;

  status = 'connecting';
  
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const wsUrl = `${protocol}//${window.location.host}/ws/jarvis`;
  
  ws = new WebSocket(wsUrl);

  ws.onopen = () => {
    status = 'connected';
    reconnectDelay = 1000;
    // Clear the active messages before receiving the fresh history stream from server
    messages = [];
  };

  ws.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data);
      if (data.type === 'status') {
        if (data.status === 'offline') {
          status = 'offline';
          paneId = '';
          return;
        }
        status = 'online';
        paneId = data.paneId || '';
        return;
      }

      if (data.type === 'message' && data.message) {
        const msg = data.message;
        messages = [
          ...messages,
          {
            sender: msg.sender,
            content: msg.content,
            timestamp: new Date(msg.timestamp)
          }
        ];

        if (typeof localStorage !== 'undefined') {
          localStorage.setItem('jarvis_chat_history', JSON.stringify(messages));
        }
        
        // Update thinking state based on who sent the last message
        if (msg.sender === 'jarvis') {
          isThinking = false;
        } else if (msg.sender === 'user') {
          isThinking = true;
        }
      }
    } catch (e) {
      console.error('Failed to parse Jarvis message:', e);
    }
  };

  ws.onclose = () => {
    status = 'disconnected';
    cleanupWS();
    triggerReconnect();
  };

  ws.onerror = (err) => {
    console.error('Jarvis WS error:', err);
    status = 'reconnecting';
    cleanupWS();
    triggerReconnect();
  };
}

function cleanupWS() {
  if (ws) {
    try {
      ws.close();
    } catch (e) {}
    ws = null;
  }
}

function triggerReconnect() {
  if (reconnectTimeout) return;
  reconnectTimeout = setTimeout(() => {
    reconnectTimeout = null;
    reconnectDelay = Math.min(reconnectDelay * 2, 30000);
    connectJarvis();
  }, reconnectDelay);
}

export function sendJarvisCommand(cmd) {
  if (!ws || ws.readyState !== WebSocket.OPEN) {
    console.error('Cannot send command: WebSocket not connected');
    return;
  }

  isThinking = true;
  ws.send(JSON.stringify({ content: cmd }));
}

export function sendIntervene() {
  if (!ws || ws.readyState !== WebSocket.OPEN) {
    console.error('Cannot send intervention: WebSocket not connected');
    return;
  }
  isThinking = false;
  ws.send(JSON.stringify({ type: 'intervene' }));
}

export function disconnectJarvis() {
  cleanupWS();
  if (reconnectTimeout) {
    clearTimeout(reconnectTimeout);
    reconnectTimeout = null;
  }
  messages = [];
  isThinking = false;
}
