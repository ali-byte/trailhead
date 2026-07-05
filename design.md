# [Project Name] — Design System
#
# INSTRUCTIONS FOR USE:
# 1. Copy this file to your project root as design.md during Phase E
# 2. Fill in every section before any Phase F Dispatch session builds
#    frontend code — incomplete sections will cause Dispatch to invent
#    values, defeating the purpose of this file
# 3. Every Dispatch session that touches UI reads this file first and
#    builds exactly to the system defined here
# 4. When any design decision changes, update the relevant section AND
#    add a changelog entry at the bottom — never edit silently
# 5. design.md is version-controlled — treat changes to it like code
#    changes: intentional, documented, reviewed

---

## Project Context

Product type: [webapp | dashboard | marketing site | internal tool |
               mobile web | other — choose one]
Primary audience: [who uses this — be specific, e.g. "DBAs at mid-size
                  fintech companies" not just "developers"]
Tone: [choose one:
       brutally minimal | editorial/magazine | luxury/refined |
       playful/toy-like | industrial/utilitarian | brutalist/raw |
       organic/natural | retro-futuristic | maximalist |
       art deco/geometric | soft/pastel | other: describe]
Personality in three words: [e.g. "precise, warm, modern"]
What this product is NOT: [describe the aesthetic to explicitly avoid,
                           e.g. "not corporate SaaS blue, not startup-generic"]

---

## Typography

Display font: [font name — used for headlines and hero text]
  Load from: [Google Fonts URL or self-hosted path]
Body font: [font name — used for body text, labels, UI copy]
  Load from: [Google Fonts URL or self-hosted path]
Monospace font: [font name — used for code, metrics, data]
  Load from: [Google Fonts URL or self-hosted path, or "system-ui monospace"]

Banned in this project:
  [list any fonts explicitly never to use, e.g.:]
  - Inter (overused AI default)
  - Space Grotesk (overused AI default)
  - Geist (overused AI default)
  - [add any others]

Type scale:
  xs:  [size]px — [usage, e.g. "captions, metadata"]
  sm:  [size]px — [usage, e.g. "labels, secondary text"]
  base:[size]px — [usage, e.g. "body text"]
  lg:  [size]px — [usage, e.g. "lead text, card titles"]
  xl:  [size]px — [usage, e.g. "section headings"]
  2xl: [size]px — [usage, e.g. "page headings"]
  3xl: [size]px — [usage, e.g. "display, hero text"]

Weight usage:
  Regular (400): [what uses it]
  Medium (500):  [what uses it, or "not used"]
  Semibold (600):[what uses it]
  Bold (700):    [what uses it]

---

## Color Palette

[Define all colors as CSS custom properties. Every color used in the
project must be defined here — no magic hex values in component code.]

Background colors:
  --color-bg:          [hex] — [usage, e.g. "page background"]
  --color-surface:     [hex] — [usage, e.g. "card backgrounds"]
  --color-surface-2:   [hex] — [usage, e.g. "nested card backgrounds"]
  --color-raised:      [hex] — [usage, e.g. "dropdowns, tooltips"]

Text colors:
  --color-text:        [hex] — [usage, e.g. "primary body text"]
  --color-text-dim:    [hex] — [usage, e.g. "secondary/muted text"]
  --color-text-bright: [hex] — [usage, e.g. "headings, emphasis"]

Brand colors:
  --color-primary:     [hex] — [usage]
  --color-primary-dim: [hex] — [usage, e.g. "primary hover states"]
  --color-accent:      [hex] — [usage, e.g. "highlights, CTAs"]

Semantic colors:
  --color-border:      [hex] — [usage]
  --color-border-hi:   [hex] — [usage, e.g. "focused/active borders"]
  --color-success:     [hex] — [usage]
  --color-warning:     [hex] — [usage]
  --color-error:       [hex] — [usage]

Color philosophy: [describe the overall approach in 1-2 sentences,
e.g. "Dark background with a single warm gold accent. High contrast
text. No gradient backgrounds. Accents used sparingly."]

---

## Spacing & Shape

Base unit: [4px | 8px — all spacing is multiples of this]

Border radius:
  sm:  [size]px — [usage, e.g. "badges, tags"]
  md:  [size]px — [usage, e.g. "buttons, inputs"]
  lg:  [size]px — [usage, e.g. "cards, panels"]
  xl:  [size]px — [usage, e.g. "modals, large containers"]
  full: 9999px  — [usage, e.g. "pills, avatars"]

Shadow philosophy: [choose one:
  "no shadows — borders and background changes convey elevation" |
  "subtle only — low-opacity shadows for essential elevation" |
  "expressive — prominent shadows on key interactive elements" |
  other: describe]

---

## Component Conventions

Buttons:
  Primary: [describe style — background, text, border, hover state]
  Secondary: [describe style]
  Destructive: [describe style — must be visually distinct from primary]
  Disabled: [describe style — must meet contrast requirements]
  Size: [describe padding, font size, minimum touch target]

Inputs:
  Style: [describe border, background, focus ring]
  Label position: [above | inline | floating]
  Placeholder: [describe usage — note: placeholder is not a label]
  Validation: [inline below field | tooltip | other]
  Error state: [describe visual treatment]

Cards:
  Style: [describe border, background, padding, radius]
  Hover state: [describe if interactive, or "static — no hover"]
  Shadow: [describe or "none"]

Navigation:
  Pattern: [top nav | sidebar | tabs | breadcrumb | combination]
  Active state: [describe visual treatment]
  Mobile behaviour: [describe how navigation adapts]

Loading states:
  Approach: [skeleton screens | spinner | progress bar | combination]
  Skeleton style: [describe animation — pulse, shimmer, or none]

Error states:
  Inline errors: [describe field-level error treatment]
  Toast/notification: [describe global error treatment]
  Page-level errors: [describe full-page error treatment]
  Message format: [describe — error messages must say what happened
                  AND what the user can do, not just "error occurred"]

Empty states:
  Style: [describe visual treatment]
  Content: [describe — empty states must explain what goes here and
            how to add content, not just show a blank area]

Destructive actions:
  Confirmation: [describe — all destructive actions require explicit
                 confirmation before executing]

---

## Layout Rules

Grid: [describe — e.g. "12-column, 8px gutters" or "CSS grid with
       named areas" or "flexbox-based, no fixed column count"]
Max content width: [e.g. 1200px | 860px | full-width]
Page padding:
  Mobile:  [e.g. 16px]
  Tablet:  [e.g. 24px]
  Desktop: [e.g. 48px]

Asymmetry policy: [choose one:
  "prefer asymmetric — deliberate imbalance creates interest" |
  "symmetric — balanced layouts with deliberate exceptions" |
  "content-driven — layout follows content structure"]

Anti-patterns banned in this project:
  [list layout patterns explicitly not to use, e.g.:]
  - Three-column icon-feature grid (generic marketing layout)
  - Full-width hero with centered headline and soft gradient background
  - Card grid with identical card sizes and equal gutters
  [add any others specific to this project's tone]

---

## Motion

Philosophy: [choose one:
  "minimal — animation only where it communicates state change" |
  "purposeful — meaningful transitions and micro-interactions" |
  "expressive — rich motion as part of the product personality"]

Page load: [describe entry animation or "none"]
Route transitions: [describe or "none"]
Component mount: [e.g. "fade-in, 150ms ease-out" or "none"]
Interactive feedback: [describe hover and click animations]
Scroll: [describe scroll-triggered effects or "none"]

Reduced-motion rule (non-negotiable):
  All animations must respect prefers-reduced-motion.
  When prefers-reduced-motion: reduce is active:
    - Remove or minimize all transitions and animations
    - Preserve functional state changes (e.g. dropdown open/close)
    - Never remove information — only remove decorative motion

---

## Do Not Use

Fonts:
  [list all banned fonts for this project]

Colors:
  [list any specific colors to avoid, e.g. "no pure black (#000000)
  backgrounds", "no unsaturated grays for interactive elements"]

Design patterns:
  [list visual patterns explicitly banned, e.g.:]
  - Gradient overlays on text (legibility)
  - Stock photo style hero images
  - Decorative divider lines between every section
  - Modal on top of modal
  [add any others]

Layout patterns:
  [list layout patterns not to use — see Layout Rules anti-patterns above,
  add any additional ones specific to this project]

Copy patterns:
  [list any UI copy patterns to avoid, e.g.:]
  - Vague error messages ("Something went wrong")
  - Placeholder text used as labels
  - Jargon not defined in GLOSSARY.md

---

## Reference & Inspiration

Visual references:
  [List 2-4 URLs or describe products/sites whose aesthetic is similar
  to what this project should feel like. Specific is better than vague.]

Anti-references:
  [List 1-2 products/sites whose aesthetic this project should NOT
  resemble, and why. This helps Dispatch understand what to avoid.]

---

## Accessibility Baseline

(Non-negotiable — these apply to all projects regardless of tone)

Contrast:
  Text on background: minimum 4.5:1 (WCAG AA)
  Large text on background: minimum 3:1
  Interactive elements: minimum 3:1

Focus states:
  All interactive elements must have a visible focus ring.
  [Describe the specific focus ring style for this project, e.g.
  "2px solid --color-accent, 2px offset"]
  Never use outline: none without a custom focus indicator.

Semantic HTML:
  Use correct heading hierarchy (h1 → h2 → h3, no skipping)
  Use landmark regions (main, nav, header, footer)
  Use button for actions, a for navigation
  Use label elements for all form inputs

Motion:
  prefers-reduced-motion respected on all animations (see Motion section)

Touch targets:
  Minimum 44x44px for all interactive elements on mobile

Screen readers:
  Icon-only buttons must have aria-label or visually hidden text
  Images must have meaningful alt text or alt="" if decorative
  Form errors must be announced (aria-live or aria-describedby)

---

## Changelog

[Update this section whenever any design decision changes.
Format: - [YYYY-MM-DD]: description of what changed and why]

- [YYYY-MM-DD]: Initial design system created
