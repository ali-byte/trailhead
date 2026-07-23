package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"regexp"

	"github.com/go-chi/chi/v5"

	"trailhead/internal/adapter"
	"trailhead/internal/domain"
)

// maxBodyBytes is the POST body size cap, applied before decode - see Wire
// Contract "Body size cap" (16 KiB).
const maxBodyBytes = 16384

// uuidPattern matches a syntactically well-formed UUID (8-4-4-4-12 hex
// digits) - shape only, not version/variant bits. Deliberately looser than
// RFC 4122 v4 validation: docs/issues/05-api-move.md's own test fixtures
// use well-formed-but-not-v4 values (e.g. "11111111-...-1111") for the
// "well-formed but nonexistent" case, so a stricter v4-only check would
// reject legitimate input this Wire Contract requires accepting.
var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func isWellFormedUUID(s string) bool {
	return uuidPattern.MatchString(s)
}

// createBookmarkRequest is the strict decode target for POST
// /api/bookmarks. DisallowUnknownFields rejects any field not listed here
// (including "tags", which is intentionally not settable on create) - see
// Wire Contract "Decode phase".
type createBookmarkRequest struct {
	URL   string  `json:"url"`
	Title *string `json:"title"`
}

func (h *handler) createBookmark(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var req createBookmarkRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "payload_too_large", "request body exceeds the size limit")
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", "request body could not be parsed")
		return
	}

	if req.URL == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "url is required")
		return
	}

	created, err := h.repo.Create(r.Context(), domain.NewBookmark{
		OriginalURL: req.URL,
		Title:       req.Title,
	})
	if err != nil {
		h.writeCreateError(w, r.Context(), err)
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

// writeCreateError maps a Create error to its wire response - see Wire
// Contract "Create". repoErr.Message may embed the raw submitted URL (see
// FakeBookmarkRepository.Create); never forward it to the client - compose
// a generic message here instead and log the real error server-side.
func (h *handler) writeCreateError(w http.ResponseWriter, ctx context.Context, err error) {
	var repoErr *adapter.RepositoryError
	if errors.As(err, &repoErr) {
		switch repoErr.Kind {
		case adapter.ErrKindDuplicate:
			writeJSON(w, http.StatusConflict, struct {
				Error    string          `json:"error"`
				Existing domain.Bookmark `json:"existing"`
			}{Error: "duplicate", Existing: *repoErr.Existing})
			return
		case adapter.ErrKindInvalidURL:
			writeError(w, http.StatusBadRequest, "invalid_url", "the submitted url is not valid")
			return
		}
	}

	slog.ErrorContext(ctx, "createBookmark: repository error", "error", err)
	writeInternalError(w)
}

func (h *handler) board(w http.ResponseWriter, r *http.Request) {
	board, err := h.repo.Board(r.Context(), adapter.BoardFilter{})
	if err != nil {
		slog.ErrorContext(r.Context(), "board: repository error", "error", err)
		writeInternalError(w)
		return
	}

	writeJSON(w, http.StatusOK, board)
}

// moveBookmarkRequest is the strict decode target for POST
// /api/bookmarks/{id}/move. TargetStatus is decoded as any (not string) so
// that a present-but-non-string JSON value is distinguishable from a
// missing key and routed through the same required-field message - see
// Wire Contract "target_status validation" (Q1). Before/After are
// pointer-optional: a nil pointer means the key was omitted or explicit
// JSON null, both treated identically (Q2/Q3).
type moveBookmarkRequest struct {
	TargetStatus any     `json:"target_status"`
	Before       *string `json:"before"`
	After        *string `json:"after"`
}

// moveBookmark has no per-request entry/exit log of its own - by
// established convention in this package (see createBookmark/board, issue
// #3), the api layer logs error paths with context and leaves I/O
// entry/exit tracing to the adapter that actually performs it; the
// postgres adapter's Move logs "postgres.Move: enter"/"exit" at DEBUG one
// layer down (internal/adapter/postgres/repository.go).
//
// The body below is a linear validation pipeline (nesting depth <= 2), not
// deeply nested logic: each guard clause maps 1:1 to one locked Wire
// Contract step ({id} format -> decode/size -> target_status
// required-field -> target_status IsValid -> before/after format), in the
// same order as the Prompt Plan that specified this handler. It is kept
// inline rather than further extracted - the two genuinely non-trivial
// steps (target_status parsing, before/after UUID-format gating) are
// already split into parseTargetStatus/parseNeighbor; extracting the
// remaining steps would only relocate, not reduce, the sequence a reader
// has to follow.
func (h *handler) moveBookmark(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	if !isWellFormedUUID(rawID) {
		// Malformed {id} collapses with "does not exist" -> 404, not 400 -
		// see Wire Contract Q4. Resolved before ever reading the body.
		writeError(w, http.StatusNotFound, "not_found", "resource not found")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	var req moveBookmarkRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "payload_too_large", "request body exceeds the size limit")
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", "request body could not be parsed")
		return
	}

	targetStatus, ok := parseTargetStatus(req.TargetStatus)
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "target_status is required")
		return
	}
	if !targetStatus.IsValid() {
		writeError(w, http.StatusBadRequest, "bad_request", "target_status is not a recognized status")
		return
	}

	before, ok := parseNeighbor(req.Before)
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "before must be a well-formed id")
		return
	}
	after, ok := parseNeighbor(req.After)
	if !ok {
		writeError(w, http.StatusBadRequest, "bad_request", "after must be a well-formed id")
		return
	}

	cmd := adapter.MoveCommand{
		ID:           domain.BookmarkID(rawID),
		TargetStatus: targetStatus,
		Before:       before,
		After:        after,
	}

	updated, err := h.repo.Move(r.Context(), cmd)
	if err != nil {
		h.writeMoveError(w, r.Context(), err)
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

// parseTargetStatus extracts a non-empty string from the decoded
// target_status value. ok is false for a missing key (nil), a
// present-but-non-string value, or an empty string - all three share the
// same required-field message (Wire Contract Q1), checked before
// domain.Status.IsValid() ever runs.
func parseTargetStatus(raw any) (status domain.Status, ok bool) {
	s, isString := raw.(string)
	if !isString || s == "" {
		return "", false
	}
	return domain.Status(s), true
}

// parseNeighbor converts an optional before/after string into a
// *domain.BookmarkID, applying the UUID-format gate at the API boundary -
// see Wire Contract "before/after" (Q2/Q3). A nil raw (key omitted or JSON
// null) means no neighbor constraint. ok is false when raw is non-nil but
// empty or not UUID-shaped - existence is never checked here, only shape.
func parseNeighbor(raw *string) (id *domain.BookmarkID, ok bool) {
	if raw == nil {
		return nil, true
	}
	if !isWellFormedUUID(*raw) {
		return nil, false
	}
	bid := domain.BookmarkID(*raw)
	return &bid, true
}

// writeMoveError maps a Move error to its wire response - mirrors
// writeCreateError's redaction discipline (Wire Contract "Redaction scar",
// Q7): the 500 fallthrough is always the exact constant envelope via
// writeInternalError, never the wrapped Move error, which can carry
// Postgres text or an echoed bad value.
func (h *handler) writeMoveError(w http.ResponseWriter, ctx context.Context, err error) {
	var repoErr *adapter.RepositoryError
	if errors.As(err, &repoErr) && repoErr.Kind == adapter.ErrKindNotFound {
		writeError(w, http.StatusNotFound, "not_found", "resource not found")
		return
	}

	slog.ErrorContext(ctx, "moveBookmark: repository error", "error", err)
	writeInternalError(w)
}
