/// <reference types="vitest/config" />
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// build.outDir: cmd/trailhead/dist, NOT web/dist. Go's //go:embed directive
// cannot traverse ".." out of the embedding file's own directory subtree
// (cmd/trailhead is not an ancestor of a sibling web/dist), so the SPA
// build output is placed directly under cmd/trailhead instead, where
// cmd/trailhead/spa_embed.go can embed it with a same-directory `dist`
// pattern. See that file's doc comment for the full reasoning and the
// `spa` build-tag split that keeps this optional for plain `go build`.
export default defineConfig({
  plugins: [react()],
  build: {
    outDir: '../cmd/trailhead/dist',
    emptyOutDir: true,
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    css: true,
    exclude: ['e2e/**', 'node_modules/**'],
  },
});
