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

### Move — Neighbor Fallback (Generalized)

**Decision:** `MoveCommand.Before`/`After` express intent, not a validated reference. Move never fails the request because of a bad neighbor. A single fallback rule covers every way a neighbor reference can fail to resolve: (1) missing/stale — the ID no longer exists (deleted between drag-start and drop, the original Phase B gate finding); (2) cross-status — the ID exists but is not in `cmd.TargetStatus`; (3) self-referential — the ID equals `cmd.ID` itself. In every one of these cases, the repository falls back to inserting at the end of the target column rather than erroring the whole request. Separately — and this is a tie-break, not a fourth fallback case — if both `Before` and `After` are set on the same call, `Before` takes precedence and `After` is ignored outright; the repository does not detect or reconcile the two disagreeing with each other, and does not fall back to end-of-column for that combination specifically.
**Reason:** Round-2 (2026-07-04) ratified only the missing/stale case, discovered because `FakeBookmarkRepository.Move` had already implemented that one case without the decision being formally recorded. Round-3 Codex review (2026-07-05) generalized the rule to cover cross-status and self-referential neighbors, which `FakeBookmarkRepository.Move`'s existing search-and-fallback-to-`len(targetOrder)` logic already handled for free. Round-3's wording also claimed the same fallback applied when `Before` and `After` "disagree with each other" — round-4 Codex review (2026-07-05) correctly found this did not match the actual implementation, which checks `Before` first and never inspects `After` at all once `Before != nil`. Rather than add new consistency-checking logic to a Tier 3 test double to make the contract's prose true, this decision is narrowed to document the simpler behavior the code already has: `Before` wins, `After` is silently ignored when both are set. Adding real precedence-conflict handling (e.g. erroring, or a documented merge rule) is Phase F implementation detail if it ever proves necessary, not a Phase B gate blocker.
**Decided by:** Developer (ratified from reviewer-flagged implicit behavior, generalized at Phase B gate round 3, 2026-07-05; narrowed at Phase B gate round 4, 2026-07-05)
**Date:** 2026-07-04 (original scope); generalized 2026-07-05; narrowed 2026-07-05
**Locked:** no

---

### Author Field — Clearing via Patch

**Decision:** `BookmarkPatch.Author` alone cannot represent "clear this field back to absent," since a nil pointer already means "leave unchanged" for every other patch field. `BookmarkPatch` gets a `ClearAuthor bool` field: `ClearAuthor = true` clears `Author` to nil regardless of the `Author` field's value; otherwise a non-nil `Author` sets a new value and a nil `Author` leaves the existing value unchanged. If a caller sets both `ClearAuthor = true` and a non-nil `Author`, `ClearAuthor` takes precedence (clearing wins) — this is a defensive tie-break, not an expected caller input.
**Reason:** Round-2 Phase B gate review (Codex) surfaced that this project's own "Locked From Brief" absence-modeling rule (`FinishedAt`/`Author` must be genuinely absent, never a zero-value stand-in) was not actually reachable through `BookmarkRepository.Update` once set — there was no way to patch an existing `Author` back to nil. A second nilable pointer (`**string`) would work but is a less obvious read than an explicit bool for a single optional field; the explicit-field approach matches this project's existing convention of wrapping intent in named struct fields (`BoardFilter`, `MoveCommand`) rather than overloading pointer depth.
**Decided by:** Developer (ratified from reviewer-flagged gap, Phase B gate, 2026-07-05)
**Date:** 2026-07-05
**Locked:** yes — this is the interface-level mechanism for a Locked absence-modeling rule; changing it requires a filed RFC per the `internal/adapter/ports.go` READ-ONLY header.

---

### Repository Error Taxonomy — Infrastructure Failures

**Decision:** `ErrorKind`/`RepositoryError` classify only failures the API layer must distinguish into different HTTP response shapes: `Duplicate` (409), `NotFound` (404), `InvalidURL` (400). Infrastructure failures — PostgreSQL unreachable, network errors, context cancellation/timeout — are NOT added as a new `ErrorKind`. Instead, `BookmarkRepository` methods return them as a plain wrapped `error` (the built-in interface, never a `*RepositoryError`). The `internal/api` layer's translation logic is: attempt `errors.As(err, &repoErr)`; on success, map `repoErr.Kind` to its specific 4xx; on failure, treat the error as an infrastructure failure and return a uniform 5xx. This reconciles PRD.md's "PostgreSQL unreachable → 5xx" Error Condition with the `RepositoryError` taxonomy.
**Reason:** Round-3 Codex review flagged that the PRD requires a 5xx response for infrastructure failure but the taxonomy only defined three classified kinds, leaving infra failures unrepresentable. Adding a fourth `ErrKind` (e.g. `ErrKindUnavailable`) was considered but rejected: every infrastructure failure maps to the same undifferentiated 5xx regardless of specific cause, so there is no HTTP-status-relevant distinction for a Kind to carry — unlike Duplicate/NotFound/InvalidURL, which each drive a genuinely different response. This also matches the on-disk precedent already built at Phase B: `FakeBookmarkRepository.checkContext` already returns `ctx.Err()` directly (a bare error), not a `*RepositoryError` — the fake was already exercising this exact pattern before the decision was formally ratified, same as the neighbor-fallback precedent above.
**Decided by:** Developer (ratified from reviewer-flagged gap, Phase B gate, 2026-07-05)
**Date:** 2026-07-05
**Locked:** yes — this is the interface-level error-handling contract for the Tier 1 `BookmarkRepository`; changing it requires a filed RFC per the `ports.go` READ-ONLY header.

---

### UpdatedAt — Write-Path Contract

**Decision:** Every mutating `BookmarkRepository` method sets `UpdatedAt` to the current time on every successful call: `Create` sets it (equal to `CreatedAt` at creation time), `Move` sets it on every successful move (regardless of whether `Status` actually changed), `Update` sets it on every successful patch application (regardless of whether any field's value actually changed). `Delete` removes the row entirely — no `UpdatedAt` semantics apply to a resource that no longer exists.
**Reason:** Round-3 Codex review flagged that `UpdatedAt` was API-visible and always-present per the Serialization Spec, but no document stated which write paths actually update it, leaving room for a Phase F implementation to update it inconsistently (e.g. only on `Update`, not `Move`). `internal/testutil/fake_repository.go`'s existing implementation already sets `UpdatedAt` on `Create`, `Move`, and `Update` — this decision formalizes that already-built behavior as the locked contract rather than an implementation detail a future session might diverge from.
**Decided by:** Developer (ratified from reviewer-flagged gap, Phase B gate, 2026-07-05)
**Date:** 2026-07-05
**Locked:** yes — part of the Tier 1 `BookmarkRepository` contract.

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
