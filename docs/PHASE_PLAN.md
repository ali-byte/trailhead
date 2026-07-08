# Phase Plan: Trailhead

**PRD:** docs/PRD.md
**Status:** Draft
**Date:** 2026-07-07

Note on phase lettering: this document's Phase A/B/C/... labels are
tracer-bullet implementation phases (per workflow/prd-to-plan), a distinct
letter-scheme from CLAUDE.md's top-level workshop phases (also A-I). Where
both are referenced in the same sentence below, "workshop Phase X" means
the CLAUDE.md gate; a bare "Phase X" means this document's own phase.

---

## Summary

Trailhead ships in seven tracer-bullet phases after the skeleton delivered
at the workshop Phase B gate: a thinnest-path Postgres-backed Create/Board
round-trip, then Move/reorder, then Update/Delete/Tags, then a REST
contract hardening pass, then two frontend phases gated behind workshop
Phase E (design.md). Every phase after the skeleton is strictly sequential
— all backend phases touch the same Tier 1 `internal/adapter/postgres`
package, and this is a single-developer project, so no parallel-safety
isolation strategy is needed (see Dependency Map).

---

## Phase A — Skeleton (DELIVERED — workshop Phase B gate, closed 2026-07-06)

**Goal:** Nothing behavioral works yet, but everything compiles and the
locked contract exists on disk.

**Slices (already committed, not re-planned here):**
- `internal/domain` — all types (`Bookmark` with JSON tags, `Board`,
  `NewBookmark`, `BookmarkPatch`, etc.) per docs/GLOSSARY.md
- `internal/domain/canonicalize.go` — `Canonicalize`, `DeriveIdentityHash`,
  `DefaultTitle`, implemented (Scope Boundary exception, not stubbed)
- `internal/adapter/ports.go` — `BookmarkRepository` interface, READ-ONLY
  after gate
- `internal/testutil/fake_repository.go` — `FakeBookmarkRepository`,
  UUID-format IDs (fixed at Phase B gate round 5)
- `cmd/trailhead/main.go` — minimal compiling entry point (health check
  only)
- `go.mod`/`go.sum`, `.gitignore`, `Makefile`, CI workflows

**Gate condition:** `go build ./...` and `go vet ./...` pass; `ports.go`
matches ARCHITECTURE_RFC.md's "Locked Interfaces" block exactly (B3
verification, already performed).

**Status:** CLOSED. Confirmed via Terminal by the developer, 2026-07-07.

---

## Phase 1 — Postgres Adapter & Thinnest Path (Add Bookmark)

**Goal:** A pasted URL survives a real round trip through a real
PostgreSQL instance and comes back out via the board query — the
project's first genuinely working vertical slice, no mocks. Satisfies
PRD Goal 1.

**Slices:**
- `internal/adapter/postgres/repository.go` — new package; implement only
  `Create` and `Board` (not the full interface yet — tracer-bullet
  discipline, per prd-to-plan's Phase 2 rule). Uses `pgx/v5` directly
  (ARCHITECTURE_RFC.md "Postgres Driver"). Duplicate detection via the
  unique index on `identity_hash`, per ARCHITECTURE_RFC.md "Persistence
  Schema" notes — catch the unique-violation and re-query rather than
  check-then-insert.
- `scripts/migrate/main.go` + a migrations directory — applies the exact
  DDL from ARCHITECTURE_RFC.md "Persistence Schema" (the `bookmarks` table,
  all three indexes). Wires into the Makefile's existing `migrate` target.
- `cmd/trailhead/main.go` — replace the Phase A stub's TODO with real
  `pgxpool` wiring, reading `DATABASE_URL` from the environment.
- `internal/api` — new package; a single Wire-Contract-scoped slice: one
  create-bookmark route and one get-board route. Maps
  `RepositoryError{Kind: ErrKindDuplicate}` to 409 (with `Existing` in the
  body) and `ErrKindInvalidURL` to 400, per DECISIONS.md "Duplicate
  Detection Response" and the Error Taxonomy.
- `docker-compose.yml` — local Postgres for dev/integration testing
  (standard Phase D scaffolding pattern, pulled forward here since this
  phase is the first that needs a real database).
- `.env.example` — documents `DATABASE_URL` and `PORT` (the only two env
  vars `cmd/trailhead/main.go` currently reads). **Phase 1 completeness
  item** — did not exist at the Phase A skeleton; added here since this is
  the first phase that actually uses `DATABASE_URL`.
- `README.md` stub — minimal now (project name, one-line description, "see
  docs/ for architecture"); workshop Phase H will add deployment
  instructions into this same file later. **Phase 1 completeness item.**
- `.github/workflows/integration.yml` — recreate (deleted at the Phase D
  CI-fix, 2026-07-07: it kept firing red on every push with no real
  `tests/integration/` content behind it, even after its `push` trigger
  was removed — the developer removed the file outright rather than debug
  a workflow with nothing to test). This phase is the first to produce
  real integration test files, so it's the right place to stand the
  workflow back up.

**Open item this phase must resolve (not deferred further):** the exact
`internal/api` route(s), request body, and response body for create-
bookmark and get-board. Per the Wire Contract gate (workflow/prd-to-issues
Phase 3d), the issue for this slice must include a Wire Contract section
and get explicit developer confirmation before its Pre-Phase F test file
is committed — this is the first of several PRD Open Questions
("exact HTTP routes, request bodies, response bodies") this phase
actually resolves, not just references.

**Gate condition:** `go test -tags integration ./tests/integration/...
-run TestCreateBookmark` passes against a real local Postgres (via
`docker-compose up` + `make migrate`): POSTing a new URL creates a row
whose `CanonicalURL`/`IdentityHash`/`Title` match DECISIONS.md's rules,
and re-POSTing the same URL (post-normalization) returns 409 with the
pre-existing bookmark's data, not a second row.

**Integration test targets:** [to be generated by test-engine before this
phase begins — provisional name: `TestCreateBookmark_DedupeAndPersist`,
`internal/adapter/postgres/repository_test.go` for adapter-level tests
(build tag `integration`)]

**Developer task (not a Dispatch issue):** stand up local Postgres via
`docker-compose up` and confirm `make migrate` runs clean. Complete before
Pre-Phase F for this phase's issue begins.

---

## Phase 2 — Move & Reorder

**Goal:** A card can move between columns and within a column, and the
`FinishedAt` invariant and neighbor-fallback rule both hold against a real
database. Satisfies PRD Goals 2 and 3.

**Slices:**
- `internal/adapter/postgres/repository.go` — implement `Move`, including
  the fractional-rank position algorithm (the one piece ARCHITECTURE_RFC.md
  explicitly deferred past Phase 1 — "Scope Boundary": "the fractional-rank
  position algorithm explicitly NOT implemented yet"). This phase is where
  that open architecture question gets resolved in code.
- Neighbor-fallback per DECISIONS.md "Move — Neighbor Fallback
  (Generalized)", including the round-5 narrowed Before/After
  precedence tie-break.
- **This phase is where E1 (Phase B gate's accepted risk — invalid
  `MoveCommand.TargetStatus` handling) must actually be decided**, per
  ARCHITECTURE_RFC.md's own boundary note routing that decision to "each
  `internal/api` issue that constructs a `MoveCommand`... at its Wire
  Contract section." Decide here: does `internal/api` validate
  `TargetStatus` before calling `Move`, or does `Move` itself return a new
  classified error? Must be resolved and documented before this issue's
  Pre-Phase F test file is committed.
- `internal/api` — move/reorder route, Wire-Contract-scoped.

**Gate condition:** `go test -tags integration ./tests/integration/...
-run TestMoveBookmark` passes: moving a bookmark into/out of Done sets/
clears `FinishedAt` correctly against real Postgres; a stale/cross-status/
self-referential neighbor falls back to end-of-column; a `Before`+`After`
combination resolves to `Before` (not an error); reordering within a
column round-trips `Position` through a real read after write.

**Integration test targets:** [to be generated by test-engine before this
phase begins]

---

## Phase D — Update, Delete, Tags & Filter

**Goal:** A bookmark's title/tags/author can be edited (including clearing
author), deleted permanently, and the board can be filtered by tag against
a real database. Satisfies PRD Goal 4 and the edit/delete half of Goal 5.

**Slices:**
- `internal/adapter/postgres/repository.go` — implement `Update`
  (including `ClearAuthor` tri-state precedence) and `Delete` (hard
  delete, per DECISIONS.md).
- Tag normalization (lowercase, dedupe, drop empty strings) on the write
  path. **Planning note, not yet a decision:** this logic currently only
  exists in `internal/testutil/fake_repository.go`'s unexported
  `normalizeTags` — the Postgres adapter needs the same rule applied and
  should not silently re-implement it a second time with room to drift.
  Whether to extract a shared `domain.NormalizeTags` pure function (so
  both the fake and the real adapter call one implementation) or duplicate
  it intentionally is a real architecture question this phase's Pre-Phase
  F prep must decide and record — flagged here so it isn't decided
  silently mid-Dispatch-session.
- `internal/api` — update and delete routes (Wire-Contract-scoped); board
  route gains a `tags` filter query parameter, OR-semantics per
  DECISIONS.md "Multi-Tag Filter Logic".

**Gate condition:** integration tests for update (including author-clear
round-trip), delete (row actually gone, `IdentityHash` freed for re-add),
and tag-filter OR-semantics (using the real GIN index) all pass against
real Postgres.

**Integration test targets:** [to be generated by test-engine before this
phase begins]

---

## Phase E — REST API Hardening

**Goal:** The `internal/api` surface built incrementally across Phases
B-D is reviewed as a whole, not just endpoint-by-endpoint: full route
table, consistent error-mapping, and the accepted-risk item from the Phase
B gate closed out. Not a new user-facing capability — a consolidation
gate before frontend work begins.

**Slices:**
- Full Wire Contract review across all five endpoints (create, board+
  filter, move, update, delete) in one document, superseding the
  per-issue Wire Contract sections scattered across Phases B-D.
- Confirm `RepositoryError.Kind` → HTTP status mapping is applied
  consistently everywhere (409/404/400), and that infrastructure failures
  (Postgres unreachable, context timeout) map uniformly to 5xx everywhere,
  per DECISIONS.md "Repository Error Taxonomy."
- Confirm E1's `TargetStatus` validation decision (made in Phase 2) is
  applied consistently and documented in DECISIONS.md if it wasn't
  already promoted from Phase 2's issue-local note.
- Revisit E2 (RISK_TIER_REGISTER.md's accepted risk on `internal/api`
  being Tier 2, not Tier 1) with real code in hand: confirm the Wire
  Contract + per-issue test coverage actually materialized as the
  mitigating safety net the Phase B gate assumed it would.

**Gate condition:** a single reviewed Wire Contract document covers all
five endpoints; integration tests for every documented error path (400,
404, 409, and a simulated 5xx) pass.

**Integration test targets:** [to be generated by test-engine before this
phase begins]

---

## Workshop Phase E Gate (prerequisite for Phases F and G below)

Per CLAUDE.md's "FRONTEND PROJECTS — UI DESIGN SYSTEM" addendum: no
Phase F Dispatch session may touch frontend code until `design.md` is
committed. This project's `design.md` (checked at Phase B gate round 4)
is currently an unfilled Phase E template. **Workshop Phase E must run
and close — copying `.template/design.md`, filling every section, and
committing it — before Phase F below can begin.** This is a workshop-level
gate, not one of this document's own lettered phases; it's called out here
because it blocks Phase F/G specifically.

**Developer task (not a Dispatch issue):** resolve the PRD Open Question
on drag-and-drop library choice (dnd-kit vs. hand-rolled HTML5 DnD).
Complete before Pre-Phase F prep for Phase F begins.

**Developer task (not a Dispatch issue):** run workshop Phase E (copy
`.template/design.md`, fill every section, commit) and confirm `design.md`
is committed. Complete before Pre-Phase F prep for Phase F begins.

---

## Phase F — Frontend: Board & Add Bar

**Prerequisite:** workshop Phase E gate closed (`design.md` committed);
drag-and-drop library chosen; Phase E (REST hardening, above) closed.

**Goal:** The three-column board renders, a pasted URL creates a card via
the Add bar, and cards can be dragged across columns and reordered.
Satisfies PRD Goals 1, 2, 3, 6 (persistence — already structurally true
once the backend is done, verified here at the UI level via reload), and
the empty/loading/error-state half of Goal 7.

**Slices:**
- `web/` SPA scaffold (React/TS/Tailwind, per the Locked tech stack),
  built exactly to `design.md`, no invented typography/color/spacing.
- Board view (three columns), Add bar, drag-and-drop wired to Phase 2's
  move/reorder endpoint.
- Empty, loading, and error states per `design.md`.
- `cmd/trailhead/main.go` — `go:embed` the built SPA into the single
  binary.

**Gate condition:** manual verification (screenshot or live check) of
PRD's Board/Add-bar/reload acceptance criteria; a page reload after a
drag shows the same state.

**Integration test targets:** [frontend test strategy to be defined at
this phase's Pre-Phase F prep — component/E2E tooling choice is itself an
open item this phase must resolve, not assumed here]

---

## Phase G — Frontend: Card Detail, Tag Filter

**Prerequisite:** Phase F closed.

**Goal:** A card's detail view supports viewing/editing/deleting, and the
board can be filtered by tag. Satisfies PRD Goals 4, 5, and the responsive/
narrow-width half of Goal 7.

**Slices:**
- Card detail modal (view, edit title/tags/author including clearing
  author, delete) wired to Phase D's update/delete endpoints.
- Tag filter UI wired to Phase D's board-filter query param.
- Responsive stacked-column layout below the three-column breakpoint.

**Gate condition:** manual verification of PRD's card-detail, tag-filter,
and responsive-layout acceptance criteria.

**Integration test targets:** [to be defined at this phase's Pre-Phase F
prep, consistent with Phase F's choice]

---

## Dependency Map

- Phase A → Phase 1 → Phase 2 → Phase D → Phase E: **strictly sequential.**
  All four touch `internal/adapter/postgres/repository.go` and
  `internal/api`, Trailhead's only Tier 1 and Tier 2 packages
  respectively — no isolation strategy is written because no parallelism
  is claimed. This is also a single-developer project (RISK_TIER_REGISTER.md
  has no multi-session concurrency scenario), so sequential execution has
  no throughput cost worth trading away for parallel-safety complexity.
- Phase E (backend) → workshop Phase E gate (design.md) → Phase F → Phase
  G: **strictly sequential.** Frontend phases additionally require the
  workshop-level design.md gate, which sits outside this document's own
  phase lettering (see the callout above).
- No phase pair in this plan is marked parallel-safe. (Phase 4b's parallel
  phase safety check is therefore trivially satisfied — there is nothing
  to check for shared-resource conflicts because no parallel pair is
  proposed.)

---

## Open Items Carried Forward From PRD.md (cross-referenced per phase)

- **Deny-list exhaustiveness** (PRD Open Questions) — affects Phase 1's
  `Create` dedup logic; golden-testing is Pre-Phase F prep for that issue,
  not a blocker for filing it.
- **Exact HTTP routes/payloads** (PRD Open Questions) — affects every
  phase touching `internal/api` (1, 2, D, E); each gets its own Wire
  Contract section per workflow/prd-to-issues Phase 3d.
- **Drag-and-drop library choice** (PRD Open Questions) — affects Phase F
  specifically; called out again in that phase's prerequisites above.
- **MV-01 through MV-04** (docs/PRD.md "Methodology Validation Criteria",
  Locked — approved by developer, 2026-07-07) — not tied to any single
  phase; evaluated at workshop Phase H per the four-sub-condition Phase H
  gate locked alongside them.
