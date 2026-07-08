# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-07)

**Core value:** Eden customers can collaboratively edit documents in the browser on infrastructure they control — privacy-first, with no third-party document cloud.
**Current focus:** Objective 2 — EdenDocs Rebrand of the Web UI

## Current Position

Objective: 2 of 5 (EdenDocs Rebrand of the Web UI) — Objective 1 (From-Source Build Pipeline) complete, all 3 jobs done (01-01, 01-02, 01-03)
Job: 02-02 complete (backstage header + admin Version-tab trademark patches); next 02-03 (Eden branding assets)
Status: Executing — wave 1 of 3, TRD 02-02 done
Last activity: 2026-07-08 — TRD 02-02 complete: two isolated single-file upstream-patch commits removed the remaining trademark strings BRAND-05 targets — `browser/src/control/backstage/Sidebar.tsx` (always-visible Backstage header "Collabora Office" → "EdenDocs", commit `f13d4afc092`) and `browser/admin/adminSettings.html` (Version-tab headers "Collabora Online" → "EdenDocs (coolwsd)" and "Collabora Office Engine" → "LibreOffice Core Engine", commit `b2759f21f4a`). Sidebar.tsx's MPL/SPDX license header ("Copyright the Collabora Online contributors.") was left byte-intact per the legal-attribution rule; the file-body-only negative grep confirmed no other Collabora strings remain. Both commits verified as single-file diffs and recorded as Objective 5 divergence-allowlist candidates in `02-02-SUMMARY.md`. BRAND-05 intentionally left incomplete pending TRD 02-05's full trademark-checklist verification pass. Previously: TRD 02-01 replaced the static `<title>` tags in `cool.html.m4` and `admintemplate.html` (commits `3aa5befe6c5`, `d27976178bb`). Objective 1 finished all 3 waves — build.yml push-triggered CI GREEN on eden-main (cold run 28910753718 14m28s, warm run 28911376952 8m19s with 100% ccache hits), BUILDING-EdenDocs.md build doc added and cross-verified against scripts/CI. BUILD-01/02/03/04 all log-evidenced.

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
