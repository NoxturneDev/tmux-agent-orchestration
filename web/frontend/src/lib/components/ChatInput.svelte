<script>
  let { disabled = false, onsend } = $props();

  let text = $state('');

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
    on:keydown={handleKeyDown}
    class="chat-textarea"
    rows="2"
  ></textarea>

  <button 
    {disabled} 
    on:click={submit} 
    class="send-btn"
    class:active={text.trim() && !disabled}
  >
    <svg viewBox="0 0 24 24" class="send-icon">
      <path d="M2,21L23,12L2,3V10L17,12L2,14V21Z" />
    </svg>
  </button>
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
</style>
