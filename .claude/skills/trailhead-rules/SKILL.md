---
name: trailhead-rules
description: "Project-specific guardrails for Trailhead: package structure,
  ubiquitous-language terms, the locked adapter interface's location and
  READ-ONLY status, error-kind/wire-key naming, and build/lint commands.
  Load in every Trailhead Dispatch session (per the Dispatch Bootstrap
  Template's 'Load: .claude/skills/[project]-rules/SKILL.md (always)')."
status: draft
category: project
---

# Trailhead Rules

## The Principle

Every fact in this skill already exists, locked, somewhere in this
project's own docs (DECISIONS.md, docs/GLOSSARY.md, docs/ARCHITECTURE_RFC.md,
`internal/adapter/ports.go`). This skill does not introduce new rules -- it
collects the ones a Dispatch session needs constantly into one place so it
doesn't have to re-derive them from four different files every time, and so
it can't quietly drift from them mid-session. If this skill and one of
those source documents ever disagree, the source document wins -- fix this
skill's wording, don't treat this skill as an independent authority.

---

## When to Use This Skill

Load this skill at the start of every Trailhead Dispatch session -- Phase F
build-loop, Pre-Phase F test generation, and any Cowork session touching
project code. This is a project-local skill: it lives at
`.claude/skills/trailhead-rules/` inside this project's own worktree, not
in the shared `~/dev/.claude/skills/` workshop library, because none of it
applies outside Trailhead.

Do NOT use this for architecture decisions not yet made (see
docs/ARCHITECTURE_RFC.md and DECISIONS.md directly for those) or for
generic Go patterns not specific to this project (see the shared
`languages/go-patterns` skill instead).

---

## Rule 1 -- Package Structure (Locked, ARCHITECTURE_RFC.md "Package Organization")

```
trailhead/
├── cmd/trailhead/           main.go -- wiring only, entry point
├── internal/domain/         types + Canonicalize/DeriveIdentityHash/DefaultTitle
├── internal/adapter/        ports.go -- BookmarkRepository (Locked Interface)
│   └── postgres/            Postgres implementation
├── internal/api/             chi handlers
├── internal/testutil/        FakeBookmarkRepository -- test helpers only
├── web/                       React/TS/Tailwind SPA
└── tests/
    └── integration/          end-to-end tests, //go:build integration
```

Import direction (any violation is a P1 merge blocker):
`adapter -> domain`, `adapter/postgres -> adapter, domain`,
`api -> adapter, domain`, `testutil -> adapter, domain`,
`cmd -> all packages`. Forbidden both ways: `domain` importing anything
outside the standard library; `api` importing `adapter/postgres` directly
(must go through the `adapter` interface); anything importing `testutil`
in production code.

Unit tests are co-located `*_test.go` files next to the code they test --
there is no separate `tests/unit/` directory in this project (a deliberate
Go-convention deviation from the generic workshop template; see
ARCHITECTURE_RFC.md "tests/ Directory Structure").

---

## Rule 2 -- Ubiquitous Language (Locked, docs/GLOSSARY.md)

Use these exact identifiers in code, tests, and issue text. Never
substitute the brief's prose synonyms as field/type/variable names:

| Canonical term | Never use as an identifier |
|---|---|
| `Status` | "column" |
| `Position` | "priority" |

Core types (package `domain` unless noted): `Bookmark`, `Status`
(`StatusInbox`/`StatusReading`/`StatusDone`), `Position`, `CanonicalURL`,
`IdentityHash`, `Tags`, `Board`, `BookmarkID`, `NewBookmark`,
`BookmarkPatch` (has `ClearAuthor bool` -- tri-state, not a plain
pointer-optional field). Package `adapter`: `BoardFilter`, `MoveCommand`,
`RepositoryError`, `ErrorKind`.

---

## Rule 3 -- Adapter Interface Location and READ-ONLY Status

`internal/adapter/ports.go` defines `BookmarkRepository` and is
**READ-ONLY as of the Phase B gate close (2026-07-06)**. Any modification
to this file -- including a doc-comment change -- requires a filed RFC and
explicit developer approval before the file is touched. A Dispatch session
that finds itself editing `ports.go` should stop immediately, revert, and
notify the developer before doing anything else (same rule as any other
READ-ONLY file named in a Dispatch bootstrap).

`internal/testutil/fake_repository.go`'s `FakeBookmarkRepository` is NOT
read-only -- it is a Tier 3 test double, and its own doc comment names two
deliberate test-double simplifications that do NOT apply to production
code: `Position`'s zero-padded-index bookkeeping (real fractional-rank
algorithm is Phase F work behind the Postgres adapter) and, as of the
Phase B gate round 5 fix, `nextID`'s real version-4 UUID generation (this
one is NOT a simplification -- it was corrected specifically because the
fake's IDs flow into real code paths and must match the locked
`BookmarkID` representation).

---

## Rule 4 -- Error Kinds and Wire-Key Naming

`ErrorKind` constants and their HTTP mapping (`internal/api`, Phase F --
not yet built, but the mapping is locked): `ErrKindDuplicate` -> 409,
`ErrKindNotFound` -> 404, `ErrKindInvalidURL` -> 400. Infrastructure
failures (Postgres unreachable, context cancellation/timeout) are a plain
wrapped `error`, never a `*RepositoryError` -- see DECISIONS.md "Repository
Error Taxonomy -- Infrastructure Failures." These map uniformly to 5xx.

JSON wire keys are locked snake_case via struct tags, not Go's default
capitalized field names -- every `domain.Bookmark` and `domain.Board` field
already carries the correct tag (`id`, `original_url`, `canonical_url`,
`identity_hash`, `title`, `tags`, `status`, `position`, `finished_at`,
`author`, `created_at`, `updated_at`, and `inbox`/`reading`/`done` on
`Board`). `CreatedAt`/`UpdatedAt` are always-present `time.Time` (never
pointers); `FinishedAt`/`Author` are `*time.Time`/`*string` and always
included in the response body even when `null` (no `omitempty`) -- see
ARCHITECTURE_RFC.md "Serialization Spec."

`Create` sets `UpdatedAt` equal to `CreatedAt` (not just "some current
time" independently derived) -- see DECISIONS.md "UpdatedAt -- Write-Path
Contract."

---

## Rule 5 -- Build, Lint, and Test Commands

```
make build       # go build -o bin/trailhead ./cmd/trailhead/
make lint         # golangci-lint run ./...  (v2 schema, .golangci.yml at repo root)
make test         # go test ./... -count=1 -race
make test-int     # go test -tags integration ./tests/integration/... -count=1 -v
make check        # lint + test + build, run before opening any PR
```

`.golangci.yml` uses the v2 config schema (`version: "2"`,
`linters.default`, `linters.settings`, `linters.exclusions.rules`,
separate top-level `formatters:` block) -- do not add v1-schema keys
(`linters-settings`, `issues.exclude-rules`, `disable-all`/`enable-all`)
when editing it. `go build`/`go vet`/`go mod tidy` do not run inside a
Cowork sandbox session -- they run in Terminal on the developer's Mac (Go
Toolchain Note, CLAUDE.md Phase D).

---

## Hard Rules

- `internal/adapter/ports.go` is READ-ONLY. No exceptions without a filed
  RFC and explicit developer approval.
- Never use "column" or "priority" as a Go identifier, DB column, or API
  field name -- use `Status` / `Position`.
- Every mutating `BookmarkRepository` method sets `UpdatedAt`; `Create`
  sets it equal to `CreatedAt`.
- Classified failures (`Duplicate`/`NotFound`/`InvalidURL`) are always
  `*RepositoryError`; infrastructure failures are always a plain wrapped
  `error`. Never mix the two.
- Unit tests are co-located `*_test.go` files. Do not create a
  `tests/unit/` directory for this project.
- `go build`/`go vet`/`go mod tidy` run in Terminal, never inside a Cowork
  sandbox session.
