# PRD: Trailhead — Personal Read-It-Later Board

**Status:** Draft
**Phase:** A (whole-product MVP spec; sliced into build phases at Phase C)
**Author:** Claude + Ali
**Date:** 2026-07-02
**Related issue:** TBD — filed at Phase C once repo/issue tracker exists

---

## Summary

Trailhead is a personal, single-user read-it-later board. Pasting a URL saves
it with a title (user-supplied or derived from the URL) and tags, and drops
it into an Inbox column. The user drags cards across three fixed columns —
Inbox, Reading, Done — to track progress, reorders cards within a column to
set reading priority, and filters the board by tag. It replaces the ad-hoc
combination of browser bookmarks and open tabs with one calm, always-current
view of "what should I read next." It ships as a single Go binary (chi API +
embedded React/TS/Tailwind SPA) backed by PostgreSQL, with no accounts and no
sharing.

## Goals

1. Pasting a valid URL into the Add bar creates a bookmark in Inbox within
   one request/response cycle, with a title either supplied by the user or
   derived from the URL string, and with duplicate URLs (per the locked
   canonical-URL rules) rejected with a clear "already on your board" signal
   instead of creating a second card.
2. A card can be dragged between any of the three columns, and its status
   (and, for Done, its `FinishedAt` timestamp per the locked invariant)
   updates and persists immediately — verified by a page reload showing the
   same state.
3. A card can be dragged to any position within a column, and that exact
   order survives a reload (position round-trips through storage) — verified
   at a working-set scale of roughly 500 bookmarks without visibly janky
   drag behavior.
4. A bookmark can be tagged with zero or more free-text tags (lowercased,
   deduplicated, no empty strings), and the board can be filtered to show
   only bookmarks matching any of one or more selected tags, with a clear way
   to clear the filter back to the full board.
5. Opening a card reveals its full URL, all tags, status, and (if in Done)
   its finished date, and allows editing title/tags/author (including
   clearing author back to unset) or deleting the bookmark — deletion is
   permanent (hard delete).
6. The entire board (bookmarks, statuses, positions, tags, timestamps)
   survives an application restart because everything lives in PostgreSQL,
   not in memory or browser storage.
7. The UI presents a coherent, deliberate visual design system (Phase E
   `design.md`) rather than default/unstyled browser controls, with designed
   empty, loading, and error states, and is usable down to a narrow
   (stacked-column) window width.

Each goal above must be verifiable by a test, a screenshot, or a manual user
action producing an observable, checkable result.

## Non-Goals

Everything in the brief's "Out of scope" list, locked in DECISIONS.md:
user accounts / auth / multi-user / sharing; fetching remote page metadata
or screenshots (no outbound HTTP to saved URLs — title is user-supplied or
derived from the URL string only); full-text search; browser extension;
real-time sync/WebSockets; native mobile apps; any column beyond the three
named; pagination/infinite scroll; analytics. These are permanent
non-goals for this project, not deferred-to-a-later-phase items — see Idea
Factory Condition 2 (holding this scope line is itself a gate condition).

Also non-goals for this PRD specifically (not because they're forbidden,
but because they're not part of the MVP contract): soft-delete/trash/undo;
global tag rename or tag identity across bookmarks; any formal
latency/throughput SLA (a qualitative "feels instant" bar at a ~500-bookmark
working set is the target, verified manually, not load-tested).

## Inputs

- DECISIONS.md decisions this PRD depends on: Position/ordering
  representation (fractional rank string); Canonical URL rules (scheme,
  query params, trailing slash/www, fragments); Identity hash (SHA-256 of
  canonical URL); Tags storage (JSONB array); Title defaulting (hostname +
  de-slugified last path segment); Author field (user-editable only,
  usually absent); Reorder/move API shape (server computes position);
  Duplicate detection response (409 Conflict with existing bookmark); Delete
  semantics (hard delete); Multi-tag filter logic (OR); Invalid URL
  validation bar (must parse as absolute http/https URL with a host); the
  `FinishedAt ⟺ Status == Done` invariant; the "Locked From Brief" tech
  stack and scope boundaries.
- GLOSSARY.md terms used: `Bookmark`, `Status`, `Position`, `CanonicalURL`,
  `IdentityHash`, `Tags`, `FinishedAt`, `Author`, `Board`.

## Affected Modules

Greenfield project — no existing codebase. Package layout below is a Phase A
proposal; it must be verified against what actually exists on disk at the
Phase B B3 read-back gate (per CLAUDE.md) before being treated as locked.

| Module / Package | Change type | Notes |
|---|---|---|
| `internal/domain` | new | `Bookmark`, `Status`, `Position`, `CanonicalURL`, `IdentityHash`, `Tags`, `Board` types per GLOSSARY.md |
| `internal/adapter` (`ports.go`) | new | Repository interface(s) for bookmark CRUD, board query, move/reorder — locked at Phase B |
| `internal/adapter/postgres` | new | PostgreSQL implementation of the repository interface, JSONB tags column, GIN index |
| `internal/api` (chi handlers) | new | REST/JSON endpoints: add bookmark, list board, move/reorder, edit, delete, tag filter |
| `internal/canonicalize` (or under `domain`) | new | Pure canonical-URL + identity-hash derivation functions, golden-testable |
| `web/` (React/TS/Tailwind SPA) | new | Board view, Add bar, card detail modal, tag filter UI, drag-and-drop |
| `cmd/trailhead` | new | `main.go` wiring chi router + Postgres + `go:embed` of the built SPA into one binary |
| `tests/unit`, `tests/integration`, `internal/testutil` | new | Per Phase B gate requirement — fake repository implementations, canonical-URL golden tests, integration tests against a real Postgres |

## Acceptance Criteria

  Given the Add bar is empty
  When the user pastes a valid http/https URL and submits
  Then a new bookmark appears in the Inbox column with a title (user-supplied or derived from the URL) and no tags, and the response/UI does not error

  Given a bookmark already exists whose canonical URL matches a newly pasted URL
  When the user submits that URL again (even with different tracking params, trailing slash, www prefix, or http vs https scheme)
  Then no new bookmark is created, the API returns 409 Conflict with the existing bookmark's data, and the UI shows "already on your board" inline

  Given the Add bar contains text that does not parse as an absolute http/https URL with a host
  When the user submits
  Then the UI shows inline "invalid URL" feedback and no bookmark is created

  Given a bookmark card in any column
  When the user drags it to a different column and drops it
  Then the bookmark's Status updates to the target column immediately, and if the target is Done, FinishedAt is set to the current time; if the source was Done and the target is not, FinishedAt is cleared

  Given a column containing multiple cards
  When the user drags a card to a new position within that column and drops it
  Then the card's Position updates so it renders in the dropped position, and reloading the page shows the same order (Position round-trips through storage)

  Given a bookmark has zero tags
  When the user adds one or more free-text tags via the card detail view
  Then the tags are stored lowercased and deduplicated, and any empty-string tag entry is not stored

  Given the board has bookmarks with various tags
  When the user selects one or more tags in the tag filter
  Then the board shows only bookmarks matching at least one (OR) of the selected tags, and a visible control clears the filter back to the full board

  Given a bookmark card
  When the user opens its detail view
  Then the full URL, all tags, current status, and (if Status is Done) the finished date are shown, with controls to edit title/tags/author or delete

  Given a bookmark's detail view is open in edit mode
  When the user changes the title, tags, and/or author (including clearing a previously-set author back to unset) and saves
  Then the bookmark's title/tags/author update immediately in the UI, and reloading the page shows the same edited values (the edit round-trips through storage the same way position and status do)

  Given a bookmark's detail view is open
  When the user deletes it
  Then the bookmark is permanently removed (hard delete) and no longer appears on the board

  Given the application has bookmarks persisted in PostgreSQL
  When the application process restarts
  Then the board renders identically to its state before restart (same bookmarks, statuses, positions, tags, timestamps)

  Given a column has no bookmarks
  When the board renders
  Then a designed calm empty state is shown for that column, not a blank area

  Given the browser window is narrowed below the point where three columns fit side by side
  When the board renders
  Then columns stack responsively rather than clipping or requiring horizontal scroll of the whole page

## Error Conditions

  Condition: pasted text is not a syntactically valid http/https absolute URL
  Expected behaviour: reject with a 4xx response; no bookmark row is created
  User-visible: inline "invalid URL" message in the Add bar, no page navigation

  Condition: pasted URL's canonical form matches an existing bookmark's IdentityHash
  Expected behaviour: reject the create; return the existing bookmark's data
  User-visible: inline "already on your board" message, optionally linking to / highlighting the existing card

  Condition: a move/reorder request references neighbor IDs that no longer exist (e.g. deleted between drag start and drop)
  Expected behaviour: server recomputes a valid position (e.g. treats missing neighbor as column boundary) rather than failing the whole request; this exact fallback behavior is an Open Question below
  User-visible: the drag completes without an error dialog; worst case the card lands at an edge of the column rather than exactly where dropped

  Condition: PostgreSQL is unreachable at request time
  Expected behaviour: API returns a 5xx; no partial writes
  User-visible: a designed (not default-browser) error state in the UI, not a raw stack trace or blank screen

## Open Questions

- ~~Exact fallback behavior when a move/reorder request's neighbor references are stale~~ — **Resolved at Phase B gate (2026-07-04):** falls back to inserting at the end of the target column. See DECISIONS.md "Move — Stale Neighbor Fallback."
- ~~Whether Author can be explicitly cleared back to unset via the edit view~~ — **Resolved at Phase B gate (2026-07-05):** `BookmarkPatch.ClearAuthor bool`. See DECISIONS.md "Author Field — Clearing via Patch."
- Exact deny-list contents for tracking-parameter stripping in canonical-URL derivation — DECISIONS.md locks the *rule* (strip known tracking params, sort the rest) as Locked, and `internal/domain/canonicalize.go` now implements a starter deny-list (`utm_source`, `utm_medium`, `utm_campaign`, `utm_term`, `utm_content`, `utm_id`, `gclid`, `fbclid`, `mc_eid`, `mc_cid`, `ref`, `igshid`) that is real and in force, not a placeholder. What remains open and Pre-Phase F is golden-testing this list's behavior — not whether the rule holds. Extending the list with additional entries beyond golden-testing is itself a change to a Locked decision and requires a DECISIONS.md amendment, not a silent Pre-Phase F code change (see DECISIONS.md "Canonical URL — Query Parameters" for the reconciled wording; this was flagged as an apparent locked-vs-open contradiction in Codex's round-2 Phase B review and is resolved by this distinction).
- Exact HTTP routes, request bodies, and response bodies for the `internal/api` REST/JSON contract — ARCHITECTURE_RFC.md's Data Flow and Serialization Spec sections define the shape of the contract (timestamps, null-vs-missing, ID representation) but not concrete routes/methods/payloads. This is deliberately Pre-Phase F scope per CLAUDE.md's Phase B Scope Boundary (implementation detail, not an open architecture question) and per the REST Adapter Wire Contract gate — each `internal/api` issue's Pre-Phase F prep must include a Wire Contract section before that issue is assigned to Dispatch.
- Drag-and-drop library choice (e.g. dnd-kit vs. a hand-rolled HTML5 DnD implementation) — deferred to Phase B/E as an architecture and design-system decision, not a Phase A domain question, but must be resolved before any Phase F issue touching the board UI begins.

## Out of Scope

See Non-Goals above (permanent) and DECISIONS.md "Locked From Brief" (permanent). No additional deferred-to-future-PRD items are being tracked at this time — this is a deliberately narrow, throwaway (DELETE-retention) methodology-validation build, not a product with a roadmap.
