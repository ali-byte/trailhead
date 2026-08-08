// Direct-Postgres fixture helpers for Playwright specs — mirrors
// tests/integration/postgres/setup_test.go's own pattern (testDSN's loud
// t.Fatal-equivalent, insertRawBookmarkAt, testUUID) as closely as a
// Node/pg client can, per the UI Contract's "Playwright backend choice."
// Not a spec file itself — imported by web/e2e/*.spec.ts.
import { Pool } from 'pg';

const dsn = process.env.TEST_DATABASE_URL;
if (!dsn) {
  throw new Error(
    'TEST_DATABASE_URL is not set — e2e tests require an explicit test database, mirroring ' +
      "tests/integration/postgres/setup_test.go's testDSN (loud failure, never a silent skip).",
  );
}

export const pool = new Pool({ connectionString: dsn });

/** Resets table state between tests — same role as the Go suite's
 * truncateBookmarks, called from each spec's test.beforeEach. */
export async function resetBookmarks(): Promise<void> {
  await pool.query('TRUNCATE TABLE bookmarks');
}

/** Deterministic, validly-shaped UUID string for fixture rows — mirrors
 * setup_test.go's testUUID (bookmarks.id is a native Postgres uuid
 * column, so fixture ids must parse as UUIDs, not arbitrary strings). */
export function testUUID(n: number): string {
  return `00000000-0000-0000-0000-${String(n).padStart(12, '0')}`;
}

export interface FixtureBookmark {
  id: string;
  status: 'inbox' | 'reading' | 'done';
  /** Position is a plain sortable string in these raw fixture rows (e.g.
   * 'b' < 'c' < 'd') — the real fractional-rank algorithm lives behind
   * Repository.Create/Move and is irrelevant to a raw INSERT bypassing
   * both, same rationale as setup_test.go's insertRawBookmarkAt.
   *
   * MUST NOT be 'a' (H-31 / H-17): 'a' is the rank floor in
   * internal/adapter/postgres/rank.go's base-26 scheme — a value
   * production's real Create/Move never emit for an existing row, since
   * the algorithm always leaves room below whatever it hands out. A raw
   * fixture row seeded at 'a' front-drops the real adapter into
   * midpoint("", "a") on any Move that inserts before it, which the
   * algorithm can't subdivide and panics on (recovered as a 500 by chi's
   * Recoverer, but still a failed request) — this is what caused the
   * intermittent e2e flake tracked as H-17. Start raw fixture positions
   * at 'b' or later so there's always synthesizable room below the
   * front-most row, the same room a real Create/Move-populated column
   * would have. */
  position: string;
  title?: string;
  url?: string;
  /** Only meaningful (and only set) when status === 'done' — mirrors the
   * FinishedAt <-> Done invariant so Done fixture rows look like
   * realistic pre-existing data, even though a raw INSERT doesn't itself
   * enforce the invariant the way Repository.Move does. */
  finishedAt?: string;
}

/** Bypasses Repository.Create/Move entirely, same rationale as
 * setup_test.go's insertRawBookmarkAt: e2e drag specs need direct control
 * over status/position fixture rows independent of the Create/Move code
 * paths under test. */
export async function insertBookmark(bm: FixtureBookmark): Promise<void> {
  const url = bm.url ?? `https://example.com/${bm.id}`;
  await pool.query(
    `INSERT INTO bookmarks
       (id, original_url, canonical_url, identity_hash, title, status, position, finished_at, created_at, updated_at)
     VALUES ($1, $2, $2, $3, $4, $5, $6, $7, now(), now())`,
    [bm.id, url, bm.id, bm.title ?? 'Fixture bookmark', bm.status, bm.position, bm.status === 'done' ? (bm.finishedAt ?? new Date().toISOString()) : null],
  );
}

// Every e2e spec file's own test.afterAll calls this. Under this
// project's playwright.config.ts (workers: 1, fullyParallel: false),
// every spec file runs inside ONE shared worker process and therefore
// shares this exact module instance — including `pool` — across files
// (Node's module cache is per-process, not per-file; Playwright does not
// reset it between files run by the same worker). Actually calling
// pool.end() here, as this function originally did, ends the pool for
// good the moment the FIRST spec file finishes, leaving it unusable for
// every file that runs after it ("Cannot use a pool after calling end on
// the pool" surfacing in a later file's beforeEach — observed empirically
// running the full suite together, though never running any single spec
// file alone). There is no reliable "this is the last file" signal
// available from inside this module, so this intentionally no-ops
// instead: the worker process's own exit (after every spec file
// completes) drops the pool's TCP connections, which Postgres handles
// the same as any other ungraceful client disconnect — harmless for a
// throwaway test database. Do not carry this no-close pattern into
// production code.
export async function closePool(): Promise<void> {
  // Intentionally not calling pool.end() — see doc comment above.
}
