# tests/integration/

Placeholder — this directory is materialized at the Phase B gate per
ARCHITECTURE_RFC.md "tests/ Directory Structure" and "Greenfield Skeleton",
but its actual test files are a Phase F deliverable, built alongside
`internal/adapter/postgres` (the Postgres implementation of
`BookmarkRepository`).

Tests here will be gated behind `//go:build integration` and will run
against a real PostgreSQL instance (not `internal/testutil.FakeBookmarkRepository`),
exercising the schema in ARCHITECTURE_RFC.md "Persistence Schema" directly —
in particular the unique-violation-driven duplicate-detection path flagged
as a Phase B reviewer-agent risk (see the Phase B gate observation entry in
`workshop/observations/trailhead-methodology-observations.md`).
