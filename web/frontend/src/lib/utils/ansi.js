/**
 * Converts terminal raw text with ANSI escape sequences to styled HTML.
 * Supports SGR 8-color, 256-color, and RGB 24-bit truecolor modes.
 * Escapes HTML characters to prevent XSS.
 * 
 * @param {string} ansiStr - Raw ANSI terminal string
 * @returns {string} Styled HTML string
 */
export function ansiToHtml(ansiStr) {
  if (!ansiStr) return '';

  // Strip non-SGR sequences like cursor hide/show and cursor positioning
  let cleanStr = ansiStr
    .replace(/\x1b\[\?25[hl]/g, '')
    .replace(/\x1b\[[0-9]*[ABCDEFGJKSThlmpqn]/g, '');

  const parts = cleanStr.split(/\x1b\[([0-9;]*)m/);
  let html = '';

  let bold = false;
  let dim = false;
  let italic = false;
  let underline = false;
  let fg = null;
  let bg = null;

  const colors = {
    0: '#000000', 1: '#cd0000', 2: '#00cd00', 3: '#cdcd00',
    4: '#0000ee', 5: '#cd00cd', 6: '#00cdcd', 7: '#e5e5e5',
    8: '#7f7f7f', 9: '#ff0000', 10: '#00ff00', 11: '#ffff00',
    12: '#5c5cff', 13: '#ff00ff', 14: '#00ffff', 15: '#ffffff'
  };

  const getStyleString = () => {
    let styles = [];
    if (bold) styles.push('font-weight: bold');
    if (dim) styles.push('opacity: 0.60');
    if (italic) styles.push('font-style: italic');
    if (underline) styles.push('text-decoration: underline');
    if (fg) styles.push(`color: ${fg}`);
    if (bg) styles.push(`background-color: ${bg}`);
    return styles.length > 0 ? `style="${styles.join('; ')}"` : '';
  };

  for (let i = 0; i < parts.length; i++) {
    if (i % 2 === 0) {
      // Text segment - escape html
      let text = parts[i]
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;');
      
      if (text) {
        const styleStr = getStyleString();
        if (styleStr) {
          html += `<span ${styleStr}>${text}</span>`;
        } else {
          html += text;
        }
      }
    } else {
      // SGR parameters
      const codes = parts[i].split(';').map(Number);
      if (codes.length === 0 || (codes.length === 1 && codes[0] === 0)) {
        bold = false;
        dim = false;
        italic = false;
        underline = false;
        fg = null;
        bg = null;
        continue;
      }

      for (let j = 0; j < codes.length; j++) {
        const code = codes[j];
        if (code === 0) {
          bold = false; dim = false; italic = false; underline = false; fg = null; bg = null;
        } else if (code === 1) {
          bold = true;
        } else if (code === 2) {
          dim = true;
        } else if (code === 3) {
          italic = true;
        } else if (code === 4) {
          underline = true;
        } else if (code === 22) {
          bold = false; dim = false;
        } else if (code === 23) {
          italic = false;
        } else if (code === 24) {
          underline = false;
        } else if (code >= 30 && code <= 37) {
          fg = colors[code - 30];
        } else if (code === 39) {
          fg = null;
        } else if (code >= 40 && code <= 47) {
          bg = colors[code - 40];
        } else if (code === 49) {
          bg = null;
        } else if (code >= 90 && code <= 97) {
          fg = colors[code - 90 + 8];
        } else if (code >= 100 && code <= 107) {
          bg = colors[code - 100 + 8];
        } else if (code === 38 || code === 48) {
          const isFg = code === 38;
          if (codes[j + 1] === 5) {
            // 256-color mode: 38;5;index
            const idx = codes[j + 2];
            let color = colors[idx] || `rgb(${idx},${idx},${idx})`;
            if (idx >= 16 && idx <= 231) {
              const val = idx - 16;
              const r = Math.floor(val / 36) * 51;
              const g = Math.floor((val % 36) / 6) * 51;
              const b = (val % 6) * 51;
              color = `rgb(${r},${g},${b})`;
            } else if (idx >= 232 && idx <= 255) {
              const gray = (idx - 232) * 10 + 8;
              color = `rgb(${gray},${gray},${gray})`;
            }
            if (isFg) fg = color; else bg = color;
            j += 2;
          } else if (codes[j + 1] === 2) {
            // RGB true-color mode: 38;2;r;g;b
            const r = codes[j + 2];
            const g = codes[j + 3];
            const b = codes[j + 4];
            const color = `rgb(${r},${g},${b})`;
            if (isFg) fg = color; else bg = color;
            j += 4;
          }
        }
      }
    }
  }

  return html;
}
