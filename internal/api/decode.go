package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// maxBodyBytes is the POST body size cap, applied before decode - see Wire
// Contract "Body size cap" (16 KiB).
const maxBodyBytes = 16384

// decodeStrictJSON decodes r's body into dst under the strict-body
// contract shared by every POST handler in this package: MaxBytesReader(16
// KiB), DisallowUnknownFields, and a trailing-data check (Codex #5 finding,
// 2026-07-23). json.Decoder.Decode only consumes one JSON value and
// silently leaves anything after it unread - a body like
// `{"target_status":"reading"} garbage` would otherwise decode the valid
// prefix and let the handler proceed on a malformed request, the same gap
// existed in createBookmark (#3). A second Decode call into a discard
// target must hit io.EOF (only trailing whitespace remains); any other
// outcome - a successfully decoded second value, or a syntax error - means
// there was more than one JSON value in the body, rejected the same way as
// any other malformed body.
//
// On success, returns true and dst is populated. On failure, writes the
// appropriate error response (413 for an oversized body, 400 for anything
// else - unparseable, unknown field, or trailing data) and returns false;
// callers must return immediately.
func decodeStrictJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "payload_too_large", "request body exceeds the size limit")
			return false
		}
		writeError(w, http.StatusBadRequest, "bad_request", "request body could not be parsed")
		return false
	}

	if err := dec.Decode(new(struct{})); err != io.EOF {
		writeError(w, http.StatusBadRequest, "bad_request", "request body could not be parsed")
		return false
	}

	return true
}
