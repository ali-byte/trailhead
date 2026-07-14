package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"trailhead/internal/adapter"
	"trailhead/internal/domain"
)

// maxBodyBytes is the POST body size cap, applied before decode - see Wire
// Contract "Body size cap" (16 KiB).
const maxBodyBytes = 16384

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
