<script>
	import { onMount, tick } from 'svelte';
	import { marked } from 'marked';
	import hljs from 'highlight.js';
	import 'highlight.js/styles/github-dark.css';
	import { VimNavController } from '../review/vimnav.svelte.js';
	import { PlanNavController, splitMarkdownBlocks } from '../review/planNav.svelte.js';
	import VimCheatsheet from './VimCheatsheet.svelte';

	let { slug } = $props();

	let loading = $state(true);
	let error = $state(null);
	let reviewData = $state(null);
	let searchInputEl = $state(null);
	let isPlanMode = $state(false);
	let planBlocks = $state([]);

	async function openNeovim(file, line) {
		try {
			const res = await fetch(`/api/review/${slug}/open`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ file, line })
			});
			if (!res.ok) {
				const errData = await res.json();
				console.error('Failed to open in neovim:', errData.error);
			}
		} catch (err) {
			console.error('Error calling open endpoint:', err);
		}
	}

	const vimNav = new VimNavController(openNeovim);
	const planNav = new PlanNavController();

	function escapeHtml(str) {
		return str.replace(/&/g, '&amp;')
			.replace(/</g, '&lt;')
			.replace(/>/g, '&gt;')
			.replace(/"/g, '&quot;')
			.replace(/'/g, '&#039;');
	}

	function highlightLine(content, language) {
		if (!language || !hljs.getLanguage(language)) return escapeHtml(content);
		try {
			return hljs.highlight(content, { language }).value;
		} catch (e) {
			return escapeHtml(content);
		}
	}

	function prepareRowsPerFile(data) {
		if (!data || !data.diff || !data.diff.files) return [];
		
		return data.diff.files.map((file, fileIdx) => {
			const fileRows = [];
			
			fileRows.push({
				type: 'file_header',
				fileIdx,
				path: file.new_path || file.old_path,
				file
			});

			if (file.hunks) {
				file.hunks.forEach((hunk, hunkIdx) => {
					fileRows.push({
						type: 'hunk_header',
						fileIdx,
						hunkIdx,
						header: hunk.header,
						hunk
					});

					if (hunk.lines) {
						hunk.lines.forEach((line, lineIdx) => {
							fileRows.push({
								type: 'line',
								fileIdx,
								hunkIdx,
								lineIdx,
								kind: line.kind,
								content: line.content,
								old_lineno: line.old_lineno,
								new_lineno: line.new_lineno,
								file
							});
						});
					}
				});
			}
			return fileRows;
		});
	}

	let activeKeyHandler = null;

	onMount(async () => {
		try {
			const res = await fetch(`/api/review/${slug}`);
			if (!res.ok) {
				throw new Error(`Review bundle not found (${res.status})`);
			}
			reviewData = await res.json();
			isPlanMode = reviewData.meta.type === 'plan';

			if (isPlanMode) {
				planBlocks = splitMarkdownBlocks(reviewData.summary_md);
				planNav.setBlocks(planBlocks);
				activeKeyHandler = planNav.handleKeyDown;
			} else {
				const rowsPerFile = prepareRowsPerFile(reviewData);
				vimNav.setRowsPerFile(rowsPerFile, reviewData.diff.files || []);
				activeKeyHandler = vimNav.handleKeyDown;
			}
			loading = false;
		} catch (err) {
			error = err.message;
			loading = false;
		}

		if (activeKeyHandler) {
			window.addEventListener('keydown', activeKeyHandler);
		}
		return () => {
			if (activeKeyHandler) {
				window.removeEventListener('keydown', activeKeyHandler);
			}
		};
	});

	$effect(() => {
		if ((vimNav.searchFocused || planNav.searchFocused) && searchInputEl) {
			tick().then(() => searchInputEl.focus());
		}
	});
</script>

{#if loading}
	<div class="status-screen">
		<div class="loader"></div>
		<p>Loading review changeset '{slug}'...</p>
	</div>
{:else if error}
	<div class="status-screen error-screen">
		<div class="error-icon">&times;</div>
		<p>Error: {error}</p>
		<a href="/" class="home-btn">Return to Dashboard</a>
	</div>
{:else}
	<div class="review-layout">
		<!-- Top Bar -->
		<header class="review-header glass-panel">
			<div class="header-left">
				<a href="/" class="back-link">
					<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="19" y1="12" x2="5" y2="12"/><polyline points="12 19 5 12 12 5"/></svg>
					Dashboard
				</a>
				<h1>{reviewData.meta.title || `Review: ${slug}`}</h1>
				<span class="project-tag">{reviewData.meta.project}</span>
			</div>
			<div class="header-right">
				{#if isPlanMode}
					<div class="stat-badge plan-badge">
						<span>Type</span>
						<strong>Plan</strong>
					</div>
					<button class="icon-btn" onclick={() => planNav.showCheatsheet = true} title="Keyboard shortcuts (?)">
						?
					</button>
				{:else}
					<div class="stat-badge changed">
						<span>Files</span>
						<strong>{reviewData.meta.stat.files_changed}</strong>
					</div>
					<div class="stat-badge additions">
						<span>+</span>
						<strong>{reviewData.meta.stat.additions}</strong>
					</div>
					<div class="stat-badge deletions">
						<span>-</span>
						<strong>{reviewData.meta.stat.deletions}</strong>
					</div>
					<button class="icon-btn" onclick={() => vimNav.showCheatsheet = true} title="Keyboard shortcuts (?)">
						?
					</button>
				{/if}
			</div>
		</header>

		{#if !isPlanMode}
		<!-- Top Sticky Tabline of Changed Files -->
		<div class="tabline glass-panel">
			{#each reviewData.diff.files as file, idx}
				{@const filename = (file.new_path || file.old_path).split('/').pop()}
				<button
					class="tabline-tab"
					class:active={vimNav.activeFileIndex === idx}
					data-index={idx}
					onclick={() => vimNav.setActiveFile(idx)}
					title={file.new_path || file.old_path}
				>
					<span class="tab-status-dot {file.status}"></span>
					<span class="tab-filename">{filename}</span>
					<div class="tab-stats">
						<span class="add-count">+{file.additions}</span>
						<span class="del-count">-{file.deletions}</span>
					</div>
				</button>
			{/each}
		</div>
		{/if}

		{#if isPlanMode}
			<!-- Plan-Review Mode: full-width markdown, vim-navigable by block -->
			<div class="workspace-body plan-workspace">
				<div class="plan-pane">
					{#each planBlocks as block, idx}
						<div class="plan-block" class:active-row={planNav.cursorIndex === idx} data-index={idx}>
							<div class="markdown-body">{@html marked.parse(block)}</div>
						</div>
					{/each}
				</div>

				{#if planNav.searchFocused}
					<div class="search-bar glass-panel">
						<span class="search-prefix">/</span>
						<input
							bind:this={searchInputEl}
							bind:value={planNav.searchTerm}
							placeholder="Search plan (Enter to jump, Esc to cancel)"
						/>
						{#if planNav.searchResults.length > 0}
							<span class="search-status">
								{planNav.currentSearchIndex + 1}/{planNav.searchResults.length}
							</span>
						{:else}
							<span class="search-status empty">No matches</span>
						{/if}
					</div>
				{/if}
			</div>

			<VimCheatsheet bind:visible={planNav.showCheatsheet} mode="plan" />
		{:else}
		<!-- Main Workspace -->
		<div class="workspace-body">
			<!-- Diff Viewer Container -->
			<div class="diff-container" class:collapsed={vimNav.sidebarCollapsed}>
				<div class="diff-scroll-area">
					{#if vimNav.rowsPerFile[vimNav.activeFileIndex]}
						{#each vimNav.rowsPerFile[vimNav.activeFileIndex] as row, idx}
							{#if row.type === 'file_header'}
								<div class="diff-row file-header-row" class:active-row={vimNav.cursorIndex === idx} data-index={idx}>
									<div class="file-info">
										<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="file-icon"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
										<span class="file-path">{row.path}</span>
										<span class="file-status-badge {row.file.status}">{row.file.status}</span>
									</div>
									<div class="file-stats">
										<span class="add-count">+{row.file.additions}</span>
										<span class="del-count">-{row.file.deletions}</span>
									</div>
								</div>
							{:else if row.type === 'hunk_header'}
								<div class="diff-row hunk-header-row" class:active-row={vimNav.cursorIndex === idx} data-index={idx}>
									<span class="hunk-sig">{row.header}</span>
								</div>
							{:else if row.type === 'line'}
								<div class="diff-row line-row {row.kind}" class:active-row={vimNav.cursorIndex === idx} data-index={idx}>
									<div class="ln ln-old">{row.old_lineno !== null ? row.old_lineno : ''}</div>
									<div class="ln ln-new">{row.new_lineno !== null ? row.new_lineno : ''}</div>
									<div class="line-marker">{row.kind === 'add' ? '+' : row.kind === 'del' ? '-' : ' '}</div>
									<pre class="line-content"><code>{@html highlightLine(row.content, row.file.language)}</code></pre>
								</div>
							{/if}
						{/each}
					{/if}
				</div>

				<!-- Floating Search Bar -->
				{#if vimNav.searchFocused}
					<div class="search-bar glass-panel">
						<span class="search-prefix">/</span>
						<input
							bind:this={searchInputEl}
							bind:value={vimNav.searchTerm}
							placeholder="Search active diff (Enter to jump, Esc to cancel)"
						/>
						{#if vimNav.searchResults.length > 0}
							<span class="search-status">
								{vimNav.currentSearchIndex + 1}/{vimNav.searchResults.length}
							</span>
						{:else}
							<span class="search-status empty">No matches</span>
						{/if}
					</div>
				{/if}
			</div>

			<!-- Summary Sidebar Panel -->
			{#if !vimNav.sidebarCollapsed}
				<aside class="summary-sidebar glass-panel">
					<div class="sidebar-header">
						<h3>Summary Report</h3>
						<button class="sidebar-toggle" onclick={() => vimNav.sidebarCollapsed = true}>
							&rarr;
						</button>
					</div>
					<div class="sidebar-content markdown-body">
						{@html marked.parse(reviewData.summary_md || '*No summary provided.*')}
					</div>
				</aside>
			{/if}
		</div>

		<!-- File Palette (Fuzzy Picker Overlay) -->
		{#if vimNav.showFilePalette}
			<div class="palette-overlay" onclick={() => vimNav.showFilePalette = false}>
				<div class="palette-content glass-panel" onclick={e => e.stopPropagation()}>
					<div class="palette-header">
						<h4>Jump to File</h4>
						<span class="palette-hint">Use j/k to select, Enter to jump</span>
					</div>
					<div class="palette-list">
						{#each vimNav.files as file, idx}
							<div
								class="palette-item"
								class:selected={vimNav.paletteSelectedIndex === idx}
								onclick={() => {
									vimNav.showFilePalette = false;
									vimNav.setActiveFile(idx);
								}}
							>
								<span class="file-path">{file.new_path || file.old_path}</span>
								<div class="item-stats">
									<span class="add-count">+{file.additions}</span>
									<span class="del-count">-{file.deletions}</span>
								</div>
							</div>
						{/each}
					</div>
				</div>
			</div>
		{/if}

		<!-- Vim Shortcuts overlay -->
		<VimCheatsheet bind:visible={vimNav.showCheatsheet} mode="diff" />
		{/if}
	</div>
{/if}

<style>
	.status-screen {
		display: flex;
		flex-direction: column;
		justify-content: center;
		align-items: center;
		height: 100vh;
		background: #070a13;
		color: var(--text-secondary);
		gap: 16px;
	}

	.loader {
		width: 48px;
		height: 48px;
		border: 3px solid rgba(0, 229, 255, 0.1);
		border-radius: 50%;
		border-top-color: var(--accent-cyan);
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	.error-screen {
		color: #ef4444;
	}

	.error-icon {
		font-size: 3rem;
		border: 2px solid #ef4444;
		width: 60px;
		height: 60px;
		display: flex;
		align-items: center;
		justify-content: center;
		border-radius: 50%;
		line-height: 1;
	}

	.home-btn {
		margin-top: 16px;
		padding: 8px 16px;
		background: rgba(255, 255, 255, 0.1);
		color: #ffffff;
		text-decoration: none;
		border-radius: 6px;
		border: 1px solid var(--border-color);
		transition: all var(--transition-fast);
	}

	.home-btn:hover {
		background: var(--accent-cyan);
		color: #000000;
	}

	.review-layout {
		display: flex;
		flex-direction: column;
		height: 100vh;
		background: #070a13;
		overflow: hidden;
		font-family: var(--font-family, sans-serif);
	}

	.review-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		height: 56px;
		padding: 0 20px;
		border-bottom: 1px solid var(--border-color);
		flex-shrink: 0;
		background: rgba(13, 18, 34, 0.5);
	}

	.header-left {
		display: flex;
		align-items: center;
		gap: 16px;
	}

	.back-link {
		display: flex;
		align-items: center;
		gap: 6px;
		color: var(--text-secondary);
		text-decoration: none;
		font-size: 0.9rem;
		transition: color var(--transition-fast);
	}

	.back-link:hover {
		color: #ffffff;
	}

	.header-left h1 {
		margin: 0;
		font-size: 1.1rem;
		font-weight: 600;
		color: #ffffff;
	}

	.project-tag {
		font-size: 0.8rem;
		color: var(--accent-cyan);
		background: rgba(0, 229, 255, 0.1);
		padding: 2px 8px;
		border-radius: 4px;
	}

	.header-right {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.stat-badge {
		display: flex;
		align-items: center;
		gap: 6px;
		background: rgba(255, 255, 255, 0.05);
		border: 1px solid var(--border-color);
		padding: 4px 10px;
		border-radius: 6px;
		font-size: 0.85rem;
	}

	.stat-badge span {
		color: var(--text-secondary);
	}

	.stat-badge.additions strong {
		color: #10b981;
	}

	.stat-badge.deletions strong {
		color: #ef4444;
	}

	.icon-btn {
		background: none;
		border: 1px solid var(--border-color);
		color: var(--text-secondary);
		width: 32px;
		height: 32px;
		display: flex;
		align-items: center;
		justify-content: center;
		border-radius: 6px;
		cursor: pointer;
		font-weight: bold;
		transition: all var(--transition-fast);
	}

	.icon-btn:hover {
		color: #ffffff;
		border-color: #ffffff;
	}

	/* Top Tabline of Changed Files */
	.tabline {
		display: flex;
		gap: 12px;
		padding: 8px 16px;
		background: rgba(13, 18, 34, 0.3);
		border-bottom: 1px solid var(--border-color);
		overflow-x: auto;
		white-space: nowrap;
		flex-shrink: 0;
		scrollbar-width: thin;
	}

	.tabline::-webkit-scrollbar {
		height: 4px;
	}

	.tabline::-webkit-scrollbar-thumb {
		background: rgba(0, 229, 255, 0.2);
		border-radius: 2px;
	}

	.tabline-tab {
		display: flex;
		align-items: center;
		gap: 8px;
		background: rgba(255, 255, 255, 0.02);
		border: 1px solid var(--border-color);
		padding: 6px 12px;
		border-radius: 6px;
		color: var(--text-secondary);
		cursor: pointer;
		font-size: 0.85rem;
		transition: all var(--transition-fast);
		position: relative;
	}

	.tabline-tab:hover {
		background: rgba(255, 255, 255, 0.05);
		color: #ffffff;
	}

	.tabline-tab.active {
		color: #ffffff;
		background: rgba(0, 229, 255, 0.05);
		border-color: rgba(0, 229, 255, 0.3);
	}

	.tabline-tab.active::after {
		content: '';
		position: absolute;
		bottom: -1px;
		left: 8px;
		right: 8px;
		height: 2px;
		background: var(--accent-cyan);
	}

	.tab-status-dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		flex-shrink: 0;
	}

	.tab-status-dot.added { background-color: #10b981; }
	.tab-status-dot.deleted { background-color: #ef4444; }
	.tab-status-dot.modified { background-color: #f59e0b; }
	.tab-status-dot.renamed { background-color: #3b82f6; }

	.tab-filename {
		font-family: monospace;
	}

	.tab-stats {
		display: flex;
		gap: 6px;
		font-size: 0.75rem;
		font-weight: bold;
	}

	.workspace-body {
		display: flex;
		flex-grow: 1;
		min-height: 0;
	}

	.diff-container {
		display: flex;
		flex-direction: column;
		flex-grow: 1;
		min-width: 0;
		position: relative;
	}

	.diff-scroll-area {
		flex-grow: 1;
		overflow-y: auto;
		background: #0a0d1a;
	}

	.summary-sidebar {
		width: 380px;
		border-left: 1px solid var(--border-color);
		display: flex;
		flex-direction: column;
		background: rgba(13, 18, 34, 0.3);
		flex-shrink: 0;
	}

	.sidebar-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 16px 20px;
		border-bottom: 1px solid var(--border-color);
	}

	.sidebar-header h3 {
		margin: 0;
		font-size: 0.95rem;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: #ffffff;
	}

	.sidebar-toggle {
		background: none;
		border: none;
		color: var(--text-secondary);
		cursor: pointer;
		font-size: 1.2rem;
		transition: color var(--transition-fast);
	}

	.sidebar-toggle:hover {
		color: #ffffff;
	}

	.sidebar-content {
		flex-grow: 1;
		overflow-y: auto;
		padding: 20px;
		font-size: 0.9rem;
		line-height: 1.6;
		color: var(--text-secondary);
	}

	.diff-row {
		display: flex;
		align-items: stretch;
		font-family: monospace;
		font-size: 0.85rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.02);
		position: relative;
	}

	.diff-row.active-row {
		background: rgba(0, 229, 255, 0.1) !important;
		outline: 1px solid var(--accent-cyan);
		z-index: 5;
	}

	.file-header-row {
		background: #111425;
		padding: 10px 16px;
		justify-content: space-between;
		align-items: center;
		position: sticky;
		top: 0;
		z-index: 10;
		border-bottom: 1px solid var(--border-color);
	}

	.file-info {
		display: flex;
		align-items: center;
		gap: 10px;
	}

	.file-icon {
		color: var(--accent-cyan);
	}

	.file-path {
		color: #ffffff;
		font-weight: 500;
	}

	.file-status-badge {
		font-size: 0.75rem;
		padding: 1px 6px;
		border-radius: 3px;
		text-transform: uppercase;
		font-weight: bold;
	}

	.file-status-badge.added { background: rgba(16, 185, 129, 0.2); color: #10b981; }
	.file-status-badge.deleted { background: rgba(239, 68, 68, 0.2); color: #ef4444; }
	.file-status-badge.modified { background: rgba(245, 158, 11, 0.2); color: #f59e0b; }
	.file-status-badge.renamed { background: rgba(59, 130, 246, 0.2); color: #3b82f6; }

	.file-stats {
		display: flex;
		gap: 12px;
		font-size: 0.8rem;
		font-weight: bold;
	}

	.add-count { color: #10b981; }
	.del-count { color: #ef4444; }

	.hunk-header-row {
		background: #151a30;
		color: #8b9bb4;
		padding: 6px 16px;
	}

	.hunk-sig {
		color: #798eb3;
	}

	.line-row {
		background: #0a0d1a;
	}

	.line-row.add {
		background: rgba(16, 185, 129, 0.08);
	}

	.line-row.del {
		background: rgba(239, 68, 68, 0.08);
	}

	.ln {
		width: 48px;
		text-align: right;
		padding-right: 12px;
		color: #4b526d;
		user-select: none;
		border-right: 1px solid rgba(255, 255, 255, 0.05);
		flex-shrink: 0;
	}

	.line-row.add .ln-new { color: #10b981; }
	.line-row.del .ln-old { color: #ef4444; }

	.line-marker {
		width: 20px;
		display: flex;
		align-items: center;
		justify-content: center;
		color: #4b526d;
		user-select: none;
		font-weight: bold;
		flex-shrink: 0;
	}

	.line-row.add .line-marker { color: #10b981; }
	.line-row.del .line-marker { color: #ef4444; }

	.line-content {
		margin: 0;
		padding: 4px 12px;
		overflow-x: auto;
		flex-grow: 1;
	}

	.line-content code {
		color: #c9d1d9;
		background: none;
		padding: 0;
		font-size: 0.85rem;
		font-family: inherit;
		white-space: pre-wrap;
	}

	.search-bar {
		position: absolute;
		bottom: 24px;
		left: 50%;
		transform: translateX(-50%);
		width: 450px;
		background: rgba(13, 18, 34, 0.95);
		border: 1px solid var(--border-color);
		border-radius: 8px;
		padding: 8px 16px;
		display: flex;
		align-items: center;
		gap: 12px;
		z-index: 100;
	}

	.search-prefix {
		color: var(--accent-cyan);
		font-family: monospace;
		font-weight: bold;
		font-size: 1.1rem;
	}

	.search-bar input {
		background: none;
		border: none;
		color: #ffffff;
		flex-grow: 1;
		font-family: inherit;
		outline: none;
		font-size: 0.9rem;
	}

	.search-status {
		font-size: 0.8rem;
		color: var(--text-secondary);
	}

	.search-status.empty {
		color: #ef4444;
	}

	.palette-overlay {
		position: fixed;
		top: 0;
		left: 0;
		width: 100vw;
		height: 100vh;
		background: rgba(4, 6, 12, 0.8);
		display: flex;
		justify-content: center;
		align-items: flex-start;
		padding-top: 10vh;
		z-index: 900;
	}

	.palette-content {
		width: 90%;
		max-width: 500px;
		background: rgba(13, 18, 34, 0.95);
		border: 1px solid var(--border-color);
		border-radius: 12px;
		padding: 16px;
		display: flex;
		flex-direction: column;
		max-height: 60vh;
	}

	.palette-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 12px;
		border-bottom: 1px solid var(--border-color);
		padding-bottom: 8px;
	}

	.palette-header h4 {
		margin: 0;
		color: #ffffff;
		font-size: 1rem;
	}

	.palette-hint {
		font-size: 0.75rem;
		color: var(--text-secondary);
	}

	.palette-list {
		overflow-y: auto;
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.palette-item {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 10px 12px;
		border-radius: 6px;
		cursor: pointer;
		transition: all var(--transition-fast);
		border: 1px solid transparent;
	}

	.palette-item.selected, .palette-item:hover {
		background: rgba(0, 229, 255, 0.1);
		border-color: rgba(0, 229, 255, 0.2);
	}

	.palette-item .file-path {
		font-size: 0.85rem;
		font-family: monospace;
	}

	.item-stats {
		display: flex;
		gap: 8px;
		font-size: 0.75rem;
		font-weight: bold;
	}

	:global(.markdown-body h1), :global(.markdown-body h2), :global(.markdown-body h3) {
		color: #ffffff;
		margin-top: 20px;
		margin-bottom: 10px;
	}
	:global(.markdown-body h1) { font-size: 1.4rem; }
	:global(.markdown-body h2) { font-size: 1.15rem; }
	:global(.markdown-body h3) { font-size: 1rem; }
	:global(.markdown-body ul) {
		padding-left: 20px;
		margin-bottom: 16px;
	}
	:global(.markdown-body li) {
		margin-bottom: 6px;
	}
	:global(.markdown-body code) {
		background: rgba(255, 255, 255, 0.1);
		padding: 2px 4px;
		border-radius: 4px;
		font-size: 0.85em;
	}

	.plan-badge strong {
		color: var(--accent-cyan);
	}

	.plan-workspace {
		flex-grow: 1;
		overflow-y: auto;
		display: flex;
		justify-content: center;
		background: #0a0d1a;
	}

	.plan-pane {
		width: 100%;
		max-width: 860px;
		padding: 32px 40px 40vh 40px;
	}

	.plan-block {
		padding: 10px 12px;
		border-radius: 4px;
		border: 1px solid transparent;
	}

	.plan-block.active-row {
		background: rgba(0, 229, 255, 0.08);
		border-color: var(--accent-cyan);
	}

	.plan-block .markdown-body {
		font-size: 0.95rem;
		line-height: 1.7;
		color: var(--text-secondary);
	}
</style>
