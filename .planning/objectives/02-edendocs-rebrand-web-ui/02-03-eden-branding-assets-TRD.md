---
objective: 02-edendocs-rebrand-web-ui
trd: "03"
type: standard
wave: 1
depends_on: []
files_modified:
  - eden-branding/README.md
  - eden-branding/branding.css
  - eden-branding/branding.js
  - eden-branding/favicon.ico
  - eden-branding/images/eden-emblem.svg
  - eden-branding/images/eden-logo.svg
  - eden-branding/images/collabora-office-white.svg
  - eden-branding/images/toolbar-bg-logo.svg
  - eden-branding/welcome/welcome.html
  - eden-branding/welcome/welcome.css
  - eden-branding/welcome/eden-emblem.svg
autonomous: true
requirements: [BRAND-01, BRAND-03, BRAND-04]

must_haves:
  truths:
    - "eden-branding/ is a fully additive directory — zero upstream files touched by this TRD"
    - "The committed logo assets embed the REAL AO gold-gradient emblem (gradient #fce88d→#efb32c) downloaded from aocyber.ai — not a text placeholder or recolored stand-in"
    - "branding.css overrides the primary palette to Eden gold with !important on every custom property (survives the runtime dark-theme CSS injection)"
    - "branding.js defines window.brandProductName = 'EdenDocs' (the ONLY possible producer of that global for the admin console)"
    - "The replacement welcome page and all new text assets contain zero Collabora trademarks"
  artifacts:
    - path: "eden-branding/images/eden-emblem.svg"
      provides: "Verbatim official AO emblem (source of truth, from https://aocyber.ai/images/ao-icon.svg)"
      contains: "svg"
    - path: "eden-branding/images/collabora-office-white.svg"
      provides: "Eden emblem under the upstream filename (intentional collision → overwrites the Backstage logo via the images/*.svg glob)"
      contains: "efb32c"
    - path: "eden-branding/images/toolbar-bg-logo.svg"
      provides: "Required branding hook file (Makefile cp renames it to images/toolbar-bg.svg; build FAILS if absent)"
      contains: "svg"
    - path: "eden-branding/images/eden-logo.svg"
      provides: "Emblem + 'EdenDocs' wordmark composition (logoURL target / general use)"
      contains: "EdenDocs"
    - path: "eden-branding/branding.css"
      provides: "Eden gold palette overrides with !important + MPL header"
      contains: "--color-primary: #D4A853 !important"
    - path: "eden-branding/branding.js"
      provides: "window.brandProductName global for editor + admin console + MPL header"
      contains: "window.brandProductName = 'EdenDocs'"
    - path: "eden-branding/favicon.ico"
      provides: "Multi-size ICO derived from the AO emblem (16/32/48)"
    - path: "eden-branding/welcome/welcome.html"
      provides: "Collabora-free replacement welcome page with MPL source-availability link"
      contains: "github.com/AO-Cyber-Systems/EdenDocs"
    - path: "eden-branding/README.md"
      provides: "Mechanism map + intentional-filename-collision warning"
      contains: "collabora-office-white.svg"
  key_links:
    - from: "eden-branding/images/collabora-office-white.svg"
      to: "browser/dist/images/collabora-office-white.svg"
      via: "browser/Makefile.am:1014-1050 images/*.svg same-filename overwrite (wired in TRD 02-04)"
      pattern: "efb32c"
    - from: "eden-branding/branding.js"
      to: "admin console header/title"
      via: "%BRANDING_JS% script tag executing before adminBody.html inline script (wsd/FileServer.cpp:2579-2582)"
      pattern: "window\\.brandProductName"
    - from: "eden-branding/branding.css"
      to: "editor color palette (light AND dark mode)"
      via: "%BRANDING_CSS% link + !important beating runtime-appended color-palette-dark.css"
      pattern: "!important"
---

<objective>
Create the self-contained, additive `eden-branding/` directory: the real AO
gold-gradient emblem and derived logo SVGs, the gold palette CSS overrides, the
mandatory `branding.js` product-name global, a derived favicon, a Collabora-free
replacement welcome page, and a README documenting the non-obvious mechanics.

Purpose: This directory is the payload for BRAND-01/03/04. TRD 02-04 wires it
into the build via `--with-app-branding` + overlay steps; nothing here touches
an upstream file.

Output: 11 new files under `eden-branding/`, all committed, all MPL-headed
where the format allows.
</objective>

<file_tree>
eden-branding/                          ← CREATE (entire directory is new)
├── README.md                           ← CREATE (mechanism map, collision warning)
├── branding.css                        ← CREATE (gold !important overrides, MPL header)
├── branding.js                         ← CREATE (window.brandProductName, MPL header)
├── favicon.ico                         ← CREATE (derived from emblem, 16/32/48)
├── images/
│   ├── eden-emblem.svg                 ← CREATE (verbatim aocyber.ai download)
│   ├── eden-logo.svg                   ← CREATE (emblem + EdenDocs wordmark)
│   ├── collabora-office-white.svg      ← CREATE (emblem under upstream filename — intentional)
│   └── toolbar-bg-logo.svg             ← CREATE (required hook; renamed to toolbar-bg.svg on copy)
└── welcome/
    ├── welcome.html                    ← CREATE (Collabora-free, self-contained)
    ├── welcome.css                     ← CREATE
    └── eden-emblem.svg                 ← CREATE (copy for the welcome page)
</file_tree>

<execution_context>
@~/.claude/devflow/workflows/execute-trd.md
@~/.claude/devflow/templates/summary.md
</execution_context>

<embedded_context>

<codebase_examples>
MPL header convention for new files (per README.FILENOTICES.md; match upstream
CSS/JS headers, add the AO copyright line):
```
/* -*- js -*- */
/*
 * Copyright the EdenDocs contributors / AO Cyber Systems.
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */
```

Upstream palette values being overridden (browser/css/color-palette.css —
NEVER edited in place; identical property names exist in color-palette-dark.css):
```css
--color-primary: #0b87e7;          /* Collabora blue */
--color-primary-dark: #0063b1;
--color-primary-darker: #004b86;
--color-primary-lighter: #83beec;
--color-primary-text: #fff;        /* used ON --color-primary-lighter bg */
--color-btn-primary-hover-bg: #e6f3fd;
--color-overlay: #1c5fa814;        /* alpha-blue overlay */
```

Backstage logo consumer (browser/css/backstage.css:56-58 — NOT edited; the
image file is replaced by filename collision):
```css
.backstage-header-title::before {
	background-image: url('images/collabora-office-white.svg');
}
```
The upstream SVG is a small white glyph, viewBox 0 0 16.23 22, rendered at
~btn-img-size with `background-size: contain` on the doc-type-colored header
(purple/blue/green per app). Any viewBox aspect works; the gold emblem reads
correctly on those saturated header colors (AO brand: gold works on dark AND
light).
</codebase_examples>

<anti_patterns>
- NEVER a text placeholder, recolored stand-in, or invented mark — the emblem
  is downloaded from `https://aocyber.ai/images/ao-icon.svg` and its gold
  gradient (#fce88d → #efb32c) is preserved verbatim (locked decision).
- Do NOT rename `collabora-office-white.svg` to something "more honest" — the
  filename collision IS the zero-edit overwrite mechanism (research Pitfall 5).
  The README must say this loudly.
- Do NOT copy any upstream `browser/welcome/` asset (slides, welcome.js) into
  eden-branding/welcome/ — the replacement is written from scratch,
  self-contained (html+css+svg only).
- Do NOT reuse Collabora's proprietary `online-branding` artwork (explicitly
  out of scope in REQUIREMENTS.md).
- Do NOT omit !important on any --color-* override (research Pitfall 2: the
  dark-theme script appendChild's color-palette-dark.css AFTER our link at
  runtime; only !important custom properties survive).
</anti_patterns>

<error_recovery>
- If `curl https://aocyber.ai/images/ao-icon.svg` fails: retry with
  `-L --retry 3`; as fallback fetch the site favicon SVG source noted in the AO
  brand guide (same file). Do NOT proceed with a placeholder — this TRD blocks
  until the real asset is obtained.
- If `rsvg-convert` mis-renders the gradient when rasterizing for the favicon,
  use `magick -background none -density 512 eden-emblem.svg -resize NxN out.png`
  instead (ImageMagick 7 with rsvg delegate is installed).
- If the downloaded SVG uses `<style>`/class-based gradient definitions that
  break when fragments are copied into composed SVGs, embed the emblem via
  nested `<svg>` (paste the whole file inside a group) rather than extracting
  paths.
</error_recovery>

</embedded_context>

<context>
@.planning/PROJECT.md
@.planning/objectives/02-edendocs-rebrand-web-ui/OBJECTIVE.md
@.planning/objectives/02-edendocs-rebrand-web-ui/2-RESEARCH.md
</context>

<research_context>
- `--with-app-branding` copy semantics (research §1, browser/Makefile.am:1014-1050):
  `branding*` glob → `browser/dist/` root (NOT a subdir); `images/*.svg` →
  `browser/dist/images/` with same-filename overwrite;
  `images/toolbar-bg-logo.svg` is cp'd + RENAMED to `images/toolbar-bg.svg`
  unconditionally — if the source file is missing the recipe FAILS, so it must
  exist even though nothing in our build references toolbar-bg.svg yet.
- `branding.js` is MANDATORY for BRAND-03: `window.brandProductName` is a bare
  global with NO other producer for the admin console (research §4). The
  `%BRANDING_JS%` script tag executes synchronously before adminBody.html's
  inline script by template order.
- Gold ramp mapping (research §5): primary #D4A853 (500), dark #A67A38 (700),
  darker #856131 (800), lighter #F4D5AA (200), btn-hover #FAECD5 (100),
  overlay #D4A85314 (alpha-hex, mirrors upstream's #1c5fa814 format).
  CONTRAST FLAG: --color-primary-text is the fg used on --color-primary-lighter
  backgrounds; upstream white fails on light gold → use gold-950 #3D2A14.
- --color-hyperlink stays upstream (standard link affordance, not a trademark;
  document the choice in the README).
- favicon + welcome/ are NOT consumed by --with-app-branding on Linux
  (research Pitfall 3) — TRD 02-04 overlays them from this directory via
  scripts/eden/build.sh.
</research_context>

<gotchas>
- HARD RULE: never use or reference port 8080 anywhere (local web checks: 8091;
  coolwsd native: 9980). No server is needed in this TRD.
- SVG `<text>` elements depend on viewer fonts. For the eden-logo.svg wordmark
  prefer converting "EdenDocs" to paths (e.g. render via a tool) OR use
  `<text>` with `font-family="Parabolica, Montserrat, system-ui, sans-serif"
  font-weight="600"` — acceptable since the SVG is used as a browser-rendered
  logo where system-ui fallback is fine. Wordmark fill: gold #D4A853.
- The ICO must contain real 16/32/48 px layers with transparency — not a
  single-resolution rename.
- All text-format files (css/js/html/README) carry the MPL header + AO
  copyright line; binary/ico and downloaded-verbatim SVG do not need one.
</gotchas>

<tasks>

<task type="auto">
  <name>Task 1: Download the real AO emblem and derive the logo SVGs + favicon</name>
  <files>eden-branding/images/eden-emblem.svg, eden-branding/images/eden-logo.svg, eden-branding/images/collabora-office-white.svg, eden-branding/images/toolbar-bg-logo.svg, eden-branding/favicon.ico, eden-branding/welcome/eden-emblem.svg</files>
  <action>
1. `mkdir -p eden-branding/images eden-branding/welcome`
2. Download the official emblem:
   `curl -fsSL --retry 3 https://aocyber.ai/images/ao-icon.svg -o eden-branding/images/eden-emblem.svg`
   Assert it is real: file is valid SVG AND contains both gradient stops
   (`fce88d` and `efb32c`, case-insensitive). If either assertion fails, STOP
   (see error_recovery) — never substitute a placeholder.
3. `eden-branding/images/collabora-office-white.svg` — the emblem under the
   upstream filename (intentional collision, research Pitfall 5). Content: the
   emblem verbatim (gradient preserved; do NOT recolor to white — gold reads on
   the doc-type-colored backstage header). Simplest correct implementation:
   `cp eden-branding/images/eden-emblem.svg eden-branding/images/collabora-office-white.svg`.
4. `eden-branding/images/toolbar-bg-logo.svg` — required hook file (Makefile cp
   fails without it). Content: the emblem verbatim (copy again). Nothing in the
   default build references the renamed `toolbar-bg.svg` output, so verbatim
   emblem is safe.
5. `eden-branding/images/eden-logo.svg` — composed horizontal lockup: the
   emblem (nested `<svg>` embed of the downloaded file, gradient intact) +
   "EdenDocs" wordmark to its right (see gotchas for font/paths guidance;
   fill #D4A853, weight 600). ViewBox sized to the lockup; transparent
   background.
6. `eden-branding/welcome/eden-emblem.svg` — copy of the emblem for the welcome
   page (welcome/ is fully self-contained after TRD 02-04's full-replacement
   overlay).
7. `eden-branding/favicon.ico` — derive from the emblem at 16/32/48 with
   transparency:
   `for s in 16 32 48; do rsvg-convert -w $s -h $s eden-branding/images/eden-emblem.svg -o /tmp/eden-fav-$s.png; done && magick /tmp/eden-fav-16.png /tmp/eden-fav-32.png /tmp/eden-fav-48.png eden-branding/favicon.ico`
Commit all six files together:
`feat(02-03): real AO emblem + derived EdenDocs logo assets (additive)`.
  </action>
  <verify>
grep -qi 'efb32c' eden-branding/images/eden-emblem.svg && grep -qi 'fce88d' eden-branding/images/eden-emblem.svg && grep -qi 'efb32c' eden-branding/images/collabora-office-white.svg && grep -qi 'svg' eden-branding/images/toolbar-bg-logo.svg && grep -q 'EdenDocs' eden-branding/images/eden-logo.svg && grep -qi 'efb32c' eden-branding/welcome/eden-emblem.svg && file eden-branding/favicon.ico | grep -qi 'ico' && magick identify eden-branding/favicon.ico | grep -c 'ICO' | grep -q '3'
  </verify>
  <done>All 5 SVGs + favicon.ico exist; every emblem-bearing file carries the real gradient stops; eden-logo.svg carries the wordmark; the ICO has 3 size layers.</done>
  <recovery>If magick identify reports fewer than 3 layers, rebuild the ICO from the three PNGs explicitly; if rsvg-convert fails on the SVG, switch to `magick -background none -density 512 ... -resize` per error_recovery.</recovery>
</task>

<task type="auto">
  <name>Task 2: Author branding.css (gold !important overrides) and branding.js (mandatory globals)</name>
  <files>eden-branding/branding.css, eden-branding/branding.js</files>
  <action>
Create `eden-branding/branding.css` (MPL header per embedded_context, then):
```css
:root {
  /* Eden UI gold ramp (PROJECT.md locked decision). !important is REQUIRED:
   * the ?darkTheme=true script appendChild()s color-palette-dark.css AFTER
   * this file in <head>, silently reverting non-!important overrides to
   * Collabora blue (2-RESEARCH.md Pitfall 2). */
  --color-primary: #D4A853 !important;          /* gold-500 */
  --color-primary-dark: #A67A38 !important;     /* gold-700 */
  --color-primary-darker: #856131 !important;   /* gold-800 */
  --color-primary-lighter: #F4D5AA !important;  /* gold-200 (background use) */
  /* Upstream white fails WCAG on the light-gold -lighter background: */
  --color-primary-text: #3D2A14 !important;     /* gold-950 */
  --color-btn-primary-hover-bg: #FAECD5 !important; /* gold-100 */
  --color-overlay: #D4A85314 !important;        /* gold @ 8% alpha */
}
```
(--color-hyperlink intentionally NOT overridden — standard link affordance,
not a trademark; record in README/SUMMARY.)

Create `eden-branding/branding.js` (MPL header, then):
```js
/* MANDATORY for BRAND-03: window.brandProductName is a bare global with NO
 * other producer for the admin console — coolwsd.xml's brandProductName key
 * only reaches the editor via the hidden-input path (2-RESEARCH.md §4).
 * This script executes (via the %BRANDING_JS% tag) before adminBody.html's
 * inline script by template order. */
window.brandProductName = 'EdenDocs';
window.brandProductURL = 'https://aocyber.ai';
```
Commit: `feat(02-03): eden-branding branding.css (gold palette) + branding.js (product-name global)`.
  </action>
  <verify>
grep -qF -- '--color-primary: #D4A853 !important' eden-branding/branding.css && [ "$(grep -c '!important' eden-branding/branding.css)" -ge 7 ] && grep -qF -- '--color-primary-text: #3D2A14 !important' eden-branding/branding.css && grep -qF "window.brandProductName = 'EdenDocs'" eden-branding/branding.js && grep -q 'Mozilla Public' eden-branding/branding.css && grep -q 'Mozilla Public' eden-branding/branding.js && node --check eden-branding/branding.js
  </verify>
  <done>branding.css carries all 7 gold overrides with !important + MPL header; branding.js parses cleanly and sets both globals with MPL header.</done>
  <recovery>If node --check fails, fix syntax; keep branding.js to plain ES5 global assignments (it runs unbundled in old-style script context in BOTH editor and admin console).</recovery>
</task>

<task type="auto">
  <name>Task 3: Replacement welcome page + eden-branding/README.md</name>
  <files>eden-branding/welcome/welcome.html, eden-branding/welcome/welcome.css, eden-branding/README.md</files>
  <action>
Create `eden-branding/welcome/welcome.html` — a minimal, fully self-contained
single page (MPL header comment): dark background `#0a0a0a`, centered
`eden-emblem.svg`, heading "Welcome to EdenDocs", one sentence ("Collaborative
documents on infrastructure you control — part of the Eden Productivity Suite
by AO Cyber Systems."), and a footer line with the MPL source-availability
link: `Source code: https://github.com/AO-Cyber-Systems/EdenDocs` (open-question
resolution 1). `<title>Welcome to EdenDocs</title>`. References ONLY
`welcome.css` and `eden-emblem.svg` (no JS, no slide images). Zero Collabora
strings.

Create `eden-branding/welcome/welcome.css` (MPL header): dark theme, gold
accent #D4A853 for the heading rule/link color, system-ui font stack.

Create `eden-branding/README.md` documenting (concise, one section each):
- What each file is and which mechanism consumes it (`branding*` glob →
  dist root; `images/*.svg` glob → dist/images same-filename overwrite;
  `toolbar-bg-logo.svg` → renamed `toolbar-bg.svg` on copy; favicon + welcome/
  NOT covered by --with-app-branding on Linux → overlaid by
  scripts/eden/build.sh, see TRD 02-04).
- LOUD warning: `images/collabora-office-white.svg` keeps the upstream
  filename ON PURPOSE — it is the zero-edit overwrite target referenced by
  backstage.css; renaming it silently restores the Collabora logo.
- Emblem provenance: downloaded from https://aocyber.ai/images/ao-icon.svg
  (official AO emblem, gradient #fce88d→#efb32c preserved; never recolor).
- Brand tokens table (the 7 overridden properties + values + why
  --color-primary-text flipped dark; --color-hyperlink intentionally left
  upstream).
- Port rule reminder: never 8080; local web checks 8091; coolwsd 9980.
Commit: `feat(02-03): EdenDocs welcome page + eden-branding README (additive)`.
  </action>
  <verify>
! grep -rqi 'collabora' eden-branding/welcome/welcome.html eden-branding/welcome/welcome.css && grep -q 'github.com/AO-Cyber-Systems/EdenDocs' eden-branding/welcome/welcome.html && grep -q 'Welcome to EdenDocs' eden-branding/welcome/welcome.html && ! grep -q 'welcome.js' eden-branding/welcome/welcome.html && grep -q 'collabora-office-white.svg' eden-branding/README.md && grep -q 'toolbar-bg' eden-branding/README.md && grep -qi 'never.*8080\|8080.*never' eden-branding/README.md
  </verify>
  <done>Welcome page is Collabora-free, self-contained, carries the source-availability link; README documents every mechanism including the filename-collision warning and the port rule.</done>
  <recovery>None destructive — re-edit and re-verify. If the 8080-prohibition grep fails on phrasing, reword to contain "never use port 8080".</recovery>
</task>

</tasks>

<validation_gates>
<lint>node --check eden-branding/branding.js</lint>
<test>! grep -rqi 'collabora' eden-branding/branding.css eden-branding/branding.js eden-branding/welcome/ eden-branding/README.md || { echo 'trademark leak in eden-branding text files (README references to the filename are allowed only in README.md)'; false; }</test>
<build>true # dist integration proof runs in CI via TRD 02-04</build>
</validation_gates>

<verification>
- All 11 files exist with the contents asserted per-task.
- Every emblem-bearing SVG contains the real gradient stops (fce88d + efb32c).
- `git status` shows no upstream file modified by this TRD (purely additive).
- NOTE on the test gate: `eden-branding/README.md` legitimately contains the
  string "collabora-office-white.svg" (it documents the collision). If the
  blanket grep trips only on README.md's filename references, that is
  acceptable — narrow the gate to exclude README.md and record it in the
  SUMMARY.
</verification>

<success_criteria>
`eden-branding/` exists as a committed, additive, self-documented directory
containing the real AO emblem (gradient intact), gold !important palette
overrides, the mandatory brandProductName global, a 3-layer favicon, and a
trademark-free welcome page — ready for TRD 02-04 to wire into the build.
</success_criteria>

<output>
After completion, create `.planning/objectives/02-edendocs-rebrand-web-ui/02-03-SUMMARY.md`.
</output>
