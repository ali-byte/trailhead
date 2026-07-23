// Regression test for the Codex #5 review finding, 2026-07-23:
// json.Decoder.Decode only consumes one JSON value and silently leaves
// anything after it unread - a body like `{"target_status":"reading"}
// garbage` would otherwise decode the valid prefix and let the handler
// proceed on a malformed request. Both createBookmark (#3) and
// moveBookmark (#5) shared this gap until fixed via the shared
// decodeStrictJSON helper (internal/api/decode.go).
//
// This is a build-created test file, not a locked Pre-Phase F fixture -
// see handlers_create_board_test.go and handlers_move_test.go for those.
// It may be modified or extended by future sessions, unlike its locked
// counterparts.

package api_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateBookmark_TrailingDataAfterJSON_Returns400BadRequest(t *testing.T) {
	srv, _ := newTestServer(t)

	resp, err := http.Post(srv.URL+"/api/bookmarks", "application/json", strings.NewReader(`{"url": "https://example.com/a"} garbage`))
	require.NoError(t, err)
	assertJSONContentType(t, resp)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var body errorEnvelope
	decodeJSON(t, resp, &body)
	assert.Equal(t, "bad_request", body.Error)
}

func TestMoveBookmark_TrailingDataAfterJSON_Returns400BadRequest(t *testing.T) {
	srv, _ := newTestServer(t)
	created := createBookmarkViaAPI(t, srv, "https://example.com/a")

	resp := moveBookmark(t, srv, string(created.ID), `{"target_status": "reading"} garbage`)
	assertJSONContentType(t, resp)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var body errorEnvelope
	decodeJSON(t, resp, &body)
	assert.Equal(t, "bad_request", body.Error)
}

// TestCreateBookmark_TrailingDataOverCap_Returns413PayloadTooLarge is the
// Codex re-review regression: a valid JSON prefix followed by enough
// trailing bytes to exceed maxBodyBytes must still be 413, not 400 - the
// MaxBytesReader cap can be hit on decodeStrictJSON's trailing-data check
// just as easily as on its first Decode, and both must map the same way.
// One route suffices since decodeStrictJSON is shared.
//
// The trailing bytes are whitespace, not garbage characters: an invalid
// non-whitespace byte fails fast as a syntax error as soon as the decoder
// sees it (often already sitting in its read-ahead buffer from the first
// Decode, well under the cap - too fast to prove this finding). Trailing
// whitespace instead forces the decoder to keep reading, looking for the
// next token or EOF, for as long as more whitespace remains - reliably
// exhausting the MaxBytesReader cap regardless of internal buffer
// boundaries.
func TestCreateBookmark_TrailingDataOverCap_Returns413PayloadTooLarge(t *testing.T) {
	srv, _ := newTestServer(t)

	body := `{"url": "https://example.com/a"}` + strings.Repeat(" ", 20000)
	resp, err := http.Post(srv.URL+"/api/bookmarks", "application/json", strings.NewReader(body))
	require.NoError(t, err)
	assert.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode)

	var errBody errorEnvelope
	decodeJSON(t, resp, &errBody)
	assert.Equal(t, "payload_too_large", errBody.Error)
}
