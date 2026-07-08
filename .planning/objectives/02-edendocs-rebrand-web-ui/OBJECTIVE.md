---
work: feature
objective: 02-edendocs-rebrand-web-ui
requirements: [BRAND-01, BRAND-02, BRAND-03, BRAND-04, BRAND-05, BRAND-06]
depends_on_objectives: [01-from-source-build-pipeline]
---

# Objective 2: EdenDocs Rebrand of the Web UI

## Goal

Every user-facing surface of the editor and admin console — name, logo, colors,
favicon, About dialog, help links — shows EdenDocs branding, delivered through
config keys and an additive `eden-branding/` directory rather than scattered
source edits, with Collabora trademarks removed and MPL/legal attribution
preserved.

## Must-Haves (the 5 roadmap Success Criteria)

1. Opening the editor shows the EdenDocs product name and logo, driven by
   `coolwsd.xml`'s `brandProductName`/`brandProductURL`/`logoURL` plus the
   additive `eden-branding/` directory (`branding.css`, `branding.js`, logo
   SVGs, `toolbar-bg-logo`) wired via `--with-app-branding` — no upstream
   source files edited to achieve it.
2. The browser tab title and About dialog read "EdenDocs" (via the two isolated
   `<title>` patches in `cool.html.m4`/`admintemplate.html`, kept as their own
   commits, plus compiled `--with-app-name="EdenDocs"`/`--with-vendor="AO Cyber
   Systems"` fallbacks).
3. The admin console shows the same EdenDocs branding as the editor via the
   shared `%BRANDING_JS%` hook, with the favicon replaced and the color palette
   overridden via a branding-dir CSS file (upstream `color-palette.css` never
   edited in place).
4. No Collabora trademarks (names, logos, `help_url`, container `LABEL`
   metadata) remain on any user-facing surface, verified against a written
   trademark/MPL checklist confirming MPL headers, `COPYING`, and
   `THIRDPARTYLICENSES` attribution are still intact.
5. `INFOBAR_URL`/`FEEDBACK_URL`/`INFO_URL` are confirmed unset and
   `--with-support-public-key`/`--with-welcome-url` are confirmed never passed
   to the build (no phone-home, no silently-dropped branding keys).

## Requirement Traceability

| Requirement | Delivered by | Verified by |
|---|---|---|
| BRAND-01 (name/logo via config + eden-branding) | 02-03 (assets), 02-04 (wiring) | 02-04 smoke, 02-05 gate |
| BRAND-02 (titles + About) | 02-01 (title patches), 02-04 (app-name/vendor flags) | 02-05 gate |
| BRAND-03 (admin console via %BRANDING_JS%) | 02-03 (branding.js), 02-04 (wiring) | 02-04 smoke, 02-05 gate |
| BRAND-04 (favicon + palette) | 02-03 (css/favicon assets), 02-04 (overlays) | 02-04 smoke, 02-05 gate |
| BRAND-05 (trademarks out, MPL intact, checklist) | 02-02 (string patches), 02-04 (help_url) | 02-05 checklist + sweep gate |
| BRAND-06 (no phone-home, mechanical proof) | 02-04 (omit list + canary assertion) | 02-05 negative-flag gate |

## Locked Decisions (do not relitigate)

- `eden-main` tracks `upstream/main`; every upstream-file edit is an isolated,
  minimal, own-commit patch listed for the future Objective 5 divergence
  allowlist. Prefer config keys + additive `eden-branding/` files.
- Eden brand tokens: Eden UI gold ramp, primary `#D4A853` (50–950 ramp per
  PROJECT.md / 2-RESEARCH.md). Use darker shades (600/700/800) where
  gold-on-light contrast fails.
- Logo: the REAL official AO gold-gradient emblem downloaded from
  `https://aocyber.ai/images/ao-icon.svg` (gradient `#fce88d → #efb32c`),
  committed into `eden-branding/`, plus an "EdenDocs" wordmark. NEVER a text
  placeholder or recolored stand-in. Preserve the gold gradient.
- Product name "EdenDocs", vendor "AO Cyber Systems".
- Open-question resolutions (adopted from planning guidance):
  1. MPL source-availability/provenance link → `https://github.com/AO-Cyber-Systems/EdenDocs`.
  2. Collabora SDK/forum/community links → accepted checklist exception for v1
     (documented, not silent); home/docs-style links repoint to `https://aocyber.ai`.
  3. `help_url` → `--with-help-url="https://aocyber.ai"` for now (no dedicated
     EdenDocs docs site yet).
  4. Newly discovered hardcoded surfaces (`browser/src/control/backstage/Sidebar.tsx`
     "Collabora Office"; `browser/admin/adminSettings.html` Version-tab headers)
     → filed under BRAND-05, each its own isolated patch commit, flagged for the
     divergence allowlist.

## Hard Constraints (apply to every TRD)

- NEVER use or reference port 8080 — local web verification uses 8091; coolwsd
  native port is 9980.
- MPL-2.0: MPL headers, `COPYING`, `THIRDPARTYLICENSES` attribution remain
  intact — trademark removal never touches legal attribution.
- Internal identifiers/protocol strings/config-key names stay
  upstream-compatible (COOLWSD agent strings, `cool.html`/`coolwsd.xml`
  filenames, class names) — only user-facing surfaces change.
- The Objective 1 CI pipeline stays green; branding verification extends it,
  never forks it.

## TRDs

| TRD | Wave | Scope |
|---|---|---|
| 02-01-upstream-title-patches | 1 | Two isolated `<title>` patches (BRAND-02) |
| 02-02-upstream-trademark-patches | 1 | Backstage header + admin Version-tab patches (BRAND-05) |
| 02-03-eden-branding-assets | 1 | Additive `eden-branding/` dir: real AO emblem, CSS/JS, favicon, welcome (BRAND-01/03/04) |
| 02-04-build-wiring | 2 | Configure flags, coolwsd.xml brand keys, favicon/welcome overlays, smoke-test branding checks, CI green (BRAND-01/02/03/04/06) |
| 02-05-trademark-checklist-verification | 3 | Written trademark/MPL checklist + mechanical CI gate `verify-branding.sh` (BRAND-05/06) |

---
*Created: 2026-07-08 (auto-scaffold via bootstrapObjectiveMd)*
*Planned: 2026-07-08 — 5 TRDs, 3 waves*
