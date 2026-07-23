// Package domain contains Trailhead's core domain types and the pure
// canonical-URL / identity-hash / default-title derivations (see
// canonicalize.go). It has no dependency on any storage engine, HTTP
// framework, or external service, and no import outside the standard
// library — see ARCHITECTURE_RFC.md "Package Organization" for the locked
// import-direction rules this package must not violate.
//
// See docs/GLOSSARY.md for the canonical definition of every exported type
// in this package.
package domain

import "time"

// BookmarkID uniquely identifies a Bookmark. Backed by a Postgres native
// uuid column (see ARCHITECTURE_RFC.md "Persistence Schema"). Represented
// as a plain string (the UUID's canonical text form) rather than a
// third-party uuid.UUID type, so that this package stays standard-library
// only — see ARCHITECTURE_RFC.md "ID Type and Representation" for the
// reasoning.
type BookmarkID string

// Status is which of the three fixed stages a Bookmark is currently in.
// "Status" is the canonical code term — the UI calls this concept
// "column"; "column" must never appear as a Go field, DB column, or API
// field name (see docs/GLOSSARY.md "Status", DECISIONS.md "Locked From
// Brief"). Backed by a Postgres text column with a CHECK constraint, not a
// native Postgres enum — see ARCHITECTURE_RFC.md "Persistence Schema" for
// why.
type Status string

// StatusInbox, StatusReading, and StatusDone are the three fixed values
// Status can hold - see the type's own doc comment above and DECISIONS.md
// "Locked From Brief" (exactly three columns, not user-configurable, no
// column beyond these three).
const (
	StatusInbox   Status = "inbox"
	StatusReading Status = "reading"
	StatusDone    Status = "done"
)

// IsValid reports whether s is one of the three fixed Status values. See
// docs/issues/05-api-move.md "TargetStatus validation" (E1, resolved):
// internal/api calls IsValid before constructing any adapter.MoveCommand,
// so BookmarkRepository.Move trusts TargetStatus is always valid and does
// not itself defend against an invalid value at the Go level.
func (s Status) IsValid() bool {
	switch s {
	case StatusInbox, StatusReading, StatusDone:
		return true
	default:
		return false
	}
}

// Position is a Bookmark's fractional rank within its Status, unique
// within (Status). Lower sorts first. "Position" is the canonical code
// term — the brief's prose calls this concept "priority"; "priority" must
// never appear as a Go field, DB column, or API field name (see
// docs/GLOSSARY.md "Position"). The concrete rank-generation algorithm
// (how a new Position is computed between two neighbors) is Phase F
// implementation detail behind the BookmarkRepository port — see
// ARCHITECTURE_RFC.md "Scope Boundary" — and is intentionally not
// implemented in this package.
type Position string

// CanonicalURL is the normalized form of a Bookmark's URL, produced by
// Canonicalize (see canonicalize.go). See DECISIONS.md "Canonical URL"
// entries for the full rule set — Locked, do not change without a filed
// RFC; changing it re-buckets every existing bookmark.
type CanonicalURL string

// IdentityHash is the SHA-256 hash (hex-encoded) of a Bookmark's
// CanonicalURL, produced by DeriveIdentityHash (see canonicalize.go), used
// as the indexed lookup key for duplicate detection. See DECISIONS.md
// "Identity Hash" — Locked.
type IdentityHash string

// Tags is the free-text tag set attached to a Bookmark: lowercased on
// save, deduplicated, no empty strings (enforced by the repository
// implementation, not by this type) — see DECISIONS.md "Locked From
// Brief". Stored as a Postgres JSONB array — see DECISIONS.md "Tags
// Storage" (fixed cross-reference at Pre-Phase F, issue #2, 2026-07-09:
// this comment previously cited only "Tags Storage" for the
// lowercase/dedup/no-empty-string rule, which actually lives under
// "Locked From Brief" — "Tags Storage" covers JSONB representation only).
type Tags []string

// Bookmark is Trailhead's core persisted entity.
//
// JSON field names are locked snake_case, matching the Postgres column
// names in ARCHITECTURE_RFC.md "Persistence Schema" and the explicit
// examples in "Serialization Spec" (id, created_at, updated_at,
// finished_at). Resolved at Phase B gate round 4 (2026-07-05) — this
// struct previously carried no tags at all, so default Go marshaling
// would have emitted the capitalized Go field names instead, same class
// of gap as domain.Board's round-3 fix (Codex round-4 finding A1).
type Bookmark struct {
	ID           BookmarkID   `json:"id"`
	OriginalURL  string       `json:"original_url"`
	CanonicalURL CanonicalURL `json:"canonical_url"`
	IdentityHash IdentityHash `json:"identity_hash"`
	Title        string       `json:"title"`
	Tags         Tags         `json:"tags"`
	Status       Status       `json:"status"`
	Position     Position     `json:"position"`

	// FinishedAt is set if and only if Status == StatusDone (see
	// DECISIONS.md "FinishedAt <-> Done invariant" — Locked). It is a
	// pointer so that "absent" (nil) is distinguishable from the zero
	// value of time.Time — absent for every Bookmark not currently in
	// Done, never a zero-value stand-in.
	FinishedAt *time.Time `json:"finished_at"`

	// Author is user-editable and absent (nil) on the large majority of
	// Bookmarks — nothing in this system auto-populates it (see
	// DECISIONS.md "Author Field - Population Path"). A pointer so that
	// "absent" is distinguishable from "author is an empty string."
	Author *string `json:"author"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewBookmark is the input to BookmarkRepository.Create. Title is optional
// — when nil, DefaultTitle (see canonicalize.go) is used to derive one
// before persisting. Tags are raw, pre-normalization input; the repository
// implementation is responsible for lowercasing, deduplicating, and
// dropping empty-string tags per DECISIONS.md "Locked From Brief".
type NewBookmark struct {
	OriginalURL string
	Title       *string
	Tags        Tags
}

// BookmarkPatch describes an edit to an existing Bookmark. For Title and
// Tags, a nil field means "leave unchanged" - standard pointer-optional
// semantics. Author is tri-state and does not follow that same rule alone:
// ClearAuthor = true clears Author to nil (absent) regardless of the
// Author field's value; otherwise a non-nil Author sets a new value and a
// nil Author leaves the existing value unchanged. ClearAuthor exists
// because a bare *string cannot distinguish "leave unchanged" from
// "clear to absent" - both would be a nil pointer. See DECISIONS.md
// "Author Field - Clearing via Patch" (Phase B gate fix, 2026-07-05).
type BookmarkPatch struct {
	Title       *string
	Tags        *Tags
	Author      *string
	ClearAuthor bool
}

// Board is the full three-column view: every Bookmark grouped by Status,
// ordered by Position within each Status, optionally filtered by tag (see
// BoardFilter in internal/adapter/ports.go). Not a persisted entity — a
// query/response shape over Bookmark rows.
//
// JSON field names are locked lowercase (`inbox`, `reading`, `done`) via
// the struct tags below — see ARCHITECTURE_RFC.md "Serialization Spec".
// Without an explicit tag, Go's default json marshaling would emit the
// capitalized Go field names instead, contradicting GLOSSARY.md's stated
// wire shape (Codex round-3 finding A2).
type Board struct {
	Inbox   []Bookmark `json:"inbox"`
	Reading []Bookmark `json:"reading"`
	Done    []Bookmark `json:"done"`
}
