export class VimNavController {
	rowsPerFile = $state([]); // array of arrays: [ [rows for file 0], [rows for file 1], ... ]
	cursorIndex = $state(0);
	activeFileIndex = $state(0);
	searchFocused = $state(false);
	searchTerm = $state('');
	searchResults = $state([]);
	currentSearchIndex = $state(-1);
	showCheatsheet = $state(false);
	showFilePalette = $state(false);
	paletteSelectedIndex = $state(0);
	sidebarCollapsed = $state(false);
	files = $state([]);
	onOpenNeovim = null;

	lastKey = null;
	lastKeyTime = 0;
	lastGTime = 0;

	constructor(onOpenNeovim) {
		this.onOpenNeovim = onOpenNeovim;
		this.handleKeyDown = this.handleKeyDown.bind(this);
	}

	setRowsPerFile(rowsPerFile, files) {
		this.rowsPerFile = rowsPerFile;
		this.files = files;
		this.activeFileIndex = 0;
		this.cursorIndex = 0;
		this.searchResults = [];
		this.currentSearchIndex = -1;
		this.showFilePalette = false;
		this.showCheatsheet = false;
	}

	setActiveFile(i) {
		if (i < 0 || i >= this.files.length) return;
		this.activeFileIndex = i;
		this.cursorIndex = 0;
		this.searchResults = [];
		this.currentSearchIndex = -1;
		this.centerOnCursor();
		
		setTimeout(() => {
			const tabEl = document.querySelector(`.tabline-tab[data-index="${i}"]`);
			if (tabEl) {
				tabEl.scrollIntoView({ inline: 'center', behavior: 'smooth' });
			}
			const diffArea = document.querySelector(`.diff-scroll-area`);
			if (diffArea) {
				diffArea.scrollTop = 0;
			}
		}, 0);
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

		if (this.showFilePalette) {
			if (e.key === 'Escape') {
				e.preventDefault();
				this.showFilePalette = false;
			} else if (e.key === 'j' || e.key === 'ArrowDown') {
				e.preventDefault();
				this.paletteSelectedIndex = (this.paletteSelectedIndex + 1) % this.files.length;
			} else if (e.key === 'k' || e.key === 'ArrowUp') {
				e.preventDefault();
				this.paletteSelectedIndex = (this.paletteSelectedIndex - 1 + this.files.length) % this.files.length;
			} else if (e.key === 'Enter') {
				e.preventDefault();
				this.showFilePalette = false;
				this.setActiveFile(this.paletteSelectedIndex);
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
				this.moveCursor(20);
			} else if (e.key === 'u') {
				e.preventDefault();
				this.moveCursor(-20);
			} else if (e.key === 'p') {
				e.preventDefault();
				this.showFilePalette = true;
				this.paletteSelectedIndex = this.activeFileIndex;
			}
			return;
		}

		// Capital H / Shift+H and Capital L / Shift+L for buffer switching
		if (e.key === 'H') {
			e.preventDefault();
			this.setActiveFile(Math.max(0, this.activeFileIndex - 1));
			return;
		}
		if (e.key === 'L') {
			e.preventDefault();
			this.setActiveFile(Math.min(this.files.length - 1, this.activeFileIndex + 1));
			return;
		}

		const currentRows = this.rowsPerFile[this.activeFileIndex] || [];

		switch (e.key) {
			case 'j':
				e.preventDefault();
				this.moveCursor(1);
				break;
			case 'k':
				e.preventDefault();
				this.moveCursor(-1);
				break;
			case 'g':
				if (this.lastGTime && Date.now() - this.lastGTime < 500) {
					e.preventDefault();
					this.cursorIndex = 0;
					this.centerOnCursor();
					this.lastGTime = 0;
				} else {
					this.lastGTime = Date.now();
				}
				break;
			case 'G':
				e.preventDefault();
				if (currentRows.length > 0) {
					this.cursorIndex = currentRows.length - 1;
				}
				this.centerOnCursor();
				break;
			case '{':
				e.preventDefault();
				this.jumpParagraph(-1);
				break;
			case '}':
				e.preventDefault();
				this.jumpParagraph(1);
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
			case 's':
				e.preventDefault();
				this.sidebarCollapsed = !this.sidebarCollapsed;
				break;
			case '?':
				e.preventDefault();
				this.showCheatsheet = true;
				break;
			case 'o':
				e.preventDefault();
				this.triggerOpenNeovim();
				break;
			case 'Tab':
				e.preventDefault();
				this.setActiveFile(Math.min(this.files.length - 1, this.activeFileIndex + 1));
				break;
			default:
				if (e.key === ']' || e.key === '[') {
					this.lastKey = e.key;
					this.lastKeyTime = Date.now();
				} else if (this.lastKey && Date.now() - this.lastKeyTime < 1000) {
					if (this.lastKey === ']' && e.key === 'c') {
						e.preventDefault();
						this.jumpHunk(1);
					} else if (this.lastKey === '[' && e.key === 'c') {
						e.preventDefault();
						this.jumpHunk(-1);
					} else if (this.lastKey === ']' && e.key === 'f') {
						e.preventDefault();
						this.setActiveFile(Math.min(this.files.length - 1, this.activeFileIndex + 1));
					} else if (this.lastKey === '[' && e.key === 'f') {
						e.preventDefault();
						this.setActiveFile(Math.max(0, this.activeFileIndex - 1));
					}
					this.lastKey = null;
				}
		}
	}

	moveCursor(offset) {
		const currentRows = this.rowsPerFile[this.activeFileIndex] || [];
		if (currentRows.length === 0) return;
		this.cursorIndex = Math.min(currentRows.length - 1, Math.max(0, this.cursorIndex + offset));
		this.centerOnCursor();
	}

	jumpHunk(direction) {
		const currentRows = this.rowsPerFile[this.activeFileIndex] || [];
		if (currentRows.length === 0) return;
		let idx = this.cursorIndex;
		while (true) {
			idx += direction;
			if (idx < 0 || idx >= currentRows.length) break;
			if (currentRows[idx].type === 'hunk_header') {
				this.cursorIndex = idx;
				this.centerOnCursor();
				return;
			}
		}
	}

	performSearch() {
		if (!this.searchTerm) {
			this.searchResults = [];
			this.currentSearchIndex = -1;
			return;
		}
		const q = this.searchTerm.toLowerCase();
		const results = [];
		const currentRows = this.rowsPerFile[this.activeFileIndex] || [];
		currentRows.forEach((row, idx) => {
			let text = '';
			if (row.type === 'file_header') {
				text = row.path;
			} else if (row.type === 'hunk_header') {
				text = row.header;
			} else if (row.type === 'line') {
				text = row.content;
			}
			if (text.toLowerCase().includes(q)) {
				results.push(idx);
			}
		});

		this.searchResults = results;
		if (results.length > 0) {
			this.currentSearchIndex = 0;
			this.cursorIndex = results[0];
			this.centerOnCursor();
		} else {
			this.currentSearchIndex = -1;
		}
	}

	jumpSearch(direction) {
		if (this.searchResults.length === 0) return;
		this.currentSearchIndex = (this.currentSearchIndex + direction + this.searchResults.length) % this.searchResults.length;
		this.cursorIndex = this.searchResults[this.currentSearchIndex];
		this.centerOnCursor();
	}

	triggerOpenNeovim() {
		const currentRows = this.rowsPerFile[this.activeFileIndex] || [];
		if (currentRows.length === 0 || !this.onOpenNeovim) return;
		const row = currentRows[this.cursorIndex];
		if (!row) return;

		let file = '';
		let line = 1;

		if (row.type === 'file_header') {
			file = row.path;
			line = 1;
		} else if (row.type === 'hunk_header') {
			const f = this.files[this.activeFileIndex];
			file = f.new_path || f.old_path;
			line = row.hunk.new_start;
		} else if (row.type === 'line') {
			const f = this.files[this.activeFileIndex];
			file = f.new_path || f.old_path;
			line = row.new_lineno || row.old_lineno || 1;
		}

		if (file) {
			this.onOpenNeovim(file, line);
		}
	}

	jumpParagraph(direction) {
		const currentRows = this.rowsPerFile[this.activeFileIndex] || [];
		if (currentRows.length === 0) return;

		let idx = this.cursorIndex;
		const isBlank = (i) => {
			const r = currentRows[i];
			return r && r.type === 'line' && r.content.trim() === '';
		};

		if (direction > 0) {
			while (idx < currentRows.length - 1 && isBlank(idx)) {
				idx++;
			}
			while (idx < currentRows.length - 1 && !isBlank(idx)) {
				idx++;
			}
			this.cursorIndex = idx;
		} else {
			while (idx > 0 && isBlank(idx)) {
				idx--;
			}
			while (idx > 0 && !isBlank(idx)) {
				idx--;
			}
			this.cursorIndex = idx;
		}
		this.centerOnCursor();
	}

	centerOnCursor() {
		setTimeout(() => {
			const activeEl = document.querySelector(`.diff-row.active-row`);
			if (activeEl) {
				activeEl.scrollIntoView({ block: 'center', behavior: 'auto' });
			}
		}, 0);
	}
}
