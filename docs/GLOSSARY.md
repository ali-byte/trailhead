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

**Definition:** A bookmark's order within its current `Status`, represented as a fractional/lexicographically-sortable rank string, unique within (Bookmark, Status). Lower sorts first. This is the same concept the brief's prose calls "priority" — **position** is the canonical code term (Phase A decision, 2026-07-02); "priority" is UI/prose language describing the user-facing meaning ("what should I read next") and must never appear as a Go field, DB column, or API field name.

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

**Go type name:** `IdentityHash` (in package `domain`), a fixed-length string (hex-encoded SHA-256, 64 chars) or `[32]byte`.

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
- `CreatedAt` — when the bookmark was first added (always present, not optional). Not yet formally defined as a glossary term since it's a standard audit timestamp, but noted here to avoid confusing the two.

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

**Go type name:** `Board` (in package `domain` or a response DTO in the API layer — to be finalized at Phase B; likely `map[Status][]Bookmark` or an explicit `Board{Inbox, Reading, Done []Bookmark}` struct for clearer JSON shape).

**Used in:**
- Primary board GET endpoint response

**Not to be confused with:**
- `Bookmark` — an individual card; `Board` is the aggregate of all of them.

---

## Adapter / Engine Concepts

(N/A — Trailhead has a single storage engine, PostgreSQL. No cross-engine
disambiguation is needed. This section is retained per the skill's template
for consistency with other workshop projects and will stay empty unless a
future extension adds a second engine.)

---

## Phase A Interview Log

Two naming forks (status-vs-column, position-vs-priority) presented to Ali
for disambiguation, both resolved 2026-07-02 (recommendation accepted in both
cases). No other term conflicts were found — this is a greenfield domain
with no existing codebase or prior naming to reconcile against.
