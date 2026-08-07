import { defineConfig, devices } from '@playwright/test';

// UI Contract "Playwright backend choice" (docs/issues/06-frontend-board-
// add-bar.md): a throwaway test Postgres, mirroring
// tests/integration/postgres/setup_test.go's own pattern (TEST_DATABASE_URL
// env var, TRUNCATE-based per-test isolation — see e2e/fixtures.ts) rather
// than a FakeBookmarkRepository test-mode wiring. webServer starts the
// actual built binary (go:embed SPA + API) — this is expected red until
// Phase F builds cmd/trailhead's real wiring and the web/dist build output
// (see UI Contract "cmd/trailhead requirements").
const PORT = process.env.PLAYWRIGHT_PORT ?? '8090';
const BASE_URL = `http://127.0.0.1:${PORT}`;

export default defineConfig({
  testDir: './e2e',
  // Not fullyParallel — every spec shares one Postgres database and each
  // test truncates it in beforeEach (see e2e/fixtures.ts resetBookmarks);
  // parallel workers would truncate out from under each other.
  fullyParallel: false,
  workers: 1,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [['github'], ['html', { open: 'never' }]] : 'list',
  use: {
    baseURL: BASE_URL,
    trace: 'on-first-retry',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  webServer: {
    command: '../bin/trailhead',
    url: BASE_URL,
    reuseExistingServer: !process.env.CI,
    timeout: 30_000,
    env: {
      PORT,
      DATABASE_URL: process.env.TEST_DATABASE_URL ?? '',
    },
  },
});
