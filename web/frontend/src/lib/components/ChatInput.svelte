<script>
  import { onMount } from 'svelte';

  let { disabled = false, onsend } = $props();

  let text = $state('');
  let isRecording = $state(false);
  let isSttSupported = $state(false);
  let recognition = null;

  onMount(() => {
    const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition;
    isSttSupported = typeof SpeechRecognition !== 'undefined';
    
    if (isSttSupported) {
      recognition = new SpeechRecognition();
      recognition.continuous = false;
      recognition.lang = 'en-US';
      recognition.interimResults = false;

      recognition.onstart = () => {
        isRecording = true;
      };

      recognition.onresult = (event) => {
        const transcript = event.results[0][0].transcript;
        text = text + (text.trim() ? ' ' : '') + transcript;
      };

      recognition.onerror = (err) => {
        console.error('Speech recognition error:', err);
        isRecording = false;
      };

      recognition.onend = () => {
        isRecording = false;
      };
    }
  });

  function toggleSpeech() {
    if (!recognition) return;
    if (isRecording) {
      recognition.stop();
    } else {
      try {
        recognition.start();
      } catch (e) {
        console.error(e);
      }
    }
  }

  const handleKeyDown = (e) => {
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

<div class="chat-input-container">
  <textarea
    {disabled}
    placeholder={disabled ? 'JARVIS is offline. Start the jarvis agent to begin...' : 'Send command or query to JARVIS...'}
    bind:value={text}
    onkeydown={handleKeyDown}
    class="chat-textarea"
    rows="2"
  ></textarea>

  {#if text.trim()}
    <button 
      {disabled} 
      onclick={submit} 
      class="send-btn"
      class:active={true}
    >
      <svg viewBox="0 0 24 24" class="send-icon">
        <path d="M2,21L23,12L2,3V10L17,12L2,14V21Z" />
      </svg>
    </button>
  {:else if isSttSupported}
    <button 
      type="button"
      class="mic-btn" 
      class:recording={isRecording} 
      onclick={toggleSpeech}
      {disabled}
      title={isRecording ? "Stop listening" : "Dictate command"}
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
      disabled
      class="send-btn"
    >
      <svg viewBox="0 0 24 24" class="send-icon">
        <path d="M2,21L23,12L2,3V10L17,12L2,14V21Z" />
      </svg>
    </button>
  {/if}
</div>

<style>
  .chat-input-container {
    display: flex;
    gap: 12px;
    align-items: flex-end;
    background: rgba(3, 6, 15, 0.6);
    border: 1px solid var(--border-color);
    padding: 8px 12px;
    border-radius: 8px;
    transition: border-color var(--transition-fast);
  }

  .chat-input-container:focus-within {
    border-color: var(--accent-cyan);
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
  }

  .chat-textarea::placeholder {
    color: var(--text-muted);
  }

  .chat-textarea:disabled {
    cursor: not-allowed;
    opacity: 0.6;
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

  @keyframes mic-pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.6; }
  }
</style>
