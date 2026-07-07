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
| `internal/testutil`, `tests/integration` | new | Per Phase B gate requirement — fake repository implementations live in `internal/testutil`; unit tests are co-located `*_test.go` files next to the code they cover (Go convention, not a separate `tests/unit/` directory — see ARCHITECTURE_RFC.md "tests/ Directory Structure"); `tests/integration` holds end-to-end tests against a real Postgres |

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

## Methodology Validation Criteria

Added at Phase C planning (2026-07-07), per CLAUDE.md's Phase C Second Review
gate condition ("MV criteria — the Phase H gate condition must include all 4
methodology validation criteria (MV-01 through MV-04) from PRD.md. Product AC
alone is not sufficient."). Distinct from the product Acceptance Criteria
above: those verify the *board* works; these verify the *workshop process*
that built it worked. This project's own Retention Policy (DECISIONS.md,
locked at Idea Factory) already frames Trailhead as a throwaway
methodology-validation build whose lasting value is the workshop's routed
observations — these four criteria make that purpose checkable rather than
implicit.

**Locked: yes — approved by developer, 2026-07-07.** Drafted by Claude,
reviewed, and approved without wording changes. Changing MV-01 through
MV-04 or the Phase H gate condition below after this point requires a
deliberate re-decision, same as any other Locked PRD/DECISIONS.md content
— not a silent edit during a later phase.

**MV-01 — Reviewer Agent Signal Quality.** Does the same-model reviewer
agent, spawned fresh at each phase gate, catch real gaps whose direction
is later corroborated by the independent different-model (Codex) pass,
without a high rate of either false negatives (real gaps Codex catches
that the reviewer agent missed) or false positives (the reviewer agent
blocking on something Codex doesn't consider a finding)? Checkable at
Phase H by comparing every reviewer-agent verdict issued across Trailhead's
gates (Phase B's five rounds are the first data points: rounds 2-5 each
had a reviewer-agent pass before or alongside a Codex pass) against the
corresponding Codex verdict from the same round, and noting agreement vs.
divergence.

**MV-02 — Phase Gate Effectiveness.** Does locking interfaces, schemas, and
decisions at Phase B measurably reduce Phase F rework — specifically, how
many Phase F Dispatch sessions (per RISK_TIER_REGISTER.md-tiered issue)
hit an implementation-vs-locked-contract mismatch that required amending
DECISIONS.md/ARCHITECTURE_RFC.md/`ports.go` (a contract change), as opposed
to a plain implementation bug fixable without touching the locked contract?
Checkable at Phase H by counting contract-amendment incidents across all
Phase F issues and comparing the rate against Phase B's own experience
(five rounds of gate-fix loops before close, all resolved *before* any
Phase F code was written) as an internal before/after signal for this one
project, since no directly comparable prior-project Phase F baseline was
identified during Phase C planning.

**MV-03 — Developer Override Discipline.** When the developer exercises
explicit override authority to close a gate against a FAIL verdict (as
happened at the Phase B gate, round 5 — closed by developer decision
without a further Codex re-submission), is every open finding at that
moment explicitly classified — fixed, accepted risk, or deferred — rather
than silently dropped? Checkable at Phase H by grepping every gate-closure
addendum in DECISIONS.md / ARCHITECTURE_RFC.md / RISK_TIER_REGISTER.md for
an explicit disposition on each finding named in the corresponding review
round; a finding with no disposition anywhere on disk is a failure of this
criterion.

**MV-04 — Observation Loop Completion.** Do methodology findings surfaced
during Trailhead's build actually get routed back into the shared workshop
(CLAUDE.md, the skill library, `.workshop-review/observations.md`) rather
than staying trapped in this project's own chat/session history? Checkable
at Close-Out via the existing gate condition (zero outstanding findings
left un-routed in `observations.md`) plus a stronger bar this criterion
adds: at least one concrete CLAUDE.md or skill-file change must be
traceable to a Trailhead-specific finding by the time Trailhead closes out
(candidates already on the table from Phase B alone: the
`export-phase-b-review.sh` root-vs-`docs/` path-lookup behavior, Finding
47's current status, and the "developer can close a FAIL-verdict gate via
explicit per-finding risk acceptance" pattern from round 5).

**Phase H gate condition (LOCKED — approved by developer, 2026-07-07):**
Phase H (Deployment) does not close until each of the four sub-conditions
below is independently satisfied and recorded in
`workshop/observations/trailhead-methodology-observations.md`, in addition
to (not instead of) the product Acceptance Criteria this PRD already
defines and CLAUDE.md's standard Pre-Deployment Review gate. This
evaluation runs as its own pass — a Cowork session pointed at `~/dev`,
same reviewer-agent-style "read and report, don't fix" mode used at every
other phase gate — completed *before* the Pre-Deployment Review skill
runs, not folded into it. Each sub-condition gets an explicit PASS/FAIL,
not just a narrative summary:

- **MV-01 sub-condition:** a table exists in
  `trailhead-methodology-observations.md` listing every reviewer-agent
  verdict issued across every Trailhead gate (Phase B's rounds 2-5 at
  minimum, plus one row per Phase F issue's reviewer-agent pass) against
  the corresponding Codex/different-model verdict from the same round or
  issue, with an explicit agree/diverge column filled in for every row —
  zero blank rows. PASS requires the table to exist and be complete, not
  requiring 100% agreement (a documented divergence is a valid data point,
  not a failure of this criterion).
- **MV-02 sub-condition:** a count of "contract-amendment incidents"
  (Phase F Dispatch sessions that required amending DECISIONS.md /
  ARCHITECTURE_RFC.md / `ports.go` — a contract change — as opposed to a
  plain implementation-only fix) is recorded against the total number of
  Phase F issues, and stated explicitly next to Phase B's own baseline
  (five gate-fix rounds, all resolved before any Phase F code existed).
  PASS requires the count and comparison to exist on the record; there is
  no target rate to hit — this criterion measures whether the comparison
  was made, not what it found.
- **MV-03 sub-condition:** grep evidence (the exact command run, plus its
  output) demonstrating every finding named in every gate-closure addendum
  across DECISIONS.md / ARCHITECTURE_RFC.md / RISK_TIER_REGISTER.md carries
  an explicit disposition — fixed, accepted risk, or deferred — with no
  finding left undispositioned anywhere on disk. PASS requires zero
  undispositioned findings; any found must be resolved (dispositioned)
  before this sub-condition can pass, not merely noted.
- **MV-04 sub-condition:** the existing Close-Out gate (zero outstanding
  findings left un-routed in `observations.md`) is satisfied, AND at least
  one CLAUDE.md or skill-file change is cited by file path and commit hash
  as directly attributable to a Trailhead-specific finding. PASS requires
  a specific citation (path + hash), not a general claim that "lessons
  were learned."

**Overall Phase H gate:** all four sub-conditions PASS. A sub-condition
that cannot pass (e.g. MV-01's table can't be built because no reviewer-
agent pass was ever run at some gate) is itself a finding — route it
through the normal observation-routing process rather than waiving the
gate silently.

## Error Conditions

  Condition: pasted text is not a syntactically valid http/https absolute URL
  Expected behaviour: reject with a 4xx response; no bookmark row is created
  User-visible: inline "invalid URL" message in the Add bar, no page navigation

  Condition: pasted URL's canonical form matches an existing bookmark's IdentityHash
  Expected behaviour: reject the create; return the existing bookmark's data
  User-visible: inline "already on your board" message, optionally linking to / highlighting the existing card

  Condition: a move/reorder request's neighbor reference does not resolve — missing/stale (deleted between drag start and drop), cross-status (exists but not in the target column), or self-referential (equal to the bookmark being moved)
  Expected behaviour: server falls back to inserting at the end of the target column rather than failing the whole request, for every case above — one rule, not a case-by-case one. Separately, if both `Before` and `After` are supplied and disagree, that is not a fallback case: `Before` takes precedence and `After` is silently ignored (a tie-break, not an error and not an end-of-column fallback). Resolved at Phase B gate, narrowed at round 5 (2026-07-06) to stop conflating the tie-break with the fallback rule. See DECISIONS.md "Move — Neighbor Fallback (Generalized)."
  User-visible: the drag completes without an error dialog; worst case the card lands at the end of the column (unresolved neighbor) or at the position implied by `Before` (both neighbors supplied) rather than exactly where dropped

  Condition: PostgreSQL is unreachable at request time
  Expected behaviour: API returns a 5xx; no partial writes. `BookmarkRepository` returns a plain wrapped error (not a `*RepositoryError`) for this and other infrastructure failures — see DECISIONS.md "Repository Error Taxonomy — Infrastructure Failures."
  User-visible: a designed (not default-browser) error state in the UI, not a raw stack trace or blank screen

## Open Questions

- ~~Exact fallback behavior when a move/reorder request's neighbor references are stale, cross-status, or self-referential~~ — **Resolved at Phase B gate (2026-07-04, generalized 2026-07-05, narrowed 2026-07-06):** falls back to inserting at the end of the target column in each of those three cases. A separate, non-fallback rule covers `Before`/`After` disagreement: `Before` takes precedence and `After` is ignored (a tie-break). Round 4's wording had folded this tie-break into the same "falls back to end-of-column in every case" sentence, which round-5 Codex review correctly flagged as broader than what DECISIONS.md and `ports.go` actually specify. See DECISIONS.md "Move — Neighbor Fallback (Generalized)."
- ~~Whether Author can be explicitly cleared back to unset via the edit view~~ — **Resolved at Phase B gate (2026-07-05):** `BookmarkPatch.ClearAuthor bool`. See DECISIONS.md "Author Field — Clearing via Patch."
- Exact deny-list contents for tracking-parameter stripping in canonical-URL derivation — DECISIONS.md locks the *rule* (strip known tracking params, sort the rest) as Locked, and `internal/domain/canonicalize.go` now implements a starter deny-list (`utm_source`, `utm_medium`, `utm_campaign`, `utm_term`, `utm_content`, `utm_id`, `gclid`, `fbclid`, `mc_eid`, `mc_cid`, `ref`, `igshid`) that is real and in force, not a placeholder. What remains open and Pre-Phase F is golden-testing this list's behavior — not whether the rule holds. Extending the list with additional entries beyond golden-testing is itself a change to a Locked decision and requires a DECISIONS.md amendment, not a silent Pre-Phase F code change (see DECISIONS.md "Canonical URL — Query Parameters" for the reconciled wording; this was flagged as an apparent locked-vs-open contradiction in Codex's round-2 Phase B review and is resolved by this distinction).
- Exact HTTP routes, request bodies, and response bodies for the `internal/api` REST/JSON contract — ARCHITECTURE_RFC.md's Data Flow and Serialization Spec sections define the shape of the contract (timestamps, null-vs-missing, ID representation) but not concrete routes/methods/payloads. This is deliberately Pre-Phase F scope per CLAUDE.md's Phase B Scope Boundary (implementation detail, not an open architecture question) and per the REST Adapter Wire Contract gate — each `internal/api` issue's Pre-Phase F prep must include a Wire Contract section before that issue is assigned to Dispatch.
- Drag-and-drop library choice (e.g. dnd-kit vs. a hand-rolled HTML5 DnD implementation) — deferred to Phase B/E as an architecture and design-system decision, not a Phase A domain question, but must be resolved before any Phase F issue touching the board UI begins.

## Out of Scope

See Non-Goals above (permanent) and DECISIONS.md "Locked From Brief" (permanent). No additional deferred-to-future-PRD items are being tracked at this time — this is a deliberately narrow, throwaway (DELETE-retention) methodology-validation build, not a product with a roadmap.
