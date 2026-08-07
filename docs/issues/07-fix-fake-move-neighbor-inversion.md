## Context

`FakeBookmarkRepository.Move` (`internal/testutil/fake_repository.go`) inserts a
moved card on the **wrong side** of its neighbors — the inverse of the locked
`ports.go` contract and the real Postgres adapter:

- `ports.go` `MoveCommand`: "Before and After are nil to mean 'first in column'
  / 'last in column'" ⇒ **`Before` = predecessor** (moved lands *after* it),
  **`After` = successor** (moved lands *before* it).
- Real adapter (`resolveNeighborBounds`): `Before` → `midpoint(Before.pos,
  Before's-successor.pos)` → moved lands **after** `Before`. #4's locked test
  says so explicitly: *"Before=cardB says 'insert right after B, before C'."*
- The fake: `Before` → `insertAt = index(Before)` → moved lands **before**
  `Before`; `After` → `insertAt = index(After)+1` → moved lands **after**
  `After`. **Both inverted.**

Because #5's API tests run against this fake, #5 locked a **false contract** —
`TestMoveBookmark_ValidBeforeNeighbor_PlacedBeforeIt` asserts the fake's
inverted behavior. **Production #5 is correct** (it runs the real adapter); only
the fake and that one test are wrong. Surfaced during #6 (frontend) Pre-Phase F,
when `deriveNeighbors` had to reconcile against the API and the coordinator
reviewer-agent gate cross-read #4 vs #5. See ab-validation backlog H-30.

This is a **minimal test-infrastructure correction, not a Dispatch build** — it
is the H-11 locked-test-amendment escape hatch (a bug in the test double + a
locked test, corrected and re-approved as a distinct commit).

## Scope

1. **`internal/testutil/fake_repository.go` — `Move`:** flip both neighbor
   branches to match `ports.go` / the real adapter:
   - `Before` set → `insertAt = index(Before) + 1` (moved lands **after** it).
   - `After` set → `insertAt = index(After)` (moved lands **before** it).
   Both-nil → end-of-column is already correct (unchanged). Update the method's
   doc comment to state the predecessor/successor semantic explicitly.
2. **`internal/api/handlers_move_test.go` — `TestMoveBookmark_ValidBeforeNeighbor_PlacedBeforeIt`:**
   flip the assertion to the correct semantic — with `before: anchor`, the moved
   card lands **after** the anchor (`board.Reading[0] == anchor`,
   `board.Reading[1] == moving`) — and rename to `…_PlacedAfterIt`. This is a
   locked-test amendment; developer re-approves it as a distinct, auditable
   commit.

## Out of scope

- No production code change. The real adapter and the #5 handler are already
  correct; this fixes the test double + the false test only.
- `#6` (frontend) `deriveNeighbors` is corrected separately, pre-commit, in its
  own Pre-Phase F session (not here).

## Verification

- `make test` — `TestMoveBookmark_ValidBeforeNeighbor_PlacedAfterIt` passes
  against the corrected fake; every other `#5` API test still passes (the
  both-nil and stale/fallback cases are direction-agnostic — they resolve to
  end-of-column regardless).
- `make test-int` — the #4 adapter suite is untouched and stays green (the real
  adapter was always correct).
- Confirm no other consumer of the fake asserts the old direction:
  `grep -rn "immediately before the anchor\|land.*before" internal/`.

## Acceptance Criteria

- The fake's `Move` places a card with `before: X` immediately **after** `X`,
  and with `after: Y` immediately **before** `Y`, matching `ports.go` and the
  real adapter on the same fixture.
- #5's neighbor test asserts the corrected (real-adapter-matching) order and is
  green; the full `go test ./...` suite is green.

## Labels

fix, tier-2, test-infra, locked-test-amendment
