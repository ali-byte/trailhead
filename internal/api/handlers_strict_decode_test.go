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
