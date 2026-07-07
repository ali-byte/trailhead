# Source -- trailhead-rules

## External Source
Original -- no external source. This is a project-local skill synthesized
entirely from Trailhead's own locked documents.

## Last Reviewed Commit/Date
2026-07-07 (Phase E gate, written against the working-tree state after the
Phase B gate round 5 fixes and Phase C planning -- see git log for the
exact commit this corresponds to once committed).

## What to Read
Not applicable in the external-source sense -- there is no upstream
document to review periodically. Instead, this skill's own content must be
re-synced whenever any of its four source-of-truth documents change:
  - DECISIONS.md
  - docs/GLOSSARY.md
  - docs/ARCHITECTURE_RFC.md
  - internal/adapter/ports.go
If any of those four files changes in a way that would make a sentence in
SKILL.md wrong, update SKILL.md in the same PR that changes the source
document -- do not let this skill drift out of sync silently.

## Build Notes
Written at the Phase E gate (2026-07-07), triggered by an explicit CLAUDE.md
process-sequence check: the developer caught that Phase C planning had
jumped straight to Pre-Phase F test generation, skipping Phase D (Quality
Infrastructure) and Phase E (Skills Setup) entirely. This skill is Phase
E's required "project-specific skill" deliverable -- CLAUDE.md's Phase E
gate condition is "all project skills in .claude/skills/, committed."
Content was pulled directly from the four source documents listed above,
not invented -- every fact here (package structure, GLOSSARY terms, the
ports.go READ-ONLY rule, ErrorKind HTTP mapping, JSON wire-key tags, build
commands) already existed, locked, in one of those documents before this
skill was written.

## What We Changed from Source
- Nothing changed from any external source -- there is none.
- Relative to the workshop's generic write-a-skill template, this skill
  uses `category: project` (not workflow/database/language/platform) and
  lives at the project-local `.claude/skills/trailhead-rules/` path rather
  than the shared `~/dev/.claude/skills/[category]/` library, matching the
  Dispatch Bootstrap Template's `.claude/skills/[project]-rules/SKILL.md`
  convention (project-relative, not shared-library).
- Structured as Rules (flat, guardrail-style) rather than numbered Phases,
  per write-a-skill's own Phase 2 guidance ("USE RULES WHEN: the skill
  modifies ongoing behaviour rather than running a process... about what
  NOT to do (guardrails)") -- this skill is exactly that: constant
  constraints a Dispatch session must not violate, not a sequential
  procedure.

## Why We Changed It
The generic template assumes a shared, cross-project skill; this skill is
deliberately single-project and colocated with the code it governs so it
travels with the project (and is deleted with it, per this project's
DELETE retention policy) rather than accumulating in the shared workshop
library where it would have zero value to any other project.

## SKILLS_REGISTRY.md Exemption

This skill is intentionally NOT registered in the shared
`~/dev/.claude/skills/SKILLS_REGISTRY.md`. That registry is scoped to the
shared, cross-project workshop skill library (`~/dev/.claude/skills/`);
project-local skills living inside a project's own worktree
(`.claude/skills/[project]-rules/`, per the Dispatch Bootstrap Template)
are out of that registry's scope by construction -- they travel with the
project and are deleted with it (per this project's DELETE retention
policy), not tracked centrally. write-a-skill's Phase 5 registration step
applies to shared-library skills; this note documents the exemption for
project-local skills explicitly rather than leaving it an open question
(Phase E gate reviewer-agent finding, 2026-07-07).
