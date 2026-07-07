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
//
// ErrorKind does NOT cover infrastructure failures (Postgres unreachable,
// network errors, context cancellation/timeout) - see DECISIONS.md
// "Repository Error Taxonomy - Infrastructure Failures". Those are
// returned as a plain wrapped error (the built-in error interface,
// never *RepositoryError), because the API layer maps every
// infrastructure failure to the same 5xx response regardless of the
// specific cause - there is no HTTP-status-relevant distinction to carry
// in a Kind the way there is for Duplicate/NotFound/InvalidURL. The
// on-disk precedent is FakeBookmarkRepository's checkContext, which
// already returns ctx.Err() directly (a bare error), not a
// *RepositoryError - see fake_repository.go.
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

// RepositoryError is the classified error type BookmarkRepository methods
// return for failures the API layer must map to a specific 4xx
// (Duplicate/NotFound/InvalidURL) - it is NOT the only error a method can
// return. Infrastructure failures (Postgres unreachable, network errors,
// context cancellation/timeout) are returned as a plain wrapped error
// instead - see ErrorKind's doc comment and DECISIONS.md "Repository Error
// Taxonomy - Infrastructure Failures" (Phase B gate round 4 fix,
// 2026-07-05 - this comment previously read "the sole error type," which
// contradicted that decision). Always returned as the built-in error
// interface (never as a concrete *RepositoryError return type) - see
// go-patterns "Hard Rules": a nil *RepositoryError boxed in a non-pointer
// error-typed return would be a non-nil interface, a classic Go footgun.
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
// Error taxonomy: every method's error return is either (a) a
// *RepositoryError with a classified Kind (Duplicate/NotFound/InvalidURL -
// the API layer maps these to 409/404/400 respectively), or (b) a plain
// wrapped error representing an infrastructure failure (Postgres
// unreachable, network error, context cancellation/timeout), which the
// API layer maps to a uniform 5xx. Callers distinguish the two with
// errors.As(err, &repoErr): success means a classified failure, failure
// means an infrastructure failure. See DECISIONS.md "Repository Error
// Taxonomy - Infrastructure Failures".
//
// Mutating-method write contract: Create, Move, and Update each set the
// returned Bookmark's UpdatedAt to the current time on every successful
// call, whether or not any field's *value* actually changed. On Create
// specifically, UpdatedAt is set equal to CreatedAt (both timestamp the
// same creation instant) - not merely "some current time" independently
// derived. Delete removes the row entirely - no UpdatedAt semantics apply
// to a deleted resource. See DECISIONS.md "UpdatedAt - Write-Path Contract"
// (Phase B gate round 5 fix, 2026-07-06 - this comment previously omitted
// the Create-equals-CreatedAt equality requirement that DECISIONS.md
// already stated, an under-specification Codex round-5 finding A1).
//
// Context Completeness Check (design-an-interface Phase 5): Board is the
// only output-affecting method: BoardFilter.Tags carries the sole optional
// user-supplied context that shapes its output, and it is present on every
// call.
type BookmarkRepository interface {
	// Create persists a new Bookmark in Status = StatusInbox at the front
	// of that column's ordering. CreatedAt and UpdatedAt are both set to
	// the same current-time value (UpdatedAt == CreatedAt on the returned
	// Bookmark) - see the write contract above and DECISIONS.md "UpdatedAt
	// - Write-Path Contract". If a Bookmark with the same
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
	// Neighbor fallback: cmd.Before and cmd.After express intent, not a
	// validated reference - Move never fails the request because of a
	// bad neighbor. Any of the following - a missing/stale ID (deleted
	// between drag-start and drop), an ID that exists but is not in
	// cmd.TargetStatus (cross-status), or an ID equal to cmd.ID itself
	// (self-referential) - falls back to inserting at the end of the
	// target column. If both Before and After are set, Before takes
	// precedence and After is ignored outright (a tie-break, not a
	// consistency check) - Move does not detect or specially handle the
	// two disagreeing with each other. See DECISIONS.md "Move - Neighbor
	// Fallback (Generalized)".
	// Returns a *RepositoryError with Kind = ErrKindNotFound if cmd.ID
	// does not exist. Behavior for an invalid cmd.TargetStatus (a
	// domain.Status value outside StatusInbox/StatusReading/StatusDone)
	// is an accepted open item, not specified by this contract - see
	// ARCHITECTURE_RFC.md "Scope Boundary" (Phase B gate round 5 note,
	// Codex finding E1); ownership of validating TargetStatus is decided
	// per internal/api issue at its Pre-Phase F Wire Contract review.
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
