# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-07)

**Core value:** Eden customers can collaboratively edit documents in the browser on infrastructure they control — privacy-first, with no third-party document cloud.
**Current focus:** Objective 2 — EdenDocs Rebrand of the Web UI

## Current Position

Objective: 2 of 5 (EdenDocs Rebrand of the Web UI) — Objective 1 (From-Source Build Pipeline) complete, all 3 jobs done (01-01, 01-02, 01-03)
Job: 02-04 complete (build wiring, CI green); next 02-05 (trademark/MPL checklist + verify-branding.sh — wave 3)
Status: Executing — waves 1+2 of 3 done (02-01/02/03/04 complete); next wave 3 (TRD 02-05)
Last activity: 2026-07-08 — TRD 02-04 complete: wired the eden-branding/ assets into the from-source build entirely inside the two Eden-owned scripts (zero upstream files touched). `scripts/eden/build.sh` now configures with `--with-app-branding=$PWD/eden-branding --with-app-name="EdenDocs" --with-vendor="AO Cyber Systems" --with-help-url=https://aocyber.ai` (never any of the 4 phone-home flags — BRAND-06), asserts the Pitfall-1 `brandProductName` canary in generated coolwsd.xml, injects EdenDocs/https://aocyber.ai/eden-logo.svg into the brand keys via python3 re.sub, overlays repo-root favicon.ico (build-time only, NEVER committed — divergence-allowlist REMARK) and fully replaces browser/dist/welcome (rm -rf + cp -R), with loud dist assertions. `scripts/eden/smoke-test.sh` gained 6 runtime branding checks against live coolwsd on native 9980 (cool.html branding.css + `<title>EdenDocs</title>` + vendor string, favicon byte-compare, eden-logo.svg gold `efb32c`, Basic-Auth admin.html branding.js). CI: first run 28959752077 FAILED on a `curl | grep -q` pipefail EPIPE false negative (curl exit 23 on debug-mode cool.html; build side fully green) → fix cycle 1 converted all checks to fetch-to-file → run 28960901457 GREEN with `BRANDING SMOKE PASSED` + `SMOKE TEST PASSED` + POCO fallback line. The green vendor check empirically confirmed the AC_SUBST(VENDOR)→automake-preamble→m4 `-DVENDOR` chain. Commits: `252f98ece5a` (flags+canary+injection), `60daa2727fc` (overlays+smoke checks), `2bb08f0350f` (fetch-to-file fix). BRAND-01/02/03/04/06 marked complete; BRAND-05 closes in 02-05. Previously: TRD 02-03 complete: created the fully additive `eden-branding/` directory (11 new files, 3117 insertions / 0 deletions, zero upstream files touched, all on eden-main). The **real** AO gold-gradient emblem was downloaded verbatim from https://aocyber.ai/images/ao-icon.svg (76398 bytes; gradient stops `#fce88d` + `#efb32c` asserted present) and derived into `images/{eden-emblem,eden-logo,collabora-office-white,toolbar-bg-logo}.svg`, `welcome/eden-emblem.svg`, and a 3-layer (16/32/48) `favicon.ico`. `eden-logo.svg` nests the emblem as a `<svg>` (class-based gradients preserved) plus an "EdenDocs" gold wordmark. `branding.css` overrides all 7 primary-palette custom properties to the Eden gold ramp (#D4A853 gold-500 … `--color-primary-text` flipped to gold-950 #3D2A14 for WCAG), each carrying `!important` to survive the dark-theme runtime CSS injection; `--color-hyperlink` intentionally left upstream. `branding.js` sets `window.brandProductName = 'EdenDocs'` (sole admin-console producer) + `brandProductURL`. A Collabora-free self-contained welcome page (html+css+svg, source-availability link to github.com/AO-Cyber-Systems/EdenDocs) and a README documenting the intentional `collabora-office-white.svg` filename-collision overwrite mechanism round it out. Commits: `938f8bf3714` (assets), `160335432f3` (css+js), `25ce2fbc786` (welcome+README), `cd6a1d2da9f` (fix: drop "Collabora blue" from the shipped branding.css comment). Two Rule-1 auto-fixes: the English word "Collaborative" in the welcome lede and a "Collabora blue" css comment both tripped the mechanical zero-Collabora gate → reworded. BRAND-01/03/04 targeted but NOT marked complete — they close after TRD 02-04 wires these assets into the build. Test-gate note: the blanket collabora grep legitimately trips on README.md's required filename documentation; the narrowed (non-README) gate passes. Previously: 02-02 removed backstage/admin-Version-tab trademark strings (`f13d4afc092`, `b2759f21f4a`); 02-01 patched editor + admin `<title>` tags (`3aa5befe6c5`, `d27976178bb`); Objective 1 finished all 3 waves (CI GREEN on eden-main).

Progress: [███████░░░] 24%

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

Last session: 2026-07-08
Stopped at: TRD 02-04 complete (CI green, run 28960901457); next TRD 02-05 (wave 3)
Resume file: None
