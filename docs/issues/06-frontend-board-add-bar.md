## Context

Build the frontend — the board UI, the Add bar, and drag-and-drop — as a
React/TS/Tailwind SPA in `web/`, embedded into the single Go binary via
`go:embed`. This is the **first frontend issue** in the project and the first
to exercise `design.md`, the `ui/frontend-design` + `ui/web-design-guidelines`
skills, and the UI DESIGN SYSTEM + UI CORRECTNESS middle-loop checks.

It wires to the **already-built** API (Create/Board/Move, issues #3 and #5) and
needs nothing from project Phases D/E (Update/Delete/Tags/REST-hardening), which
are **descoped for this UI-path build** — the board here is read + create + move
only. Card detail, edit/delete, and tag filter are deferred to Phase G.

References: `docs/PRD.md` (Goals 1–3, the UI acceptance criteria) |
`docs/PHASE_PLAN.md` (Phase F — Frontend: Board & Add Bar) | `design.md` (the
LOCKED design system — build exactly to it) | `docs/issues/03-api-create-board.md`
and `docs/issues/05-api-move.md` (the API wire contracts this SPA consumes).

## Locked tech stack

React + TypeScript + Tailwind (PRD "Locked From Brief"). Vite build. **dnd-kit**
for drag-and-drop (chosen at the workshop Phase E gate, 2026-07-23). The built
SPA (`web/dist`) is `go:embed`ed into `cmd/trailhead` and served at `/`, with
the existing API under `/api`. Single binary, no separate frontend server in
production.

## Scope (this issue — Board & Add Bar)

- **`web/` SPA scaffold** — Vite + React + TS + Tailwind, built EXACTLY to
  `design.md` (no invented typography, color, spacing, radius, shadow, or
  component pattern; every CSS variable/font must trace to `design.md`).
- **Board view** — three columns (Inbox / Reading / Done) rendered from
  `GET /api/board`. Empty columns (the API returns `[]`, never null) render the
  designed per-column empty state from `design.md`, not a blank area.
- **Card** — title (Fraunces), host/domain line, tag pills, and a "finished
  {date}" line for Done cards — per `design.md` Cards. No status label on the
  card (the column conveys status). The card's title links to the real URL
  (`<a>`).
- **Add bar** — `POST /api/bookmarks`. On 201, optimistically prepend the new
  card to Inbox; on 409 (duplicate), inline "This is already on your board" with
  a link to the existing card; on 400 (`invalid_url`), inline "That doesn't look
  like a web address"; on 413/500, a toast. Per `design.md` Inputs + Error
  states. A visually-hidden label; the placeholder is a hint only.
- **Drag-and-drop (dnd-kit)** — dragging a card across columns issues
  `POST /api/bookmarks/{id}/move` with `target_status` + the `before`/`after`
  neighbor ids derived from the drop position; dragging within a column issues a
  move with `before`/`after` for reorder. **Optimistic UI**: apply the move
  immediately, reconcile with the server response, roll back + toast on failure.
  Keyboard-operable (dnd-kit KeyboardSensor) and announced via an aria-live
  region using the UI column names — per `design.md` Accessibility.
- **States** — loading (skeleton board), page-level error ("Couldn't load your
  board… Try again"), and per-column empty states, all per `design.md`.
- **`cmd/trailhead/main.go`** — `go:embed web/dist`, serve the SPA at `/`, keep
  the API under `/api`. A `make` target builds the SPA then the Go binary.

## Out of scope (Phase G / needs project Phase D)

- Card-detail modal (full URL, edit title/tags/author, delete) — Phase G, needs
  Update/Delete (project Phase D).
- Tag-filter UI — Phase G, needs the board-filter query param (project Phase D).
- Update / Delete / Tags backend (project Phase D) — **descoped** for this
  UI-path build; logged as descoped in PHASE_PLAN.md so Close-Out's
  plan-completeness cross-check (H-29) accounts for it.

## Frontend test strategy (DECIDED — two layers, both locked at Pre-Phase F)

The methodology's test-engine is Go/backend-shaped; this issue is the first to
apply it to a frontend, so the two layers below are the locked-test target the
build must pass.

1. **Vitest + React Testing Library + MSW** (Mock Service Worker) — the primary
   component/interaction layer: fast, deterministic, no browser or server.
   Locks: the board renders three columns from a mocked `GET /api/board`; cards
   render title/host/tags and the Done finished-date; the Add bar create (201 →
   optimistic Inbox card), duplicate (409 → inline message), and invalid (400 →
   inline message) paths; the loading/empty/error states; keyboard operability
   of the Add bar and controls. MSW mocks every `/api` response.
   Files: `web/src/**/*.test.tsx`.
2. **Playwright** — the e2e layer for the drag-and-drop acceptance criteria that
   jsdom cannot exercise (dnd-kit depends on real pointer/keyboard events and DOM
   measurement). Locks: drag Inbox→Reading updates status and persists after a
   reload (Done sets finished-date; leaving Done clears it); drag within a column
   reorders and persists; keyboard-drag works and is announced. Runs against the
   **built binary** (`go:embed` SPA + API) backed by a throwaway test Postgres
   (mirroring `tests/integration/postgres` DB setup) OR a test-mode wiring to
   `FakeBookmarkRepository` for speed — the Pre-Phase F session picks one and
   documents it.
   Files: `web/e2e/*.spec.ts`.

Rationale: rendering/forms/states are the lockable fast layer (Vitest+RTL); the
drag-and-drop is the board's central, hardest behavior and is untestable in
jsdom, so it must be Playwright against a real browser. Both together are the
complete frontend-path test — and exercising test-engine on a frontend for the
first time is expected to surface methodology findings (log them).

CI: add a **Vitest** job (runs on every PR, fast) and a **Playwright** job
(headless browser; on every PR, or gated if runtime is heavy) to `ci.yml`.

## UI Contract

**CONFIRMED by developer, 2026-08-07 (Pre-Phase F interview, 4 questions,
Tier 2).** The frontend analog of #3/#5's Wire Contract — locks the DOM
surface, the DnD→API mapping, optimistic-UI behavior, and the exact copy
the locked test files (`web/src/**/*.test.tsx`, `web/e2e/*.spec.ts`)
assert against. Dispatch may not deviate from anything below without
amending this section and re-running the locked tests.

### Locked module map

Only these paths are locked, because the committed test files import them
directly by name — everything else (further component decomposition, a
`Column` subcomponent, CSS structure) is Dispatch's own choice:

- `web/src/App.tsx` — exports `App`. Owns the `GET /api/board` fetch,
  loading/error/board state, the header (h1 + `AddBar`), the three
  columns, and the `dnd-kit` `DndContext`.
- `web/src/components/AddBar.tsx` — exports `AddBar`, props
  `{ onCreated: (bookmark: Bookmark) => void; onTransientError: (message: string) => void }`.
  Owns `POST /api/bookmarks` and every inline (400/409) error
  presentation. Never touches board state itself.
- `web/src/components/Card.tsx` — exports `Card`, props
  `{ bookmark: Bookmark }`. Pure presentational.
- `web/src/dnd/deriveNeighbors.ts` — exports
  `deriveNeighbors(targetColumnIds: string[], dropIndex: number): { before: string | null; after: string | null }`,
  a pure function (see `web/src/dnd/deriveNeighbors.test.ts` for the full
  locked contract).
- `web/src/api/types.ts` — the locked wire-contract TS types (already
  committed at Pre-Phase F scaffold, mirrors #3/#5 exactly).

### DOM / accessibility contract

- One `h1`, text exactly `"Trailhead"` (the wordmark doubles as the page
  heading — design.md Accessibility Baseline).
- One `h2` per column, text exactly `"Inbox"` / `"Reading"` / `"Done"`, in
  that order — never the internal `Status` values.
- Each column is a `<ul>` with an accessible name equal to its column name
  (`aria-labelledby` pointing at its `h2`, or an equivalent `aria-label`),
  queryable via `getByRole('list', { name: 'Inbox' })`. Each card is a
  native `<li>` inside it (implicit `listitem` role — no `role="..."`
  needed).
- Each card's root element carries `data-testid="card-{id}"` (Playwright
  locator target) and `id="bookmark-{id}"` (anchor-link target for the
  409 duplicate message below). The card's title is a single `<a>` whose
  text is the title and whose `href` is `bookmark.original_url` (not
  `canonical_url` — the saved page, not the dedup key).
- Each column's droppable container carries `data-testid="column-{status}"`
  (`status` = `inbox` | `reading` | `done`).
- The Add bar's input has a visually-hidden `<label>` with the accessible
  name `"Add a bookmark by URL"`, and `placeholder="Paste a link to save
  it…"` (real ellipsis character, not three periods — web-design-
  guidelines). It sits inside a `<form>` with a submit `<button type="submit">`
  whose accessible name is `"Save"` at rest and `"Saving…"` while a create
  request is in flight (disabled during that state).
- An inline field error sets `aria-invalid="true"` and
  `aria-describedby` on the input, pointing at the element holding the
  error text (web-design-guidelines Checklist 1 — a sighted-only inline
  error is a P0).
- `dnd-kit`'s `DndContext` renders its own `role="status"` live region
  automatically once mounted — no custom live-region markup is needed.
  `App` supplies a custom `announcements` object (see "Keyboard / drag
  announcements" below) rather than dnd-kit's default English text, so
  announcements use the UI's column names.
- Loading skeletons: three elements carrying `data-testid="column-skeleton"`
  (one per column) while `GET /api/board` is in flight; gone once it
  resolves (success or error).

### DnD → API mapping

Locked as the pure function `deriveNeighbors` (see
`web/src/dnd/deriveNeighbors.test.ts` for the full test-locked contract).
Summary: `targetColumnIds` is the target column's bookmark-id array with
the dragged bookmark already removed by the caller; `dropIndex` is the
0-based index the dragged card should land at (out-of-range clamps into
`[0, targetColumnIds.length]`). `before` = the **predecessor** — the id
immediately preceding `dropIndex` (or `null` at the front); the moved
card lands immediately *after* it. `after` = the **successor** — the id
currently at `dropIndex` (or `null` at the end); the moved card lands
immediately *before* it. Both are sent on every
`POST /api/bookmarks/{id}/move` call — per the Move Wire Contract
("Before wins; null = end-of-column"), this means a normal drop is fully
resolved by real neighbor references and never needs to rely on Move's
stale/cross-status fallback; that fallback exists for concurrent-edit
edge cases, not the common path.

**Direction verified against the REAL adapter, not the fake (corrected
2026-08-07, pre-commit — see issue #7):** `deriveNeighbors` was originally
built with `before`/`after` swapped, matching
`FakeBookmarkRepository.Move`'s inverted interpretation rather than
`internal/adapter/postgres/repository.go`'s `resolveNeighborBounds` (the
adapter Playwright and production actually run against). Confirmed by
reading `resolveNeighborBounds` directly — when `Before` is set, the
computed bounds are `(pos, succ)` where `pos` is `Before`'s own position,
i.e. the moved card's new rank falls between `Before` and `Before`'s
current successor (immediately *after* `Before`) — and by
`tests/integration/postgres/move_test.go`'s own comment:
`"Before=cardB says 'insert right after B, before C'"`. The fake's
inversion is tracked separately as issue #7 (not fixed here — this
issue's frontend scope only fixes `deriveNeighbors` to the correct,
real-adapter side); the two are the same root cause on both sides of the
Wire Contract, filed and fixed independently since they're different
packages.

**Separate note for the Phase F build** (methodology finding, not
actionable here — `internal/adapter/ports.go` is READ-ONLY, and this is
independent of the direction fix above): `MoveCommand`'s own doc comment
reads *"Before and After are nil to mean 'first in column' / 'last in
column' respectively,"* which contradicts `Move`'s own more detailed doc
comment and `DECISIONS.md` ("falls back to inserting at the **end** of
the target column" when both are nil) — under the correct direction,
`Before` nil (with `After` also nil) falls back to **end**-of-column, not
front, in both the fake and the real adapter alike, so the comment's
"first in column" claim doesn't hold regardless of which side (fake vs.
real) is used as the reference. Flagged for routing, not silently
"fixed," since the file is locked.

### Optimistic UI + rollback

**Add bar (developer-confirmed: "No card until 201"):** submitting does
**not** show a placeholder card before the response arrives — the client
doesn't duplicate the server's `DefaultTitle` derivation, so there is
nothing honest to show yet. Instead: disable the input/button and show
`"Saving…"` while in flight; on `201`, prepend the returned `Bookmark`
directly to local `Inbox` state (no `GET /api/board` refetch) and clear
the input; on `409`/`400`, show the inline message (below) and leave the
input's text as-is so the user can see what they typed; on `413`/`500`,
call `onTransientError` (App shows a toast) and leave the input as-is.

**Drag/move (true pre-response optimism — design.md "apply immediately,
reconcile"):** the full `Bookmark` object is already known client-side
(only its `Status`/`Position` change), so the move applies to local state
the instant the drop resolves, before the network call returns. On `200`,
reconcile the moved card with the server's authoritative response
(`status`/`position`/`finished_at`/`updated_at`). On any failure, roll
back to the pre-move board state and show the toast: `"Couldn't save
that move — put it back?"` with a `"Retry"` action (design.md's own
literal example — not a Pre-Phase F addition).

### Empty / error copy

Verbatim from design.md where design.md gives it verbatim:

- Empty states — Inbox: `"Nothing saved yet. Paste a link above to start
  your board."` · Reading: `"Drag a card here when you start reading
  it."` · Done: `"Finished something? Drag it here to check it off."`
- Page-level board-load error: `"Couldn't load your board. Check your
  connection and try again."` with a `"Try again"` button that retries
  `GET /api/board`.
- Add bar duplicate (409): `"This is already on your board."`
- Add bar invalid URL (400 `invalid_url`): `"That doesn't look like a web
  address."`
- Move failure toast: `"Couldn't save that move — put it back?"` +
  `"Retry"`.

**Pre-Phase F additions (design.md doesn't give these verbatim — written
to match its locked message-format rule, "says what happened AND what to
do next"; flagged for the developer to confirm/amend at approval, same as
any other copy gap):**

- Add bar empty submit (400 `bad_request`): `"Paste a link before
  saving."`
- Add bar transient failure (413/500) toast: a message containing
  `"Try again"` (exact wording left to Dispatch; `AddBar.test.tsx`
  asserts only that `onTransientError`'s argument contains that phrase,
  not an exact string).

**409 "link to the existing card" (developer-confirmed: scroll +
highlight, no modal):** rendered as a real `<a href="#bookmark-{existing.id}">Show
it</a>` immediately after the inline duplicate message. No cross-component
JS wiring is needed — every card already carries `id="bookmark-{id}"`
(DOM contract above), so the anchor is native fragment navigation. The
"highlight" itself is a Phase F CSS concern (`:target` pseudo-class on the
card, styled per design.md when its token theme lands) — out of scope for
this Pre-Phase F session, not asserted by any locked test.

### Add bar fields (developer-confirmed: URL-only)

The Add bar has exactly one input (the URL). `title` is always sent as
`null` in the `POST /api/bookmarks` body — `domain.DefaultTitle` always
derives it server-side. No title input exists in this issue's scope.

### Keyboard / drag announcements

`dnd-kit`'s `KeyboardSensor` with a **custom `coordinateGetter`**
(required for cross-container movement — dnd-kit's default coordinate
getter only handles single-container reordering) implementing:

- `ArrowRight` / `ArrowLeft` — move the lifted card to the next /
  previous column, landing at the end of that column's current order.
- `ArrowDown` / `ArrowUp` — move the lifted card later / earlier within
  its current column's order.
- `Space` — pick up (if not dragging) / drop (if dragging).
- `Escape` — cancel the drag; the card returns to its original position.

`announcements` (passed to `DndContext`, replacing dnd-kit's default
English text so the UI's column names are used — design.md Accessibility
Baseline's example phrasing):

- `onDragStart`: `"Picked up {title}."`
- `onDragOver`: `"Moved to {ColumnName}, position {index} of {total}."`
- `onDragEnd` (successful drop): `"Dropped."`
- `onDragCancel`: `"Movement cancelled."`

### MSW handlers (already committed — `web/src/mocks/handlers.ts`)

Default (happy-path) handlers for `GET /api/board`, `POST /api/bookmarks`
(201), and `POST /api/bookmarks/{id}/move` (200), plus named error-scenario
factories (`errorHandlers.createDuplicate`, `.createInvalidUrl`,
`.createBadRequest`, `.createPayloadTooLarge`, `.createInternalError`,
`.boardOk`, `.boardInternalError`, `.moveNotFound`, `.moveInternalError`)
individual tests layer in via `server.use(...)`. Response shapes read
directly from `internal/api/handlers.go`/`response.go` at Pre-Phase F, not
guessed.

### Playwright backend (developer-confirmed: throwaway test Postgres)

Mirrors `tests/integration/postgres/setup_test.go`'s own pattern —
`TEST_DATABASE_URL` env var (loud failure if unset, never a silent skip),
`TRUNCATE`-based per-test isolation, a deterministic `testUUID` helper —
via the `pg` npm client in `web/e2e/fixtures.ts`, rather than a
`FakeBookmarkRepository` test-mode wiring. Chosen over the fake for
production-fidelity: the e2e layer's entire purpose is proving the real
pointer/keyboard drag persists through the real `Repository.Move`, and a
second CI Postgres service container is cheap given `integration.yml`
already pays that cost.

**`cmd/trailhead` requirements this implies for the Phase F build** (not
built at Pre-Phase F — `main.go`'s `go:embed`/wiring is explicitly out of
scope for this session, per this issue's own instructions):

- When `DATABASE_URL` is set, `main.go` must call
  `postgres.RunMigrations(cfg.DatabaseURL)` before the HTTP server starts
  listening — Playwright's `webServer.url` health-check waits for the
  port to respond, so fixture-seeding INSERTs in `web/e2e/fixtures.ts`
  can safely assume the schema already exists once Playwright proceeds.
  (Note: `Makefile`'s `migrate:` target already references
  `./scripts/migrate/main.go`, which does not exist on disk — a
  pre-existing gap, not introduced here; `RunMigrations` being called
  from `main.go` directly sidesteps needing that script for e2e/
  production startup, though the gap itself is worth a separate
  observation-routing note.)
- `go:embed web/dist` serves the built SPA at `/`; the API stays mounted
  under `/api` (already true of `NewRouter`).
- `playwright.config.ts`'s `webServer.command` (`../bin/trailhead`)
  expects the binary at `bin/trailhead` relative to `web/` — i.e.
  `../bin/trailhead` from `web/`, matching `make build`'s existing output
  path (`bin/$(BINARY_NAME)`).

### Finished-date formatting

`Card`'s `"finished {date}"` line formats `finished_at` with
`timeZone: 'UTC'` explicitly (e.g.
`Intl.DateTimeFormat('en-US', { month: 'short', day: 'numeric', year: 'numeric', timeZone: 'UTC' })`
→ `"finished Aug 6, 2026"`). Explicit UTC, not the runner's local
timezone — this project has already been scarred once by a TZ-dependent
test that passed in CI but would fail locally
(`trailhead-rules` Rule 3 / `FakeBookmarkRepository`'s UTC-clock fix,
issue #3 Pre-Phase F, 2026-07-09); this is the same failure class applied
to a new surface (client-side date formatting) before it ships, not after.

## Acceptance Criteria (from PRD Goals 1–3, at the UI level)

- Given the Add bar and a valid, non-duplicate URL, when submitted, then a card
  appears in Inbox with a title (supplied or derived) and the UI does not error.
- Given a duplicate URL, when submitted, then no new card is created and an
  inline "already on your board" message shows (backed by the 409).
- Given text that is not an absolute http/https URL, when submitted, then an
  inline "not a web address" message shows and no card is created (400).
- Given a card in any column, when dragged to a different column and dropped,
  then its status updates immediately, Done sets `FinishedAt` (shown as the
  finished-date), leaving Done clears it, and the change persists across reload.
- Given a column with multiple cards, when a card is dragged to a new position
  and dropped, then its order updates and persists across reload.
- Given an empty column, when the board renders, then its designed calm empty
  state shows, not a blank area.
- Given the window is narrowed below the three-column width, when the board
  renders, then columns stack responsively (no clipping, no whole-page
  horizontal scroll).
- All of the above are operable by keyboard, and drag state is announced via an
  aria-live region.

## Design-system adherence (mandatory gate)

Build exactly to `design.md`. The UI DESIGN SYSTEM check (every color/font/space
traces to `design.md`) and UI CORRECTNESS check (`web-design-guidelines` P0s
resolved) apply in the middle loop; the build session loads `ui/frontend-design`
+ `ui/web-design-guidelines` and reads `design.md` before any aesthetic or
structural decision. Verify the contrast pairs `design.md` flags (white on
`--color-primary`; `--color-text-dim` on `--color-bg`) at ≥ 4.5:1.

## Parallel Safety

Can run alongside: none.
Must wait for: #3 (Create/Board API), #5 (Move API), workshop Phase E gate
(`design.md` committed). All complete.
Blocks: Phase G (card detail, tag filter) — which additionally needs project
Phase D (Update/Delete/Tags).

## Tier

Tier 2 (frontend). No new entry existed in `RISK_TIER_REGISTER.md` (it tiered
the backend packages); add `web/` as Tier 2 — UX/glue bugs, mitigated
procedurally by `design.md`, the UI checks, and the two-layer test suite, not by
Tier-1 line-by-line review. Interview depth at Pre-Phase F: Tier 2 (max 12
questions).

## Labels

phase-f, tier-2, frontend, ui
