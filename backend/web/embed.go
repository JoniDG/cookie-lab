// Package web embeds the frontend build. dist/ is filled by `make web` or the
// Dockerfile; only dist/.gitkeep is versioned so the package always compiles.
package web

import "embed"

// Dist holds the frontend build under the "dist" directory.
//
//go:embed all:dist
var Dist embed.FS
