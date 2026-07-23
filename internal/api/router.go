package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"trailhead/internal/adapter"
)

// NewRouter builds Trailhead's HTTP API over repo. This is the sole
// exported entry point this package locks - see docs/issues/03-api-create-
// board.md "Required exported surface".
func NewRouter(repo adapter.BookmarkRepository) http.Handler {
	h := &handler{repo: repo}

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)

	r.Post("/api/bookmarks", h.createBookmark)
	r.Post("/api/bookmarks/{id}/move", h.moveBookmark)
	r.Get("/api/board", h.board)

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "resource not found")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	})

	return r
}

// handler is stateless per request - repo is the only field, set once at
// construction, never mutated.
type handler struct {
	repo adapter.BookmarkRepository
}
