## Context

Expose `Move` (issue #4) over REST/JSON, following the same routing/
error-mapping conventions issue #3 established.

References: docs/PRD.md (Goals 2, 3) | docs/PHASE_PLAN.md (Phase 2) |
DECISIONS.md ("Reorder / Move Endpoint") | docs/ARCHITECTURE_RFC.md
("Locked Interfaces")

## Wire Contract

**Mandatory endpoints:**
- `POST /api/bookmarks/{id}/move` — request body: `{"target_status":
  string, "before": string | null, "after": string | null}`. Response
  200: the updated `domain.Bookmark`. Response 404 when
  `RepositoryError.Kind == ErrKindNotFound` (unknown `id`).

**Optional endpoints:** none.

**Response shapes:** as above.

**Pagination semantics:** N/A.

**TargetStatus validation — RESOLVED (E1, issue #4 Pre-Phase F,
2026-07-13):** `internal/api` owns validation. This issue must add
`domain.Status.IsValid() bool` (new exported method on `internal/domain`)
and call it before constructing any `MoveCommand`, rejecting an
unrecognized `target_status` string with 400 `bad_request` (same
envelope shape issue #3 established). See `DECISIONS.md`
"MoveCommand.TargetStatus — Validation Ownership (E1, resolved)".

**Developer confirmation required:** Developer must confirm the endpoint
layout above matches the intended Phase F implementation plan before the
Pre-Phase F test file is committed.

**CONFIRMED by developer, 2026-07-22 — full contract locked at Pre-Phase F
interview (7 questions, Tier 2), amending/detailing the shape above:**

- **`target_status` (Q1):** missing key, non-string value, and empty
  string `""` all → `400 bad_request` via the same required-field path,
  checked *before* `domain.Status.IsValid()` — mirrors `url`'s treatment
  in issue #3 exactly. A present-but-unrecognized value (e.g.
  `"archived"`) goes through `IsValid()` and is also `400 bad_request`,
  but via a distinct code path with a distinct message (same envelope
  shape, non-generic wording) — the two cases must not share one message.
- **`before`/`after` (Q2, Q3 — the core open decision, amended once):**
  opaque pass-through for *existence*, but UUID-**format**-gated at the
  boundary. `null` (or the key omitted entirely — equivalent) → no
  neighbor constraint. A syntactically well-formed UUID that doesn't
  correspond to any real bookmark → passed straight into
  `MoveCommand.Before`/`.After` untouched; `Move`'s already-locked
  fallback (see issue #4) handles it — end-of-column, `200`, **not** a
  400. An empty string or any non-UUID-shaped string → `400 bad_request`,
  rejected before ever reaching `Move`. Reason for the format gate: `id`
  is a native Postgres `uuid` column — passing a non-UUID string into
  `Move`'s `WHERE id = $1` would 22P02 against real Postgres while
  `FakeBookmarkRepository` would just no-op-fallback (map miss), a
  fake-vs-real divergence this gate closes at the API boundary so both
  backends agree. Existence is never checked in `internal/api` — only
  shape. `Before` still wins over `After` unconditionally per issue #4's
  locked `ports.go` contract; that's unchanged by this amendment.
- **`{id}` path param (Q4):** same UUID-cast risk as `before`/`after`,
  one layer up, but a different resolution: malformed `{id}` → `404
  not_found`, collapsed with the "does not exist" case, not `400`. Role
  distinction: `{id}` is the resource address (unresolvable → `404`);
  `before`/`after` are request fields (malformed → `400`). Implementation
  note (not binding on Dispatch's exact mechanism, just the outcome): a
  handler-level UUID-parse failure on `{id}` should produce the same
  `not_found` envelope as a valid-but-nonexistent id, before ever calling
  `Move`, so `FakeBookmarkRepository` and real Postgres agree.
- **Response `finished_at` (Q5):** Move-specific, not a re-test of #3's
  generic null-serialization invariant (which stays out of scope here —
  `author` in particular is pure redundancy, `Move` never touches it).
  Transition into Done → non-null RFC3339 timestamp. Transition out of
  Done → explicit `null`.
- **Out of scope (Q6):** an HTTP-level Done→Done `finished_at`-preserve
  resend test. That invariant belongs to `Move` itself, already locked at
  #4's adapter layer with an advanceable clock; the API has no code path
  that touches `finished_at`, so re-testing it here would re-prove
  adapter logic through serialization without covering new API-owned
  surface.
- **Redaction scar (Q7):** the #2/#3 scar applies here too — Move's
  underlying errors can be detail-rich (Postgres text, a 22P02 echoing
  the bad value, `identity_hash`-adjacent internals). The `500 internal`
  fallthrough must be the exact constant envelope
  `{"error":"internal","message":"internal server error"}`, never the
  wrapped `Move` error. `400`/`404` messages must not echo the submitted
  `{id}`, `before`, or `after` value back to the client.

Full response set locked: `200` (updated `domain.Bookmark`) · `404
not_found` (`ErrKindNotFound` from `Move`, or a malformed `{id}`) · `400
bad_request` (target_status missing/non-string/empty/unrecognized;
before/after empty or malformed; unparseable body; unknown JSON field) ·
`413 payload_too_large` (body over 16 KiB, matching #3's cap) · `500
internal` (any other error, generic message only) · `405
method_not_allowed` (wrong method on this route) · `404 not_found` (any
unknown path, router-level, unchanged from #3).

## Acceptance Criteria

Given a valid `id` and a valid `target_status`
When `POST /api/bookmarks/{id}/move` is called
Then a 200 response returns the updated bookmark with the correct
`status`/`finished_at`/`position`

Given an `id` that does not exist
When `POST /api/bookmarks/{id}/move` is called
Then a 404 response is returned

## Out of Scope

Update/delete/tag-filter routes (Phase D). Frontend drag-and-drop
integration (Phase F).

## Test Targets

Drafted at Pre-Phase F, 2026-07-22 — 23 tests (2 acceptance-criteria, 21
interview-generated) in `internal/api/handlers_move_test.go`, package
`api_test`, black-box against `httptest.NewServer(api.NewRouter(...))`,
no build tag (runs in the plain `go test ./...` unit job, like #3).
`FakeBookmarkRepository.Move` (`internal/testutil`) needed no changes —
already correctly implements the full locked contract.

Final test list: `TestMoveBookmark_ValidRequest_Returns200WithUpdatedBookmark`,
`TestMoveBookmark_UnknownID_Returns404NotFound`,
`TestMoveBookmark_MissingTargetStatus_Returns400BadRequest`,
`TestMoveBookmark_NonStringTargetStatus_Returns400BadRequest`,
`TestMoveBookmark_EmptyStringTargetStatus_Returns400BadRequest`,
`TestMoveBookmark_UnrecognizedTargetStatus_Returns400BadRequest`,
`TestMoveBookmark_MissingVsUnrecognizedTargetStatus_DistinctMessages`,
`TestMoveBookmark_NoBeforeAfterKeys_EndOfColumn`,
`TestMoveBookmark_ValidBeforeNeighbor_PlacedBeforeIt`,
`TestMoveBookmark_WellFormedNonexistentNeighbor_FallsBackEndOfColumn_Returns200`,
`TestMoveBookmark_EmptyStringBefore_Returns400BadRequest`,
`TestMoveBookmark_MalformedBefore_Returns400BadRequest`,
`TestMoveBookmark_MalformedAfter_Returns400BadRequest`,
`TestMoveBookmark_MalformedID_Returns404NotFound`,
`TestMoveBookmark_IntoDone_ResponseFinishedAtIsNonNullTimestamp`,
`TestMoveBookmark_OutOfDone_ResponseFinishedAtIsNull`,
`TestMoveBookmark_ContextCanceled_Returns500Internal_GenericMessage`,
`TestMoveBookmark_MalformedIDMessage_DoesNotEchoSubmittedID`,
`TestMoveBookmark_MalformedBeforeMessage_DoesNotEchoSubmittedValue`,
`TestMoveBookmark_WrongMethod_Returns405MethodNotAllowed`,
`TestMoveBookmark_OversizedBody_Returns413PayloadTooLarge`,
`TestMoveBookmark_UnparseableJSON_Returns400BadRequest`,
`TestMoveBookmark_UnknownField_Returns400BadRequest`.

Gate condition from PHASE_PLAN.md (Phase 2): part of the overall Phase 2
gate — `TestMoveBookmark*` passes end-to-end through this API layer.

## Parallel Safety

Can run alongside: none
Must wait for: Postgres Adapter — Move (issue #4), API — Create & Board
routes (issue #3) [reuses its router/error-mapping conventions]
Blocks: none within this plan (Phase D's issues are the next dependents)

## Labels

phase-2, tier-2, api
