<script>
  import { onMount } from 'svelte';
  import { fleet } from '../stores/fleet.svelte.js';

  let { disabled = false, onsend } = $props();

  let text = $state('');
  let isRecording = $state(false);
  let isSttSupported = $state(false);
  let micPermission = $state('granted'); // 'granted', 'denied', 'prompt', 'unsupported', 'insecure'
  let permissionError = $state('');
  let bannerDismissed = $state(false);
  let recognition = null;
  let textareaEl = $state(null);

  const BASE_COMMANDS = [
    { cmd: '/spawn', desc: 'Spawn a new agent' },
    { cmd: '/kill', desc: 'Terminate an active agent pane' },
    { cmd: '/status', desc: 'Check system status' },
    { cmd: '/intercom', desc: 'Send intercom query to another agent' },
    { cmd: '/agents', desc: 'List active agent instances' },
    { cmd: '/clear', desc: 'Clear console logs' },
    { cmd: '/plan', desc: 'Generate or review active plans' },
    { cmd: '/help', desc: 'Show list of available commands' }
  ];

  function getActiveCommandSegment(str) {
    for (let i = str.length - 1; i >= 0; i--) {
      if (str[i] === '/' && (i === 0 || /\s/.test(str[i - 1]))) {
        return {
          index: i,
          segment: str.slice(i)
        };
      }
    }
    return null;
  }

  let hiddenSegment = $state('');

  let suggestions = $derived.by(() => {
    const active = getActiveCommandSegment(text);
    if (!active) return [];

    const val = active.segment;
    if (val === hiddenSegment) return [];

    if (val === '/') {
      return BASE_COMMANDS;
    }

    if (val === '/kill' || val.startsWith('/kill ')) {
      const targetNames = fleet.panes.map(p => p.Command || p.PaneID).filter(Boolean);
      const paneSuggestions = fleet.panes.map(p => ({
        cmd: `/kill ${p.Command || p.PaneID}`,
        desc: `Kill ${p.Command || p.PaneID} (Pane ${p.PaneID})`
      }));
      if (val === '/kill' || val === '/kill ') {
        return paneSuggestions;
      }
      const query = val.slice(6).toLowerCase();
      const trimmedQuery = query.trim();
      const targetNamesLower = targetNames.map(name => name.toLowerCase());
      if (trimmedQuery.includes(' ') || (query.endsWith(' ') && targetNamesLower.includes(trimmedQuery))) {
        return [];
      }
      return paneSuggestions.filter(s => s.cmd.toLowerCase().includes(val.toLowerCase()));
    }

    if (val === '/intercom' || val.startsWith('/intercom ')) {
      const targetNames = fleet.panes.map(p => p.Command || p.PaneID).filter(Boolean);
      const paneSuggestions = fleet.panes.map(p => ({
        cmd: `/intercom ${p.Command || p.PaneID} `,
        desc: `Send intercom message to ${p.Command || p.PaneID}`
      }));
      if (val === '/intercom' || val === '/intercom ') {
        return paneSuggestions;
      }
      const query = val.slice(10).toLowerCase();
      const trimmedQuery = query.trim();
      const targetNamesLower = targetNames.map(name => name.toLowerCase());
      if (trimmedQuery.includes(' ') || (query.endsWith(' ') && targetNamesLower.includes(trimmedQuery))) {
        return [];
      }
      return paneSuggestions.filter(s => s.cmd.toLowerCase().includes(val.toLowerCase()));
    }

    if (val === '/agents' || val.startsWith('/agents ')) {
      const targetNames = fleet.panes.map(p => p.Command || p.PaneID).filter(Boolean);
      const uniqueCommands = Array.from(new Set(targetNames));
      const paneSuggestions = uniqueCommands.map(cmd => ({
        cmd: `/intercom ${cmd} `,
        desc: `Active agent: ${cmd}`
      }));
      fleet.panes.forEach(p => {
        if (!p.Command && p.PaneID) {
          paneSuggestions.push({
            cmd: `/intercom ${p.PaneID} `,
            desc: `Active pane: ${p.PaneID}`
          });
        }
      });
      if (val === '/agents' || val === '/agents ') {
        return paneSuggestions;
      }
      const query = val.slice(8).toLowerCase();
      const trimmedQuery = query.trim();
      const targetNamesLower = targetNames.map(name => name.toLowerCase());
      if (trimmedQuery.includes(' ') || (query.endsWith(' ') && targetNamesLower.includes(trimmedQuery))) {
        return [];
      }
      return paneSuggestions.filter(s => s.cmd.toLowerCase().includes(`/intercom ${query}`));
    }

    return BASE_COMMANDS.filter(s => s.cmd.toLowerCase().startsWith(val.toLowerCase()));
  });

  let showSuggestions = $derived(suggestions.length > 0 && getActiveCommandSegment(text) !== null);
  let focusedIndex = $state(0);

  $effect(() => {
    if (suggestions) {
      focusedIndex = 0;
    }
  });

  $effect(() => {
    const active = getActiveCommandSegment(text);
    if (!active || active.segment !== hiddenSegment) {
      hiddenSegment = '';
    }
  });

  function applySuggestion(item) {
    const active = getActiveCommandSegment(text);
    if (active) {
      text = text.slice(0, active.index) + item.cmd;
    } else {
      text = item.cmd;
    }

    if (item.cmd !== '/agents' && item.cmd !== '/kill' && item.cmd !== '/intercom') {
      hiddenSegment = item.cmd;
    }

    if (textareaEl) {
      textareaEl.focus();
    }
  }

  async function checkPermission() {
    if (typeof window !== 'undefined') {
      if (!window.isSecureContext) {
        micPermission = 'insecure';
        permissionError = 'Speech-to-text requires a secure context (HTTPS or localhost). Accessing via HTTP on a non-localhost IP disables browser microphone APIs.';
        return;
      }
    }
    if (typeof navigator === 'undefined') return;
    if (!navigator.mediaDevices) {
      micPermission = 'unsupported';
      permissionError = 'Microphone media devices are not supported in this browser context.';
      return;
    }
    if (navigator.permissions && navigator.permissions.query) {
      try {
        const status = await navigator.permissions.query({ name: 'microphone' });
        micPermission = status.state;
        status.onchange = () => {
          micPermission = status.state;
          if (micPermission === 'granted') {
            permissionError = '';
            bannerDismissed = false;
          }
        };
      } catch (e) {
        console.warn('Microphone permission query not supported:', e);
      }
    }
  }

  async function ensureMicPermission() {
    if (typeof window !== 'undefined' && !window.isSecureContext) {
      micPermission = 'insecure';
      permissionError = 'Speech-to-text requires a secure context (HTTPS or localhost). Accessing via HTTP on a non-localhost IP disables browser microphone APIs.';
      return;
    }
    if (typeof navigator === 'undefined' || !navigator.mediaDevices) return;
    try {
      if (navigator.permissions && navigator.permissions.query) {
        const status = await navigator.permissions.query({ name: 'microphone' });
        micPermission = status.state;
        if (status.state === 'prompt') {
          const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
          stream.getTracks().forEach(track => track.stop());
          micPermission = 'granted';
          permissionError = '';
          bannerDismissed = false;
        } else if (status.state === 'denied') {
          micPermission = 'denied';
          permissionError = 'Microphone permission blocked. Please enable it in your browser settings (click lock icon next to URL) and retry.';
        }
      } else {
        const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
        stream.getTracks().forEach(track => track.stop());
        micPermission = 'granted';
        permissionError = '';
        bannerDismissed = false;
      }
    } catch (e) {
      try {
        const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
        stream.getTracks().forEach(track => track.stop());
        micPermission = 'granted';
        permissionError = '';
        bannerDismissed = false;
      } catch (err) {
        micPermission = 'denied';
        permissionError = 'Microphone permission blocked. Please enable it in your browser settings (click lock icon next to URL) and retry.';
        throw err;
      }
    }
  }

  async function requestMicPermission() {
    permissionError = '';
    bannerDismissed = false;
    if (typeof window !== 'undefined' && !window.isSecureContext) {
      permissionError = 'Speech-to-text requires a secure context (HTTPS or localhost). Accessing via HTTP on a non-localhost IP disables browser microphone APIs.';
      return;
    }
    if (typeof navigator === 'undefined' || !navigator.mediaDevices) {
      permissionError = 'Media devices are not supported in this environment.';
      return;
    }
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      stream.getTracks().forEach(track => track.stop());
      micPermission = 'granted';
      permissionError = '';
      initSpeechRecognition();
    } catch (err) {
      console.error('Error requesting mic permission:', err);
      micPermission = 'denied';
      permissionError = 'Microphone permission blocked. Please enable it in your browser settings (click lock icon next to URL) and retry.';
    }
  }

  function initSpeechRecognition() {
    if (recognition) return;
    const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition;
    isSttSupported = typeof SpeechRecognition !== 'undefined';
    
    if (isSttSupported) {
      recognition = new SpeechRecognition();
      recognition.continuous = false;
      recognition.lang = 'en-US';
      recognition.interimResults = false;

      recognition.onstart = () => {
        isRecording = true;
        permissionError = '';
        bannerDismissed = false;
      };

      recognition.onresult = (event) => {
        const transcript = event.results[0][0].transcript;
        text = text + (text.trim() ? ' ' : '') + transcript;
      };

      recognition.onerror = (err) => {
        console.error('Speech recognition error:', err);
        isRecording = false;
        if (err.error === 'not-allowed') {
          micPermission = 'denied';
          permissionError = 'Microphone permission blocked. Please enable it in your browser settings (click lock icon next to URL) and retry.';
          bannerDismissed = false;
        } else if (err.error === 'no-speech') {
          // No speech detected, quietly stop
        } else {
          permissionError = `Speech recognition error: ${err.error}`;
          bannerDismissed = false;
        }
      };

      recognition.onend = () => {
        isRecording = false;
      };
    }
  }

  onMount(async () => {
    await checkPermission();
    const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition;
    isSttSupported = typeof SpeechRecognition !== 'undefined';
    
    if (isSttSupported) {
      if (micPermission !== 'denied' && micPermission !== 'insecure') {
        try {
          await ensureMicPermission();
        } catch (err) {
          console.warn('Initial mic permission check failed:', err);
        }
      }
      initSpeechRecognition();
    }
  });

  function toggleSpeech() {
    bannerDismissed = false; // Reset dismissal state on new click attempt
    if (isRecording) {
      if (recognition) {
        recognition.stop();
      }
      isRecording = false;
    } else {
      permissionError = '';
      if (micPermission === 'insecure') {
        permissionError = 'Speech-to-text requires a secure context (HTTPS or localhost). Accessing via HTTP on a non-localhost IP disables browser microphone APIs.';
      } else if (micPermission === 'denied') {
        requestMicPermission();
      } else {
        if (!recognition) {
          initSpeechRecognition();
        }
        if (recognition) {
          try {
            recognition.start();
          } catch (e) {
            console.error('Failed to start speech recognition:', e);
            permissionError = 'Could not start microphone listening. Try refreshing permissions.';
          }
        } else {
          permissionError = 'Speech recognition is not supported in this browser.';
        }
      }
    }
  }

  const handleKeyDown = (e) => {
    if (showSuggestions) {
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        focusedIndex = (focusedIndex + 1) % suggestions.length;
        return;
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault();
        focusedIndex = (focusedIndex - 1 + suggestions.length) % suggestions.length;
        return;
      }
      if (e.key === 'Tab') {
        e.preventDefault();
        applySuggestion(suggestions[focusedIndex]);
        return;
      }
      if (e.key === 'Escape') {
        e.preventDefault();
        const active = getActiveCommandSegment(text);
        if (active) {
          hiddenSegment = active.segment;
        }
        return;
      }
    }

    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      submit();
    }
  };

  const submit = () => {
    if (!text.trim() || disabled) return;
    onsend(text.trim());
    text = '';
  };
</script>

<div class="chat-input-wrapper">
  {#if (permissionError || micPermission === 'denied' || micPermission === 'insecure') && !bannerDismissed}
    <div class="permission-banner animate-fade-in">
      <div class="permission-banner-content">
        <svg class="warning-icon" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3Z"/>
          <line x1="12" y1="9" x2="12" y2="13"/>
          <line x1="12" y1="17" x2="12.01" y2="17"/>
        </svg>
        <span class="warning-text">
          {permissionError || 'Microphone access is blocked. Please enable it in browser settings (click lock icon next to URL) and retry.'}
        </span>
      </div>
      <div class="permission-banner-actions">
        {#if micPermission !== 'insecure'}
          <button type="button" class="refresh-permission-btn" onclick={requestMicPermission}>
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="refresh-icon">
              <path d="M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8"/>
              <path d="M16 3h5v5"/>
              <path d="M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16"/>
              <path d="M8 21H3v-5"/>
            </svg>
            Refresh
          </button>
        {/if}
        <button 
          type="button" 
          class="close-banner-btn" 
          onclick={() => bannerDismissed = true}
          title="Dismiss warning"
          aria-label="Dismiss warning"
        >
          &times;
        </button>
      </div>
    </div>
  {/if}

  <div class="chat-input-container" class:listening={isRecording}>
    <div class="textarea-wrapper">
      <textarea
        bind:this={textareaEl}
        {disabled}
        placeholder={disabled ? 'JARVIS is offline. Start the jarvis agent to begin...' : 'Send command or query to JARVIS...'}
        bind:value={text}
        onkeydown={handleKeyDown}
        class="chat-textarea"
        rows="2"
      ></textarea>

      {#if showSuggestions}
        <div class="autocomplete-popover animate-fade-in">
          {#each suggestions as item, idx}
            <button
              type="button"
              class="autocomplete-item"
              class:focused={idx === focusedIndex}
              onclick={() => applySuggestion(item)}
            >
              <span class="autocomplete-cmd">{item.cmd}</span>
              <span class="autocomplete-desc">{item.desc}</span>
            </button>
          {/each}
        </div>
      {/if}

      {#if isRecording}
        <div class="listening-overlay animate-fade-in">
          <div class="listening-status">
            <div class="listening-pulse-dot"></div>
            <span class="listening-text">Listening... Speak now</span>
          </div>
          <div class="sound-wave">
            <span class="bar bar-1"></span>
            <span class="bar bar-2"></span>
            <span class="bar bar-3"></span>
            <span class="bar bar-4"></span>
            <span class="bar bar-5"></span>
            <span class="bar bar-6"></span>
            <span class="bar bar-7"></span>
            <span class="bar bar-8"></span>
          </div>
          <button type="button" class="cancel-recording-btn" onclick={toggleSpeech} title="Cancel recording">
            Cancel
          </button>
        </div>
      {/if}
    </div>

    {#if text.trim()}
      <button 
        {disabled} 
        onclick={submit} 
        class="send-btn"
        class:active={true}
        aria-label="Send message"
        title="Send message"
      >
        <svg viewBox="0 0 24 24" class="send-icon">
          <path d="M2,21L23,12L2,3V10L17,12L2,14V21Z" />
        </svg>
      </button>
    {:else if isSttSupported && micPermission !== 'insecure'}
      <button 
        type="button"
        class="mic-btn" 
        class:recording={isRecording} 
        onclick={toggleSpeech}
        title={isRecording ? "Stop listening" : "Dictate command"}
        aria-label={isRecording ? "Stop listening" : "Dictate command"}
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          {#if isRecording}
            <rect x="4" y="4" width="16" height="16" rx="2" ry="2" fill="currentColor"></rect>
          {:else}
            <path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z"></path>
            <path d="M19 10v2a7 7 0 0 1-14 0v-2"></path>
            <line x1="12" y1="19" x2="12" y2="23"></line>
            <line x1="8" y1="23" x2="16" y2="23"></line>
          {/if}
        </svg>
      </button>
    {:else}
      <button 
        type="button"
        class="mic-btn disabled-unsupported" 
        onclick={toggleSpeech}
        title="Speech-to-text not supported or insecure context"
        aria-label="Speech-to-text not supported"
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <line x1="1" y1="1" x2="23" y2="23"></line>
          <path d="M9 9v3a3 3 0 0 0 5.12 2.12M15 9.34V4a3 3 0 0 0-5.94-.6"></path>
          <path d="M17 17a7 7 0 0 1-10.24-1.24"></path>
          <path d="M19 10v2a7 7 0 0 1-1.24 3.9"></path>
          <line x1="12" y1="19" x2="12" y2="23"></line>
          <line x1="8" y1="23" x2="16" y2="23"></line>
        </svg>
      </button>
    {/if}
  </div>
</div>

<style>
  .chat-input-wrapper {
    display: flex;
    flex-direction: column;
    width: 100%;
  }

  .permission-banner {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    background: rgba(244, 67, 54, 0.15);
    border: 1px solid rgba(244, 67, 54, 0.3);
    padding: 10px 14px;
    border-radius: 6px;
    margin-bottom: 8px;
    font-size: 0.8rem;
    color: #ff9992;
    font-family: var(--font-sans);
  }

  .permission-banner-content {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-grow: 1;
  }

  .warning-icon {
    flex-shrink: 0;
    color: #ff5252;
  }

  .warning-text {
    line-height: 1.4;
  }

  .permission-banner-actions {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-shrink: 0;
  }

  .close-banner-btn {
    background: none;
    border: none;
    color: #ff9992;
    font-size: 1.25rem;
    font-weight: bold;
    cursor: pointer;
    padding: 0 4px;
    line-height: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: color var(--transition-fast);
  }

  .close-banner-btn:hover {
    color: #ffffff;
  }

  .refresh-permission-btn {
    background: rgba(255, 82, 82, 0.2);
    border: 1px solid rgba(255, 82, 82, 0.35);
    color: #ffffff;
    padding: 6px 12px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.75rem;
    font-weight: 500;
    display: flex;
    align-items: center;
    gap: 6px;
    transition: all var(--transition-fast);
    white-space: nowrap;
  }

  .refresh-permission-btn:hover {
    background: rgba(255, 82, 82, 0.3);
    border-color: rgba(255, 82, 82, 0.6);
    transform: translateY(-1px);
  }

  .refresh-icon {
    transition: transform 0.5s ease;
  }

  .refresh-permission-btn:hover .refresh-icon {
    transform: rotate(180deg);
  }

  .chat-input-container {
    display: flex;
    gap: 12px;
    align-items: flex-end;
    background: rgba(3, 6, 15, 0.6);
    border: 1px solid var(--border-color);
    padding: 8px 12px;
    border-radius: 8px;
    transition: all var(--transition-fast);
  }

  .chat-input-container:focus-within {
    border-color: var(--accent-cyan);
  }

  .chat-input-container.listening {
    border-color: rgba(255, 82, 82, 0.5);
    box-shadow: 0 0 15px rgba(255, 82, 82, 0.15);
    animation: border-pulse-red 2s infinite ease-in-out;
  }

  @keyframes border-pulse-red {
    0%, 100% {
      border-color: rgba(255, 82, 82, 0.4);
      box-shadow: 0 0 10px rgba(255, 82, 82, 0.05);
    }
    50% {
      border-color: rgba(255, 82, 82, 0.8);
      box-shadow: 0 0 20px rgba(255, 82, 82, 0.25);
    }
  }

  .textarea-wrapper {
    position: relative;
    flex-grow: 1;
    display: flex;
    min-height: 40px;
  }

  .chat-textarea {
    flex-grow: 1;
    background: none;
    border: none;
    outline: none;
    color: #ffffff;
    font-size: 0.9rem;
    font-family: var(--font-sans);
    resize: none;
    max-height: 120px;
    line-height: 1.4;
    padding-top: 4px;
    width: 100%;
  }

  .chat-textarea::placeholder {
    color: var(--text-muted);
  }

  .chat-textarea:disabled {
    cursor: not-allowed;
    opacity: 0.6;
  }

  .listening-overlay {
    position: absolute;
    inset: 0;
    background: rgba(3, 6, 15, 0.95);
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    border-radius: 4px;
    z-index: 5;
  }

  .listening-status {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .listening-pulse-dot {
    width: 10px;
    height: 10px;
    background-color: #ff5252;
    border-radius: 50%;
    box-shadow: 0 0 0 0 rgba(255, 82, 82, 0.7);
    animation: pulse-red 1.2s infinite;
  }

  .listening-text {
    font-size: 0.9rem;
    color: #ffffff;
    font-weight: 500;
  }

  .sound-wave {
    display: flex;
    align-items: center;
    gap: 3px;
    height: 20px;
    margin-left: auto;
    margin-right: 16px;
  }

  .sound-wave .bar {
    display: inline-block;
    width: 3px;
    height: 4px;
    background-color: var(--accent-cyan);
    border-radius: 10px;
    animation: wave 1.2s ease-in-out infinite;
  }

  .sound-wave .bar-1 { animation-delay: 0.1s; }
  .sound-wave .bar-2 { animation-delay: 0.2s; }
  .sound-wave .bar-3 { animation-delay: 0.3s; }
  .sound-wave .bar-4 { animation-delay: 0.4s; }
  .sound-wave .bar-5 { animation-delay: 0.3s; }
  .sound-wave .bar-6 { animation-delay: 0.2s; }
  .sound-wave .bar-7 { animation-delay: 0.1s; }
  .sound-wave .bar-8 { animation-delay: 0.15s; }

  @keyframes wave {
    0%, 100% {
      height: 4px;
      background-color: var(--accent-cyan);
    }
    50% {
      height: 20px;
      background-color: var(--accent-blue);
    }
  }

  @keyframes pulse-red {
    0% {
      transform: scale(0.95);
      box-shadow: 0 0 0 0 rgba(255, 82, 82, 0.7);
    }
    70% {
      transform: scale(1);
      box-shadow: 0 0 0 6px rgba(255, 82, 82, 0);
    }
    100% {
      transform: scale(0.95);
      box-shadow: 0 0 0 0 rgba(255, 82, 82, 0);
    }
  }

  .cancel-recording-btn {
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.15);
    color: var(--text-secondary);
    padding: 4px 10px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.75rem;
    transition: all var(--transition-fast);
  }

  .cancel-recording-btn:hover {
    background: rgba(255, 255, 255, 0.1);
    color: #ffffff;
    border-color: rgba(255, 255, 255, 0.25);
  }

  .send-btn {
    background: rgba(255, 255, 255, 0.02);
    border: 1px solid var(--border-color);
    padding: 10px;
    border-radius: 6px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all var(--transition-fast);
    height: 40px;
    width: 40px;
  }

  .send-btn:hover:not(:disabled) {
    background: linear-gradient(135deg, var(--accent-cyan), var(--accent-blue));
    border-color: transparent;
    transform: translateY(-1px);
    box-shadow: 0 0 10px rgba(0, 229, 255, 0.2);
  }

  .send-btn:hover:not(:disabled) .send-icon {
    fill: #000000;
  }

  .send-btn:disabled {
    opacity: 0.3;
    cursor: not-allowed;
  }

  .send-icon {
    width: 18px;
    height: 18px;
    fill: var(--text-muted);
    transition: fill var(--transition-fast);
  }

  .send-btn.active .send-icon {
    fill: var(--accent-cyan);
  }

  .mic-btn {
    background: rgba(255, 255, 255, 0.02);
    border: 1px solid var(--border-color);
    padding: 10px;
    border-radius: 6px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all var(--transition-fast);
    height: 40px;
    width: 40px;
    color: var(--text-muted);
  }

  .mic-btn:hover:not(:disabled) {
    background: rgba(0, 229, 255, 0.05);
    border-color: rgba(0, 229, 255, 0.3);
    color: var(--accent-cyan);
  }

  .mic-btn.recording {
    background: rgba(244, 67, 54, 0.2);
    border-color: #ff5252;
    color: #ff5252;
    animation: mic-pulse 1.5s infinite;
  }

  .mic-btn.disabled-unsupported {
    background: rgba(255, 255, 255, 0.01);
    border-color: rgba(255, 255, 255, 0.05);
    color: var(--text-muted);
    cursor: pointer;
  }

  .mic-btn.disabled-unsupported:hover {
    background: rgba(244, 67, 54, 0.05);
    border-color: rgba(244, 67, 54, 0.2);
    color: #ff8a80;
  }

  @keyframes mic-pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.6; }
  }

  /* Autocomplete popover */
  .autocomplete-popover {
    position: absolute;
    bottom: 100%;
    left: 0;
    width: 100%;
    max-height: 200px;
    overflow-y: auto;
    background: rgba(13, 18, 34, 0.95);
    backdrop-filter: blur(12px);
    border: 1px solid var(--border-color);
    border-radius: 6px;
    box-shadow: 0 -4px 24px rgba(0, 0, 0, 0.5);
    z-index: 20;
    margin-bottom: 8px;
    display: flex;
    flex-direction: column;
    padding: 4px;
  }

  .autocomplete-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    padding: 8px 12px;
    background: none;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    text-align: left;
    width: 100%;
    color: var(--text-primary);
    transition: all var(--transition-fast);
  }

  .autocomplete-item.focused, .autocomplete-item:hover {
    background: rgba(0, 229, 255, 0.1);
    color: var(--accent-cyan);
  }

  .autocomplete-cmd {
    font-family: var(--font-mono);
    font-size: 0.85rem;
    font-weight: 600;
  }

  .autocomplete-desc {
    font-size: 0.75rem;
    color: var(--text-secondary);
  }
</style>
