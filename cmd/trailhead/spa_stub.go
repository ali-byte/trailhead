//go:build !spa

// The default (untagged) build's spaFS — see spa_embed.go's doc comment
// for why the real embed is gated behind the `spa` tag. main.go falls
// back to a minimal placeholder response for non-API routes when
// spaEmbedded is false, so a plain `go build ./cmd/trailhead` still
// produces a working (API-only) binary.
package main

import (
	"io/fs"
)

const spaEmbedded = false

func spaFS() (fs.FS, error) {
	return nil, errSPANotEmbedded
}
