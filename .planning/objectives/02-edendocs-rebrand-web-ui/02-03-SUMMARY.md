---
objective: 02-edendocs-rebrand-web-ui
trd: "03"
subsystem: ui
tags: [branding, svg, css-custom-properties, favicon, mpl, ao-emblem, welcome-page]

# Dependency graph
requires:
  - objective: 02-edendocs-rebrand-web-ui
    provides: "Upstream trademark strings already patched (02-01 titles, 02-02 backstage/admin headers); eden-branding/ is the additive counterpart"
provides:
  - "eden-branding/ — additive brand overlay directory (11 files), zero upstream edits"
  - "Real AO gold-gradient emblem (verbatim from aocyber.ai) + derived logo/favicon assets"
  - "Gold !important palette overrides (branding.css) surviving the dark-theme runtime injection"
  - "window.brandProductName = 'EdenDocs' global (branding.js) — sole producer for the admin console"
  - "Collabora-free self-contained welcome page (html+css+svg)"
  - "README documenting the images/*.svg filename-collision overwrite mechanism"
affects: [02-04-build-wiring, 02-05-trademark-checklist-verification]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Additive-only branding overlay: no upstream file touched; assets consumed by build-time copy/overlay"
    - "Intentional filename collision (collabora-office-white.svg) as a zero-edit image overwrite"
    - "!important on every --color-* custom property to survive runtime dark-theme CSS appendChild"
    - "Emblem embedded as nested <svg> to preserve class-based gradient defs (no path extraction)"

key-files:
  created:
    - eden-branding/images/eden-emblem.svg
    - eden-branding/images/eden-logo.svg
    - eden-branding/images/collabora-office-white.svg
    - eden-branding/images/toolbar-bg-logo.svg
    - eden-branding/favicon.ico
    - eden-branding/branding.css
    - eden-branding/branding.js
    - eden-branding/welcome/welcome.html
    - eden-branding/welcome/welcome.css
    - eden-branding/welcome/eden-emblem.svg
    - eden-branding/README.md
  modified: []

key-decisions:
  - "Emblem downloaded verbatim from https://aocyber.ai/images/ao-icon.svg (76398 bytes, gradient #fce88d→#efb32c intact); never recolored or placeholdered"
  - "eden-logo.svg embeds the emblem as a nested <svg> (class-based gradients preserved) + 'EdenDocs' wordmark in gold #D4A853"
  - "--color-primary-text flipped to gold-950 #3D2A14 (upstream white fails WCAG on the light-gold -lighter background)"
  - "--color-hyperlink intentionally left at upstream value (standard link affordance, not a trademark surface)"
  - "Reworded welcome lede: 'Collaborative' → 'Real-time, multi-author' (the English word contains the case-insensitive substring 'collabora', tripping the zero-Collabora gate)"
  - "Reworded a branding.css comment to drop 'Collabora blue' → 'the upstream blue #0b87e7' (branding.css ships to dist via the branding* glob, so it must not carry the trademark even in a comment)"

patterns-established:
  - "Filename-collision overwrite: ship images/collabora-office-white.svg with the upstream name so the images/*.svg copy glob overwrites the Backstage glyph without editing backstage.css"
  - "Mandatory branding.js product-name global as the sole admin-console producer of window.brandProductName"

requirements-completed: []  # BRAND-01/03/04 targeted but NOT marked complete — they close after 02-04 build wiring

# Verification evidence
verification:
  gates_defined: 3
  gates_passed: 3          # lint + build pass; test passes on the narrowed (non-README) gate per TRD verification note
  auto_fix_cycles: 1       # one trademark-substring fix cycle (welcome lede + branding.css comment)
  tdd_evidence: false
  test_pairing: false

# Metrics
duration: 6min
completed: 2026-07-08
---

# Objective 02 TRD 03: Eden Branding Assets Summary

**Additive `eden-branding/` overlay (11 files): the real AO gold-gradient emblem verbatim from aocyber.ai, a derived EdenDocs logo lockup + 3-layer favicon, gold `!important` palette overrides, the mandatory `window.brandProductName` global, and a Collabora-free self-contained welcome page — zero upstream files touched.**

## Performance

- **Duration:** 6 min
- **Started:** 2026-07-08T15:57:06Z
- **Completed:** 2026-07-08T16:03:51Z
- **Tasks:** 3 (+1 trademark auto-fix cycle)
- **Files created:** 11 (3117 insertions, 0 deletions — purely additive)

## Accomplishments
- Downloaded the **real** AO emblem verbatim (76398 bytes, gradient `#fce88d→#efb32c` asserted present) and derived four SVGs + a 3-layer favicon from it — no placeholder, no recolor.
- `branding.css` overrides all 7 primary-palette custom properties to the Eden gold ramp, every one carrying `!important` so it survives the `?darkTheme=true` runtime `appendChild()` of `color-palette-dark.css`.
- `branding.js` defines `window.brandProductName = 'EdenDocs'` (+ `brandProductURL`) — the sole producer of that global for the admin console.
- Trademark-free, JS-free, slide-free welcome page carrying the MPL source-availability link `https://github.com/AO-Cyber-Systems/EdenDocs`.
- `README.md` documents every consumption mechanism, the LOUD filename-collision warning, emblem provenance, the 7-token table, and the port-8080 rule.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: Emblem download + derived logo/favicon | `grep -qi efb32c …eden-emblem.svg && … && magick identify …favicon.ico \| grep -c ICO \| grep -q 3` | 0 | PASS |
| 2: branding.css (gold !important) + branding.js (globals) | `grep -qF -- '--color-primary: #D4A853 !important' … && [ "$(grep -c '!important' …css)" -ge 7 ] && grep -qF "window.brandProductName = 'EdenDocs'" …js && node --check …js` | 0 | PASS |
| 3: Welcome page + README | `! grep -rqi collabora …welcome/{html,css} && grep -q github.com/AO-Cyber-Systems/EdenDocs …html && grep -q collabora-office-white.svg …README.md && grep -qi 'never.*8080' …README.md` | 0 | PASS |

_Task 3 first run returned exit 1 (English word "Collaborative" in the lede matched the `-i 'collabora'` gate); reworded to "Real-time, multi-author" and re-verified exit 0._

## Task Commits

Each task was committed atomically:

1. **Task 1: real AO emblem + derived EdenDocs logo assets** — `938f8bf3714` (feat)
2. **Task 2: branding.css (gold palette) + branding.js (product-name global)** — `160335432f3` (feat)
3. **Task 3: EdenDocs welcome page + eden-branding README** — `25ce2fbc786` (feat)
4. **Auto-fix: drop Collabora trademark from shipped branding.css comment** — `cd6a1d2da9f` (fix)

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| lint | `node --check eden-branding/branding.js` | 0 | PASS |
| test (blanket) | `! grep -rqi collabora branding.css branding.js welcome/ README.md` | 1 | EXPECTED-FAIL (README-only) |
| test (narrowed) | `! grep -rqi collabora branding.css branding.js welcome/` | 0 | PASS |
| build | `true` (dist integration proof runs in CI via TRD 02-04) | 0 | PASS |

**Test-gate note (documented per TRD `<verification>`):** The blanket test gate includes `eden-branding/README.md`, which legitimately contains `collabora-office-white.svg` (the required filename-collision documentation) plus prose describing the collision. Narrowing the gate to exclude `README.md` passes (exit 0). After the auto-fix, `README.md` is the **only** Collabora-bearing text file — `branding.css`, `branding.js`, and everything under `welcome/` are clean. This is the accepted resolution the TRD anticipated.

## Post-TRD Verification

- **Auto-fix cycles used:** 1
- **Must-haves verified:** 9/9 artifacts + 5/5 truths
- **Gate failures:** None (blanket test-gate trip is the documented README-only exception; narrowed gate passes)
- **Additivity:** `git diff --name-only 61143393c8b..HEAD` → all 11 paths under `eden-branding/`; 3117 insertions, 0 deletions. Zero upstream files modified.
- **Emblem authenticity:** downloaded from `https://aocyber.ai/images/ao-icon.svg`, 76398 bytes, both gradient stops (`fce88d`, `efb32c`) asserted present in every emblem-bearing SVG.
- **Favicon:** `magick identify` reports 3 layers (16/32/48), `file` reports "MS Windows icon resource - 3 icons".
- **Branch/worktree:** still on `eden-main`; no worktree created or entered by this TRD (a pre-existing unrelated agent worktree exists on disk, untouched); no push performed.

## Files Created/Modified
- `eden-branding/images/eden-emblem.svg` — official AO emblem, verbatim from aocyber.ai (gradient intact)
- `eden-branding/images/collabora-office-white.svg` — emblem under the upstream filename (intentional collision → overwrites the Backstage logo)
- `eden-branding/images/toolbar-bg-logo.svg` — required hook file (Makefile renames it to toolbar-bg.svg; build FAILS if absent)
- `eden-branding/images/eden-logo.svg` — emblem (nested `<svg>`) + "EdenDocs" gold wordmark lockup
- `eden-branding/favicon.ico` — 3-layer (16/32/48) ICO derived from the emblem with transparency
- `eden-branding/branding.css` — 7 gold-ramp `!important` palette overrides + MPL header
- `eden-branding/branding.js` — `window.brandProductName`/`brandProductURL` globals + MPL header
- `eden-branding/welcome/welcome.html` — Collabora-free self-contained welcome page (MPL header, source link)
- `eden-branding/welcome/welcome.css` — dark theme, gold accent, system-ui stack + MPL header
- `eden-branding/welcome/eden-emblem.svg` — emblem copy for the welcome page
- `eden-branding/README.md` — mechanism map, filename-collision warning, token table, port rule (MPL header)

## Decisions Made
- **Emblem embedding:** `eden-logo.svg` nests the whole downloaded emblem as a positioned `<svg>` rather than extracting paths — the emblem uses `<style>`/class-based gradient defs that break under fragment extraction (per TRD error_recovery).
- **`--color-primary-text` flip:** set to gold-950 `#3D2A14` because upstream white is the foreground used on the light-gold `--color-primary-lighter` background and fails WCAG there.
- **`--color-hyperlink` left upstream:** standard link affordance, not a trademark surface (documented in README + branding.css).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Trademark-gate substring collision on the English word "Collaborative"**
- **Found during:** Task 3 (welcome page) — first `<verify>` run returned exit 1.
- **Issue:** The TRD's suggested welcome sentence began "Collaborative documents…". The English word "Collaborative" contains the case-insensitive substring "collabora", so the authoritative `! grep -rqi 'collabora'` verify/validation gate flagged it as a trademark leak.
- **Fix:** Reworded the lede to "Real-time, multi-author documents on infrastructure you control — …", preserving meaning and passing the mechanical zero-Collabora gate.
- **Files modified:** `eden-branding/welcome/welcome.html`
- **Verification:** Task 3 `<verify>` re-run exit 0.
- **Committed in:** `25ce2fbc786` (Task 3 commit — fix applied before commit).

**2. [Rule 1 - Bug] "Collabora blue" trademark string in shipped branding.css comment**
- **Found during:** Post-task validation-gate run (narrowed test gate).
- **Issue:** A `branding.css` explanatory comment read "…reverting … to Collabora blue". `branding.css` is copied into `browser/dist/` by the `branding*` glob and ships, so the trademark string would appear in a distributed artifact — failing the non-README trademark gate and the must-have "all new text assets contain zero Collabora trademarks".
- **Fix:** Reworded to "…reverting … back to the upstream blue #0b87e7". README (documentation, gate-excluded) still explains the mechanism.
- **Files modified:** `eden-branding/branding.css`
- **Verification:** Narrowed test gate exit 0; README confirmed as the only remaining Collabora-bearing text file.
- **Committed in:** `cd6a1d2da9f` (fix commit).

---

**Total deviations:** 2 auto-fixed (2 Rule-1 trademark-substring corrections)
**Impact on plan:** Both fixes strengthen the zero-Collabora requirement the TRD explicitly enforces. No scope creep; asset set and mechanics unchanged.

## Issues Encountered
- `file://` navigation is blocked in the browser tool and the TRD needs no server, so the welcome page was validated via its passing mechanical gate plus a confirmed standalone render of the emblem SVG rather than a live browser screenshot. The page is trivial self-contained flexbox.

## Requirements Note
BRAND-01, BRAND-03, BRAND-04 are **targeted** by this TRD but intentionally **not** marked complete — they close after TRD 02-04 wires these assets into the build (`--with-app-branding` flags, `coolwsd.xml` brand keys, favicon/welcome overlays) and CI proves the dist integration. Per the executor instructions, `requirements mark-complete` was NOT run here.

## Next Objective Readiness
- `eden-branding/` is committed, additive, and self-documented — ready for TRD 02-04 to wire into the build.
- 02-04 must: reference `branding.css`/`branding.js` via `--with-app-branding`; confirm the `images/*.svg` overwrite lands `collabora-office-white.svg`; ensure `toolbar-bg-logo.svg` is present before the Makefile `cp`; overlay `favicon.ico` + `welcome/` via `scripts/eden/build.sh` (not covered by `--with-app-branding` on Linux).

---
*Objective: 02-edendocs-rebrand-web-ui*
*Completed: 2026-07-08*

## Self-Check: PASSED

- All 11 eden-branding/ files + 02-03-SUMMARY.md exist on disk.
- All 4 commits present in git history: `938f8bf3714`, `160335432f3`, `25ce2fbc786`, `cd6a1d2da9f`.
