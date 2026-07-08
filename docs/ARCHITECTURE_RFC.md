# ARCHITECTURE_RFC.md — Trailhead

**Phase:** B
**Status:** Locked — Phase B gate CLOSED (2026-07-06). Reviewer-agent pass and different-model (Codex) review both completed across five rounds; round 5 returned FAIL with four fixable findings and two accepted risks, all resolved per the B3 Addendum, round 5 below. Closed by developer decision on round 5, without a further Codex re-submission — see that addendum for the developer's explicit reasoning.
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

**Accepted risk, not fixed at this gate (Phase B gate round 5, 2026-07-06
— Codex finding E1):** `BookmarkRepository.Move`'s contract defines
behavior for an unresolved *neighbor* (`Before`/`After`) but not for an
invalid `cmd.TargetStatus` — a `domain.Status` value outside
`StatusInbox`/`StatusReading`/`StatusDone`. Since `Status` is a plain
string-backed type (not a closed Go enum the compiler can enforce), a
caller could construct one. Two places could own rejecting this: `Move`
itself (returning a new classified error), or `internal/api` validating
`TargetStatus` against the three known constants before ever calling
`Move`. This RFC does not pick one now — deciding *where* input
validation for `internal/api` requests lives is Pre-Phase F Wire Contract
scope (same category as exact routes/payloads above), not a Phase B
architecture question, since it doesn't change the locked
`BookmarkRepository` interface either way. Flagged here so it isn't lost:
each `internal/api` issue that constructs a `MoveCommand` from a request
body must explicitly decide and document this validation ownership in
its Wire Contract section before reaching Dispatch.

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

**Error taxonomy (resolved Phase B gate round 3, 2026-07-05):**
`RepositoryError`/`ErrorKind` classify only failures the API layer must
map to a *distinct* HTTP status: `Duplicate` (409), `NotFound` (404),
`InvalidURL` (400). Infrastructure failures (Postgres unreachable,
network errors, context cancellation/timeout) are deliberately NOT a
fourth `ErrorKind` — every infrastructure failure maps to the same
undifferentiated 5xx, so there is no status-relevant distinction for a
Kind to carry. `BookmarkRepository` methods return these as a plain
wrapped `error` instead; `internal/api` distinguishes the two cases with
`errors.As(err, &repoErr)` (success → classified 4xx via `Kind`; failure
→ uniform 5xx). See DECISIONS.md "Repository Error Taxonomy —
Infrastructure Failures." This matches `FakeBookmarkRepository`'s
existing `checkContext`, which already returns `ctx.Err()` directly
rather than wrapping it in a `*RepositoryError`.

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
  or document this guarantee.) **Pre-Phase F boundary note (round 3,
  2026-07-05):** this RFC locks only that the invariant is application-
  enforced by the repository as sole writer — it does not lock a specific
  concurrent-write enforcement mechanism (e.g. transaction isolation
  level, `SELECT ... FOR UPDATE`, advisory locks, retry-on-conflict). The
  exact mechanism, if one is added beyond the accepted last-write-wins
  risk, is Postgres adapter implementation detail for its Phase F issue,
  not a Phase B architecture question — consistent with the Scope
  Boundary above.

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
- `domain.Board`'s three fields are locked to lowercase JSON keys —
  `inbox`, `reading`, `done` — via struct tags on `internal/domain/bookmark.go`'s
  `Board` type (`json:"inbox"`, `json:"reading"`, `json:"done"`). Resolved
  at the Phase B gate round 3 (2026-07-05): GLOSSARY.md already asserted
  this casing, but the struct itself carried no tags, so default Go
  marshaling would have emitted the capitalized field names (`Inbox`,
  `Reading`, `Done`) instead (Codex round-3 finding A2).
- `updated_at` write-path contract: `Create`, `Move`, and `Update` each
  set the returned Bookmark's `UpdatedAt` to the current time on every
  successful call, regardless of whether any field's value actually
  changed. On `Create` specifically, `UpdatedAt` is set equal to
  `CreatedAt` — both timestamp the same creation instant, not two
  independently-derived "current time" values — matching
  `FakeBookmarkRepository.Create`'s existing `now := f.now()` /
  `CreatedAt: now, UpdatedAt: now` implementation. `Delete` removes the
  row entirely, so no `UpdatedAt` semantics apply. See DECISIONS.md
  "UpdatedAt — Write-Path Contract" (resolved Phase B gate round 3,
  2026-07-05 — the schema and this spec already established `updated_at`
  as always-present and API-visible, but no document previously stated
  which write paths actually update it; the Create-equals-CreatedAt
  equality requirement was added at Phase B gate round 5, 2026-07-06 —
  DECISIONS.md already stated it, but this spec and `ports.go` only said
  "current time," omitting the equality — Codex round-5 finding A1).

---

## tests/ Directory Structure

Per CLAUDE.md Phase B gate condition (Go projects use `internal/testutil/`
+ `tests/integration/`, not `conftest.py`):

- `internal/testutil/fake_repository.go` — `FakeBookmarkRepository`, built
  now (this RFC), satisfying the four adversarial invariants documented in
  its method doc comments (exact IdentityHash match; typed
  `*RepositoryError` — never an untyped string error — for every
  *classified* failure, i.e. `Duplicate`/`NotFound`/`InvalidURL`;
  context-cancellation checked before mutating state; nil-pointer
  optional fields never masqueraded as zero values). Invariant 2's
  "typed error" guarantee applies to classified failures specifically —
  it does NOT mean every failure path returns `*RepositoryError`.
  Infrastructure failures (context cancellation, and in the eventual
  Postgres adapter, connection/network errors) are returned as a plain
  wrapped `error` per DECISIONS.md "Repository Error Taxonomy —
  Infrastructure Failures," exactly as `checkContext` already does below.
  (Phase B gate round 5 fix, 2026-07-06 — the previous shorthand "typed
  `*RepositoryError` never a plain error" read as a blanket claim that
  appeared to contradict the infra-taxonomy carve-out elsewhere in this
  same file; narrowed to state Invariant 2's actual scope — Codex
  round-5 finding B1.)
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

**B3 Addendum, round 3 (2026-07-05):** Codex's round-3 review returned
FAIL with no locked-decision violations and no missing gate artifacts —
every finding was under-specification or an edge case. Four were lock-
fixes: (A1) infra-error taxonomy — documented that infrastructure
failures are a plain wrapped error, not a fourth `ErrorKind`; (A2) Board
JSON casing — added `json:"inbox"`/`"reading"`/`"done"` tags to
`domain.Board`, previously asserted in GLOSSARY.md but not actually
encoded; (E1) malformed-neighbor Move — generalized the stale-neighbor
rule to one rule covering missing, cross-status, self-referential, and
otherwise inconsistent neighbors, discovering that
`FakeBookmarkRepository.Move`'s existing implementation already handles
all four cases correctly by construction (searches `targetOrder`, falls
back to end-of-column whenever no match is found); (E3) `updated_at`
write-path contract — documented that `Create`/`Move`/`Update` all set it
on every successful call, matching the fake's already-built behavior.
One item (E2, position uniqueness under concurrent writes) was a
Pre-Phase F boundary note only — locking that the invariant is
application-enforced without locking a specific concurrency-control
mechanism. None of the four lock-fixes required behavioral changes to
`internal/testutil/fake_repository.go` — A2 required one struct-tag
addition to `bookmark.go`; the rest were documentation of behavior the
fake already exhibited. `go build ./...` / `go vet ./...` confirmation
is provided in the Phase B gate round-3 evidence package (Terminal, per
the Go Toolchain Note — not run in this session).

**B3 Addendum, round 4 (2026-07-05):** Round-4 Codex review returned FAIL
again, and the developer explicitly directed an exhaustive sweep of the
entire skeleton against the locked contracts — not a reactive patch of
only the cited findings — to break the round-over-round pattern of new
implementation-vs-contract drift surfacing each time. Confirmed and fixed
all six cited findings: (A1) `domain.Bookmark` had no JSON struct tags at
all — added the full `id`/`original_url`/`canonical_url`/`identity_hash`/
`title`/`tags`/`status`/`position`/`finished_at`/`author`/`created_at`/
`updated_at` tag set, same class of gap as `domain.Board`'s round-3 fix;
(B1) the `Move` doc comment and DECISIONS.md claimed a Before/After-
disagreement case fell back to end-of-column, but
`FakeBookmarkRepository.Move`'s actual code only checks `Before` and
ignores `After` outright whenever `Before != nil` — narrowed both
documents to describe the real Before-takes-precedence tie-break instead
of adding new consistency-checking logic to the Tier 3 fake; (B2) the
`RepositoryError` struct's own doc comment still said "the sole error
type," contradicting the infra-taxonomy language added elsewhere in the
same file at round 3 — corrected; (B3) PRD.md's Affected Modules table
still listed a `tests/unit` directory that contradicts this RFC's locked
co-located-`*_test.go` convention — corrected to reference
`internal/testutil` and `tests/integration` only; (B4) this section's own
Greenfield Skeleton claims about `go.mod`/`go.sum` were checked against
the actual files on disk and found wrong on both counts — corrected
above; (C1) `FakeBookmarkRepository.nextID`'s `"fake-id-N"` format doesn't
match the locked UUID canonical-text-form ID representation — added a
doc-comment note (mirroring the existing Position precedent) that this is
a test-double simplification, not a claim about the production format.
The exhaustive sweep beyond the cited findings additionally found and
fixed three previously-uncited mechanical bugs, all instances of the same
underlying pattern (generic template values never customized for this
project, or a `test`/`tests` directory-name mismatch): the `Makefile`'s
`BINARY_NAME ?= app` and `MODULE ?= github.com/org/project` placeholders,
and its `test-int` target referencing the nonexistent singular
`./test/integration/...` path; `.github/workflows/integration.yml`'s
identical singular-path bug; and `.github/workflows/ci.yml`'s gosec step
using `-exclude-dir=test` (singular), which silently excluded nothing
since the actual directory is `tests/integration/` (plural). No behavioral
changes were made to `internal/testutil/fake_repository.go`'s `Move`
logic itself — every fix either added a struct tag, corrected doc-comment/
decision prose to match already-built behavior, or fixed a path/placeholder
string in a non-Go tooling file. `go build ./...` / `go vet ./...`
confirmation is provided in the Phase B gate round-4 evidence package
(Terminal, per the Go Toolchain Note — not run in this session).

**B3 Addendum, round 5 (2026-07-06) — gate closed:** Round-5 Codex review
returned FAIL with four substantive findings (A1, B1, B2, C1) and two
risks (E1, E2); no missing deliverables, no missing acceptance criteria,
and no `ports.go`-vs-RFC signature mismatch was found. The developer
reviewed the findings directly and made the closing call: fix all four
substantive findings, accept E1 and E2 as documented risks rather than
fixing them now, and close the gate on this round without a further
Codex re-submission. (A1) `Create`'s `UpdatedAt == CreatedAt` equality
requirement was already correctly stated in DECISIONS.md but missing from
`ports.go`'s interface-level and `Create`-specific doc comments and from
this file's Serialization Spec — added to both; no code change needed,
`FakeBookmarkRepository.Create` already sets both fields from the same
`now := f.now()` value. (B1) The tests/ Directory Structure bullet's
"typed `*RepositoryError` never a plain error" shorthand read as a
blanket claim in tension with the infra-error-taxonomy carve-out
documented elsewhere in this same file — narrowed to state Invariant 2's
actual scope (classified failures only). (B2) PRD.md's Error Conditions
and Open Questions wording had folded the round-4 Before/After
precedence tie-break into the same "falls back to end-of-column in every
case" sentence describing the three genuine fallback cases (missing/
stale, cross-status, self-referential) — split apart to match what
DECISIONS.md and `ports.go` actually specify since round 4. (C1) Unlike
round 4's treatment of this finding, round-5 Codex correctly rejected
"document it as a test-double simplification" as sufficient for the ID
*format* itself (as opposed to Position's fractional-rank *algorithm*,
which genuinely is Phase F-deferred implementation detail no test needs
to assume a specific string form for) — `FakeBookmarkRepository` is a
registered `adapter.BookmarkRepository` implementor, so an out-of-
contract ID format is a real Locked-Decision violation, not a cosmetic
convenience. Fixed in code: `nextID` now generates a random version-4
UUID in canonical text form via `crypto/rand` (no third-party dependency
added; `go.mod` is unaffected). (E1) invalid `MoveCommand.TargetStatus`
handling and (E2) `internal/api`'s Tier 2 depth against handler-level
mutation bugs were both reviewed and accepted as documented, non-blocking
risks rather than fixed now — see the boundary note in Scope Boundary
above and the accepted-risk note on `internal/api` in
RISK_TIER_REGISTER.md respectively. `go build ./...` / `go vet ./...`
confirmation is provided in the Phase B gate round-5 evidence package
(Terminal, per the Go Toolchain Note — not run in this session).

---

## Greenfield Skeleton

Materialized on disk before this gate closes (new repo, no prior code to
build on):

- `go.mod` — module `trailhead`, Go 1.25 (bumped from 1.22 at the Phase D
  CI-fix, 2026-07-07 — govulncheck flagged 21 stdlib vulnerabilities fixed
  by the version bump alone; `.github/workflows/ci.yml` and
  `integration.yml` both pin `go-version: '1.25'` to match), currently
  requires only
  `github.com/go-chi/chi/v5 v5.0.12` — the only import the on-disk
  skeleton actually has (`cmd/trailhead/main.go`'s chi router). `go.sum`
  already exists on disk with chi's hash entries. `jackc/pgx/v5` (see
  "Postgres Driver" above) is NOT yet a `go.mod` requirement — it will be
  added by `go get`/`go mod tidy` when `internal/adapter/postgres` is
  actually built and imports it in Phase F; adding it as an unused
  `require` now would just be stripped by the `go mod tidy` step in the
  Terminal commands at the end of the Phase B gate summary. (Phase B gate
  round 4 fix, 2026-07-05 — this section previously claimed pgx was
  already required and go.sum was not yet generated, neither of which
  matched the on-disk go.mod/go.sum — Codex round-4 finding B4.)
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
