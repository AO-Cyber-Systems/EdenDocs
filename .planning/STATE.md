# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-07)

**Core value:** Eden customers can collaboratively edit documents in the browser on infrastructure they control — privacy-first, with no third-party document cloud.
**Current focus:** Objective 1 — From-Source Build Pipeline

## Current Position

Objective: 1 of 5 (From-Source Build Pipeline)
Job: 01-02 complete (CI green end-to-end); next 01-03 (build doc)
Status: Executing — wave 2 of 3 complete
Last activity: 2026-07-08 — Wave 2 done: build.yml push-triggered CI GREEN on eden-main (cold run 28910753718 14m28s, warm run 28911376952 8m19s with 100% ccache hits). 5 fix cycles: +POCO Zip module, +LDFLAGS=/usr/local/lib, explicit `make -C browser` (build-nocheck never recurses SUBDIRS), setcap on coolforkit-caps/coolmount for the smoke test. BUILD-01/02/03/04 all log-evidenced.

Progress: [████░░░░░░] 13%

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
