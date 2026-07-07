## Context

Phase 1 (see docs/PHASE_PLAN.md) is the first phase that runs against a
real PostgreSQL instance. This issue is the housekeeping prerequisite:
local Postgres via Docker Compose, the env vars `cmd/trailhead/main.go`
will read, and a README stub workshop Phase H will later extend. No
production Go code changes in this issue — pure scaffolding, low risk,
unblocks every other Phase 1/2 issue.

References: docs/PRD.md | docs/PHASE_PLAN.md (Phase 1) | DECISIONS.md

## Acceptance Criteria

Given a fresh clone of the repo
When the developer runs `docker-compose up -d` then `make migrate`
Then a local Postgres instance is running and reachable at the
`DATABASE_URL` documented in `.env.example`, with the `bookmarks` table
and all three indexes created per ARCHITECTURE_RFC.md "Persistence Schema"

Given `.env.example`
When a developer copies it to `.env` and fills in real values
Then every env var `cmd/trailhead/main.go` reads (`DATABASE_URL`, `PORT`)
is documented with a comment explaining its purpose and an example value

## Out of Scope

Any Go code change (repository implementation, API routes, cmd wiring)
— those are issues #2 and #3. Production deployment config (Phase H).

## Test Targets

No automated test — this is infrastructure/config only. Manual
verification: `docker-compose up -d && make migrate` succeeds with no
errors on a clean checkout.

Gate condition from PHASE_PLAN.md (Phase 1, partial — this issue covers
the scaffolding sub-slice only): N/A for this issue alone; contributes to
Phase 1's overall gate.

## Parallel Safety

Can run alongside: none filed yet (first issue)
Must wait for: none — can start immediately
Blocks: Postgres Adapter — Create & Board (issue #2), API — Create & Board
routes (issue #3)

## Labels

phase-1, tier-3, infra
