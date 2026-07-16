# CODEBASE_MAP.md — Trailhead

Mandatory Phase B deliverable (CLAUDE.md). Updated whenever a new package
or interface is added — a PR that adds one without updating this file
cannot merge.

---

## 1. Package Registry

| Package | Tier | Direct dependencies | What breaks if it changes |
|---|---|---|---|
| `internal/domain` | 1 | none (stdlib only) | Everything — every other package imports domain types. A signature change to `Bookmark`, `Status`, `Position`, etc. ripples into `adapter`, `adapter/postgres`, `api`, `testutil`, `cmd`, and `web`'s JSON contract. A rule change in `Canonicalize`/`DeriveIdentityHash` re-buckets every stored bookmark's duplicate-detection identity (see DECISIONS.md — Locked). |
| `internal/adapter` (`ports.go`) | 1 | `internal/domain` | Every caller of `BookmarkRepository` (`internal/api`, `cmd/trailhead` at wiring time) and every implementor (`internal/adapter/postgres`, `internal/testutil`). Locked after Phase B gate — changes require a filed RFC. |
| `internal/adapter/postgres` | 1 | `internal/adapter`, `internal/domain` | The running application's actual data — this is the only package that writes to PostgreSQL. Built (issue #2): `Create` + `Board` + `RunMigrations` + `IsDuplicateConstraintViolation`; `Move`/`Update`/`Delete` pending issues #4/#5. |
| `internal/api` | 2 | `internal/adapter`, `internal/domain`, `github.com/go-chi/chi/v5` | The HTTP contract `web/` depends on — a handler bug produces a wrong status code or malformed JSON, but does not touch storage directly. Built (issue #3): `NewRouter(repo adapter.BookmarkRepository) http.Handler` (`router.go`) is the sole exported entry point, wiring `POST /api/bookmarks` + `GET /api/board` (`handlers.go`) and the JSON error envelope (`response.go`), plus custom `not_found`/`method_not_allowed` handlers. `Move`/`Update`/`Delete` routes pending issues #5+. |
| `internal/testutil` | 3 | `internal/adapter`, `internal/domain` | Only test correctness — Pre-Phase F test-engine sessions and any future `api`-layer unit tests depend on `FakeBookmarkRepository` behaving like the real repository's documented contract. Never imported by production code. Its injectable clock defaults to UTC (fixed at issue #3 Pre-Phase F, 2026-07-09) to mirror the real adapter's contract. |
| `cmd/trailhead` | 3 | all packages (wiring only) | The running binary's startup behavior — if this breaks, the app doesn't start, but no other package's correctness is affected. |
| `web/` | 2 | none (Go side) — communicates with `internal/api` over REST/JSON only | The user-facing board experience — the brief's "what good looks like" bar (drag-and-drop feel, empty/loading/error states). Not yet built (Phase E/F). |
| `tests/integration/` | 3 | `internal/adapter/postgres` (via build tag `integration`) | Only CI confidence that the Postgres adapter works against a real database — no production impact. Built (issue #2): the `tests/integration/postgres` suite (Create/Board + migrations), run in CI by `.github/workflows/integration.yml` against a real Postgres service. |

---

## 2. Interface Registry

| Interface | Defined in | Implementors | Consumers |
|---|---|---|---|
| `BookmarkRepository` | `internal/adapter/ports.go` | `internal/adapter/postgres.Repository` (built issue #2: `Create` + `Board`; `Move`/`Update`/`Delete` pending #4/#5); `internal/testutil.FakeBookmarkRepository` (built) | `internal/api` handlers (built issue #3 — `handlers.go`'s `createBookmark`/`board` methods on `handler`, wired via `NewRouter`); `cmd/trailhead/main.go` at wiring time (currently a TODO — no repository wired yet); any test file importing `internal/testutil` |

Only one Interface exists in this project — Trailhead has a single storage
engine (PostgreSQL) and a single external-facing contract (the
`BookmarkRepository` port). There is no cross-engine adapter comparison to
make, unlike multi-engine workshop projects (Mongo/Elastic/Redis/Cassandra).

---

## 3. Blast Radius Quick Reference

"If I change X, what breaks?" — read top to bottom, most consequential first.

- **Change a `domain.Bookmark` field (add/remove/rename/retype):** breaks
  `internal/adapter/ports.go` signatures that reference it, the Postgres
  schema (`ARCHITECTURE_RFC.md` "Persistence Schema"), the JSON API
  contract (`ARCHITECTURE_RFC.md` "Serialization Spec"), `web/`'s
  TypeScript types, and every test fixture in `internal/testutil` and
  future `tests/integration/`. Requires a filed RFC per the Locked
  Interface rule, plus a DECISIONS.md amendment if the field is one of the
  Locked ones (`FinishedAt`, `Author`, `Position`, `CanonicalURL`,
  `IdentityHash`).
- **Change the `Canonicalize` rule set (scheme, tracking-param deny-list,
  trailing-slash/www, fragment handling):** re-buckets every existing
  bookmark's `IdentityHash` — bookmarks that were previously distinct may
  become "duplicates" of each other or vice versa. Locked in DECISIONS.md;
  changing it requires a deliberate re-decision logged there, not a
  silent code change.
- **Change the `BookmarkRepository` interface (`internal/adapter/ports.go`):**
  breaks every implementor (`postgres.Repository`,
  `testutil.FakeBookmarkRepository`) and every consumer (`internal/api`
  handlers, `cmd/trailhead` wiring). READ-ONLY after the Phase B gate —
  requires a filed RFC via improve-codebase-architecture.
- **Change the Postgres schema (`bookmarks` table):** breaks
  `internal/adapter/postgres` (once built) and requires a migration;
  does not directly break `internal/domain`, `internal/api`, or `web/` as
  long as the Go-level `BookmarkRepository` contract is preserved across
  the schema change.
- **Change `internal/api` handler behavior:** breaks `web/`'s expectations
  of the REST/JSON contract (status codes, response shapes) — does not
  touch storage or the repository interface itself.
- **Change `FakeBookmarkRepository`'s internal ordering/position
  bookkeeping:** breaks any test asserting on the fake's exact `Position`
  string values (a testing anti-pattern this project should avoid per the
  fake's own doc comment — tests should assert relative order and
  round-trip behavior, not the fake's internal string format). Does not
  affect production code, since `testutil` is never imported there.
- **Change `cmd/trailhead/main.go` wiring:** breaks only the running
  binary's startup — no other package's correctness is affected.
- **Change `web/`:** breaks only the user-facing experience — no backend
  contract is affected as long as `web/` continues to speak the locked
  REST/JSON shape.

---

## Update Log

- 2026-07-04 — Initial version, Phase B gate. All packages tiered, single
  Interface (`BookmarkRepository`) registered, greenfield skeleton
  materialized (`internal/domain`, `internal/adapter/ports.go`,
  `internal/testutil`, `cmd/trailhead/main.go`).
- 2026-07-13 (backfilled — see note below) — Issue #2 merged
  (`eef6d7a`): `internal/adapter/postgres.Repository` built, `Create` +
  `Board` + `RunMigrations` + `IsDuplicateConstraintViolation` real.
  `tests/integration/postgres/` suite added (build tag `integration`).
- 2026-07-13 (backfilled — see note below) — Issue #3 merged (`b35b9ba`):
  `internal/api` built. `NewRouter(repo adapter.BookmarkRepository)
  http.Handler` (`router.go`) is the sole exported entry point;
  `handlers.go`/`response.go` implement `POST /api/bookmarks` +
  `GET /api/board` and the JSON error envelope. `internal/testutil`'s
  fake clock changed to default UTC in the same window (fidelity fix,
  not a new package/interface).

**Note (added at issue #4 Pre-Phase F, 2026-07-13):** the #2 and #3
entries above were backfilled now, not appended at merge time — this
file's inline table cells were opportunistically kept current by each
Dispatch session, but the Update Log itself (the audit trail CLAUDE.md's
Phase B mandate actually requires — "Updated whenever a new package or
interface is added... a PR that adds one without updating this file
cannot merge") was not appended to at either merge. Caught via a
disk cross-check at #4's Pre-Phase F context loading, not by the #3 PR
review. Logged as a process gap: the merge-blocking check needs to name
the Update Log section specifically, not just "the file," since the
table cells alone can look current while the audit trail silently lapses.
