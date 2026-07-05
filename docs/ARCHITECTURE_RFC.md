# ARCHITECTURE_RFC.md — Trailhead

**Phase:** B
**Status:** Locked (pending Phase B gate: reviewer-agent pass + different-model review)
**Date:** 2026-07-04

Locked Vocabulary in force throughout this document (per
improve-codebase-architecture): **Module**, **Interface**, **Implementation**,
**Depth**, **Seam**, **Adapter**, **Leverage**, **Locality**. No synonyms
("component", "service", "boundary") are used.

---

## Scope Boundary

Per CLAUDE.md Phase B: this RFC LOCKS Modules, Interfaces (ports), the
persistence schema, and architectural decisions. It DEFERS per-issue
implementation detail — function bodies, internal helpers, concrete
algorithms — to per-issue Pre-Phase F prep. Two deliberate exceptions,
justified below because their exact rules are already fully pinned in
DECISIONS.md (not open architecture questions): `domain.Canonicalize`,
`domain.DeriveIdentityHash`, and `domain.DefaultTitle` are implemented now,
not stubbed, because DECISIONS.md already specifies their rules
completely and the brief calls for them to be golden-testable pure
functions from the start. The fractional-rank position algorithm is
explicitly NOT implemented yet — its concrete form is a genuine open
architecture question, deferred to the Postgres adapter's Phase F issue.

Two boundary notes added at the Phase B gate fix (2026-07-05), reconciling
apparent locked-vs-open tension Codex's round-2 review flagged:

- The canonical-URL tracking-param deny-list *rule* (strip known
  trackers, sort the remainder) is Locked and implemented now, same as
  the other `Canonicalize` sub-rules. The current 12-entry list is real
  and in force, not a placeholder. Golden-testing that list's behavior is
  legitimate Pre-Phase F work; *extending* the list is a change to a
  Locked decision requiring a DECISIONS.md amendment. See DECISIONS.md
  "Canonical URL — Query Parameters" and docs/PRD.md "Open Questions."
- Exact `internal/api` HTTP routes, request bodies, and response bodies
  are explicitly Pre-Phase F implementation detail, not an open
  architecture question this RFC needs to resolve — the Serialization
  Spec below locks the wire-level conventions (timestamp format,
  null-vs-missing, ID representation) that any route must follow; the
  routes themselves are specified per-issue via the REST Adapter Wire
  Contract gate before that issue reaches Dispatch.

---

## Package Organization

```
trailhead/
├── cmd/trailhead/           main.go — wiring only, entry point
├── internal/domain/         types + Canonicalize/DeriveIdentityHash/DefaultTitle
├── internal/adapter/        ports.go — BookmarkRepository (Locked Interface)
│   └── postgres/            Postgres implementation — Phase F, not yet built
├── internal/api/             chi handlers — Phase F, not yet built
├── internal/testutil/        FakeBookmarkRepository — built at Phase B gate
├── web/                       React/TS/Tailwind SPA — Phase E/F, not yet built
└── tests/
    └── integration/          end-to-end tests against a real Postgres, //go:build integration
```

### Import Direction Rules (Locked)

```
adapter    -> domain                    (allowed — ports.go references domain types)
adapter/postgres -> adapter, domain     (allowed — implements the port)
api        -> adapter, domain           (allowed — depends on the interface, not the implementation)
testutil   -> adapter, domain           (allowed — test helpers only)
cmd        -> all packages              (allowed — entry point only, sole wiring site)
```

Forbidden (any of these is a P1 merge blocker):

```
domain          -> adapter              (domain must not know about ports)
domain          -> adapter/postgres     (domain must not know about persistence)
domain          -> api                  (domain must not know about HTTP)
adapter         -> adapter/postgres     (the ports file must not import its own implementor)
api             -> adapter/postgres     (api must depend on the interface only, wired via cmd)
testutil        -> (nothing imports testutil in production code)
```

The `api -> adapter/postgres` prohibition is why `BookmarkRepository` lives
in `internal/adapter`, not in `internal/adapter/postgres`: if the interface
lived in the implementor's own package, `internal/api` would have to import
the Postgres-specific package just to get the interface type, defeating the
Seam the interface exists to create (go-patterns "Interface Location").

### Module Responsibilities

- `internal/domain`: types only, plus the three pure functions named in
  Scope Boundary above. No I/O. No imports outside the standard library —
  this is the Depth-maximizing choice: every other Module can depend on
  domain without inheriting any external dependency.
- `internal/adapter`: the `BookmarkRepository` Interface, `BoardFilter`,
  `MoveCommand`, `RepositoryError`, `ErrorKind`. Imports domain only.
- `internal/adapter/postgres`: the Postgres Implementation of
  `BookmarkRepository`. Imports adapter, domain. Not yet built (Phase F).
- `internal/api`: chi handlers translating HTTP requests into
  `BookmarkRepository` calls and `RepositoryError.Kind` into HTTP status
  codes (409 for `ErrKindDuplicate`, 404 for `ErrKindNotFound`, 400 for
  `ErrKindInvalidURL`). Imports adapter, domain. Not yet built (Phase F).
- `internal/testutil`: `FakeBookmarkRepository`. Imports adapter, domain.
  Test helpers only — never imported by production code. Built now (this
  RFC), per the Phase B gate requirement that fakes exist before close.
- `cmd/trailhead`: wiring and the HTTP server entry point. Imports all
  packages needed at wire-time. Sole producer of the running binary.
- `web/`: React/TS/Tailwind SPA, client-side only, communicates with
  `internal/api` over REST/JSON. Not yet built (Phase E/F).

---

## ID Type and Representation

**Decision:** `domain.BookmarkID` is a plain Go `string` holding a UUID's
canonical text form, backed by a native Postgres `uuid` column
(`gen_random_uuid()` default, via the `pgcrypto` extension).
**Reason:** Avoids exposing a sequential count of bookmarks through the ID
(an auto-increment integer would). Keeping the Go type a plain `string`
rather than wrapping a third-party `uuid.UUID` type keeps `internal/domain`
free of any non-standard-library dependency, preserving its maximal Depth
and Locality.
**Decided by:** Recommendation accepted (Phase B, 2026-07-04)
**Locked:** yes

## Status — Postgres Representation

**Decision:** `status` is a Postgres `text` column with a `CHECK` constraint
(`CHECK (status IN ('inbox', 'reading', 'done'))`), not a native Postgres
`enum` type.
**Reason:** Postgres native enums are painful to alter (adding a value is
fine, but reordering or removing one requires a full type rebuild). A
`CHECK` constraint gives the same guarantee with a trivial migration path if
the three fixed columns ever needed to change — even though DECISIONS.md
locks the three columns as fixed, the migration-friction asymmetry between
the two representations makes `text` + `CHECK` the lower-risk default for
no cost today.
**Decided by:** Recommendation accepted (Phase B, 2026-07-04)
**Locked:** no — an internal schema choice, not called out as
brief-critical; open to revisit if a compelling reason surfaces.

## Postgres Driver

**Decision:** `github.com/jackc/pgx/v5`, used directly (not via
`database/sql` + `lib/pq`).
**Reason:** `pgx` has first-class support for Postgres-native types
(`uuid`, `jsonb`) without extra marshaling shims, is the more actively
maintained of the two common choices as of this project's start, and its
connection-pool (`pgxpool`) is the standard recommendation for a
single-binary Go service talking to Postgres.
**Decided by:** Recommendation accepted (Phase B, 2026-07-04)
**Locked:** no — an implementation-level choice; changing it only affects
`internal/adapter/postgres`, nothing else.

---

## Locked Interfaces

`BookmarkRepository` is the Tier 1 contract — see RISK_TIER_REGISTER.md.
The full definition lives in `internal/adapter/ports.go` (READ-ONLY after
this gate; changes require a filed RFC). Reproduced here for the record —
this copy and the on-disk file must match exactly, verified at B3 below:

```go
type BoardFilter struct {
	Tags []string
}

type MoveCommand struct {
	ID           domain.BookmarkID
	TargetStatus domain.Status
	Before       *domain.BookmarkID
	After        *domain.BookmarkID
}

type ErrorKind string

const (
	ErrKindDuplicate  ErrorKind = "Duplicate"
	ErrKindNotFound   ErrorKind = "NotFound"
	ErrKindInvalidURL ErrorKind = "InvalidURL"
)

type RepositoryError struct {
	Kind     ErrorKind
	Existing *domain.Bookmark
	Message  string
	Wrapped  error
}

func (e *RepositoryError) Error() string { return e.Message }
func (e *RepositoryError) Unwrap() error { return e.Wrapped }

type BookmarkRepository interface {
	Create(ctx context.Context, b domain.NewBookmark) (domain.Bookmark, error)
	Board(ctx context.Context, filter BoardFilter) (domain.Board, error)
	Move(ctx context.Context, cmd MoveCommand) (domain.Bookmark, error)
	Update(ctx context.Context, id domain.BookmarkID, patch domain.BookmarkPatch) (domain.Bookmark, error)
	Delete(ctx context.Context, id domain.BookmarkID) error
}
```

Design chosen: **Flexible** (from design-an-interface's four competing
designs), with Ports-and-Adapters' typed-sentinel-error convention folded
in. `BoardFilter` and `MoveCommand` absorb future fields without breaking
the method signature; `RepositoryError.Kind` lets `internal/api` use
`errors.As` and switch on `Kind` rather than string-matching error
messages. Approved by Ali, 2026-07-04 (see Phase B interview log below).
Full four-design comparison, testability assessment, and context
completeness check are in the Phase B chat record — reproduced in summary
in RISK_TIER_REGISTER.md's rationale column where relevant.

---

## Data Flow

```
Browser (web/ SPA)
   │  REST/JSON over HTTP
   ▼
internal/api (chi handlers)
   │  calls BookmarkRepository methods
   ▼
internal/adapter.BookmarkRepository  (Interface — the Seam)
   │
   ├── production: internal/adapter/postgres.Repository  ── SQL ──▶ PostgreSQL (bookmarks table)
   └── tests:       internal/testutil.FakeBookmarkRepository ── in-memory map
```

Every write (`Create`, `Move`, `Update`, `Delete`) flows through the single
`BookmarkRepository` Seam — there is no code path that writes to Postgres
except through this Interface, and no code path that lets `internal/api`
see a Postgres-specific type (`pgx.Rows`, `pgtype.UUID`, etc.). This is the
Ports-and-Adapters property carried over from the interface design.

---

## Persistence Schema

```sql
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE bookmarks (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    original_url   text NOT NULL,
    canonical_url  text NOT NULL,
    identity_hash  text NOT NULL,
    title          text NOT NULL,
    tags           jsonb NOT NULL DEFAULT '[]'::jsonb,
    status         text NOT NULL CHECK (status IN ('inbox', 'reading', 'done')),
    position       text NOT NULL,
    finished_at    timestamptz,          -- NULL unless status = 'done' (app-enforced invariant)
    author         text,                  -- NULL unless user has set one
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX bookmarks_identity_hash_idx ON bookmarks (identity_hash);
CREATE INDEX bookmarks_status_position_idx ON bookmarks (status, position);
CREATE INDEX bookmarks_tags_gin_idx ON bookmarks USING gin (tags);
```

Notes:
- `identity_hash` has a unique index — this is what makes duplicate
  detection an atomic, race-safe database-level guarantee, not just an
  application-level check-then-insert with a TOCTOU gap. `Create`'s
  implementation (Phase F) should catch the unique-violation error and
  translate it into `RepositoryError{Kind: ErrKindDuplicate}` by
  re-querying the existing row, rather than relying solely on a
  check-before-insert.
- `(status, position)` composite index supports the Board query's
  `ORDER BY status, position` access pattern directly.
- `tags` GIN index supports the `?|` (any-of) Postgres JSONB operator,
  which maps directly onto the locked OR-semantics tag filter.
- `finished_at ⟺ status = 'done'` is an **application-enforced invariant,
  not a database CHECK constraint** — expressing "column A is non-null iff
  column B equals a specific value" as a portable CHECK constraint is
  possible but adds real complexity (`CHECK ((status = 'done') = (finished_at IS NOT NULL))`)
  for a single-user local tool where the repository implementation is the
  only writer. Documented here as a conscious choice, not an oversight — a
  DB-level CHECK is a reasonable Phase F upgrade if this invariant ever
  needs a second line of defense.
- **Position uniqueness within its Status (DECISIONS.md "Position /
  Ordering Representation," Locked) is likewise an application-enforced
  invariant, not a database constraint.** `bookmarks_status_position_idx`
  is a plain (non-unique) index supporting the `ORDER BY status, position`
  access pattern only — it does not enforce uniqueness. The Postgres
  adapter's `Move` implementation (Phase F) is the sole writer of
  `position` values and is responsible for computing a new fractional
  rank that sorts strictly between its target neighbors, which by
  construction never collides with an existing value in that `Status`
  under single-writer conditions. This project's Concurrency decision
  (DECISIONS.md "Concurrency / Reorder-Status Conflict Handling" —
  last-write-wins, no optimistic locking) means a genuine collision is
  already an accepted, out-of-scope risk in the rare concurrent-tab case,
  same as any other last-write-wins field. A DB-level `UNIQUE
  (status, position)` constraint is a reasonable Phase F hardening step
  if this invariant ever needs a second line of defense — same category
  of choice as the `finished_at` invariant above. (This reconciles
  Codex's round-2 Phase B review finding that the schema did not encode
  or document this guarantee.)

---

## Serialization Spec

Required for any project that hashes, persists, or transmits domain
objects (CLAUDE.md Phase B gate condition).

**On-disk (Postgres):**
- `tags` — Postgres `jsonb`, a JSON array of lowercase strings, e.g.
  `["reading-list", "go"]`. Empty tag set stored as `[]`, never `null`.
- `position` — stored as `text`; the concrete fractional-rank string
  format is Phase F implementation detail (Scope Boundary above).
- `finished_at`, `author` — nullable columns; absence is a real SQL
  `NULL`, never an empty string or epoch-zero timestamp.

**On-the-wire (REST/JSON API):**
- Timestamps (`created_at`, `updated_at`, `finished_at`): RFC 3339 /
  ISO 8601 with UTC offset, e.g. `"2026-07-04T18:30:00Z"`. Go's
  `time.Time` JSON-marshals to this format by default via
  `encoding/json`. `CreatedAt` and `UpdatedAt` are `domain.Bookmark`
  fields (`internal/domain/bookmark.go`), always present, never optional
  — unlike `FinishedAt` they are ordinary `time.Time`, not `*time.Time`.
  Both are API-visible per this spec (resolved at the Phase B gate fix,
  2026-07-05 — previously ambiguous per Codex's round-2 review; see also
  docs/GLOSSARY.md's `FinishedAt` entry).
- `finished_at`, `author`: Go pointer types (`*time.Time`, `*string`)
  marshal to JSON `null` when nil, and are **always included in the
  response body** (no `omitempty`) — an absent key would be
  indistinguishable from a key the client forgot to check, whereas an
  explicit `null` is an unambiguous signal. This mirrors the
  null-vs-missing contract documented in go-patterns for exactly this
  reason.
- `tags`: JSON array of strings, always present (empty array `[]` when a
  bookmark has no tags, never `null`).
- `id`: JSON string, the UUID's canonical text form (matches
  `domain.BookmarkID`'s Go representation exactly — no transformation at
  the API boundary).

---

## tests/ Directory Structure

Per CLAUDE.md Phase B gate condition (Go projects use `internal/testutil/`
+ `tests/integration/`, not `conftest.py`):

- `internal/testutil/fake_repository.go` — `FakeBookmarkRepository`, built
  now (this RFC), satisfying the four adversarial invariants documented in
  its method doc comments (exact IdentityHash match, typed
  `*RepositoryError` never a plain error, context-cancellation checked
  before mutating state, nil-pointer optional fields never masqueraded as
  zero values).
- `tests/integration/` — end-to-end tests against a real Postgres
  instance, gated behind `//go:build integration`, exercising the actual
  `internal/adapter/postgres.Repository` (not the fake). Not yet
  populated — Phase F deliverable, alongside the Postgres adapter itself.
- Unit tests live alongside their package as idiomatic Go `*_test.go`
  files (e.g. `internal/domain/canonicalize_test.go`), not under a
  separate `tests/unit/` directory — this is the standard Go convention
  and is more Discoverable (Locality) than centralizing unit tests apart
  from the code they test. This is a deliberate deviation from a
  Python-project-style `tests/unit/` layout, noted explicitly per CLAUDE.md
  Phase B gate wording ("the minimum structure is tests/unit,
  tests/integration, conftest.py" — the Go equivalent substitutes
  co-located `*_test.go` files for `tests/unit/` and `internal/testutil/`
  for `conftest.py`, matching the precedent set by
  go-workflow-observations.md Finding 3).

---

## B3 Verification

Performed 2026-07-04 against the on-disk skeleton (Greenfield Skeleton
below), per CLAUDE.md Phase B gate:

- [x] Every port interface method signature in `internal/adapter/ports.go`
      matches the "Locked Interfaces" block above exactly (read back from
      disk and diffed by eye, method-by-method: `Create`, `Board`, `Move`,
      `Update`, `Delete` — all five match, including parameter names and
      order).
- [x] All imports in `ports.go` are correct: `context` and
      `trailhead/internal/domain` only, both referenced (no unused
      imports — `domain.BookmarkID`, `domain.Bookmark`, `domain.Status`,
      `domain.NewBookmark`, `domain.Board`, `domain.BookmarkPatch` are all
      used in the file).
- [x] Data flow diagram matches port signatures: `internal/api` (not yet
      built) will call the five `BookmarkRepository` methods shown in the
      diagram; no method in the diagram is absent from `ports.go` and vice
      versa.
- [x] Doc comments in `ports.go` reference the correct parameter names —
      `cmd.TargetStatus`, `cmd.ID`, `patch`, `filter.Tags`, `b.OriginalURL`
      all checked against the actual parameter names in each method
      signature.
- [x] Optional/absent-on-real-data fields decided as Go pointer types:
      `domain.Bookmark.FinishedAt *time.Time`, `domain.Bookmark.Author
      *string`, `domain.BookmarkPatch.Title *string`, `.Tags *domain.Tags`,
      `.Author *string`, `MoveCommand.Before *domain.BookmarkID`, `.After
      *domain.BookmarkID` — all pointer types, verified by reading
      `internal/domain/bookmark.go` and `internal/adapter/ports.go` back
      from disk.

No mismatches found. B3 verification: **PASS**.

**B3 Addendum (2026-07-05, Phase B gate fix batch):** Codex's round-2
Phase B review (2026-07-04 packet) returned FAIL with six substantive
cross-document findings (Board glossary staleness, Move stale-neighbor
fallback undocumented in `ports.go`, Position uniqueness scope
disagreement + no schema-level enforcement or documented contract,
Author editing omitted from PRD.md, Title Defaulting example
contradicting its own stated rule, UpdatedAt status ambiguous) plus a
newly-ratified `BookmarkPatch.ClearAuthor` mechanism. All were applied in
lockstep across DECISIONS.md, docs/GLOSSARY.md, docs/PRD.md, this file,
`internal/adapter/ports.go`, `internal/domain/bookmark.go`, and
`internal/testutil/fake_repository.go`. The `ClearAuthor bool` addition
to `BookmarkPatch` does not change any `BookmarkRepository` method
signature — only a field on an existing parameter type and doc comments
— so the B3 signature-match check above is unaffected and does not need
re-verification against `ports.go`. Two items (deny-list exhaustive
contents, exact API routes/request/response shapes) were classified as
legitimate Pre-Phase F deferrals and given boundary notes rather than
specced now, per the Scope Boundary above.

---

## Greenfield Skeleton

Materialized on disk before this gate closes (new repo, no prior code to
build on):

- `go.mod` — module `trailhead`, Go 1.22, requires `go-chi/chi/v5` and
  `jackc/pgx/v5` (versions pinned; `go.sum` not yet generated — see Terminal
  commands at the end of the Phase B gate summary).
- `internal/domain/bookmark.go` — all types from GLOSSARY.md.
- `internal/domain/canonicalize.go` — `Canonicalize`, `DeriveIdentityHash`,
  `DefaultTitle` (implemented per Scope Boundary exception above).
- `internal/adapter/ports.go` — `BookmarkRepository` and supporting types,
  READ-ONLY header comment in place.
- `internal/testutil/fake_repository.go` — `FakeBookmarkRepository`,
  compile-time interface-compliance assertion included
  (`var _ adapter.BookmarkRepository = (*FakeBookmarkRepository)(nil)`).
- `cmd/trailhead/main.go` — minimal compiling entry point (health check
  only; Postgres/API wiring deferred to Phase F per the file's own doc
  comment).

`go build ./...` and `go vet ./...` must be run and confirmed clean in
Terminal on the developer's Mac before this gate is declared closed (Go
Toolchain Note — the Cowork sandbox has no Go toolchain installed). Exact
commands are provided in the Phase B gate summary message.
