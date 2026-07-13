## Context

First `internal/api` slice: expose `Create` and `Board` (issue #2) over
REST/JSON. This issue must resolve the exact routes/request/response
bodies for these two endpoints — a PRD Open Question this issue closes
out, not defers further.

References: docs/PRD.md (Goal 1, Open Questions — "exact HTTP routes...")
| docs/PHASE_PLAN.md (Phase 1) | DECISIONS.md ("Duplicate Detection
Response") | docs/ARCHITECTURE_RFC.md ("Serialization Spec", "Locked
Interfaces")

## Wire Contract

**CONFIRMED by developer, 2026-07-09 (Pre-Phase F interview, 10 questions).**

**Mandatory endpoints:**
- `POST /api/bookmarks` — create a bookmark.
  - Request body: `{"url": string, "title": string | null}`.
    `tags` is intentionally not settable on create — deferred to `Update`
    (a later issue). Request `Content-Type` is not gated — the handler
    decodes regardless of the header; a bad body fails at decode.
    Unknown fields are rejected via strict `DisallowUnknownFields`
    decoding. Body capped at **16 KiB (16384 bytes)** via
    `http.MaxBytesReader`, applied before decode, POST only.
  - `title` semantics (locked): `null` → default via `domain.DefaultTitle`
    (existing AC1 behavior). `""` (explicit empty string) → explicit
    empty title, stored as `""`, NOT defaulted — distinct from `null`.
    Any non-string JSON value (e.g. `123`) → `bad_request`.
  - `url` semantics: missing key, non-string value, or empty string `""`
    → `bad_request` (empty url is treated as missing, unlike empty
    title). A present, non-empty url that fails `domain.Canonicalize` →
    `invalid_url`.
  - Response 201: the created `domain.Bookmark` (JSON keys per its
    locked struct tags — `id`, `original_url`, `canonical_url`,
    `identity_hash`, `title`, `tags`, `status`, `position`,
    `finished_at`, `author`, `created_at`, `updated_at`).
  - Response 409: `{"error": "duplicate", "existing": <domain.Bookmark>}`
    when `RepositoryError.Kind == ErrKindDuplicate`.
  - Response 400 (`invalid_url`): `{"error": "invalid_url", "message":
    string}` when `Kind == ErrKindInvalidURL`. `message` must NOT
    contain the submitted URL or any internal error text (see #2's own
    redaction scar — the adapter's error is already redacted, but the
    handler must not re-introduce the URL when composing its own
    message).
  - Response 400 (`bad_request`): `{"error": "bad_request", "message":
    string}` — unparseable JSON, missing/non-string/empty `url`,
    non-string `title`, or unknown fields.
  - Response 413 (`payload_too_large`): `{"error": "payload_too_large",
    "message": string}` — body exceeds 16 KiB, detected via
    `errors.As` on `*http.MaxBytesError`.
  - Response 500 (`internal`): `{"error": "internal", "message":
    "internal server error"}` — the general fallthrough for any error
    that is not a classified `*adapter.RepositoryError` (in this
    issue's scope, `FakeBookmarkRepository`'s canceled/timed-out
    context). Constant generic message, never the wrapped error text —
    log the real error server-side only. No handler-level `panic`;
    clean `errors.As` fallthrough. `chi/middleware.Recoverer` is wired
    as a top-level panic safety net separately, not test-driven here.
- `GET /api/board` — returns the full `domain.Board` (locked `inbox`/
  `reading`/`done` JSON keys, empty columns as `[]` never `null`). No
  query params in this issue (tag filter is Phase D). Only in-scope
  error is the same 500 `internal` fallthrough (context
  cancellation) — `Board` never returns a classified
  `*RepositoryError` per `ports.go`'s doc comment.

**Routing conventions (established here, reused by issue #5's Move route):**
- Unknown path → 404, JSON envelope `{"error": "not_found", "message":
  string}` via a custom chi `NotFoundHandler`.
- Wrong method on a known path (e.g. `DELETE /api/bookmarks`) → 405,
  JSON envelope `{"error": "method_not_allowed", "message": string}`
  via a custom chi `MethodNotAllowedHandler`.

**Headers:** `Content-Type: application/json` set on every response
(success and every error case), before `WriteHeader`.

**Timestamps:** RFC3339 UTC (Go's `encoding/json` default for
`time.Time`) — light format assertion, no prior scar, just confirmed.

**Optional endpoints:** none in this issue.

**Response shapes:** as above — this issue is itself the source of truth
for these two shapes; there is no upstream third-party API to reference.

**Pagination semantics:** N/A — `Board` returns the full board, no
pagination per PRD Non-Goals ("no pagination/infinite scroll").

## Acceptance Criteria

Given a valid, non-duplicate URL in a `POST /api/bookmarks` body
When the request is made
Then a 201 response returns the created bookmark's full JSON
representation, and the bookmark is retrievable via `GET /api/board`
immediately after

Given a duplicate URL in a `POST /api/bookmarks` body
When the request is made
Then a 409 response body contains the pre-existing bookmark's data

Given an invalid URL in a `POST /api/bookmarks` body
When the request is made
Then a 400 response is returned and no bookmark is created

## Out of Scope

Move/update/delete/tag-filter routes (later issues). Frontend consumption
of this API (Phase F).

## Test Targets

HTTP-level tests generated by test-engine at Pre-Phase F, 2026-07-09
(Tier 2 interview, 10 questions). File: `internal/api/handlers_create_board_test.go`
— co-located with the package per `trailhead-rules` Rule 1 (no separate
`tests/unit/` directory in this project), package `api_test` (black-box),
**no `integration` build tag** — this suite runs against
`internal/testutil.FakeBookmarkRepository`, not a live Postgres, so it's
part of the plain `go test ./...` run already covered by `ci.yml`'s
existing `test` job. No new CI/workflow artifact needed for this issue.

**Required exported surface this test file locks (Dispatch may not
deviate without a filed RFC):**
- `func NewRouter(repo adapter.BookmarkRepository) http.Handler` —
  assembles the chi router with both handlers wired, plus the custom
  `not_found` / `method_not_allowed` JSON-envelope handlers. Tests are
  black-box against this router via `httptest`, not against individual
  handler functions, so handler names/signatures below this are Dispatch's
  own choice.

Gate condition from PHASE_PLAN.md (Phase 1): part of the overall Phase 1
gate — `TestCreateBookmark` integration test passes end-to-end through
this API layer, not just at the repository layer.

## Parallel Safety

Can run alongside: none
Must wait for: Postgres Adapter — Create & Board (issue #2)
Blocks: none within Phase 1; Phase 2's move route (issue #5) reuses this
issue's router/error-mapping conventions

## Labels

phase-1, tier-2, api
