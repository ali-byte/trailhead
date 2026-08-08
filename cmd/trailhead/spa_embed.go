//go:build spa

// This file embeds the built SPA (dist/, produced by `cd web && npm run
// build` — see vite.config.ts's build.outDir) into the trailhead binary.
//
// Gated behind the `spa` build tag (see spa_stub.go for the untagged
// fallback and the Makefile's build-spa target for the tagged build) so
// that a plain, untagged `go build ./...` / `go vet ./...` / `go test
// ./...` / `golangci-lint run ./...` — what CI's existing lint/test/build
// jobs already run, and what a backend-only Terminal session should keep
// being able to run without Node installed or the SPA having been built —
// never requires dist/ to exist on disk. Go's //go:embed directive fails
// to compile against a directory that does not exist (see this package's
// original Phase B placeholder comment on this exact point); only the
// explicit `-tags spa` build (the Makefile's build-spa target, and CI's
// frontend-e2e job) opts into requiring it, after `npm run build` has
// already run.
package main

import (
	"embed"
	"io/fs"
)

//go:embed dist
var spaAssets embed.FS

const spaEmbedded = true

// spaFS returns the embedded SPA's filesystem rooted at dist/ (so
// fs.FS-consuming code sees index.html/assets/... directly, not a
// dist/-prefixed path).
func spaFS() (fs.FS, error) {
	return fs.Sub(spaAssets, "dist")
}
