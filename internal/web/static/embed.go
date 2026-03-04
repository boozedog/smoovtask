// Package static embeds and serves the web UI's static assets.
package static

import "embed"

//go:embed all:dist
var Assets embed.FS
