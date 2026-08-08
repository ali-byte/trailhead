# Trailhead — Design System
#
# INSTRUCTIONS FOR USE:
# 1. This file is the single source of truth for all UI decisions.
# 2. Every Dispatch session that touches frontend code reads this FIRST and
#    builds exactly to the system defined here — no invented typography,
#    color, spacing, or component patterns.
# 3. When any design decision changes, update the relevant section AND add a
#    changelog entry at the bottom — never edit silently.
# 4. design.md is version-controlled — treat changes like code: intentional,
#    documented, reviewed.

---

## Project Context

Product type: webapp — a single-user, single-view read-it-later board (no
              accounts, no multi-page navigation; one board is the whole app).
Primary audience: one person — the owner of their own read-it-later board,
                  using it daily to triage links they mean to read. Not a team
                  tool; no collaboration, no onboarding, no marketing surface.
Tone: calm & focused — quiet editorial restraint over warm paper neutrals. A
      reading tool, not a productivity dashboard. Closest single template
      value: editorial/magazine, warmed and softened.
Personality in three words: calm, warm, focused.
What this product is NOT: not a gamified kanban (no badges, streaks, counts,
      or bright category colors); not corporate-SaaS blue; not a dark
      "developer tool" dashboard; not Trello's loud primary-color board;
      not startup-generic.

---

## Typography

Display font: Fraunces — used for the wordmark, column headers, card titles,
              and empty-state display. A soft, warm serif that gives the board
              its editorial/reading character.
  Load from: Google Fonts — https://fonts.google.com/specimen/Fraunces
             (weights 400, 500, 600; opsz 9..144 optical sizing enabled)
Body font: Hanken Grotesk — used for all UI copy: labels, metadata, tag chips,
           buttons, timestamps, body text.
  Load from: Google Fonts — https://fonts.google.com/specimen/Hanken+Grotesk
             (weights 400, 500, 600)
Monospace font: system monospace — used only for the raw/canonical URL in the
                card-detail view.
  Load from: system-ui monospace — ui-monospace, SFMono-Regular, Menlo,
             "Cascadia Mono", monospace

Banned in this project:
  - Inter (overused AI default)
  - Space Grotesk (overused AI default)
  - Geist (overused AI default)
  - Roboto / Open Sans (generic)

Type scale (base 16px):
  xs:  12px — tag chips, timestamps, secondary metadata
  sm:  14px — labels, card host/domain line, secondary UI copy
  base:16px — body text, card title at rest, input text
  lg:  18px — card titles (Fraunces), lead copy
  xl:  22px — column headers (Fraunces)
  2xl: 28px — board heading / section titles
  3xl: 36px — "Trailhead" wordmark, empty-board display

Weight usage:
  Regular (400): body text, metadata, card host line, timestamps
  Medium (500):  labels, buttons, column headers, tag chips
  Semibold (600):card titles, the wordmark, inline emphasis
  Bold (700):    not used — Semibold is the heaviest weight

---

## Color Palette

(All colors are CSS custom properties. No magic hex values in component code —
every color a component uses must be one of these tokens.)

Background colors:
  --color-bg:          #F5F1EA — board / page background (warm paper)
  --color-surface:     #FFFFFF — bookmark card backgrounds, modal surface
  --color-surface-2:   #EDE7DC — the three column containers (a slightly
                                 deeper warm tint the cards sit on)
  --color-raised:      #FFFFFF — card-detail modal, dropdowns, toasts
                                 (elevated via shadow, not a different hue)

Text colors:
  --color-text:        #2B2926 — primary body text, card titles (warm near-black)
  --color-text-dim:    #6E665C — secondary/muted: host line, timestamps, metadata
  --color-text-bright: #1A1815 — headings, the wordmark, strong emphasis

Brand colors:
  --color-primary:     #9A4E2E — the single accent: primary button background
                                 (with white text), focus ring, active borders
  --color-primary-dim: #7F4026 — primary hover / pressed
  --color-accent:      #9A4E2E — same terracotta; there is ONE accent by design
                                 (accents used sparingly)

Semantic colors:
  --color-border:      #E4DDD1 — hairline card/input borders (warm)
  --color-border-hi:   #9A4E2E — focused / active borders (the primary)
  --color-success:     #4F7A52 — muted sage (Done column marker, success)
  --color-warning:     #9A6B12 — muted amber (rare; non-blocking notices)
  --color-error:       #A23B2E — muted brick red (destructive actions, errors)

Status column markers (quiet per-column identity — used ONLY as a small dot /
thin header underline, never as a column background fill):
  --color-status-inbox:   #6B7280 — muted slate
  --color-status-reading: #9A4E2E — terracotta (in-progress warmth)
  --color-status-done:    #4F7A52 — sage

Color philosophy: warm-paper light theme, one terracotta accent for actions and
focus, three muted status markers (slate / terracotta / sage) that give the
columns quiet identity via a small dot or header underline — never loud fills.
High-contrast warm-black text. No gradients, no pure black, no pure-white page
background, no bright saturated category colors.

CONTRAST NOTE: every foreground/background pair below the Accessibility Baseline
must be verified at build time (white on --color-primary for buttons ≥ 4.5:1;
--color-text-dim on --color-bg ≥ 4.5:1). If any pair fails, darken the token
here and add a changelog entry — do not adjust per-component.

---

## Spacing & Shape

Base unit: 4px — all spacing is a multiple of 4; the primary rhythm is 8/16/24.

Border radius:
  sm:  4px  — inline elements, focus-ring corners
  md:  8px  — buttons, the Add-bar input
  lg:  12px — bookmark cards
  xl:  16px — the card-detail modal, column containers
  full:9999px — tag chips (pills)

Shadow philosophy: subtle only. Resting cards use a 1px --color-border, NO
shadow. Shadows appear only for genuine elevation:
  --shadow-drag:      0 8px 24px rgba(43,41,38,0.14) — a card lifted while dragging
  --shadow-modal:     0 12px 40px rgba(43,41,38,0.18) — card-detail modal, dropdowns
  --shadow-highlight: 0 0 0 3px rgba(154,78,46,0.45) — the brief terracotta ring
                      on the existing card when a duplicate URL is added (409),
                      time-boxed and removed after ~1.5s; see Component
                      Conventions → Error states (409)
Never a resting drop-shadow on static cards.

---

## Component Conventions

Buttons:
  Primary: --color-primary background, #FFFFFF text, no border, radius md;
           hover → --color-primary-dim. Used for "Save" (Add bar) and the
           primary action in the card-detail (Save changes).
  Secondary: transparent background, --color-text text, 1px --color-border;
             hover → --color-surface-2 background. Used for "Cancel".
  Destructive: transparent background, --color-error text + 1px --color-error
               border; hover → --color-error background with #FFFFFF text.
               Visually distinct from primary (red, never terracotta).
  Disabled: --color-surface-2 background, --color-text-dim text, no pointer
            cursor; must still meet 3:1 contrast.
  Size: 40px min height, 16px horizontal padding, base font, Medium weight;
        on mobile, padding brings the touch target to ≥ 44×44px.

Inputs:
  Style: --color-surface background, 1px --color-border, radius md, 12px
         padding; focus → 2px --color-border-hi ring at 2px offset + border-hi.
  Label position: above the field (visually-hidden label on the Add bar, whose
                  placeholder doubles as the visible hint; visible labels on
                  card-detail edit fields).
  Placeholder: hint only — e.g. "Paste a link to save it…" — never the sole
               label.
  Validation: inline, below the field.
  Error state: --color-error border + an inline message below (e.g. duplicate
               → "This is already on your board" with a link to the existing
               card; invalid → "That doesn't look like a web address").

Cards (the bookmark card — the core element):
  Style: --color-surface background, 1px --color-border, radius lg, 14px
         padding. Contents: title (Fraunces, lg, --color-text), a host/domain
         line (Hanken, sm, --color-text-dim), tag pills; for a Done card, a
         small "finished {date}" line (sm, --color-text-dim). No status label
         on the card — the column conveys status.
  Hover state (cards are draggable): border → --color-border-hi at low opacity,
         cursor grab; no movement at rest.
  Dragging state: --shadow-drag, scale 1.02, cursor grabbing; the source slot
         shows a dashed --color-border placeholder at reduced opacity (dnd-kit
         DragOverlay).
  Shadow: none at rest; --shadow-drag only while lifted.

Navigation:
  Pattern: none in the traditional sense — the app is one view. A slim sticky
           header holds the "Trailhead" wordmark (left), the Add bar (center /
           stretch), and the tag-filter control (right).
  Active state: n/a for nav; the tag-filter chips show active (filled
           --color-primary) vs inactive (outline).
  Mobile behaviour: the header stays sticky at top; the Add bar wraps to its
           own full-width row below the wordmark; the three columns stack
           vertically (see Layout Rules).

Loading states:
  Approach: skeleton screens for the initial board load — three column
            containers, each with 2–3 skeleton cards. No spinner for the board.
            Add and drag use optimistic UI (apply immediately, reconcile on the
            server response; roll back with a toast on failure).
  Skeleton style: gentle pulse; static under prefers-reduced-motion.

Error states:
  Inline errors: field-level, directly below the Add bar (see Inputs).
  Toast/notification: a single, unobtrusive toast anchored bottom-center on
            --color-raised with --shadow-modal; used for transient failures
            (e.g. "Couldn't save that move — put it back?" with a Retry
            action). Auto-dismiss after ~6s; not a persistent red banner.
  Page-level errors: if the board itself fails to load, a calm centered message
            on --color-bg — "Couldn't load your board. Check your connection
            and try again." — with a "Try again" button. Never a stack trace.
  Message format: every message says what happened AND what to do next. Never
            "Something went wrong."

Empty states (the PRD requires a designed calm empty state per column):
  Style: centered within the column container, --color-text-dim, generous
         whitespace, at most a single small line-art glyph. Quiet, not a
         full illustration.
  Content: explain what belongs and how to add it —
    Inbox:   "Nothing saved yet. Paste a link above to start your board."
    Reading: "Drag a card here when you start reading it."
    Done:    "Finished something? Drag it here to check it off."

Destructive actions:
  Confirmation: Delete requires explicit confirmation — an inline confirm step
    within the card-detail: "Delete this bookmark? This can't be undone." with
    a Cancel (secondary) and a Delete (destructive) button. No one-click delete
    anywhere; nothing is removed without the second, deliberate click.

---

## Layout Rules

Grid: CSS Grid for the board — three equal-width columns with a fixed gutter on
      desktop/tablet; each column is a vertical flex stack of cards. Below the
      stack breakpoint (≈ 720px), the grid collapses to a single full-width
      column and the three columns stack in order (Inbox, Reading, Done).
Max content width: 1120px, centered, with page padding. Cards fill their
      column width; column width is capped by the max content width, keeping
      titles at a comfortable reading measure.
Page padding:
  Mobile:  16px
  Tablet:  24px
  Desktop: 32px
Column gutter: 20px (desktop/tablet); n/a when stacked.

Asymmetry policy: symmetric — the three-column board is deliberately balanced
      (equal columns, equal gutters). This is the intended exception to the
      generic "avoid identical card grid" caution below: a board of equal
      columns is the correct form here, not a generic marketing grid.

Anti-patterns banned in this project:
  - Loud colored column backgrounds or headers (status is a small marker only)
  - Dense/compact "productivity app" card lists — cards stay airy (14px padding,
    clear separation)
  - Horizontal scroll of the board on desktop — columns always fit or stack
  - Full-width gradient hero / marketing chrome (there is no marketing surface)
  - Sticky elements that overlap or cover cards

---

## Motion

Philosophy: purposeful — motion communicates state change (a card lifting,
neighbors reflowing, a card settling on drop, a new card arriving). No
decorative motion anywhere.

Page load: cards fade + rise in with a subtle 40ms stagger per column, ≤ 150ms
           each; skip entirely under reduced-motion.
Route transitions: none (single view).
Component mount: a newly saved bookmark fades + slides in at the top of Inbox,
           150ms ease-out. The card-detail modal fades + scales from 0.98→1.0,
           150ms ease-out; backdrop fades in.
Interactive feedback: card hover lift 100ms; drag uses dnd-kit's transforms —
           lifted card follows the pointer, neighbors reflow with a ~200ms
           ease; drop settles the card into place.
Scroll: none — no scroll-triggered effects.

Reduced-motion rule (non-negotiable):
  All animation respects prefers-reduced-motion: reduce. When active:
    - Remove the load stagger, mount slides, hover lift, and drag reflow
      transition (dragging still works — the card repositions instantly).
    - Preserve every functional state change (modal open/close, card moving
      columns, filter applying).
    - Remove decorative motion only; never remove information or function.

---

## Do Not Use

Fonts:
  - Inter, Space Grotesk, Geist, Roboto, Open Sans

Colors:
  - No pure black (#000000) — use --color-text (#2B2926)
  - No pure-white page background — the board is warm paper (--color-bg)
  - No gradients of any kind
  - No bright / saturated category colors — status markers are muted only

Design patterns:
  - Gradient overlays on text or surfaces
  - Stock-photo or illustration-heavy hero imagery
  - Decorative divider lines between every element
  - Modal on top of modal (the card-detail is the only modal; nothing stacks
    over it)
  - Badges, streaks, counts, or any gamification chrome
  - Loud colored column headers or backgrounds

Layout patterns:
  - Dense compact card lists (keep cards airy)
  - Desktop horizontal scroll of the board
  - Content-covering sticky elements

Copy patterns:
  - Vague errors ("Something went wrong", "Error occurred")
  - Placeholder text used as the only label
  - Internal code terms in UI copy — the UI says "column" and the column names
    (Inbox / Reading / Done); it never exposes "Status", "Position",
    "IdentityHash", or other GLOSSARY.md code terms

---

## Reference & Inspiration

Visual references (aim for this feel):
  - Readwise Reader's library view — calm, warm, reading-first, content over chrome
  - Are.na — quiet paper surfaces, restraint, no decorative noise
  - iA Writer — warm neutral background, editorial typographic calm
  - Linear's board interactions (the drag/reflow feel) — but warmer, softer,
    and without the cool corporate palette

Anti-references (do NOT resemble):
  - Trello — loud primary-color boards, gamified chrome, dense lists
  - A generic dark "developer dashboard" — this is a warm, light reading tool,
    not an ops console

---

## Accessibility Baseline

(Non-negotiable — applies regardless of tone.)

Contrast:
  Text on background: minimum 4.5:1 (WCAG AA)
  Large text (≥ lg / 18px semibold): minimum 3:1
  Interactive elements / borders: minimum 3:1
  Verify at build time: white on --color-primary, --color-text-dim on
  --color-bg, and each status marker against its surface. Darken the token in
  the palette (with a changelog entry) if any pair fails — never patch per
  component.

Focus states:
  Every interactive element has a visible focus ring: 2px solid
  --color-border-hi, 2px offset. Never outline: none without this custom ring.
  dnd-kit keyboard dragging must keep a visible focus indicator on the lifted
  card throughout.

Semantic HTML:
  - h1: the board / "Trailhead"; h2: each column name — correct hierarchy, no
    skips.
  - Landmarks: header, main (the board), and the card-detail modal as a
    proper dialog (role="dialog", focus-trapped, Escape closes).
  - Each column is a labelled list; each card a list item.
  - button for actions; a for the card's external link (opens the real URL).
  - label for every input (visually-hidden is fine for the Add bar).

Motion:
  prefers-reduced-motion respected on all animation (see Motion).

Touch targets:
  Minimum 44×44px for every interactive element on mobile (cards, buttons,
  tag chips, filter controls).

Screen readers:
  - Icon-only controls (delete, filter, close) have aria-label.
  - Drag-and-drop is operable by keyboard (dnd-kit KeyboardSensor) and announces
    state via an aria-live region: "Picked up {title}. Moved to Reading,
    position 2 of 4. Dropped." Announcements use the UI terms (column names),
    never code terms.
  - The tag filter announces the active filter set and result count.
  - Form errors are announced via aria-live / aria-describedby.

---

## Changelog

- 2026-08-07: Added `--shadow-highlight` (Spacing & Shape → Shadow) — the brief
  terracotta ring shown on the existing card when a duplicate URL is added (409),
  introduced during the #6 build and promoted into the token set so every shadow
  still traces to design.md (Codex round-3 FLAG).
- 2026-07-23: Initial design system created (Workshop Phase E gate). Calm &
  focused direction: warm-paper light theme, Fraunces + Hanken Grotesk, single
  terracotta accent with muted slate/terracotta/sage status markers, dnd-kit
  for drag-and-drop (keyboard-operable, announced). Every section filled; no
  placeholders remain.
