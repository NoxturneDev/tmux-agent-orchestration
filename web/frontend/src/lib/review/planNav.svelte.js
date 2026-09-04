// Lightweight vim-motion controller for plan-review mode (markdown-only,
// no diff). Operates over an array of markdown block strings instead of
// diff rows; every jump re-centers the active block, mirroring vimnav.svelte.js.
export class PlanNavController {
	blocks = $state([]);
	cursorIndex = $state(0);
	searchFocused = $state(false);
	searchTerm = $state('');
	searchResults = $state([]);
	currentSearchIndex = $state(-1);
	showCheatsheet = $state(false);

	lastGTime = 0;

	constructor() {
		this.handleKeyDown = this.handleKeyDown.bind(this);
	}

	setBlocks(blocks) {
		this.blocks = blocks;
		this.cursorIndex = 0;
		this.searchResults = [];
		this.currentSearchIndex = -1;
	}

	handleKeyDown(e) {
		if (this.searchFocused) {
			if (e.key === 'Enter') {
				e.preventDefault();
				this.searchFocused = false;
				this.performSearch();
			} else if (e.key === 'Escape') {
				e.preventDefault();
				this.searchFocused = false;
				this.searchTerm = '';
				this.searchResults = [];
				this.currentSearchIndex = -1;
			}
			return;
		}

		if (this.showCheatsheet) {
			if (e.key === 'Escape' || e.key === '?') {
				e.preventDefault();
				this.showCheatsheet = false;
			}
			return;
		}

		if (document.activeElement && (document.activeElement.tagName === 'INPUT' || document.activeElement.tagName === 'TEXTAREA')) {
			return;
		}

		if (e.ctrlKey) {
			if (e.key === 'd') {
				e.preventDefault();
				this.moveCursor(5);
			} else if (e.key === 'u') {
				e.preventDefault();
				this.moveCursor(-5);
			}
			return;
		}

		switch (e.key) {
			case 'j':
			case '}':
				e.preventDefault();
				this.moveCursor(1);
				break;
			case 'k':
			case '{':
				e.preventDefault();
				this.moveCursor(-1);
				break;
			case 'g':
				if (this.lastGTime && Date.now() - this.lastGTime < 500) {
					e.preventDefault();
					this.cursorIndex = 0;
					this.center();
					this.lastGTime = 0;
				} else {
					this.lastGTime = Date.now();
				}
				break;
			case 'G':
				e.preventDefault();
				if (this.blocks.length > 0) this.cursorIndex = this.blocks.length - 1;
				this.center();
				break;
			case '/':
				e.preventDefault();
				this.searchFocused = true;
				break;
			case 'n':
				e.preventDefault();
				this.jumpSearch(1);
				break;
			case 'N':
				e.preventDefault();
				this.jumpSearch(-1);
				break;
			case '?':
				e.preventDefault();
				this.showCheatsheet = true;
				break;
		}
	}

	moveCursor(offset) {
		if (this.blocks.length === 0) return;
		this.cursorIndex = Math.min(this.blocks.length - 1, Math.max(0, this.cursorIndex + offset));
		this.center();
	}

	performSearch() {
		if (!this.searchTerm) {
			this.searchResults = [];
			this.currentSearchIndex = -1;
			return;
		}
		const q = this.searchTerm.toLowerCase();
		const results = [];
		this.blocks.forEach((b, idx) => {
			if (b.toLowerCase().includes(q)) results.push(idx);
		});
		this.searchResults = results;
		if (results.length > 0) {
			this.currentSearchIndex = 0;
			this.cursorIndex = results[0];
			this.center();
		} else {
			this.currentSearchIndex = -1;
		}
	}

	jumpSearch(direction) {
		if (this.searchResults.length === 0) return;
		this.currentSearchIndex = (this.currentSearchIndex + direction + this.searchResults.length) % this.searchResults.length;
		this.cursorIndex = this.searchResults[this.currentSearchIndex];
		this.center();
	}

	center() {
		setTimeout(() => {
			const el = document.querySelector('.plan-block.active-row');
			if (el) el.scrollIntoView({ block: 'center', behavior: 'auto' });
		}, 0);
	}
}

// Split raw markdown into blank-line-separated blocks (paragraphs/sections).
export function splitMarkdownBlocks(raw) {
	if (!raw) return [];
	return raw
		.split(/\n\s*\n/)
		.map((b) => b.trim())
		.filter((b) => b.length > 0);
}
