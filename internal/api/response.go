// Package api implements the HTTP layer over adapter.BookmarkRepository:
// request decoding, response encoding, and error-kind -> HTTP status
// mapping. See docs/issues/03-api-create-board.md "Wire Contract" for the
// locked request/response shapes this package implements.
package api

import (
	"bytes"
	"encoding/json"
	"net/http"
)

// writeJSON encodes v to a buffer first so that an encode error can still
// produce a clean 500 rather than a half-written body - Content-Type and
// the status code are only written once encoding has succeeded.
func writeJSON(w http.ResponseWriter, status int, v any) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal","message":"internal server error"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

// errorEnvelope is the locked {"error": string, "message": string} wire
// shape for all error responses except the 409 duplicate response, which
// additionally carries "existing".
type errorEnvelope struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func writeError(w http.ResponseWriter, status int, kind, message string) {
	writeJSON(w, status, errorEnvelope{Error: kind, Message: message})
}

func writeInternalError(w http.ResponseWriter) {
	writeError(w, http.StatusInternalServerError, "internal", "internal server error")
}
