# Phase B — Independent Architecture Review
**Project:** trailhead
**Date:** 2026-07-05
**Reviewer:** ChatGPT (different-model review — Phase B gate)

---

## Instructions for ChatGPT

You are performing an independent architecture review of a software project
at the end of Phase B (Architecture). You are a REVIEWER, not a builder.
Do NOT suggest rewrites or new code. Produce a structured gap report only.

Read every section in this document, then evaluate against the five categories
below. For each finding, state the exact document name and identifier
(type name, method name, section heading, field name) so the developer can
locate it immediately.

### Review categories

**A. TYPE / FIELD / INTERFACE GAPS**
Any type, field, method, or interface named in one document that is absent,
differently named, or differently typed in another document. Include any port
signature in ARCHITECTURE_RFC.md that does not match the actual adapter
interface file below.

**B. CROSS-DOCUMENT INCONSISTENCIES**
Any place two documents disagree — e.g. a package located in different paths,
a data flow described differently, a field described as optional in one place
and required in another.

**C. LOCKED-DECISION VIOLATIONS**
Any architectural choice that contradicts a decision recorded in DECISIONS.md
without explicitly flagging the contradiction.

**D. MISSING ACCEPTANCE CRITERIA**
Any Phase B gate condition not covered by the deliverables visible in this
document. Gate conditions are:
  - ARCHITECTURE_RFC.md present and complete
  - RISK_TIER_REGISTER.md present with every package tiered
  - CODEBASE_MAP.md present with package registry, interface registry,
    and blast radius quick reference
  - Adapter interface locked (ports.go or ports.py)
  - tests/ directory structure defined in ARCHITECTURE_RFC.md (unit/,
    integration/, and conftest.py for Python or internal/testutil/ for Go)

**E. RISKS AND CONCERNS**
Anything not covered above that a careful engineer would flag before
implementation begins — unclear ownership, missing error handling contracts,
undocumented external dependencies, scalability assumptions, etc.

### Verdict

End your response with a VERDICT line:
  PASS            — no substantive findings
  PASS-WITH-GAPS  — only cosmetic / non-blocking findings
  FAIL            — one or more substantive findings that block the gate

---


---

## DECISIONS.md

# DECISIONS.md — Trailhead

Locked design decisions. Do not re-litigate without a formal RFC (per grill-me
Hard Rules). Read this file before asking any question in a future grill-me,
write-a-prd, or design session.

---

## Retention Policy (locked at Idea Factory — 2026-06-30)

RETENTION POLICY: DELETE
REASONING: Throwaway methodology-validation project per the project brief. Code, repo, worktrees, and session logs are not intended to persist past project close-out; the lasting value is the workshop's routed observations and the react-typescript / chi-router skill content produced along the way.
PRESERVE LIST: N/A (DELETE)
EPHEMERAL LIST: N/A (DELETE)
GITHUB ACCOUNT: Developer-provided — Ali creating a new repo directly. Confirm URL before first `git push` / before Phase D.

**Locked:** yes — cannot change except by a deliberate re-decision logged here with reasoning (see CLAUDE.md Retention Policy section).

---

## Locked From Brief (pre-decided, not re-litigated in grill-me)

These were explicit and unambiguous in the original project brief. Recorded
here so every later phase can grep DECISIONS.md as the single source rather
than re-reading the brief.

- **Tech stack:** Go + `chi` router + REST/JSON API backend; PostgreSQL persistence (JSONB where a document shape fits); React + TypeScript + Tailwind SPA; SPA embedded into the Go binary via `go:embed` — one binary serves API and static assets. Fixed, no substitution.
- **Columns:** exactly three, fixed: Inbox, Reading, Done. Not user-configurable. No column beyond these three.
- **Status ⟺ Column invariant:** a bookmark's column IS its status — one concept, not two fields.
- **finished_at ⟺ Done invariant:** `finished_at` is set if and only if the bookmark is currently in the Done column. Moving a card out of Done clears `finished_at`. This must hold at every write path, not just the initial move.
- **Tags:** free text, lowercased on save, deduplicated per bookmark, empty-string tags are never stored.
- **Absence modeling:** `finished_at` and `author` must be modeled as genuinely absent (nullable / pointer types), never a zero-value or empty-string standing in for "absent." Absent and present-but-empty are distinct states end to end (model, storage, API).
- **Out of scope (do not build):** user accounts/auth/multi-user/sharing; fetching remote page metadata or screenshots (no outbound HTTP to saved URLs); full-text search; browser extension; real-time sync/WebSockets; native mobile apps; any column beyond the three named; pagination/infinite scroll; analytics.
- **Single user, single board.** No multi-tenancy of any kind.

**Decided by:** Brief (developer-authored, pre-Idea-Factory)
**Date:** 2026-06-30 (brief date; confirmed still binding at Phase A grill-me, 2026-07-02)
**Locked:** yes

---

## Data Model

### Position / Ordering Representation

**Decision:** Position within a column is stored as a fractional/lexicographically-sortable rank string (LexoRank-style), one per bookmark row, unique within its column. Moving a card computes a new rank string that sorts strictly between its new neighbors' ranks (or before the first / after the last) — no other row in the column is rewritten on a move.
**Reason:** Reordering must round-trip through storage exactly and must not require rewriting the whole column on every drag (a documented anti-pattern in the brief's "Approaches that have failed before"). A fractional rank gives O(1) writes per move and handles unlimited insertions between any two existing cards without renumbering in the common case. Occasional re-spacing (a maintenance operation, not per-move) is needed only after many insertions collapse into the same gap — this is a Phase B implementation detail, not a Phase A concern.
**Decided by:** Recommendation accepted
**Date:** 2026-07-02
**Locked:** yes — this is one of the two things the brief explicitly says must not change once decided. Changing it post-hoc requires renumbering every existing row.

### Canonical URL — Scheme Normalization

**Decision:** Canonical form always normalizes to `https://`, regardless of the scheme pasted. `http://example.com/x` and `https://example.com/x` canonicalize to the same identity.
**Reason:** Handles the common case of a site migrating to HTTPS or a user pasting a stale http link — both should dedupe to one card. Meaningfully different content served on http vs https for the same host+path is rare enough today not to worry about.
**Decided by:** Recommendation accepted
**Date:** 2026-07-02
**Locked:** yes (part of the canonical-URL rule set — see lock rationale above)

### Canonical URL — Query Parameters

**Decision:** Strip a maintained deny-list of known tracking/marketing parameters (`utm_*`, `gclid`, `fbclid`, `mc_eid`, `ref`, and similar) from the query string, then sort the remaining query keys alphabetically before hashing. Non-tracking params (e.g. `?page=2`) are preserved and remain part of the identity.
**Reason:** Collapses the specific, common duplicate case (same link, different tracking params) while not silently merging bookmarks that differ by a functionally meaningful query param. The deny-list is explicit and versioned in code so canonicalization stays deterministic and reproducible by any implementation following the spec.
**Decided by:** Recommendation accepted
**Date:** 2026-07-02
**Locked:** yes — the *rule* (strip known trackers from this deny-list, sort the remainder) is locked and is what `internal/domain/canonicalize.go` implements now, per the Phase B Scope Boundary exception. The current 12-entry list (`utm_source`, `utm_medium`, `utm_campaign`, `utm_term`, `utm_content`, `utm_id`, `gclid`, `fbclid`, `mc_eid`, `mc_cid`, `ref`, `igshid`) is a real, in-force starter list, not a placeholder — but it is not asserted to be exhaustive. Golden-testing this list's behavior is legitimate Pre-Phase F work (per PRD.md "Open Questions"). *Extending* the list with new entries beyond Pre-Phase F golden-testing is itself a change to a Locked decision — it requires a DECISIONS.md amendment (per this file's own header: "do not re-litigate without a formal RFC"), not a silent code change in a later phase. This reconciles the apparent tension Codex's round-2 review flagged between "Locked: yes" here and PRD.md's "exact deny-list contents still open."

### Canonical URL — Trailing Slash and `www` Prefix

**Decision:** Strip a trailing slash from the path (except a bare root path, which normalizes to `/`). Strip a leading `www.` from the host.
**Reason:** Matches common user expectation that `example.com/x/`, `example.com/x`, and `www.example.com/x` are "the same page" — the most common cosmetic duplicate pattern besides tracking params.
**Decided by:** Recommendation accepted
**Date:** 2026-07-02
**Locked:** yes

### Canonical URL — Fragments

**Decision:** Fragments (`#section-anchor`) are always dropped from the canonical form.
**Reason:** Fragments are client-side scroll anchors never sent to the server; they don't change what page is being saved. (Explicitly not treating fragment-based SPA routing as a target use case — out of scope per the brief.)
**Decided by:** Recommendation accepted
**Date:** 2026-07-02
**Locked:** yes

### Identity Hash

**Decision:** Store both the canonical URL string (human-readable column, useful for debugging/display) and a SHA-256 hash of the canonical URL string (fixed-width, indexed column used for fast exact-match duplicate lookups on add).
**Reason:** The hash is a pure function of the canonical form — two implementations following the same canonicalization spec will always agree. Storing the canonical URL alongside it keeps the derivation inspectable rather than opaque.
**Decided by:** Recommendation accepted
**Date:** 2026-07-02
**Locked:** yes (identity/hash derivation is the other of the two things the brief says must not change once decided — changing it re-buckets every existing bookmark)

### Tags Storage

**Decision:** Tags are stored as a JSONB text array directly on the `bookmarks` row (not a normalized `tags` + join table). A GIN index on the array supports the tag-filter query.
**Reason:** Fits the brief's tech-stack note ("PostgreSQL, using JSONB where a document shape fits") and matches the simplicity of free-text, per-bookmark tagging described in the brief. No cross-bookmark tag identity is needed at single-user scale — renaming a tag globally is not a feature this brief asks for.
**Decided by:** Recommendation accepted
**Date:** 2026-07-02
**Locked:** no — reasonable to revisit if a future extension needs global tag rename/identity, but changing it requires a migration, so treat as a real decision, not a placeholder.

### Title Defaulting

**Decision:** When no title is supplied, derive a default from the URL string only (no remote fetch): hostname + de-slugified last path segment, e.g. `https://example.com/blog/my-great-post` → `"example.com - My Great Post"` (hyphens/underscores → spaces, title-cased). Note: the separator between hostname and path-derived title is a plain hyphen (` - `), matching `internal/domain/canonicalize.go`'s actual `DefaultTitle` implementation — not an em dash.
**Reason:** More readable than the raw URL or bare hostname on a card, without violating the "no outbound HTTP to saved URLs" constraint.
**Decided by:** Recommendation accepted
**Date:** 2026-07-02
**Locked:** yes

### Author Field — Population Path

**Decision:** `author` is a user-editable free-text field, settable only from the card detail/edit view. Nothing in the app auto-populates it (remote metadata fetch is out of scope). It will be absent (null) on the large majority of bookmarks.
**Reason:** Confirms the absent-vs-empty-string modeling requirement is actually exercised by a real, reachable code path (a user can choose to fill it in) rather than being dead schema no code path ever sets.
**Decided by:** Recommendation accepted
**Date:** 2026-07-02
**Locked:** yes

---

## Interface / API

### Reorder / Move Endpoint

**Decision:** The client sends the target column and the IDs of the two neighboring cards at the drop point (or a sentinel for "first"/"last" in the column). The server computes the new fractional rank and persists it; the server is the sole authority on rank validity and uniqueness.
**Reason:** Keeps the ranking scheme's invariants enforced in one place. A client-computed rank would require every consumer (frontend today, any future API client) to re-implement the ranking algorithm correctly and still be validated server-side, which is strictly more surface area for the same guarantee.
**Decided by:** Recommendation accepted
**Date:** 2026-07-02
**Locked:** no — internal API detail, can evolve without re-bucketing data as long as the position representation itself (locked above) doesn't change.

### Duplicate Detection Response

**Decision:** Adding a URL whose identity hash already exists returns HTTP 409 Conflict, with the existing bookmark's full data in the response body.
**Reason:** Clear REST semantics for a conflicting resource state; lets the frontend show "already on your board" and optionally highlight/jump to the existing card using the data it's already given.
**Decided by:** Recommendation accepted
**Date:** 2026-07-02
**Locked:** no

### Delete Semantics

**Decision:** Delete is a hard delete — the row is removed permanently. No `deleted_at` / trash / undo concept.
**Reason:** The brief never mentions an undo or trash concept, and introducing one would be scope creep against the brief's explicit "keep it to the three-column board and its CRUD" instruction (flagged as a failure-mode risk in the Idea Factory review).
**Decided by:** Recommendation accepted
**Date:** 2026-07-02
**Locked:** yes (scope boundary, ties to Idea Factory Condition 2)

### Multi-Tag Filter Logic

**Decision:** Selecting more than one tag filter shows bookmarks matching ANY of the selected tags (OR), not all of them (AND).
**Reason:** Matches the more common "broaden my view across a few related topics" mental model for tag-filter UIs; avoids the surprise of zero results when a user selects two tags meaning to see more, not less.
**Decided by:** Recommendation accepted
**Date:** 2026-07-02
**Locked:** no

---

## Error Handling

### Invalid URL Validation Bar

**Decision:** The Add bar rejects (with inline "invalid URL" feedback) anything that does not parse as an absolute URL with scheme `http` or `https` and a non-empty host. This is a syntactic check only — no reachability check, since outbound fetches to saved URLs are out of scope.
**Reason:** Matches the board's purpose (a web-link reading board) without expanding validation into schemes (`javascript:`, `mailto:`, `ftp:`) the board isn't meant to hold.
**Decided by:** Recommendation accepted
**Date:** 2026-07-02
**Locked:** no

---

### Move — Stale Neighbor Fallback

**Decision:** If a `MoveCommand`'s `Before` or `After` bookmark ID no longer exists (e.g. deleted between drag-start and drop), the repository falls back to inserting at the end of the target column rather than erroring the whole request.
**Reason:** Originally an open question deferred from Phase A's PRD; the Phase B reviewer-agent pass caught that `FakeBookmarkRepository.Move` had already implemented this exact fallback without the decision being formally ratified. A rare client-side staleness edge case that doesn't warrant more design than "pick a safe default, never fail the drag" — ratified as the real decision rather than left as an implicit behavior a future session might not know was never actually decided.
**Decided by:** Developer (ratified from reviewer-flagged implicit behavior, Phase B gate, 2026-07-04)
**Date:** 2026-07-04
**Locked:** no

---

### Author Field — Clearing via Patch

**Decision:** `BookmarkPatch.Author` alone cannot represent "clear this field back to absent," since a nil pointer already means "leave unchanged" for every other patch field. `BookmarkPatch` gets a `ClearAuthor bool` field: `ClearAuthor = true` clears `Author` to nil regardless of the `Author` field's value; otherwise a non-nil `Author` sets a new value and a nil `Author` leaves the existing value unchanged. If a caller sets both `ClearAuthor = true` and a non-nil `Author`, `ClearAuthor` takes precedence (clearing wins) — this is a defensive tie-break, not an expected caller input.
**Reason:** Round-2 Phase B gate review (Codex) surfaced that this project's own "Locked From Brief" absence-modeling rule (`FinishedAt`/`Author` must be genuinely absent, never a zero-value stand-in) was not actually reachable through `BookmarkRepository.Update` once set — there was no way to patch an existing `Author` back to nil. A second nilable pointer (`**string`) would work but is a less obvious read than an explicit bool for a single optional field; the explicit-field approach matches this project's existing convention of wrapping intent in named struct fields (`BoardFilter`, `MoveCommand`) rather than overloading pointer depth.
**Decided by:** Developer (ratified from reviewer-flagged gap, Phase B gate, 2026-07-05)
**Date:** 2026-07-05
**Locked:** yes — this is the interface-level mechanism for a Locked absence-modeling rule; changing it requires a filed RFC per the `internal/adapter/ports.go` READ-ONLY header.

---

### Concurrency / Reorder-Status Conflict Handling

**Decision:** No optimistic-locking or conflict-resolution design for concurrent writes. Last-write-wins on status/position updates. Realistic worst case is the same single user with two browser tabs open, not true multi-user contention.
**Reason:** Originally raised as an assumption (not asked) during Phase A grill-me; ratified as a real decision by Ali during the Phase A gate observation interview (2026-07-02) rather than left as a silent assumption, since it does quietly fix API behavior on a stale reorder. If concurrent-tab conflicts prove to be a real problem in use, that's a DEFER-worthy finding for a future phase, not a Phase A requirement.
**Decided by:** Developer (ratified from a flagged assumption)
**Date:** 2026-07-02
**Locked:** no

---

## Configuration

(No open configuration forks surfaced in the Phase A interview beyond the fixed tech stack already locked from the brief. Standard local dev config — env vars for Postgres connection string, port — is assumed and will be finalized at Phase D scaffolding, not treated as a Phase A decision.)

---

## Testing

(Deferred to Pre-Phase F test-engine sessions per the workshop process. No Phase A testing-approach forks were raised in this interview.)

---

## Assumptions (not asked, assumed from context)

- **Local dev/integration test environment uses a standard Postgres instance** (Docker Compose or local install) with connection config via environment variables, following the workshop's normal Phase D scaffolding pattern. (The other original Phase A assumption — concurrency/last-write-wins — was ratified into a real Decision above during the Phase A gate observation interview; see "Concurrency / Reorder-Status Conflict Handling" under Error Handling.)

---

## Phase B Architecture Decisions (summary — full detail in ARCHITECTURE_RFC.md)

Recorded in ARCHITECTURE_RFC.md rather than duplicated in full here, per
design-an-interface's Phase 6 ("write the final interface definition...
add it to docs/ARCHITECTURE_RFC.md"). Cross-referenced here so a future
session reading only this file doesn't miss them:

- **BookmarkRepository interface design:** Flexible (from design-an-interface's four competing designs) + Ports-and-Adapters typed sentinel errors. Approved by Ali, 2026-07-04. See ARCHITECTURE_RFC.md "Locked Interfaces." **Locked: yes** (interface, not just a preference — READ-ONLY after Phase B gate per `internal/adapter/ports.go`'s own header comment).
- **BookmarkID type:** UUID, represented as a plain Go `string` (not a wrapped third-party type), Postgres native `uuid` column. See ARCHITECTURE_RFC.md "ID Type and Representation." **Locked: yes.**
- **Status Postgres representation:** `text` + `CHECK` constraint, not a native Postgres enum. See ARCHITECTURE_RFC.md "Status — Postgres Representation." **Locked: no** — internal schema choice, open to revisit.
- **Postgres driver:** `jackc/pgx/v5`. See ARCHITECTURE_RFC.md "Postgres Driver." **Locked: no** — implementation-level choice, only affects `internal/adapter/postgres`.
- **`domain.Canonicalize`/`DeriveIdentityHash`/`DefaultTitle` implemented at Phase B, not stubbed** — a deliberate, justified exception to the Phase B Scope Boundary (implementation detail is normally deferred to Pre-Phase F), because DECISIONS.md already specifies their rules completely. See ARCHITECTURE_RFC.md "Scope Boundary." **Locked: yes** (the rule set itself was already locked in Phase A; this just confirms the code now matches it).
- **Fractional-rank position algorithm explicitly NOT implemented yet** — its concrete form is a genuine open architecture question correctly deferred to the Postgres adapter's Phase F issue, unlike canonicalization. See ARCHITECTURE_RFC.md "Scope Boundary."

---

## Phase A Interview Log

Interview conducted 2026-07-02 via Cowork (Ali as developer, real interview — no self-answered forks). 15 questions asked across 4 batches; all resolved (recommendation accepted on every question, no "your call" shortcuts needed, no assumptions forced by hitting a question-count limit).

---

## GLOSSARY.md

[Editorial note: export-phase-b-review.sh looks for GLOSSARY.md at the project root; this project's write-a-prd/ubiquitous-language skills place it at docs/GLOSSARY.md instead. Spliced in manually below — logged as a script/skill path-mismatch finding for the workshop (Finding 47).]

# Glossary — Ubiquitous Language

**Last updated:** 2026-07-02
**Scope:** trailhead

All identifiers in source code, test files, GitHub issues, and documentation
must use these terms exactly as written. Deviations are bugs, not style
choices. Go type names below are proposed at Phase A (pre-code) and must be
verified to match exactly at the Phase B B3 read-back gate once
`internal/domain` and `internal/adapter/ports.go` exist on disk.

---

## Core Domain Concepts

## Bookmark

**Definition:** The core persisted entity — one saved link, with its original URL, canonical URL, identity hash, display title, tags, status, position, optional finished timestamp, and optional author.

**Go type name:** `Bookmark` (in package `domain`)

**Used in:**
- `internal/adapter/ports.go` (repository interface)
- REST API request/response bodies
- All tests

**Not to be confused with:**
- `Board` — the aggregate three-column view of all bookmarks; not a persisted entity itself, just a query/response shape over `Bookmark` rows grouped by `Status`.

---

## Status

**Definition:** Which of the three fixed stages a bookmark is currently in: Inbox, Reading, or Done. This is the same concept the UI calls a "column" — **status** is the canonical code term (Phase A decision, 2026-07-02); "column" is UI/prose language only and must never appear as a Go field, DB column, or API field name.

**Go type name:** `Status` (in package `domain`), a string-backed enum with constants `StatusInbox`, `StatusReading`, `StatusDone`.

**Used in:**
- `Bookmark.Status` field
- Board query/filter parameters
- Move/reorder API requests (target status)

**Not to be confused with:**
- "Column" — UI-facing word for the same concept. Fine in prose, screenshots, and user-facing copy; never in code identifiers.

---

## Position

**Definition:** A bookmark's order within its current `Status`, represented as a fractional/lexicographically-sortable rank string, unique within its `Status` (matching DECISIONS.md "Position / Ordering Representation" wording exactly — not a per-bookmark scope, a per-column one). Lower sorts first. Uniqueness is an application-enforced invariant, not a database constraint — see ARCHITECTURE_RFC.md "Persistence Schema." This is the same concept the brief's prose calls "priority" — **position** is the canonical code term (Phase A decision, 2026-07-02); "priority" is UI/prose language describing the user-facing meaning ("what should I read next") and must never appear as a Go field, DB column, or API field name.

**Go type name:** `Position` (in package `domain`), a string type wrapping the fractional rank value (e.g. `type Position string`).

**Used in:**
- `Bookmark.Position` field
- Move/reorder API requests and the server-side rank computation
- Board rendering order (`ORDER BY position`)

**Not to be confused with:**
- "Priority" — UI-facing word for the same concept, describing the *meaning* of position to the user, never a code identifier.

---

## CanonicalURL

**Definition:** The normalized form of a bookmark's URL, produced by a deterministic rule set (scheme forced to https, tracking query params stripped, remaining query keys sorted, trailing slash and leading `www.` stripped, fragment dropped — see DECISIONS.md § Canonical URL rules). Two URLs that differ only cosmetically produce the same `CanonicalURL`.

**Go type name:** `CanonicalURL` (in package `domain`), a string type, plus a pure function `Canonicalize(rawURL string) (CanonicalURL, error)`.

**Used in:**
- `Bookmark.CanonicalURL` field (stored for debugging/display)
- Input to `IdentityHash` derivation
- Golden tests (Pre-Phase F) verifying determinism of the rule set

**Not to be confused with:**
- `OriginalURL` — the exact string the user pasted, stored unmodified alongside `CanonicalURL`.
- `IdentityHash` — the hash derived *from* the canonical form, used for fast duplicate lookups; not the canonical form itself.

---

## IdentityHash

**Definition:** SHA-256 hash of a bookmark's `CanonicalURL`, used as the indexed lookup key for duplicate-on-add detection. A pure function of `CanonicalURL` — same canonical form always yields the same hash.

**Go type name:** `IdentityHash` (in package `domain`), a fixed-length string (hex-encoded SHA-256, 64 chars). Decided at Phase B (2026-07-04) — resolves the Phase A either/or between a hex string and a raw `[32]byte`; the hex-string form is what `internal/domain/canonicalize.go`'s `DeriveIdentityHash` actually implements, matching the Postgres `text` column it's stored in.

**Used in:**
- `Bookmark.IdentityHash` field (indexed, unique constraint)
- Add-bookmark duplicate check (409 Conflict response path)

**Not to be confused with:**
- `CanonicalURL` — the human-readable input the hash is derived from.

---

## Tag

**Definition:** A free-text label attached to a bookmark. Lowercased on save, deduplicated per bookmark, empty strings never stored. No cross-bookmark tag identity — renaming a tag is per-bookmark, not global (Phase A decision: JSONB array, not a normalized tags table).

**Go type name:** `Tags` (in package `domain`), `type Tags []string` on `Bookmark.Tags`.

**Used in:**
- `Bookmark.Tags` field (Postgres JSONB array, GIN-indexed)
- Board tag-filter query (matches ANY selected tag — OR semantics, Phase A decision)

**Not to be confused with:**
- Nothing else in this domain uses "tag" — no conflict.

---

## FinishedAt

**Definition:** The timestamp a bookmark entered the Done status. Absent (nil) for every bookmark in Inbox or Reading. Set the instant a bookmark's status becomes Done; cleared the instant it leaves Done. This invariant (`FinishedAt` set ⟺ `Status == StatusDone`) is locked in DECISIONS.md and must hold at every write path.

**Go type name:** `FinishedAt *time.Time` (pointer — genuinely absent, not a zero-value `time.Time`) on `domain.Bookmark`.

**Used in:**
- `Bookmark.FinishedAt` field
- Card detail view (shown only when present)

**Not to be confused with:**
- `CreatedAt` / `UpdatedAt` — standard audit timestamps on `domain.Bookmark` (`CreatedAt time.Time`, `UpdatedAt time.Time`), always present, never optional (unlike `FinishedAt`). Both are API-visible per ARCHITECTURE_RFC.md "Serialization Spec" (RFC 3339, always included in the response body). Resolved at Phase B gate (2026-07-05) — previously not formally defined here, which left it ambiguous whether `UpdatedAt` was domain, storage-only, or API-visible (Codex round-2 finding).

---

## Author

**Definition:** An optional, user-editable free-text byline for a bookmark. Absent (nil) on the large majority of bookmarks since nothing in this system auto-populates it (remote metadata fetch is explicitly out of scope). Distinct from an empty string — absence and empty-string are different states end to end.

**Go type name:** `Author *string` (pointer — genuinely absent, not a zero-value empty string) on `domain.Bookmark`.

**Used in:**
- `Bookmark.Author` field
- Card detail / edit view (user-settable)

**Not to be confused with:**
- Nothing else in this domain uses "author" — no conflict.

---

## Board

**Definition:** The full three-column view: all bookmarks grouped by `Status`, ordered by `Position` within each status, optionally filtered by one or more tags (OR semantics). Not a persisted entity — a query/response shape over `Bookmark` rows.

**Go type name:** `Board` (in package `domain`), a struct: `type Board struct { Inbox, Reading, Done []Bookmark }`. Finalized at Phase B (2026-07-04) — see `internal/domain/bookmark.go` and ARCHITECTURE_RFC.md "Locked Interfaces" (`BookmarkRepository.Board` returns `domain.Board`). This replaces the Phase A either/or between this struct form and `map[Status][]Bookmark`; the struct form was chosen for a clearer, self-documenting JSON shape (explicit `inbox`/`reading`/`done` keys rather than a status-string-keyed map).

**Used in:**
- Primary board GET endpoint response

**Not to be confused with:**
- `Bookmark` — an individual card; `Board` is the aggregate of all of them.

---

## BookmarkID

**Definition:** Uniquely identifies a `Bookmark`. Backed by a Postgres native `uuid` column.

**Go type name:** `BookmarkID` (in package `domain`), a plain `string` holding a UUID's canonical text form — see ARCHITECTURE_RFC.md "ID Type and Representation" for why this isn't a wrapped third-party `uuid.UUID` type.

**Used in:**
- `Bookmark.ID` field
- Every `BookmarkRepository` method signature (`internal/adapter/ports.go`)

**Not to be confused with:**
- Nothing else in this domain uses "ID" ambiguously — no conflict.

---

## NewBookmark

**Definition:** The input shape to `BookmarkRepository.Create` — a not-yet-persisted bookmark: the original URL, an optional user-supplied title, and raw (pre-normalization) tags.

**Go type name:** `NewBookmark` (in package `domain`)

**Used in:**
- `BookmarkRepository.Create` parameter

**Not to be confused with:**
- `Bookmark` — the persisted entity `Create` returns. `NewBookmark` never has an `ID`, `CanonicalURL`, `IdentityHash`, `Status`, or `Position` — those are derived/assigned during `Create`.

---

## BookmarkPatch

**Definition:** Describes an edit to an existing `Bookmark`. For `Title` and `Tags`, `nil` means "leave unchanged" (not "clear this field to its zero value"). `Author` is tri-state, not a plain pointer-optional field: a separate `ClearAuthor bool` distinguishes "leave unchanged" (both `Author` nil and `ClearAuthor` false) from "clear to absent" (`ClearAuthor` true) from "set to a new value" (`Author` non-nil). See DECISIONS.md "Author Field — Clearing via Patch" (Phase B gate fix, 2026-07-05).

**Go type name:** `BookmarkPatch` (in package `domain`)

**Used in:**
- `BookmarkRepository.Update` parameter

**Not to be confused with:**
- `NewBookmark` — the creation-time input shape, not an edit to an existing row.

---

## BoardFilter

**Definition:** Narrows a `Board` query to bookmarks matching at least one (OR semantics) of a set of tags. An empty/nil tag list means no filtering.

**Go type name:** `BoardFilter` (in package `adapter`)

**Used in:**
- `BookmarkRepository.Board` parameter

**Not to be confused with:**
- `Board` (domain package) — the query result `BoardFilter` shapes, not the filter itself.

---

## MoveCommand

**Definition:** Describes a drag-and-drop move: which bookmark, its target `Status`, and the IDs of its new neighbors at the drop point (`nil` meaning "first"/"last" in the column). The server (repository implementation), not the client, computes the resulting `Position` from this command — see DECISIONS.md "Reorder/Move Endpoint."

**Go type name:** `MoveCommand` (in package `adapter`)

**Used in:**
- `BookmarkRepository.Move` parameter

**Not to be confused with:**
- `Position` — the value `Move` computes and persists; `MoveCommand` only carries the *intent* (target status + neighbors), never a literal position value.

---

## RepositoryError / ErrorKind

**Definition:** The sole error type every `BookmarkRepository` method returns on failure, always as the built-in `error` interface (never a concrete `*RepositoryError` return type — see go-patterns "Hard Rules" on the nil-interface footgun). `ErrorKind` classifies the failure (`Duplicate`, `NotFound`, `InvalidURL`) so callers use `errors.As` + a `Kind` switch rather than string-matching messages.

**Go type name:** `RepositoryError` and `ErrorKind` (in package `adapter`)

**Used in:**
- Every `BookmarkRepository` method's error return path
- `internal/api` handlers (Phase F) — translates `Kind` into HTTP status codes (409/404/400)

**Not to be confused with:**
- `domain.ErrInvalidURL` — a sentinel error wrapped *inside* `Canonicalize`'s returned error, at the `internal/domain` layer. `RepositoryError{Kind: ErrKindInvalidURL}` is the `internal/adapter`-layer error that wraps `Canonicalize`'s failure for repository callers — two different layers, both legitimately about "invalid URL," not a naming collision to resolve away.

---

## Adapter / Engine Concepts

(N/A — Trailhead has a single storage engine, PostgreSQL. No cross-engine
disambiguation is needed.)

---

## PRD.md

[Editorial note: same path mismatch — this project's docs live at docs/PRD.md. Spliced in manually below.]

# PRD: Trailhead — Personal Read-It-Later Board

**Status:** Draft
**Phase:** A (whole-product MVP spec; sliced into build phases at Phase C)
**Date:** 2026-07-02

## Summary

Trailhead is a personal, single-user read-it-later board. Pasting a URL saves
it with a title (user-supplied or derived from the URL) and tags, and drops
it into an Inbox column. The user drags cards across three fixed columns —
Inbox, Reading, Done — to track progress, reorders cards within a column to
set reading priority, and filters the board by tag. It ships as a single Go
binary (chi API + embedded React/TS/Tailwind SPA) backed by PostgreSQL, with
no accounts and no sharing.

## Goals (must each be verifiable by test, screenshot, or observable user action)

1. Paste a valid URL -> bookmark created in Inbox, duplicate URLs (per canonical-URL rules) rejected with a clear signal instead of a second card.
2. Drag a card between columns -> Status updates and persists; FinishedAt set/cleared per the Done invariant.
3. Drag a card within a column -> Position round-trips through storage, verified at ~500 bookmarks without janky behavior.
4. Tag a bookmark (lowercased, deduplicated, no empty strings) and filter the board by one or more tags (OR semantics).
5. Card detail view shows full URL/tags/status/finished-date, supports edit (title/tags/author, including clearing author back to unset) and hard delete.
6. Full board state survives an application restart (PostgreSQL-backed).
7. Coherent designed visual system (Phase E design.md), designed empty/loading/error states, responsive down to a stacked-column narrow width.

## Non-Goals

Everything in DECISIONS.md "Locked From Brief": accounts/auth/multi-user/sharing, remote metadata fetch, full-text search, browser extension, real-time sync, mobile apps, extra columns, pagination, analytics — permanent, not deferred. Also: soft-delete/trash/undo, global tag identity, formal latency SLA (qualitative "feels instant" at ~500 bookmarks is the bar).

## Affected Modules

| Module / Package | Change type | Notes |
|---|---|---|
| `internal/domain` | new | `Bookmark`, `Status`, `Position`, `CanonicalURL`, `IdentityHash`, `Tags`, `Board`, `NewBookmark`, `BookmarkPatch` |
| `internal/adapter` (`ports.go`) | new | `BookmarkRepository` interface, `BoardFilter`, `MoveCommand`, `RepositoryError`/`ErrorKind` — locked at Phase B |
| `internal/adapter/postgres` | new | PostgreSQL implementation — Phase F, not yet built |
| `internal/api` (chi handlers) | new | REST/JSON endpoints — Phase F, not yet built |
| `web/` (React/TS/Tailwind SPA) | new | Phase E/F, not yet built |
| `cmd/trailhead` | new | `main.go` — minimal skeleton exists, Postgres/API wiring deferred to Phase F |
| `internal/testutil`, `tests/integration` | new | Fake repository built at Phase B; integration tests are a Phase F deliverable |

## Acceptance Criteria (Gherkin-style — abbreviated list; see docs/PRD.md for full text)

Twelve scenarios covering: add-bookmark success, duplicate-on-add (409 + existing bookmark), invalid-URL rejection, cross-column drag (+ FinishedAt set/clear), within-column reorder (+ round-trip), tag add/normalize, tag filter (OR) + clear, card detail view (title/tags/author edit controls), edit-and-persist (including author clearing), hard delete, restart persistence, empty-column state, responsive stacking.

## Error Conditions

Invalid URL -> 4xx, inline message. Duplicate -> reject create, return existing bookmark's data. Stale move neighbor reference -> **(resolved at Phase B gate, see DECISIONS.md "Move — Stale Neighbor Fallback")** falls back to end-of-column rather than failing the request. PostgreSQL unreachable -> 5xx, designed error state.

## Open Questions

- ~~Exact fallback behavior when a move/reorder request's neighbor references are stale~~ — **Resolved at Phase B gate (2026-07-04):** falls back to inserting at the end of the target column. See DECISIONS.md "Move — Stale Neighbor Fallback."
- ~~Whether Author can be explicitly cleared back to unset via the edit view~~ — **Resolved at Phase B gate (2026-07-05):** `BookmarkPatch.ClearAuthor bool`. See DECISIONS.md "Author Field — Clearing via Patch."
- Exact deny-list contents for tracking-parameter stripping — DECISIONS.md locks the *rule* as Locked; the starter list (`utm_source`, `utm_medium`, `utm_campaign`, `utm_term`, `utm_content`, `utm_id`, `gclid`, `fbclid`, `mc_eid`, `mc_cid`, `ref`, `igshid`) is real and in force. Golden-testing it is legitimate Pre-Phase F work; extending it is a Locked-decision change requiring a DECISIONS.md amendment, not a silent code change.
- Exact HTTP routes/request/response bodies for `internal/api` — deliberately Pre-Phase F implementation detail per CLAUDE.md's Phase B Scope Boundary; resolved per-issue via the REST Adapter Wire Contract gate.
- Drag-and-drop library choice (e.g. dnd-kit vs. hand-rolled) — deferred to Phase B/E design-system decision, must resolve before any Phase F board-UI issue begins.

---

## ARCHITECTURE_RFC.md

[Editorial note: same path mismatch — this project's RFC lives at docs/ARCHITECTURE_RFC.md. Spliced in manually below.]

# ARCHITECTURE_RFC.md — Trailhead

**Phase:** B | **Date:** 2026-07-04

## Scope Boundary

This RFC LOCKS Modules, Interfaces (ports), the persistence schema, and architectural decisions. It DEFERS per-issue implementation detail (function bodies, concrete algorithms) to Pre-Phase F prep. Two deliberate exceptions: `domain.Canonicalize`, `domain.DeriveIdentityHash`, `domain.DefaultTitle` are implemented now (not stubbed) because DECISIONS.md already specifies their rules completely. The fractional-rank position algorithm is explicitly NOT implemented yet — a genuine open architecture question deferred to the Postgres adapter's Phase F issue.

Two boundary notes added at the Phase B gate fix (2026-07-05), reconciling apparent locked-vs-open tension Codex's round-2 review flagged: (1) the canonical-URL tracking-param deny-list *rule* is Locked and implemented now; the current 12-entry list is real and in force, not a placeholder; golden-testing it is legitimate Pre-Phase F work, extending it requires a DECISIONS.md amendment. (2) Exact `internal/api` HTTP routes/request/response bodies are explicitly Pre-Phase F implementation detail — the Serialization Spec below locks wire-level conventions; routes are specified per-issue via the REST Adapter Wire Contract gate.

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

Allowed: adapter -> domain; adapter/postgres -> adapter, domain; api -> adapter, domain; testutil -> adapter, domain; cmd -> all packages.

Forbidden (P1 merge blocker): domain -> adapter; domain -> adapter/postgres; domain -> api; adapter -> adapter/postgres; api -> adapter/postgres; anything importing testutil in production code.

## ID Type and Representation

**Decision:** `domain.BookmarkID` is a plain Go `string` holding a UUID's canonical text form, backed by a native Postgres `uuid` column (`gen_random_uuid()` via `pgcrypto`). **Reason:** avoids exposing a sequential bookmark count; keeps `internal/domain` free of any non-stdlib dependency. **Locked: yes.**

## Status — Postgres Representation

**Decision:** `status` is Postgres `text` + `CHECK (status IN ('inbox','reading','done'))`, not a native enum. **Reason:** native enums are painful to alter; CHECK gives the same guarantee with a trivial migration path. **Locked: no.**

## Postgres Driver

**Decision:** `github.com/jackc/pgx/v5`, used directly. **Reason:** first-class Postgres-native type support (uuid, jsonb), actively maintained, standard pooling via pgxpool. **Locked: no.**

## Locked Interfaces

`BookmarkRepository` is the Tier 1 contract. Full definition lives in `internal/adapter/ports.go` (READ-ONLY after this gate; changes require a filed RFC):

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

Design chosen: **Flexible** (from design-an-interface's four competing designs — Minimal, Flexible, Common-Case-First, Ports-and-Adapters), with Ports-and-Adapters' typed-sentinel-error convention folded in. Approved by developer, 2026-07-04.

## Data Flow

Browser (web/ SPA) --REST/JSON--> internal/api (chi handlers) --calls BookmarkRepository methods--> internal/adapter.BookmarkRepository (Interface/Seam) --> production: internal/adapter/postgres.Repository --SQL--> PostgreSQL; tests: internal/testutil.FakeBookmarkRepository (in-memory).

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
    finished_at    timestamptz,
    author         text,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX bookmarks_identity_hash_idx ON bookmarks (identity_hash);
CREATE INDEX bookmarks_status_position_idx ON bookmarks (status, position);
CREATE INDEX bookmarks_tags_gin_idx ON bookmarks USING gin (tags);
```

`identity_hash` unique index makes duplicate detection an atomic, race-safe DB guarantee — `Create`'s Phase F implementation should catch the unique-violation error and translate it to `RepositoryError{Kind: ErrKindDuplicate}` by re-querying, not rely on check-before-insert alone. `finished_at ⟺ status = 'done'` is an application-enforced invariant, not a DB CHECK constraint — a conscious choice given the repository implementation is the only writer, documented as a reasonable Phase F upgrade if a second line of defense is ever needed. **Position uniqueness within its Status is likewise application-enforced, not a database constraint** — `bookmarks_status_position_idx` is non-unique (ordering support only); the Postgres adapter's `Move` implementation (Phase F) is the sole writer responsible for computing non-colliding fractional ranks; a DB-level `UNIQUE (status, position)` constraint is a reasonable Phase F hardening step, same category of choice as `finished_at`.

## Serialization Spec

On-disk: `tags` is Postgres jsonb array of lowercase strings (empty `[]`, never null); `position` is text (concrete format deferred to Phase F); `finished_at`/`author` are nullable columns, real SQL NULL never empty-string/zero-value stand-ins.

On-the-wire (REST/JSON): timestamps RFC 3339/ISO 8601 UTC (Go's default `time.Time` marshaling); `CreatedAt`/`UpdatedAt` are ordinary (non-pointer) `domain.Bookmark` fields, always present, API-visible per this spec; `finished_at`/`author` are Go pointer types marshaling to JSON `null` when nil, always included (no `omitempty`) so an explicit null is distinguishable from a client forgetting to check; `tags` always a JSON array (never null); `id` is the UUID's canonical string form, unchanged at the API boundary.

## tests/ Directory Structure

`internal/testutil/fake_repository.go` — `FakeBookmarkRepository`, built now, satisfying four adversarial invariants (exact IdentityHash match, typed `*RepositoryError` never a plain error, context-cancellation checked before mutating state, nil-pointer optional fields never masquerading as zero values). `tests/integration/` — end-to-end tests against real Postgres, `//go:build integration`, not yet populated (Phase F). Unit tests live alongside their package as idiomatic Go `*_test.go` files, not under a separate `tests/unit/` directory — deliberate Go-convention deviation from the Python-style layout, per go-workflow-observations.md Finding 3 precedent.

## B3 Verification

Performed 2026-07-04 against the on-disk skeleton: every port interface method signature in `internal/adapter/ports.go` matches this RFC's "Locked Interfaces" block exactly (Create, Board, Move, Update, Delete — all five, including parameter names/order); all imports in ports.go are correct (context, trailhead/internal/domain, both referenced, no unused imports); data flow diagram matches port signatures; doc comments reference correct parameter names; optional/absent-on-real-data fields are Go pointer types (FinishedAt *time.Time, Author *string, BookmarkPatch.Title/.Tags/.Author, MoveCommand.Before/.After). No mismatches found. B3 verification: PASS. (Independently re-verified by the Phase B reviewer-agent sub-agent pass, which confirmed this claim by re-reading ports.go rather than trusting the self-report — see PASS-WITH-GAPS verdict in the project's Phase B gate record.)

**B3 Addendum (2026-07-05, Phase B gate fix batch):** Codex's round-2 Phase B review returned FAIL with six substantive cross-document findings (Board glossary staleness, Move stale-neighbor fallback undocumented in ports.go, Position uniqueness scope disagreement + no schema-level enforcement documented, Author editing omitted from PRD.md, Title Defaulting example contradicting its own stated rule, UpdatedAt status ambiguous) plus a newly-ratified `BookmarkPatch.ClearAuthor` mechanism. All applied in lockstep across DECISIONS.md, docs/GLOSSARY.md, docs/PRD.md, this file, ports.go, bookmark.go, and fake_repository.go. `ClearAuthor bool` does not change any `BookmarkRepository` method signature, so the B3 signature-match check is unaffected. A subsequent fresh reviewer-agent pass caught one residual cosmetic mismatch (Title Defaulting example used an em dash while the code produces a plain hyphen) — corrected. Two items (deny-list exhaustive contents, exact API routes/request/response shapes) were classified as legitimate Pre-Phase F deferrals with boundary notes, not specced now.

## Greenfield Skeleton

Materialized on disk before this gate closes: `go.mod` (module trailhead, Go 1.22, requires go-chi/chi/v5 and jackc/pgx/v5; go.sum not yet generated); `internal/domain/bookmark.go` (all types); `internal/domain/canonicalize.go` (Canonicalize, DeriveIdentityHash, DefaultTitle); `internal/adapter/ports.go` (BookmarkRepository + supporting types, READ-ONLY header); `internal/testutil/fake_repository.go` (FakeBookmarkRepository, compile-time interface-compliance assertion included); `cmd/trailhead/main.go` (minimal compiling entry point, health check only, Postgres/API wiring deferred to Phase F). `go build ./...` and `go vet ./...` must be run and confirmed clean in Terminal on the developer's Mac before this gate closes (no Go toolchain in the Cowork sandbox).

---

## RISK_TIER_REGISTER.md

# Architecture RFC — Risk Tier Register

## What This Table Is

Every package in this project is assigned a blast radius tier before
any code is written. The tier determines:

  - Interview depth in the test-engine (Tier 1: 20Q, Tier 2: 12Q, Tier 3: 6Q)
  - Review depth in the middle loop (Tier 1: line-by-line, Tier 3: tests passing is sufficient)
  - Security scan response (Tier 1 finding: block merge, Tier 3: assess first)
  - Session pairing (Tier 1 packages: developer co-builds at least one slice)

Tiers are assigned at architecture time and do not change without a
formal RFC amendment. If a package grows to include Tier 1 concerns
that weren't anticipated, file an RFC before writing that code.

---

## Blast Radius Tier Definitions

### Tier 1 — Critical
**Blast radius if wrong: data loss, security breach, or compliance failure.**

Characteristics:
  - Handles authentication, authorisation, or credential management
  - Writes to persistent storage (creates, updates, or deletes records)
  - Generates compliance evidence or audit logs
  - Processes or transmits data classified as sensitive (PII, health, financial)
  - Controls access to external systems or APIs

Review requirements:
  - Human review of every line before merge
  - Developer co-builds at least one vertical slice in this package
  - test-engine interview: maximum 20 questions, adversarial stance
  - gosec finding in Tier 1 package: block merge, no exceptions
  - Architecture checkpoint required before adding new Tier 1 packages

### Tier 2 — Standard
**Blast radius if wrong: incorrect results, degraded performance, or poor UX.**

Characteristics:
  - Core business logic that does not touch Tier 1 concerns
  - API endpoints that read (but do not write) sensitive data
  - Diagnostic and monitoring logic
  - Report generation and rendering

Review requirements:
  - Human review of the diff as a whole (not necessarily line by line)
  - test-engine interview: maximum 12 questions
  - gosec finding in Tier 2 package: assess whether it is a true positive

### Tier 3 — Low Risk
**Blast radius if wrong: incorrect formatting, minor errors, or inconvenience.**

Characteristics:
  - Formatters, utilities, and helper functions
  - CLI output formatting
  - Read-only queries that return non-sensitive data
  - Test fixtures and seed helpers

Review requirements:
  - Tests passing is sufficient for merge
  - test-engine interview: maximum 6 questions, mostly auto-generated
  - gosec finding in Tier 3 package: assess, likely acceptable with comment

---

## Package Tier Register

Fill this table during the Architecture RFC session (Phase B).
Every internal package must have a tier assigned before Phase C begins.

| Package path                    | Tier | Reason                                         |
|---------------------------------|------|------------------------------------------------|
| internal/adapter/adapter.go     | 1    | Interface contract — all data flows through it |
| internal/adapter/mongo/         | 2    | Reads engine state, does not write user data   |
| internal/adapter/elastic/       | 2    | Reads engine state, does not write user data   |
| internal/adapter/redis/         | 2    | Reads engine state, does not write user data   |
| internal/adapter/cassandra/     | 2    | Reads engine state, does not write user data   |
| internal/snapshot/              | 2    | Collects and stores operational snapshots      |
| internal/diff/                  | 2    | Core AWR comparison logic                      |
| internal/rules/                 | 2    | Evaluates diagnostic rules against diffs       |
| internal/report/                | 2    | Renders diagnostic reports (HTML/PDF/JSON)     |
| internal/storage/               | 1    | Writes all persistent data to PostgreSQL       |
| internal/monitor/               | 2    | Alerting and metric streaming                  |
| internal/audit/                 | 1    | Compliance evidence — SOC2/HIPAA artifacts     |
| internal/api/                   | 2    | REST API (reads and writes)                    |
| cmd/<project>/                    | 3    | CLI entry point, output formatting             |
| cmd/<project>d/                   | 3    | Daemon entry point                             |
| pkg/                            | 3    | Public types and client SDK                    |
| test/integration/               | 3    | Test infrastructure — no production impact     |
| web/                            | 3    | React UI — client-side only                    |

NOTE: This table is for any future project as an example. Each project must
produce its own tier register during its Phase B Architecture RFC.
Populate using the blank template below.

---

## Trailhead — Package Tier Register (Phase B, 2026-07-04)

| Package path                      | Tier | Reason |
|------------------------------------|------|--------|
| `internal/adapter` (`ports.go`)    | 1    | Interface contract — all data flows through it; a signature drift here breaks every caller silently until compile time, or worse, at runtime if types were loosely typed. |
| `internal/adapter/postgres`        | 1    | Writes all persistent data (creates/updates/deletes bookmarks); a bug here is data loss or data corruption — the canonical Tier 1 characteristic. Also the sole enforcement point for the `finished_at ⟺ Done` invariant and duplicate-detection race-safety (unique index + conflict handling). |
| `internal/domain`                  | 1    | Silent-failure-check applied (per go-workflow-observations.md Phase B lesson from a prior project): `Canonicalize`/`DeriveIdentityHash` are pure functions with no error-raising failure mode if the *rules themselves* are subtly wrong — a wrong tracking-param deny-list entry or a wrong trailing-slash rule silently lets duplicates through or silently over-merges distinct bookmarks, across every future Create call, with no exception ever thrown. This is exactly the "could a bug here degrade output quality silently without raising an error?" test that promotes a package to Tier 1. |
| `internal/api`                     | 2    | Core business logic (HTTP <-> repository translation) that does not itself touch storage or credentials; reads and writes via the Tier 1 repository interface but the interface's own contract is what's load-bearing, not the handler glue. |
| `internal/testutil`                | 3    | Test infrastructure only — no production impact. A bug here produces a misleading test result, not a production incident (though see the four adversarial invariants in fake_repository.go — those exist precisely to keep this package trustworthy despite its Tier 3 review depth). |
| `cmd/trailhead`                    | 3    | Entry point / wiring only — CLI/server bootstrap, output formatting, no business logic of its own. |
| `web/`                             | 2    | Client-side React/TS UI — incorrect drag/reorder behavior or filter logic produces degraded UX (a core brief concern — "what good looks like") but no data loss, since all persistence decisions are server-side. Not Tier 1: a UI bug is recoverable by reload; a repository bug is not. |
| `tests/integration/`               | 3    | Test infrastructure — no production impact. |

---

## Blank Template (copy for new projects)

| Package path | Tier | Reason |
|--------------|------|--------|
| | | |

---

## Tier Amendment Process

If a package's tier needs to change after the RFC is approved:

1. File a GitHub issue titled "RFC: Tier change — [package] from Tier X to Tier Y"
2. Describe: what changed about the package's responsibilities?
3. What additional test coverage is needed for the new tier?
4. What review process changes are needed?
5. Approved by developer review before any code in the package is written

Do not change a tier silently. The tier is a commitment, not a label.

---

## CODEBASE_MAP.md

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
| `internal/adapter/postgres` | 1 | `internal/adapter`, `internal/domain` | The running application's actual data — this is the only package that writes to PostgreSQL. Not yet built (Phase F). |
| `internal/api` | 2 | `internal/adapter`, `internal/domain` | The HTTP contract `web/` depends on — a handler bug produces a wrong status code or malformed JSON, but does not touch storage directly. Not yet built (Phase F). |
| `internal/testutil` | 3 | `internal/adapter`, `internal/domain` | Only test correctness — Pre-Phase F test-engine sessions and any future `api`-layer unit tests depend on `FakeBookmarkRepository` behaving like the real repository's documented contract. Never imported by production code. |
| `cmd/trailhead` | 3 | all packages (wiring only) | The running binary's startup behavior — if this breaks, the app doesn't start, but no other package's correctness is affected. |
| `web/` | 2 | none (Go side) — communicates with `internal/api` over REST/JSON only | The user-facing board experience — the brief's "what good looks like" bar (drag-and-drop feel, empty/loading/error states). Not yet built (Phase E/F). |
| `tests/integration/` | 3 | `internal/adapter/postgres` (via build tag `integration`) | Only CI confidence that the Postgres adapter works against a real database — no production impact. Not yet built (Phase F). |

---

## 2. Interface Registry

| Interface | Defined in | Implementors | Consumers |
|---|---|---|---|
| `BookmarkRepository` | `internal/adapter/ports.go` | `internal/adapter/postgres.Repository` (Phase F, not yet built); `internal/testutil.FakeBookmarkRepository` (built) | `internal/api` handlers (Phase F, not yet built); `cmd/trailhead/main.go` at wiring time (currently a TODO — no repository wired yet); any test file importing `internal/testutil` |

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

---

## PHASE_PLAN.md

_File not found: /sessions/elegant-happy-pascal/mnt/dev/projects/trailhead/PHASE_PLAN.md_

---

## Adapter Interface — ports.go (internal/adapter/ports.go)

// READ-ONLY after Phase B gate. Every signature in this file is copied
// verbatim from ARCHITECTURE_RFC.md ("Locked Interfaces"). Do not modify.
// Any proposed change requires a filed RFC (GitHub issue, via the
// improve-codebase-architecture skill) and explicit developer approval
// before any code is touched. A modification here is a P1 finding - block
// merge, no exceptions.
//
// Package adapter defines the single port through which all of Trailhead's
// persistence flows: BookmarkRepository. This file imports internal/domain
// only - see ARCHITECTURE_RFC.md "Package Organization" for the locked
// import-direction rules. Concrete implementations (the Postgres adapter)
// live in a subpackage (internal/adapter/postgres) and are NOT imported
// here - the interface belongs to the consumer side of the boundary, not
// the implementor, per go-patterns "Interface Location".
package adapter

import (
	"context"

	"trailhead/internal/domain"
)

// BoardFilter narrows a Board query. Wrapped in a struct (rather than a
// bare []string parameter) so that future filter dimensions (e.g. a search
// term, a date range) can be added without breaking BookmarkRepository's
// method signature - see ARCHITECTURE_RFC.md "Locked Interfaces" for the
// design rationale (the "Flexible" design from design-an-interface).
type BoardFilter struct {
	// Tags selects bookmarks matching at least one of these tags (OR
	// semantics - see DECISIONS.md "Multi-Tag Filter Logic"). An empty or
	// nil slice means no tag filtering - the full Board is returned.
	Tags []string
}

// MoveCommand describes a drag-and-drop move: the bookmark being moved,
// its target Status, and the IDs of its new neighbors at the drop point.
// Before and After are nil to mean "first in column" / "last in column"
// respectively. Wrapped in a struct for the same future-extension reason
// as BoardFilter.
type MoveCommand struct {
	ID           domain.BookmarkID
	TargetStatus domain.Status
	Before       *domain.BookmarkID
	After        *domain.BookmarkID
}

// ErrorKind classifies a RepositoryError. Every BookmarkRepository method
// that can fail in a way the API layer must distinguish returns a
// *RepositoryError with one of these kinds - never a plain errors.New()
// string error - so that api handlers can use errors.As and switch on Kind
// rather than string-matching error messages.
type ErrorKind string

const (
	// ErrKindDuplicate: Create was called with a URL whose IdentityHash
	// already exists. RepositoryError.Existing carries the pre-existing
	// Bookmark - see DECISIONS.md "Duplicate Detection Response" (API
	// layer responds 409 Conflict with Existing in the body).
	ErrKindDuplicate ErrorKind = "Duplicate"

	// ErrKindNotFound: the referenced BookmarkID does not exist.
	ErrKindNotFound ErrorKind = "NotFound"

	// ErrKindInvalidURL: the OriginalURL in a NewBookmark failed
	// domain.Canonicalize's validation - see DECISIONS.md "Invalid URL
	// Validation Bar".
	ErrKindInvalidURL ErrorKind = "InvalidURL"
)

// RepositoryError is the sole error type BookmarkRepository methods return
// on failure. Always returned as the built-in error interface (never as a
// concrete *RepositoryError return type) - see go-patterns "Hard Rules": a
// nil *RepositoryError boxed in a non-pointer error-typed return would be
// a non-nil interface, a classic Go footgun.
type RepositoryError struct {
	Kind ErrorKind

	// Existing is populated only when Kind == ErrKindDuplicate.
	Existing *domain.Bookmark

	Message string
	Wrapped error
}

func (e *RepositoryError) Error() string { return e.Message }
func (e *RepositoryError) Unwrap() error { return e.Wrapped }

// BookmarkRepository is the single port through which the api package
// reads and writes bookmarks. It is the Tier 1 contract for this project -
// see RISK_TIER_REGISTER.md. Every method takes context.Context as its
// first parameter (go-patterns "Context Handling" - all I/O must be
// cancellable).
//
// Context Completeness Check (design-an-interface Phase 5): Board is the
// only output-affecting method: BoardFilter.Tags carries the sole optional
// user-supplied context that shapes its output, and it is present on every
// call.
type BookmarkRepository interface {
	// Create persists a new Bookmark in Status = StatusInbox at the front
	// of that column's ordering. If a Bookmark with the same
	// domain.IdentityHash (derived from the canonicalized OriginalURL)
	// already exists, Create returns a *RepositoryError with
	// Kind = ErrKindDuplicate and Existing set to the pre-existing
	// Bookmark - it does not create a second row. If OriginalURL fails
	// domain.Canonicalize's validation, Create returns a *RepositoryError
	// with Kind = ErrKindInvalidURL.
	Create(ctx context.Context, b domain.NewBookmark) (domain.Bookmark, error)

	// Board returns every Bookmark grouped by Status and ordered by
	// Position within each Status, restricted by filter per BoardFilter's
	// semantics.
	Board(ctx context.Context, filter BoardFilter) (domain.Board, error)

	// Move changes a Bookmark's Status and/or its Position within a
	// Status, per cmd. If cmd.TargetStatus == domain.StatusDone and the
	// Bookmark's current Status is not Done, Move sets FinishedAt to the
	// current time. If the Bookmark's current Status is Done and
	// cmd.TargetStatus is not, Move clears FinishedAt to nil - see
	// DECISIONS.md "FinishedAt <-> Done invariant" (Locked - this
	// invariant must hold on every call, not just the common case).
	// If cmd.Before or cmd.After references a BookmarkID that no longer
	// exists (e.g. deleted between drag-start and drop), Move falls back
	// to inserting at the end of the target column rather than failing
	// the request - see DECISIONS.md "Move - Stale Neighbor Fallback".
	// Returns a *RepositoryError with Kind = ErrKindNotFound if cmd.ID
	// does not exist.
	Move(ctx context.Context, cmd MoveCommand) (domain.Bookmark, error)

	// Update applies patch to the Bookmark identified by id. For Title and
	// Tags, a nil field in patch is unchanged - standard pointer-optional
	// semantics. Author is tri-state: patch.ClearAuthor = true clears
	// Author to nil regardless of patch.Author; otherwise a non-nil
	// patch.Author sets a new value and nil leaves the existing value
	// unchanged - see domain.BookmarkPatch and DECISIONS.md "Author Field
	// - Clearing via Patch". Returns a *RepositoryError with
	// Kind = ErrKindNotFound if id does not exist.
	Update(ctx context.Context, id domain.BookmarkID, patch domain.BookmarkPatch) (domain.Bookmark, error)

	// Delete permanently removes the Bookmark identified by id (hard
	// delete - see DECISIONS.md "Delete Semantics", no trash/undo).
	// Returns a *RepositoryError with Kind = ErrKindNotFound if id does
	// not exist.
	Delete(ctx context.Context, id domain.BookmarkID) error
}

---

## End of context

All relevant Phase B artifacts for **trailhead** are above.
Produce your structured gap report (categories A–E) and end with a VERDICT line.
