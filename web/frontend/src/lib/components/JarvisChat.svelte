<script>
  import { onMount, tick } from 'svelte';
  import { jarvis, connectJarvis, disconnectJarvis, sendJarvisCommand, sendIntervene } from '../stores/jarvis.svelte.js';
  import { ansiToHtml } from '../utils/ansi.js';
  import ChatInput from './ChatInput.svelte';
  import { marked } from 'marked';

  let messageContainer = $state(null);

  const scrollToBottom = async () => {
    await tick();
    if (messageContainer) {
      messageContainer.scrollTop = messageContainer.scrollHeight;
    }
  };

  // Scroll to bottom when new messages arrive
  $effect(() => {
    if (jarvis.messages.length > 0) {
      scrollToBottom();
    }
  });

  let autoSpeak = $state(false);
  let currentlySpeakingIndex = $state(null);
  let isTtsSupported = $state(false);

  let showReviewModal = $state(false);
  let reviewPlanMessage = $state(null);
  let reviewFeedbackText = $state('');
  let bannerDismissed = $state(false);

  function hasPlan(content) {
    if (!content) return false;
    return content.includes('## Technical Goal') || 
           content.includes('## Key Design Decisions') || 
           content.includes('## Technical Tasks') || 
           content.includes('## Commit Strategy') || 
           content.includes('Please review this commit plan');
  }

  let latestPlanMessage = $derived.by(() => {
    for (let i = jarvis.messages.length - 1; i >= 0; i--) {
      const msg = jarvis.messages[i];
      if (msg.sender === 'jarvis' && hasPlan(msg.content)) {
        return msg;
      }
    }
    return null;
  });

  $effect(() => {
    if (latestPlanMessage) {
      bannerDismissed = false;
    }
  });

  async function approveAndProceed() {
    let msg = 'Proceed with the implementation.';
    if (reviewFeedbackText.trim()) {
      msg = `${reviewFeedbackText.trim()}`;
    }
    showReviewModal = false;
    reviewFeedbackText = '';
    await sendJarvisCommand(msg);
  }

  onMount(() => {
    isTtsSupported = typeof window !== 'undefined' && 'speechSynthesis' in window;
    connectJarvis();

    const unlockSpeaker = () => {
      if (isTtsSupported) {
        try {
          const u = new SpeechSynthesisUtterance('');
          u.volume = 0;
          window.speechSynthesis.speak(u);
        } catch (e) {}
      }
      window.removeEventListener('click', unlockSpeaker);
      window.removeEventListener('keydown', unlockSpeaker);
    };
    window.addEventListener('click', unlockSpeaker);
    window.addEventListener('keydown', unlockSpeaker);

    return () => {
      if (isTtsSupported) {
        window.speechSynthesis.cancel();
      }
      window.removeEventListener('click', unlockSpeaker);
      window.removeEventListener('keydown', unlockSpeaker);
      disconnectJarvis();
    };
  });

  function cleanMarkdown(text) {
    return text
      .replace(/_[^_]+_/g, '') // remove markdown italic descriptions (e.g. interventions)
      .replace(/`[^`]+`/g, 'code block') // simplify inline code reads
      .replace(/[#*`~_-]/g, ' ') // strip formatting chars
      .replace(/\s+/g, ' ') // normalize whitespace
      .trim();
  }

  function speakMessage(text, idx) {
    if (!isTtsSupported) return;

    if (currentlySpeakingIndex === idx) {
      window.speechSynthesis.cancel();
      currentlySpeakingIndex = null;
      return;
    }

    window.speechSynthesis.cancel();
    currentlySpeakingIndex = idx;

    const cleanedText = cleanMarkdown(text);
    const utterance = new SpeechSynthesisUtterance(cleanedText);
    
    utterance.onend = () => {
      if (currentlySpeakingIndex === idx) {
        currentlySpeakingIndex = null;
      }
    };

    utterance.onerror = () => {
      if (currentlySpeakingIndex === idx) {
        currentlySpeakingIndex = null;
      }
    };

    window.speechSynthesis.speak(utterance);
  }

  // Auto-speak new messages if enabled
  $effect(() => {
    if (autoSpeak && jarvis.messages.length > 0) {
      const lastMsg = jarvis.messages[jarvis.messages.length - 1];
      if (lastMsg.sender === 'jarvis') {
        speakMessage(lastMsg.content, jarvis.messages.length - 1);
      }
    }
  });
</script>

<div class="jarvis-chat-container">
  <div class="chat-meta-bar">
    {#if jarvis.status === 'online'}
      <div class="meta-left">
        <span class="online-indicator"></span>
        <span class="meta-title">JARVIS AGENT</span>
      </div>
      <div class="meta-right">
        {#if isTtsSupported}
          <button 
            class="tts-toggle-btn" 
            class:active={autoSpeak} 
            onclick={() => {
              autoSpeak = !autoSpeak;
              if (!autoSpeak) window.speechSynthesis.cancel();
            }}
            title={autoSpeak ? "Disable Auto-Speak" : "Enable Auto-Speak"}
          >
            <svg class="tts-icon" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"></polygon>
              {#if autoSpeak}
                <path d="M15.54 8.46a5 5 0 0 1 0 7.07"></path>
                <path d="M19.07 4.93a10 10 0 0 1 0 14.14"></path>
              {:else}
                <line x1="22" y1="9" x2="16" y2="15"></line>
                <line x1="16" y1="9" x2="22" y2="15"></line>
              {/if}
            </svg>
            <span>Auto-Speak</span>
          </button>
        {/if}
        <span class="pane-badge font-mono">Pane {jarvis.paneId}</span>
      </div>
    {:else}
      <div class="meta-left">
        <span class="offline-indicator"></span>
        <span class="meta-title">JARVIS OFFLINE</span>
      </div>
      <span class="offline-reason">Start jarvis in tmux to connect</span>
    {/if}
  </div>

  {#if latestPlanMessage && !showReviewModal && !bannerDismissed}
    <div class="floating-plan-banner animate-fade-in">
      <div class="banner-info">
        <span class="banner-icon">📋</span>
        <span class="banner-text">Implementation plan is ready for review.</span>
      </div>
      <div class="banner-actions">
        <button class="banner-action-btn" onclick={() => {
          reviewPlanMessage = latestPlanMessage;
          reviewFeedbackText = '';
          showReviewModal = true;
        }}>
          Open Review Modal
        </button>
        <button class="banner-dismiss-btn" onclick={() => bannerDismissed = true} title="Dismiss notification">
          &times;
        </button>
      </div>
    </div>
  {/if}

  <div class="chat-messages" bind:this={messageContainer}>
    {#if jarvis.messages.length === 0}
      <div class="welcome-container">
        <div class="jarvis-avatar">🧠</div>
        <h3>Jarvis Supervisor Agent</h3>
        <p class="welcome-desc">
          This panel acts as a web-based proxy console directly to the running JARVIS supervisor tmux pane.
        </p>
        <div class="help-box">
          <span class="help-title">💡 How to use:</span>
          <ul>
            <li>Type any prompt or request (e.g. "show agent status" or "explain plan progress").</li>
            <li>Commands are injected directly as standard keyboard input into the pane buffer.</li>
            <li>Output generated by JARVIS is captured and streamed back here in real-time.</li>
          </ul>
        </div>
      </div>
    {:else}
      {#each jarvis.messages as msg, idx}
        <div class="message-row" class:user-msg={msg.sender === 'user'} class:jarvis-msg={msg.sender === 'jarvis'}>
          <div class="msg-bubble animate-fade-in">
            <div class="msg-header">
              <span class="msg-sender">{msg.sender === 'user' ? 'You' : 'JARVIS Console'}</span>
              <span class="msg-time">{msg.timestamp.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })}</span>
              <div class="msg-header-actions">
                {#if hasPlan(msg.content)}
                  <button 
                    class="review-plan-btn"
                    onclick={() => {
                      reviewPlanMessage = msg;
                      reviewFeedbackText = '';
                      showReviewModal = true;
                    }}
                    title="Open plan in large review modal"
                  >
                    <svg xmlns="http://www.w3.org/2000/svg" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
                      <polyline points="14 2 14 8 20 8"></polyline>
                      <line x1="16" y1="13" x2="8" y2="13"></line>
                      <line x1="16" y1="17" x2="8" y2="17"></line>
                      <polyline points="10 9 9 9 8 9"></polyline>
                    </svg>
                    <span>Review Plan</span>
                  </button>
                {/if}
                {#if isTtsSupported && msg.sender === 'jarvis'}
                  <button 
                    class="speak-msg-btn" 
                    class:speaking={currentlySpeakingIndex === idx} 
                    onclick={() => speakMessage(msg.content, idx)}
                    title={currentlySpeakingIndex === idx ? "Stop speaking" : "Speak message"}
                  >
                    <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                      {#if currentlySpeakingIndex === idx}
                        <rect x="4" y="4" width="16" height="16" rx="2" ry="2"></rect>
                      {:else}
                        <polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"></polygon>
                        <path d="M15.54 8.46a5 5 0 0 1 0 7.07"></path>
                      {/if}
                    </svg>
                  </button>
                {/if}
              </div>
            </div>
            <div class="msg-body">
              {#if msg.sender === 'user'}
                <p class="user-text">{msg.content}</p>
              {:else}
                <div class="markdown-body animate-fade-in">
                  {@html marked.parse(msg.content)}
                </div>
              {/if}
            </div>
          </div>
        </div>
      {/each}
      {#if jarvis.isThinking}
        <div class="message-row jarvis-msg">
          <div class="msg-bubble animate-fade-in">
            <div class="msg-header">
              <span class="msg-sender">JARVIS Console</span>
              <span class="msg-time">thinking...</span>
            </div>
            <div class="msg-body">
              <div class="thinking-loader">
                <div class="dots">
                  <span class="dot"></span>
                  <span class="dot"></span>
                  <span class="dot"></span>
                </div>
                <button class="stop-btn" onclick={sendIntervene}>
                  <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round" class="stop-svg">
                    <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
                  </svg>
                  <span>Stop Response</span>
                </button>
              </div>
            </div>
          </div>
        </div>
      {/if}
    {/if}
  </div>

  <div class="chat-footer">
    <ChatInput 
      disabled={jarvis.status !== 'online'} 
      onsend={(cmd) => sendJarvisCommand(cmd)} 
    />
  </div>
</div>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
{#if showReviewModal && reviewPlanMessage}
  <div class="modal-backdrop" onclick={() => showReviewModal = false}>
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="review-modal glass-panel" onclick={(e) => e.stopPropagation()}>
      <header class="modal-header">
        <div class="modal-title">
          <h3>📋 Review Implementation Plan</h3>
          <span class="modal-subtitle">Verify features, tasks, and proposed git commit structure</span>
        </div>
        <button class="close-modal-btn" onclick={() => showReviewModal = false}>&times;</button>
      </header>

      <main class="modal-body">
        <div class="plan-content-scroll markdown-body">
          {@html marked.parse(reviewPlanMessage.content)}
        </div>
        <div class="feedback-section">
          <label for="feedback-textarea">Add additional review context / feedback (optional):</label>
          <textarea
            id="feedback-textarea"
            bind:value={reviewFeedbackText}
            placeholder="e.g. Looks good, go ahead! Or request changes here..."
            rows="3"
          ></textarea>
        </div>
      </main>

      <footer class="modal-footer">
        <button class="btn btn-secondary" onclick={() => showReviewModal = false}>Cancel</button>
        <button class="btn btn-primary" onclick={approveAndProceed}>
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="20 6 9 17 4 12"></polyline>
          </svg>
          Approve & Proceed
        </button>
      </footer>
    </div>
  </div>
{/if}

<style>
  .jarvis-chat-container {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
  }

  .chat-meta-bar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 16px;
    background: rgba(0, 0, 0, 0.2);
    border-bottom: 1px solid var(--border-color);
    flex-shrink: 0;
  }

  .meta-left {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .meta-title {
    font-size: 0.8rem;
    font-weight: 700;
    color: #ffffff;
    letter-spacing: 0.5px;
  }

  .online-indicator {
    width: 8px;
    height: 8px;
    background: #00e5ff;
    border-radius: 50%;
    box-shadow: 0 0 8px #00e5ff;
  }

  .offline-indicator {
    width: 8px;
    height: 8px;
    background: #ff5252;
    border-radius: 50%;
  }

  .pane-badge {
    background: rgba(0, 229, 255, 0.1);
    color: var(--accent-cyan);
    border: 1px solid var(--border-color);
    padding: 1px 6px;
    border-radius: 4px;
    font-size: 0.75rem;
  }

  .offline-reason {
    font-size: 0.75rem;
    color: var(--text-muted);
  }

  .chat-messages {
    flex-grow: 1;
    overflow-y: auto;
    padding: 16px;
    display: flex;
    flex-direction: column;
    gap: 16px;
    min-height: 0;
  }

  .welcome-container {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 24px;
    text-align: center;
    color: var(--text-secondary);
    height: 100%;
    gap: 12px;
  }

  .jarvis-avatar {
    font-size: 3rem;
    background: rgba(0, 229, 255, 0.05);
    border: 1px solid var(--border-color);
    width: 70px;
    height: 70px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow: var(--shadow-glow);
  }

  .welcome-container h3 {
    color: #ffffff;
    font-weight: 600;
  }

  .welcome-desc {
    font-size: 0.85rem;
    max-width: 320px;
    line-height: 1.4;
  }

  .help-box {
    margin-top: 12px;
    background: rgba(255, 255, 255, 0.02);
    border: 1px solid var(--border-color);
    border-radius: 8px;
    padding: 12px 16px;
    text-align: left;
    max-width: 380px;
  }

  .help-title {
    font-size: 0.8rem;
    font-weight: 600;
    color: #ffffff;
    display: block;
    margin-bottom: 6px;
  }

  .help-box ul {
    margin-left: 16px;
    font-size: 0.75rem;
    line-height: 1.5;
  }

  .message-row {
    display: flex;
    width: 100%;
  }

  .message-row.user-msg {
    justify-content: flex-end;
  }

  .message-row.jarvis-msg {
    justify-content: flex-start;
  }

  .msg-bubble {
    max-width: 85%;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .user-msg .msg-bubble {
    max-width: 70%;
  }

  .msg-header {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 0.75rem;
    color: var(--text-muted);
  }

  .user-msg .msg-header {
    justify-content: flex-end;
  }

  .msg-sender {
    font-weight: 600;
  }

  .user-msg .msg-sender {
    color: var(--accent-blue);
  }

  .jarvis-msg .msg-sender {
    color: var(--accent-cyan);
  }

  .msg-body {
    border-radius: 8px;
    overflow: hidden;
  }

  .user-text {
    background: rgba(41, 121, 255, 0.15);
    border: 1px solid rgba(41, 121, 255, 0.3);
    color: #ffffff;
    padding: 10px 14px;
    border-radius: 12px 12px 0 12px;
    font-size: 0.85rem;
    line-height: 1.4;
    word-break: break-word;
  }

  .terminal-block {
    margin: 0;
    background: var(--terminal-bg);
    border: 1px solid var(--terminal-border);
    padding: 12px;
    border-radius: 0 12px 12px 12px;
    font-family: var(--font-mono);
    font-size: 0.75rem;
    line-height: 1.5;
    white-space: pre-wrap;
    word-break: break-all;
    color: #cbd5e1;
    box-shadow: inset 0 2px 8px rgba(0, 0, 0, 0.5);
  }

  .markdown-body {
    background: rgba(255, 255, 255, 0.02);
    border: 1px solid var(--border-color);
    padding: 14px;
    border-radius: 0 12px 12px 12px;
    font-size: 0.85rem;
    line-height: 1.6;
    color: #e2e8f0;
    word-break: break-word;
  }

  .markdown-body :global(p) {
    margin-bottom: 12px;
  }
  .markdown-body :global(p:last-child) {
    margin-bottom: 0;
  }
  .markdown-body :global(h1), .markdown-body :global(h2), .markdown-body :global(h3) {
    color: #ffffff;
    margin-top: 16px;
    margin-bottom: 8px;
    font-weight: 600;
  }
  .markdown-body :global(h1) { font-size: 1.2rem; }
  .markdown-body :global(h2) { font-size: 1.1rem; }
  .markdown-body :global(h3) { font-size: 1rem; }
  .markdown-body :global(ul), .markdown-body :global(ol) {
    margin-left: 20px;
    margin-bottom: 12px;
  }
  .markdown-body :global(li) {
    margin-bottom: 4px;
  }
  .markdown-body :global(code) {
    font-family: var(--font-mono);
    background: rgba(0, 0, 0, 0.3);
    padding: 2px 6px;
    border-radius: 4px;
    font-size: 0.8rem;
    color: var(--accent-cyan);
  }
  .markdown-body :global(pre) {
    background: #03060f;
    border: 1px solid rgba(0, 229, 255, 0.1);
    padding: 12px;
    border-radius: 6px;
    overflow-x: auto;
    margin-bottom: 12px;
  }
  .markdown-body :global(pre code) {
    background: none;
    padding: 0;
    color: #cbd5e1;
  }

  .chat-footer {
    padding: 16px;
    border-top: 1px solid var(--border-color);
    background: rgba(0, 0, 0, 0.15);
    flex-shrink: 0;
  }

  .thinking-loader {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 16px;
    background: rgba(0, 229, 255, 0.05);
    border: 1px dashed var(--border-color);
    border-radius: 0 12px 12px 12px;
  }

  .thinking-loader .dots {
    display: flex;
    gap: 4px;
    align-items: center;
  }

  .thinking-loader .dot {
    width: 6px;
    height: 6px;
    background: var(--accent-cyan);
    border-radius: 50%;
    animation: pulse-dot-key 1.4s infinite ease-in-out both;
  }

  .thinking-loader .dot:nth-child(1) { animation-delay: -0.32s; }
  .thinking-loader .dot:nth-child(2) { animation-delay: -0.16s; }

  .stop-btn {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    background: rgba(244, 67, 54, 0.15);
    border: 1px solid rgba(244, 67, 54, 0.35);
    color: #ff8a80;
    padding: 4px 10px;
    border-radius: 6px;
    font-size: 0.75rem;
    font-family: var(--font-sans);
    font-weight: 600;
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .stop-btn:hover {
    background: rgba(244, 67, 54, 0.25);
    border-color: rgba(244, 67, 54, 0.55);
    color: #ffffff;
  }

  .stop-svg {
    fill: currentColor;
  }

  @keyframes pulse-dot-key {
    0%, 80%, 100% { transform: scale(0.6); opacity: 0.3; }
    40% { transform: scale(1.0); opacity: 1; }
  }

  .meta-right {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .tts-toggle-btn {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    background: rgba(255, 255, 255, 0.03);
    border: 1px solid var(--border-color);
    color: var(--text-secondary);
    padding: 3px 8px;
    border-radius: 4px;
    font-size: 0.7rem;
    font-weight: 600;
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .tts-toggle-btn:hover {
    background: rgba(0, 229, 255, 0.05);
    border-color: rgba(0, 229, 255, 0.3);
    color: #ffffff;
  }

  .tts-toggle-btn.active {
    background: rgba(0, 229, 255, 0.15);
    border-color: var(--accent-cyan);
    color: var(--accent-cyan);
    box-shadow: 0 0 8px rgba(0, 229, 255, 0.2);
  }

  .speak-msg-btn {
    background: none;
    border: none;
    color: var(--text-muted);
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: 2px;
    border-radius: 4px;
    transition: all var(--transition-fast);
    margin-left: 4px;
  }

  .speak-msg-btn:hover {
    color: var(--accent-cyan);
    background: rgba(255, 255, 255, 0.05);
  }

  .speak-msg-btn.speaking {
    color: var(--accent-cyan);
    animation: bounce-scale-key 1.2s infinite ease-in-out;
  }

  @keyframes bounce-scale-key {
    0%, 100% { transform: scale(1); }
    50% { transform: scale(1.15); }
  }

  /* Message Header Actions */
  .msg-header {
    display: flex;
    align-items: center;
    width: 100%;
  }

  .msg-header-actions {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-left: auto;
  }

  .review-plan-btn {
    background: rgba(0, 229, 255, 0.1);
    border: 1px solid rgba(0, 229, 255, 0.25);
    color: var(--accent-cyan);
    padding: 3px 8px;
    border-radius: 4px;
    font-size: 0.7rem;
    font-weight: 500;
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    gap: 4px;
    transition: all var(--transition-fast);
  }

  .review-plan-btn:hover {
    background: rgba(0, 229, 255, 0.2);
    border-color: var(--accent-cyan);
    transform: translateY(-1px);
    box-shadow: 0 0 6px rgba(0, 229, 255, 0.15);
  }

  /* Floating Plan Banner */
  .floating-plan-banner {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    background: rgba(0, 229, 255, 0.08);
    border-bottom: 1px solid rgba(0, 229, 255, 0.2);
    padding: 8px 16px;
    font-size: 0.8rem;
    color: var(--text-primary);
    animation: fadeIn var(--transition-normal) forwards;
    flex-shrink: 0;
  }

  .floating-plan-banner .banner-info {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .floating-plan-banner .banner-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .banner-action-btn {
    background: var(--accent-cyan);
    border: none;
    color: #000000;
    padding: 4px 10px;
    border-radius: 4px;
    font-size: 0.75rem;
    font-weight: 600;
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .banner-action-btn:hover {
    transform: translateY(-1px);
    box-shadow: 0 0 8px var(--accent-cyan);
  }

  .banner-dismiss-btn {
    background: none;
    border: none;
    color: var(--text-muted);
    font-size: 1.1rem;
    cursor: pointer;
    padding: 0 4px;
    line-height: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: color var(--transition-fast);
  }

  .banner-dismiss-btn:hover {
    color: #ffffff;
  }

  /* Review Modal Styles */
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(3, 6, 15, 0.75);
    backdrop-filter: blur(8px);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 100;
    padding: 24px;
    animation: fadeIn 0.2s ease-out;
  }

  .review-modal {
    width: 100%;
    max-width: 800px;
    max-height: calc(100vh - 48px);
    display: flex;
    flex-direction: column;
    background: var(--bg-glass);
    border: 1px solid var(--border-color);
    border-radius: 12px;
    box-shadow: 0 20px 50px rgba(0, 0, 0, 0.5);
    overflow: hidden;
  }

  .modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 20px;
    border-bottom: 1px solid var(--border-color);
    flex-shrink: 0;
  }

  .modal-title h3 {
    font-size: 1.1rem;
    font-weight: 600;
    color: #ffffff;
    margin-bottom: 4px;
  }

  .modal-subtitle {
    font-size: 0.75rem;
    color: var(--text-secondary);
  }

  .close-modal-btn {
    background: none;
    border: none;
    color: var(--text-muted);
    font-size: 1.5rem;
    cursor: pointer;
    transition: color var(--transition-fast);
    line-height: 1;
  }

  .close-modal-btn:hover {
    color: #ffffff;
  }

  .modal-body {
    flex-grow: 1;
    overflow-y: auto;
    padding: 20px;
    display: flex;
    flex-direction: column;
    gap: 16px;
    min-height: 0;
  }

  .plan-content-scroll {
    flex-grow: 1;
    overflow-y: auto;
    background: rgba(3, 6, 15, 0.4);
    border: 1px solid var(--border-color);
    border-radius: 6px;
    padding: 16px;
    min-height: 200px;
  }

  .feedback-section {
    display: flex;
    flex-direction: column;
    gap: 8px;
    flex-shrink: 0;
  }

  .feedback-section label {
    font-size: 0.8rem;
    font-weight: 500;
    color: var(--text-secondary);
  }

  .feedback-section textarea {
    background: rgba(3, 6, 15, 0.6);
    border: 1px solid var(--border-color);
    border-radius: 6px;
    padding: 10px;
    color: #ffffff;
    font-size: 0.85rem;
    font-family: var(--font-sans);
    resize: none;
    outline: none;
    transition: border-color var(--transition-fast);
  }

  .feedback-section textarea:focus {
    border-color: var(--accent-cyan);
  }

  .modal-footer {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 12px;
    padding: 12px 20px;
    border-top: 1px solid var(--border-color);
    background: rgba(13, 18, 34, 0.4);
    flex-shrink: 0;
  }

  .btn {
    padding: 8px 16px;
    border-radius: 6px;
    font-size: 0.8rem;
    font-weight: 600;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 6px;
    transition: all var(--transition-fast);
  }

  .btn-secondary {
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    color: var(--text-secondary);
  }

  .btn-secondary:hover {
    background: rgba(255, 255, 255, 0.1);
    color: #ffffff;
  }

  .btn-primary {
    background: var(--accent-cyan);
    border: 1px solid var(--accent-cyan);
    color: #000000;
  }

  .btn-primary:hover {
    transform: translateY(-1px);
    box-shadow: 0 0 12px rgba(0, 229, 255, 0.4);
  }
</style>
