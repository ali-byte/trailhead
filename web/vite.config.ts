/// <reference types="vitest/config" />
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// Pre-Phase F scaffold (issue #6) — do not add design-token theme
// configuration here; that is the Phase F build's job. This file wires the
// toolchain only: React, the dev/build pipeline, and the Vitest test
// runner (jsdom environment + the locked setup file that boots MSW).
export default defineConfig({
  plugins: [react()],
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    css: true,
    exclude: ['e2e/**', 'node_modules/**'],
  },
});
