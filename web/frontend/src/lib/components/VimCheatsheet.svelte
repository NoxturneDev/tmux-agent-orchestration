<script>
	let { visible = $bindable(false), mode = 'diff' } = $props();
</script>

{#if visible}
	<div class="cheatsheet-overlay" onclick={() => visible = false}>
		<div class="cheatsheet-content glass-panel" onclick={e => e.stopPropagation()}>
			<div class="cheatsheet-header">
				<h3>Vim Keyboard Shortcuts</h3>
				<button class="close-btn" onclick={() => visible = false}>&times;</button>
			</div>

			{#if mode === 'plan'}
				<div class="cheatsheet-grid">
					<div class="key-group">
						<h4>Navigation</h4>
						<div class="key-row"><span class="key">j</span> <span class="desc">Block down</span></div>
						<div class="key-row"><span class="key">k</span> <span class="desc">Block up</span></div>
						<div class="key-row"><span class="key">Ctrl + d</span> <span class="desc">5 blocks down + center</span></div>
						<div class="key-row"><span class="key">Ctrl + u</span> <span class="desc">5 blocks up + center</span></div>
						<div class="key-row"><span class="key">gg</span> <span class="desc">Jump to first block + center</span></div>
						<div class="key-row"><span class="key">G</span> <span class="desc">Jump to last block + center</span></div>
						<div class="key-row"><span class="key">&#123;</span> <span class="desc">Previous block (alias of k)</span></div>
						<div class="key-row"><span class="key">&#125;</span> <span class="desc">Next block (alias of j)</span></div>
					</div>

					<div class="key-group">
						<h4>Actions</h4>
						<div class="key-row"><span class="key">/</span> <span class="desc">Search plan text</span></div>
						<div class="key-row"><span class="key">n</span> <span class="desc">Next search match</span></div>
						<div class="key-row"><span class="key">N</span> <span class="desc">Previous search match</span></div>
						<div class="key-row"><span class="key">?</span> <span class="desc">Toggle this cheatsheet</span></div>
					</div>
				</div>
			{:else}
				<div class="cheatsheet-grid">
					<div class="key-group">
						<h4>Navigation</h4>
						<div class="key-row"><span class="key">j</span> <span class="desc">Line down</span></div>
						<div class="key-row"><span class="key">k</span> <span class="desc">Line up</span></div>
						<div class="key-row"><span class="key">Ctrl + d</span> <span class="desc">Half-page down + center</span></div>
						<div class="key-row"><span class="key">Ctrl + u</span> <span class="desc">Half-page up + center</span></div>
						<div class="key-row"><span class="key">gg</span> <span class="desc">Jump to first line + center</span></div>
						<div class="key-row"><span class="key">G</span> <span class="desc">Jump to last line + center</span></div>
						<div class="key-row"><span class="key">&#123;</span> <span class="desc">Previous paragraph (blank line hop)</span></div>
						<div class="key-row"><span class="key">&#125;</span> <span class="desc">Next paragraph (blank line hop)</span></div>
					</div>

					<div class="key-group">
						<h4>Jumps & Selections</h4>
						<div class="key-row"><span class="key">]c</span> <span class="desc">Next hunk + center</span></div>
						<div class="key-row"><span class="key">[c</span> <span class="desc">Previous hunk + center</span></div>
						<div class="key-row"><span class="key">H</span> <span class="desc">Previous file (buffer)</span></div>
						<div class="key-row"><span class="key">L</span> <span class="desc">Next file (buffer)</span></div>
						<div class="key-row"><span class="key">Tab</span> / <span class="key">]f</span> <span class="desc">Next file (alias of L)</span></div>
						<div class="key-row"><span class="key">Shift+Tab</span> / <span class="key">[f</span> <span class="desc">Previous file (alias of H)</span></div>
						<div class="key-row"><span class="key">Ctrl + p</span> <span class="desc">Fuzzy file finder palette</span></div>
					</div>

					<div class="key-group">
						<h4>Actions</h4>
						<div class="key-row"><span class="key">o</span> <span class="desc">Open file in Neovim at cursor line</span></div>
						<div class="key-row"><span class="key">s</span> <span class="desc">Toggle summary sidebar</span></div>
						<div class="key-row"><span class="key">/</span> <span class="desc">Search diff lines</span></div>
						<div class="key-row"><span class="key">n</span> <span class="desc">Next search match</span></div>
						<div class="key-row"><span class="key">N</span> <span class="desc">Previous search match</span></div>
						<div class="key-row"><span class="key">?</span> <span class="desc">Toggle this cheatsheet</span></div>
					</div>
				</div>
			{/if}

			<div class="cheatsheet-footer">
				<p>Press <span class="key-inline">Esc</span> or click anywhere outside to close</p>
			</div>
		</div>
	</div>
{/if}

<style>
	.cheatsheet-overlay {
		position: fixed;
		top: 0;
		left: 0;
		width: 100vw;
		height: 100vh;
		background: rgba(4, 6, 12, 0.85);
		display: flex;
		justify-content: center;
		align-items: center;
		z-index: 1000;
	}

	.cheatsheet-content {
		width: 90%;
		max-width: 650px;
		background: rgba(13, 18, 34, 0.95);
		border: 1px solid var(--border-color);
		border-radius: 12px;
		padding: 24px;
	}

	.cheatsheet-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		border-bottom: 1px solid var(--border-color);
		padding-bottom: 12px;
		margin-bottom: 20px;
	}

	.cheatsheet-header h3 {
		margin: 0;
		color: #ffffff;
		font-size: 1.25rem;
		font-weight: 600;
	}

	.close-btn {
		background: none;
		border: none;
		color: var(--text-secondary);
		font-size: 1.5rem;
		cursor: pointer;
		transition: color var(--transition-fast);
	}

	.close-btn:hover {
		color: #ffffff;
	}

	.cheatsheet-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 24px;
	}

	@media (max-width: 600px) {
		.cheatsheet-grid {
			grid-template-columns: 1fr;
		}
	}

	.key-group {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.key-group h4 {
		margin: 0 0 8px 0;
		color: var(--accent-cyan);
		font-size: 0.9rem;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.key-row {
		display: flex;
		align-items: center;
		gap: 12px;
		font-size: 0.85rem;
	}

	.key {
		background: rgba(0, 229, 255, 0.1);
		border: 1px solid rgba(0, 229, 255, 0.2);
		color: var(--accent-cyan);
		padding: 2px 6px;
		border-radius: 4px;
		font-family: monospace;
		font-weight: bold;
		min-width: 24px;
		text-align: center;
	}

	.key-inline {
		background: rgba(255, 255, 255, 0.1);
		border: 1px solid rgba(255, 255, 255, 0.2);
		color: #ffffff;
		padding: 2px 6px;
		border-radius: 4px;
		font-family: monospace;
		font-size: 0.8rem;
	}

	.desc {
		color: var(--text-secondary);
	}

	.cheatsheet-footer {
		margin-top: 24px;
		padding-top: 12px;
		border-top: 1px solid var(--border-color);
		text-align: center;
		font-size: 0.8rem;
		color: var(--text-secondary);
	}

	.cheatsheet-footer p {
		margin: 0;
	}
</style>
