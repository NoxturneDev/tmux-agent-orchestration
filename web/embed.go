package web

import "embed"

// FS embeds the Svelte static frontend build assets.
//go:embed all:frontend/dist
var FS embed.FS
