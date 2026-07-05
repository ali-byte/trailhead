# New Project Setup Prompt

Use this prompt in Cowork or Claude Code to initialise a new project
within the dev directory structure. Run once per new project.

---

## Prompt

```
I am starting a new software project within my development environment.

Dev root: ~/dev/   (or wherever this structure lives)
Project name: [PROJECT_NAME]
GitHub org: [ORG_NAME]

Step 1 — Copy the template
  Copy ~/dev/.template/ to ~/dev/projects/[PROJECT_NAME]/
  Do not copy .git/ if it exists in the template.

Step 2 — Initialise git
  cd ~/dev/projects/[PROJECT_NAME]/
  git init
  git remote add origin https://github.com/[ORG_NAME]/[PROJECT_NAME].git
  Create an initial commit: git commit --allow-empty -m "chore: initialise project"

Step 3 — Update project CLAUDE.md
  In ~/dev/projects/[PROJECT_NAME]/CLAUDE.md:
  - Set Project Name to [PROJECT_NAME]
  - Set GitHub repo to the remote URL
  - Set Started to today's date
  - Set Phase to A

Step 4 — Update root project index
  In ~/dev/CLAUDE.md, add a row to the Project Index table:
  | [PROJECT_NAME] | Active | [today's date] | https://github.com/[ORG]/[PROJECT_NAME] |

Step 5 — Verify
  Confirm: ~/dev/projects/[PROJECT_NAME]/ exists with all template files
  Confirm: CLAUDE.md in project directory is updated
  Confirm: root CLAUDE.md project index is updated

Step 6 — Report
  Print the project directory structure.
  Tell me what to do next: open Claude Code pointed at the project directory
  and begin Phase A (Domain Design Interview — see root CLAUDE.md).

Do not begin Phase A in this session. Just set up the structure.
Phase A begins in a fresh Claude Code session pointed at the project directory.
```

---

## After Running

Open Claude Code:
  `cd ~/dev/projects/[PROJECT_NAME] && claude`

The root CLAUDE.md will be automatically loaded.
Begin with the Phase A Domain Design Interview (grill-me).
