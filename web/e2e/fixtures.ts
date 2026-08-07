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
   * 'a' < 'b' < 'c') — the real fractional-rank algorithm lives behind
   * Repository.Create/Move and is irrelevant to a raw INSERT bypassing
   * both, same rationale as setup_test.go's insertRawBookmarkAt. */
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

export async function closePool(): Promise<void> {
  await pool.end();
}
