# Architecture RFC — Risk Tier Register

## What This Table Is

Every package in this project is assigned a blast radius tier before
any code is written. The tier determines:

  - Interview depth in the test-engine (Tier 1: 20Q, Tier 2: 12Q, Tier 3: 6Q)
  - Review depth in the middle loop (Tier 1: line-by-line, Tier 3: tests passing is sufficient)
  - Security scan response (Tier 1 finding: block merge, Tier 3: assess first)
  - Session pairing (Tier 1 packages: developer co-builds at least one slice)

Tiers are assigned at architecture time and do not change without a
formal RFC amendment. If a package grows to include Tier 1 concerns
that weren't anticipated, file an RFC before writing that code.

---

## Blast Radius Tier Definitions

### Tier 1 — Critical
**Blast radius if wrong: data loss, security breach, or compliance failure.**

Characteristics:
  - Handles authentication, authorisation, or credential management
  - Writes to persistent storage (creates, updates, or deletes records)
  - Generates compliance evidence or audit logs
  - Processes or transmits data classified as sensitive (PII, health, financial)
  - Controls access to external systems or APIs

Review requirements:
  - Human review of every line before merge
  - Developer co-builds at least one vertical slice in this package
  - test-engine interview: maximum 20 questions, adversarial stance
  - gosec finding in Tier 1 package: block merge, no exceptions
  - Architecture checkpoint required before adding new Tier 1 packages

### Tier 2 — Standard
**Blast radius if wrong: incorrect results, degraded performance, or poor UX.**

Characteristics:
  - Core business logic that does not touch Tier 1 concerns
  - API endpoints that read (but do not write) sensitive data
  - Diagnostic and monitoring logic
  - Report generation and rendering

Review requirements:
  - Human review of the diff as a whole (not necessarily line by line)
  - test-engine interview: maximum 12 questions
  - gosec finding in Tier 2 package: assess whether it is a true positive

### Tier 3 — Low Risk
**Blast radius if wrong: incorrect formatting, minor errors, or inconvenience.**

Characteristics:
  - Formatters, utilities, and helper functions
  - CLI output formatting
  - Read-only queries that return non-sensitive data
  - Test fixtures and seed helpers

Review requirements:
  - Tests passing is sufficient for merge
  - test-engine interview: maximum 6 questions, mostly auto-generated
  - gosec finding in Tier 3 package: assess, likely acceptable with comment

---

## Package Tier Register

Fill this table during the Architecture RFC session (Phase B).
Every internal package must have a tier assigned before Phase C begins.

| Package path                    | Tier | Reason                                         |
|---------------------------------|------|------------------------------------------------|
| internal/adapter/adapter.go     | 1    | Interface contract — all data flows through it |
| internal/adapter/mongo/         | 2    | Reads engine state, does not write user data   |
| internal/adapter/elastic/       | 2    | Reads engine state, does not write user data   |
| internal/adapter/redis/         | 2    | Reads engine state, does not write user data   |
| internal/adapter/cassandra/     | 2    | Reads engine state, does not write user data   |
| internal/snapshot/              | 2    | Collects and stores operational snapshots      |
| internal/diff/                  | 2    | Core AWR comparison logic                      |
| internal/rules/                 | 2    | Evaluates diagnostic rules against diffs       |
| internal/report/                | 2    | Renders diagnostic reports (HTML/PDF/JSON)     |
| internal/storage/               | 1    | Writes all persistent data to PostgreSQL       |
| internal/monitor/               | 2    | Alerting and metric streaming                  |
| internal/audit/                 | 1    | Compliance evidence — SOC2/HIPAA artifacts     |
| internal/api/                   | 2    | REST API (reads and writes)                    |
| cmd/<project>/                    | 3    | CLI entry point, output formatting             |
| cmd/<project>d/                   | 3    | Daemon entry point                             |
| pkg/                            | 3    | Public types and client SDK                    |
| test/integration/               | 3    | Test infrastructure — no production impact     |
| web/                            | 3    | React UI — client-side only                    |

NOTE: This table is for any future project as an example. Each project must
produce its own tier register during its Phase B Architecture RFC.
Populate using the blank template below.

---

## Trailhead — Package Tier Register (Phase B, 2026-07-04)

| Package path                      | Tier | Reason |
|------------------------------------|------|--------|
| `internal/adapter` (`ports.go`)    | 1    | Interface contract — all data flows through it; a signature drift here breaks every caller silently until compile time, or worse, at runtime if types were loosely typed. |
| `internal/adapter/postgres`        | 1    | Writes all persistent data (creates/updates/deletes bookmarks); a bug here is data loss or data corruption — the canonical Tier 1 characteristic. Also the sole enforcement point for the `finished_at ⟺ Done` invariant and duplicate-detection race-safety (unique index + conflict handling). |
| `internal/domain`                  | 1    | Silent-failure-check applied (per go-workflow-observations.md Phase B lesson from a prior project): `Canonicalize`/`DeriveIdentityHash` are pure functions with no error-raising failure mode if the *rules themselves* are subtly wrong — a wrong tracking-param deny-list entry or a wrong trailing-slash rule silently lets duplicates through or silently over-merges distinct bookmarks, across every future Create call, with no exception ever thrown. This is exactly the "could a bug here degrade output quality silently without raising an error?" test that promotes a package to Tier 1. |
| `internal/api`                     | 2    | Core business logic (HTTP <-> repository translation) that does not itself touch storage or credentials; reads and writes via the Tier 1 repository interface but the interface's own contract is what's load-bearing, not the handler glue. |
| `internal/testutil`                | 3    | Test infrastructure only — no production impact. A bug here produces a misleading test result, not a production incident (though see the four adversarial invariants in fake_repository.go — those exist precisely to keep this package trustworthy despite its Tier 3 review depth). |
| `cmd/trailhead`                    | 3    | Entry point / wiring only — CLI/server bootstrap, output formatting, no business logic of its own. |
| `web/`                             | 2    | Client-side React/TS UI — incorrect drag/reorder behavior or filter logic produces degraded UX (a core brief concern — "what good looks like") but no data loss, since all persistence decisions are server-side. Not Tier 1: a UI bug is recoverable by reload; a repository bug is not. |
| `tests/integration/`               | 3    | Test infrastructure — no production impact. |

---

## Blank Template (copy for new projects)

| Package path | Tier | Reason |
|--------------|------|--------|
| | | |

---

## Tier Amendment Process

If a package's tier needs to change after the RFC is approved:

1. File a GitHub issue titled "RFC: Tier change — [package] from Tier X to Tier Y"
2. Describe: what changed about the package's responsibilities?
3. What additional test coverage is needed for the new tier?
4. What review process changes are needed?
5. Approved by developer review before any code in the package is written

Do not change a tier silently. The tier is a commitment, not a label.
