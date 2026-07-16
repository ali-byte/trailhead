## Context

Phase 2 (docs/PHASE_PLAN.md): implement `internal/adapter/postgres.
Repository.Move`, including the fractional-rank position algorithm
ARCHITECTURE_RFC.md's Scope Boundary explicitly deferred past Phase 1 —
this issue resolves that open architecture question in code. Also
implements the `FinishedAt ⟺ Done` invariant and the neighbor-fallback
rule.

References: docs/PRD.md (Goals 2, 3) | docs/PHASE_PLAN.md (Phase 2) |
DECISIONS.md ("Position / Ordering Representation", "Move — Neighbor
Fallback (Generalized)", "FinishedAt ⟺ Done invariant") |
docs/ARCHITECTURE_RFC.md ("Scope Boundary", "Persistence Schema" —
Position-uniqueness application-enforced-invariant note)

## Acceptance Criteria

Given a bookmark in Inbox
When `Move` is called with `TargetStatus = StatusDone`
Then the bookmark's `Status` becomes `Done` and `FinishedAt` is set to the
current time

Given a bookmark in Done
When `Move` is called with a `TargetStatus` other than `Done`
Then `FinishedAt` is cleared to `nil`

Given a `MoveCommand.Before` or `.After` that is missing, cross-status, or
self-referential
When `Move` is called
Then the bookmark is inserted at the end of the target column's ordering,
the request does not fail, and no other row's `Position` in that column is
rewritten unnecessarily

Given both `Before` and `After` are set and refer to different neighbors
When `Move` is called
Then `Before` takes precedence and `After` is ignored (a tie-break, not a
fallback — see DECISIONS.md "Move — Neighbor Fallback (Generalized)",
narrowed at Phase B gate round 5)

Given a column with multiple bookmarks
When a bookmark is moved to a position between two existing neighbors
Then its new `Position` sorts strictly between the neighbors' `Position`
values without rewriting either neighbor's row

## Decision this issue must make (not deferred further)

**E1 (Phase B gate accepted risk) — RESOLVED at Pre-Phase F interview,
2026-07-13.** `internal/api` (issue #5) validates `TargetStatus` via a
new `domain.Status.IsValid() bool` method before ever constructing a
`MoveCommand`, rejecting an unrecognized value with 400. `Move` trusts
`TargetStatus` and does not defend against garbage at the Go level; the
`bookmarks` table's `CHECK` constraint is the defense-in-depth layer.
See `DECISIONS.md` "MoveCommand.TargetStatus — Validation Ownership (E1,
resolved)" and `ports.go`'s updated `Move` doc comment. This issue's own
test coverage is limited to one defensive test proving a garbage status
that somehow reaches `Move` fails cleanly via the `CHECK` constraint (a
plain infra error), not a 400 test — the 400 behavior belongs to issue #5.

**Decision B (position collision, deferred from issue #2) — RESOLVED at
Pre-Phase F interview, 2026-07-13.** Migration `000002` adds `UNIQUE
(status, position)`, enforced for both `Move` and `Create`. See
`DECISIONS.md` "Position Collision Handling (Decision B, resolved)" and
`ARCHITECTURE_RFC.md`'s "Persistence Schema" amendment, same date.
**This adds two deliverables to this issue beyond the adapter code:**
(1) migration `000002_add_bookmarks_status_position_unique.up/down.sql`;
(2) revise `Create`'s `initialPosition` constant scheme in
`repository.go` to a computed, distinct front-insert rank — `Create`'s
existing issue #2 tests still pass (they assert relative order, not the
literal rank string), but the constant-value approach no longer satisfies
the new `UNIQUE` constraint under concurrent `Create`s into an empty or
single-row column.

**Locked index name for migration `000002` (required — tests key off
this exact name, same pattern as `identityHashConstraint`):** drop
`000001`'s non-unique `bookmarks_status_position_idx` and replace it with
`CREATE UNIQUE INDEX bookmarks_status_position_unique_idx ON bookmarks
(status, position)` — one index serving both the ordering query and the
uniqueness guarantee, not two redundant indexes. The Go-level constant
analogous to `identityHashConstraint` (`repository.go`) should be named
`statusPositionConstraint = "bookmarks_status_position_unique_idx"`, used
by `Move`'s (and `Create`'s) constraint-name-specific `23505` retry logic.

## Out of Scope

`Update`, `Delete` (Phase D). The `internal/api` move route (issue #5) —
this issue is adapter-only.

## Test Targets

Drafted at Pre-Phase F, 2026-07-13 — 18 interview-driven tests in
`tests/integration/postgres/move_test.go` (corrected at Pre-Phase F —
this line previously said
`internal/adapter/postgres/repository_test.go`, the same stale-path
mistake #2's Test Targets line originally had, contradicting this same
issue's Gate condition below, which already says `./tests/integration/...`.
Co-located in the same package as issue #2's `tests/integration/postgres/`
suite, reusing its `setup_test.go` helpers — extended with `mutableClock`,
`insertRawBookmarkAt`, `testUUID` — package `postgres_test`, build tag
`integration`).

Final test list: `TestMove_MissingNeighborID_FallsBackToEndOfColumn`,
`TestMove_CrossStatusNeighborID_FallsBackToEndOfColumn`,
`TestMove_SelfReferentialNeighborID_FallsBackToEndOfColumn`,
`TestMove_BeforeAndAfterBothSet_BeforeWins`,
`TestMove_BeforeInvalidAfterValid_FallsBackIgnoringValidAfter`,
`TestMove_IntoDone_SetsFinishedAtToClockAndUpdatedAt`,
`TestMove_OutOfDone_ClearsFinishedAtToNULL`,
`TestMove_DoneToDone_PreservesFinishedAtButBumpsUpdatedAt`,
`TestMove_NonDoneToNonDone_FinishedAtStaysNil`,
`TestMove_UnknownID_ReturnsNotFound_NoSideEffect`,
`TestMove_IntoEmptyColumn_ValidStartingRank`,
`TestMove_RepeatedInsertsIntoSameGap_MaintainsStrictOrder`,
`TestMove_ConcurrentSameGapMoves_BothSucceedDistinctRanksCorrectOrder`,
`TestSchema_StatusPositionUniqueIndex_FiresOnDuplicateRawInsert`,
`TestSchema_StatusPositionColumn_CollationStillC`,
`TestBoard_DuringInFlightMoveTransaction_SnapshotConsistent`,
`TestMove_ContextCanceledBeforeCommit_RowUnchanged`,
`TestMove_InvalidTargetStatus_FailsCleanlyViaCheckConstraint`.

Gate condition from PHASE_PLAN.md (Phase 2): `go test -tags integration
./tests/integration/... -run TestMoveBookmark` passes against real
Postgres per the acceptance criteria above.

## Parallel Safety

Can run alongside: none
Must wait for: Postgres Adapter — Create & Board (issue #2)
Blocks: API — Move route (issue #5)

## Labels

phase-2, tier-1, postgres
