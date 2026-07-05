// Package testutil provides fake implementations of the interfaces
// declared in internal/adapter/ports.go, for use in tests only. Never
// imported by production code (cmd/trailhead or internal/api) - see
// ARCHITECTURE_RFC.md "Package Organization".
package testutil

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"trailhead/internal/adapter"
	"trailhead/internal/domain"
)

// FakeBookmarkRepository is an in-memory implementation of
// adapter.BookmarkRepository for use in api-layer and domain-level tests
// (Pre-Phase F test-engine sessions target this, not a live Postgres
// instance). It must satisfy four adversarial invariants - see the
// doc comment on each method below for which invariant it enforces.
//
// Position handling: FakeBookmarkRepository orders bookmarks per Status
// using a simple in-memory slice (see order field) and formats
// domain.Position as a zero-padded index string purely for its own
// bookkeeping. This is a test-double simplification, NOT the production
// fractional-rank algorithm - the real algorithm is Phase F implementation
// detail behind the Postgres adapter (see ARCHITECTURE_RFC.md "Scope
// Boundary"). Tests that assert on the exact string form of Position
// against this fake are testing the fake's bookkeeping, not the locked
// production representation.
type FakeBookmarkRepository struct {
	mu sync.Mutex

	byID   map[domain.BookmarkID]domain.Bookmark
	byHash map[domain.IdentityHash]domain.BookmarkID
	order  map[domain.Status][]domain.BookmarkID
	idSeq  int
	now    func() time.Time // injectable clock, defaults to time.Now
}

// Compile-time interface-compliance check (improve-codebase-architecture
// "Adapter Interface Compliance Check"): fails to build if
// FakeBookmarkRepository's method set drifts from adapter.BookmarkRepository.
var _ adapter.BookmarkRepository = (*FakeBookmarkRepository)(nil)

// NewFakeBookmarkRepository returns an empty fake, ready to use.
func NewFakeBookmarkRepository() *FakeBookmarkRepository {
	return &FakeBookmarkRepository{
		byID:   make(map[domain.BookmarkID]domain.Bookmark),
		byHash: make(map[domain.IdentityHash]domain.BookmarkID),
		order: map[domain.Status][]domain.BookmarkID{
			domain.StatusInbox:   {},
			domain.StatusReading: {},
			domain.StatusDone:    {},
		},
		now: time.Now,
	}
}

// SetClock overrides the fake's clock, for tests that need a deterministic
// FinishedAt value.
func (f *FakeBookmarkRepository) SetClock(now func() time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = now
}

func (f *FakeBookmarkRepository) nextID() domain.BookmarkID {
	f.idSeq++
	return domain.BookmarkID(fmt.Sprintf("fake-id-%d", f.idSeq))
}

// checkContext enforces Adversarial Invariant 3: Context Cancellation - a
// fake that ignores context cancellation silently passes tests for code
// that would hang in production. Every method checks this before touching
// shared state. Returns ctx.Err() directly (a plain error), never wrapped
// in a *adapter.RepositoryError - context cancellation is an
// infrastructure failure, not a classified repository failure. See
// DECISIONS.md "Repository Error Taxonomy - Infrastructure Failures".
func checkContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// Create implements adapter.BookmarkRepository.Create.
//
// Enforces Adversarial Invariant 1: Exact IdentityHash Match - duplicate
// detection compares domain.IdentityHash values exactly (via a map
// lookup), never a fuzzy or substring match on the URL string.
//
// Enforces Adversarial Invariant 2: Typed Error - returns
// *adapter.RepositoryError, never a plain errors.New() string error, on
// every failure path.
func (f *FakeBookmarkRepository) Create(ctx context.Context, b domain.NewBookmark) (domain.Bookmark, error) {
	if err := checkContext(ctx); err != nil {
		return domain.Bookmark{}, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	canonical, err := domain.Canonicalize(b.OriginalURL)
	if err != nil {
		return domain.Bookmark{}, &adapter.RepositoryError{
			Kind:    adapter.ErrKindInvalidURL,
			Message: fmt.Sprintf("Create(%q): invalid URL", b.OriginalURL),
			Wrapped: err,
		}
	}
	hash := domain.DeriveIdentityHash(canonical)

	if existingID, found := f.byHash[hash]; found {
		existing := f.byID[existingID]
		return domain.Bookmark{}, &adapter.RepositoryError{
			Kind:     adapter.ErrKindDuplicate,
			Existing: &existing,
			Message:  fmt.Sprintf("Create(%q): already on the board", b.OriginalURL),
		}
	}

	title := ""
	if b.Title != nil {
		title = *b.Title
	} else {
		title = domain.DefaultTitle(b.OriginalURL)
	}

	tags := normalizeTags(b.Tags)

	now := f.now()
	id := f.nextID()
	bookmark := domain.Bookmark{
		ID:           id,
		OriginalURL:  b.OriginalURL,
		CanonicalURL: canonical,
		IdentityHash: hash,
		Title:        title,
		Tags:         tags,
		Status:       domain.StatusInbox,
		Position:     domain.Position(fmt.Sprintf("%08d", 0)),
		FinishedAt:   nil,
		Author:       nil,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	f.byID[id] = bookmark
	f.byHash[hash] = id
	f.order[domain.StatusInbox] = append([]domain.BookmarkID{id}, f.order[domain.StatusInbox]...)
	f.reindexPositions(domain.StatusInbox)

	return f.byID[id], nil
}

// Board implements adapter.BookmarkRepository.Board.
func (f *FakeBookmarkRepository) Board(ctx context.Context, filter adapter.BoardFilter) (domain.Board, error) {
	if err := checkContext(ctx); err != nil {
		return domain.Board{}, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	board := domain.Board{
		Inbox:   f.columnFiltered(domain.StatusInbox, filter.Tags),
		Reading: f.columnFiltered(domain.StatusReading, filter.Tags),
		Done:    f.columnFiltered(domain.StatusDone, filter.Tags),
	}
	return board, nil
}

func (f *FakeBookmarkRepository) columnFiltered(status domain.Status, tags []string) []domain.Bookmark {
	result := make([]domain.Bookmark, 0, len(f.order[status]))
	for _, id := range f.order[status] {
		b := f.byID[id]
		if len(tags) == 0 || hasAnyTag(b.Tags, tags) {
			result = append(result, b)
		}
	}
	return result
}

func hasAnyTag(bookmarkTags domain.Tags, filterTags []string) bool {
	set := make(map[string]struct{}, len(bookmarkTags))
	for _, t := range bookmarkTags {
		set[t] = struct{}{}
	}
	for _, want := range filterTags {
		if _, ok := set[strings.ToLower(want)]; ok {
			return true
		}
	}
	return false
}

// Move implements adapter.BookmarkRepository.Move.
//
// Enforces Adversarial Invariant 4: Nil Pointer Optional Fields - FinishedAt
// is set to a real, non-nil *time.Time only on transition into Done, and
// explicitly reset to nil on transition out of Done. It is never left as a
// stale non-nil value, and never represented as a zero-value time.Time
// standing in for "absent."
func (f *FakeBookmarkRepository) Move(ctx context.Context, cmd adapter.MoveCommand) (domain.Bookmark, error) {
	if err := checkContext(ctx); err != nil {
		return domain.Bookmark{}, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	bookmark, ok := f.byID[cmd.ID]
	if !ok {
		return domain.Bookmark{}, &adapter.RepositoryError{
			Kind:    adapter.ErrKindNotFound,
			Message: fmt.Sprintf("Move(%s): not found", cmd.ID),
		}
	}

	oldStatus := bookmark.Status

	// cmd.ID is removed from its old status's order slice before the
	// neighbor search below runs. This is why a self-referential neighbor
	// (Before or After == cmd.ID) naturally falls through to the
	// not-found case even when oldStatus == cmd.TargetStatus.
	f.removeFromOrder(oldStatus, cmd.ID)

	// Neighbor fallback (generalized) - see DECISIONS.md "Move - Neighbor
	// Fallback (Generalized)". targetOrder is cmd.TargetStatus's own
	// ordering slice, so a search that doesn't find Before/After there
	// covers every inconsistent-neighbor case with one mechanism: a
	// missing/stale ID, a cross-status ID (exists, but in a different
	// status's slice), and a self-referential ID (removed above) all fail
	// the search the same way and fall back to insertAt = len(targetOrder)
	// (end of column). No case-by-case handling is needed.
	targetOrder := f.order[cmd.TargetStatus]
	insertAt := len(targetOrder)
	if cmd.Before != nil {
		for i, id := range targetOrder {
			if id == *cmd.Before {
				insertAt = i
				break
			}
		}
	} else if cmd.After != nil {
		for i, id := range targetOrder {
			if id == *cmd.After {
				insertAt = i + 1
				break
			}
		}
	}
	newOrder := make([]domain.BookmarkID, 0, len(targetOrder)+1)
	newOrder = append(newOrder, targetOrder[:insertAt]...)
	newOrder = append(newOrder, cmd.ID)
	newOrder = append(newOrder, targetOrder[insertAt:]...)
	f.order[cmd.TargetStatus] = newOrder

	bookmark.Status = cmd.TargetStatus

	switch {
	case cmd.TargetStatus == domain.StatusDone && oldStatus != domain.StatusDone:
		finishedAt := f.now()
		bookmark.FinishedAt = &finishedAt
	case cmd.TargetStatus != domain.StatusDone && oldStatus == domain.StatusDone:
		bookmark.FinishedAt = nil
	}

	bookmark.UpdatedAt = f.now()
	f.byID[cmd.ID] = bookmark
	f.reindexPositions(cmd.TargetStatus)
	if oldStatus != cmd.TargetStatus {
		f.reindexPositions(oldStatus)
	}

	return f.byID[cmd.ID], nil
}

func (f *FakeBookmarkRepository) removeFromOrder(status domain.Status, id domain.BookmarkID) {
	order := f.order[status]
	for i, existing := range order {
		if existing == id {
			f.order[status] = append(order[:i], order[i+1:]...)
			return
		}
	}
}

// reindexPositions recomputes each bookmark's Position field from its
// index in the in-memory order slice. See the FakeBookmarkRepository type
// doc comment - this is a test-double simplification, not the locked
// production ranking algorithm.
func (f *FakeBookmarkRepository) reindexPositions(status domain.Status) {
	for i, id := range f.order[status] {
		b := f.byID[id]
		b.Position = domain.Position(fmt.Sprintf("%08d", i))
		f.byID[id] = b
	}
}

// Update implements adapter.BookmarkRepository.Update.
func (f *FakeBookmarkRepository) Update(ctx context.Context, id domain.BookmarkID, patch domain.BookmarkPatch) (domain.Bookmark, error) {
	if err := checkContext(ctx); err != nil {
		return domain.Bookmark{}, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	bookmark, ok := f.byID[id]
	if !ok {
		return domain.Bookmark{}, &adapter.RepositoryError{
			Kind:    adapter.ErrKindNotFound,
			Message: fmt.Sprintf("Update(%s): not found", id),
		}
	}

	if patch.Title != nil {
		bookmark.Title = *patch.Title
	}
	if patch.Tags != nil {
		bookmark.Tags = normalizeTags(*patch.Tags)
	}
	// Author is tri-state, not a plain pointer-optional field - see
	// domain.BookmarkPatch and DECISIONS.md "Author Field - Clearing via
	// Patch". ClearAuthor takes precedence over Author if a caller somehow
	// sets both.
	if patch.ClearAuthor {
		bookmark.Author = nil
	} else if patch.Author != nil {
		bookmark.Author = patch.Author
	}
	bookmark.UpdatedAt = f.now()

	f.byID[id] = bookmark
	return bookmark, nil
}

// Delete implements adapter.BookmarkRepository.Delete.
func (f *FakeBookmarkRepository) Delete(ctx context.Context, id domain.BookmarkID) error {
	if err := checkContext(ctx); err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	bookmark, ok := f.byID[id]
	if !ok {
		return &adapter.RepositoryError{
			Kind:    adapter.ErrKindNotFound,
			Message: fmt.Sprintf("Delete(%s): not found", id),
		}
	}

	f.removeFromOrder(bookmark.Status, id)
	delete(f.byID, id)
	delete(f.byHash, bookmark.IdentityHash)
	f.reindexPositions(bookmark.Status)

	return nil
}

// normalizeTags lowercases, deduplicates, and drops empty-string tags -
// see DECISIONS.md "Locked From Brief" (tags are free text, lowercased on
// save, deduplicated per bookmark, empty strings never stored).
func normalizeTags(raw domain.Tags) domain.Tags {
	seen := make(map[string]struct{}, len(raw))
	result := make(domain.Tags, 0, len(raw))
	for _, t := range raw {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		result = append(result, t)
	}
	return result
}
