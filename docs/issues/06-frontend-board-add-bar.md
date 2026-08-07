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
