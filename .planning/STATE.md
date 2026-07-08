# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-07)

**Core value:** Eden customers can collaboratively edit documents in the browser on infrastructure they control — privacy-first, with no third-party document cloud.
**Current focus:** Objective 2 — EdenDocs Rebrand of the Web UI

## Current Position

Objective: 2 of 5 (EdenDocs Rebrand of the Web UI) — Objective 1 (From-Source Build Pipeline) complete, all 3 jobs done (01-01, 01-02, 01-03)
Job: 02-03 complete (additive eden-branding/ assets); next 02-04 (build wiring — wave 2)
Status: Executing — wave 1 of 3 done (02-01 + 02-02 + 02-03 all complete); next wave 2 (TRD 02-04)
Last activity: 2026-07-08 — TRD 02-03 complete: created the fully additive `eden-branding/` directory (11 new files, 3117 insertions / 0 deletions, zero upstream files touched, all on eden-main). The **real** AO gold-gradient emblem was downloaded verbatim from https://aocyber.ai/images/ao-icon.svg (76398 bytes; gradient stops `#fce88d` + `#efb32c` asserted present) and derived into `images/{eden-emblem,eden-logo,collabora-office-white,toolbar-bg-logo}.svg`, `welcome/eden-emblem.svg`, and a 3-layer (16/32/48) `favicon.ico`. `eden-logo.svg` nests the emblem as a `<svg>` (class-based gradients preserved) plus an "EdenDocs" gold wordmark. `branding.css` overrides all 7 primary-palette custom properties to the Eden gold ramp (#D4A853 gold-500 … `--color-primary-text` flipped to gold-950 #3D2A14 for WCAG), each carrying `!important` to survive the dark-theme runtime CSS injection; `--color-hyperlink` intentionally left upstream. `branding.js` sets `window.brandProductName = 'EdenDocs'` (sole admin-console producer) + `brandProductURL`. A Collabora-free self-contained welcome page (html+css+svg, source-availability link to github.com/AO-Cyber-Systems/EdenDocs) and a README documenting the intentional `collabora-office-white.svg` filename-collision overwrite mechanism round it out. Commits: `938f8bf3714` (assets), `160335432f3` (css+js), `25ce2fbc786` (welcome+README), `cd6a1d2da9f` (fix: drop "Collabora blue" from the shipped branding.css comment). Two Rule-1 auto-fixes: the English word "Collaborative" in the welcome lede and a "Collabora blue" css comment both tripped the mechanical zero-Collabora gate → reworded. BRAND-01/03/04 targeted but NOT marked complete — they close after TRD 02-04 wires these assets into the build. Test-gate note: the blanket collabora grep legitimately trips on README.md's required filename documentation; the narrowed (non-README) gate passes. Previously: 02-02 removed backstage/admin-Version-tab trademark strings (`f13d4afc092`, `b2759f21f4a`); 02-01 patched editor + admin `<title>` tags (`3aa5befe6c5`, `d27976178bb`); Objective 1 finished all 3 waves (CI GREEN on eden-main).

Progress: [██████░░░░] 20%

## Accumulated Context

> Decisions and performance metrics are logged in STATE_ARCHIVE.md.

### Pending Todos

[From .planning/todos/pending/ — ideas captured during sessions]

None yet.

### Blockers/Concerns

- ~~Branch-tracking rule must precede Objective 2~~ **RESOLVED 2026-07-08** — recorded as a PROJECT.md Key Decision: `eden-main` tracks `upstream/main` (weekly-minimum merge cadence). Rationale: `main` is what `eden-main` already tracks and the only branch (besides `co-25.04`) publishing the `engine-*-assets.tar.gz` prebuilt tarball Objective 1's pipeline depends on. Objective 5 still formalizes the written workflow + divergence allowlist.
- ~~Eden brand tokens are TBD~~ **RESOLVED 2026-07-08** — recorded as a PROJECT.md Key Decision: Eden UI gold `#D4A853` ramp from the canonical Eden design system (`eden-ui-flutter` `lib/src/tokens/colors.dart` `EdenColors.gold`, 50–950 shades; primary "defaults to gold"), matching AO master-brand gold. Logo = official AO emblem (`https://aocyber.ai/images/ao-icon.svg`) + "EdenDocs" wordmark. Config/CSS-driven branding keeps a future palette swap cheap.
- **POCO tarball completeness unverified**: It's unverified whether `engine-main-assets.tar.gz` ships enough of `workdir/UnpackedTarball/poco/` for `--with-lo-builddir` to resolve POCO natively. Validate early in Objective 1 (per BUILD-02) rather than discovering it mid-CI-setup.
- **CheckFileInfo schema / AOID token model not cross-checked**: The WOPI `CheckFileInfo` JSON schema (sdk.collaboraonline.com) and AOID's actual OIDC claims/token-issuance model haven't been cross-referenced yet. Flagged for `/devflow:research-objective` before Objective 3 implementation starts.
- **No prebuilt engine tarball for `distro/collabora/co-26.04`**: Only `main` and `co-25.04` currently publish `engine-*-assets.tar.gz`. If `eden-main` ever rebases onto a `co-26.04` distro branch, engine would need building from source until Collabora publishes a matching asset — factor into the Objective 5 branch-tracking decision.

## Session Continuity

Last session: 2026-07-07
Stopped at: Roadmap creation complete (.planning/ROADMAP.md, .planning/STATE.md written; REQUIREMENTS.md traceability updated)
Resume file: None
