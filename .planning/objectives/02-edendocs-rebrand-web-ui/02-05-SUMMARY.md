---
objective: 02-edendocs-rebrand-web-ui
job: "05"
subsystem: infra
tags: [trademark, mpl, checklist, verify-branding, ci-gate, sweep, brand-05, brand-06]

# Dependency graph
requires:
  - "02-01: title patches (cool.html.m4 / admintemplate.html)"
  - "02-02: trademark patches (Sidebar.tsx / adminSettings.html)"
  - "02-03: eden-branding/ assets"
  - "02-04: build wiring + smoke checks (green run 28960901457)"
provides:
  - "docs/eden/TRADEMARK-MPL-CHECKLIST.md — written BRAND-05 checklist (taxonomy, per-surface rows, justified exceptions, MPL section, BRAND-06 section, divergence allowlist)"
  - "scripts/eden/trademark-exceptions.txt — 22 justified fixed-string exception lines (every line has a checklist row)"
  - "scripts/eden/verify-branding.sh — mechanical BRAND-01..06 gate: per-occurrence dist sweep + MPL byte-integrity + no-phone-home flag asserts"
  - ".github/workflows/build.yml — 'Verify EdenDocs branding + trademark/MPL gates' step after smoke test"
affects: [05-upstream-sync-divergence-allowlist]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Per-occurrence trademark sweep: grep -oi ±40-char snippets filtered with grep -viFf against a comment-free fixed-string exceptions file (minification-safe)"
    - "Sweep scope mirrors upstream's production ship-set (Makefile.am install EXCLUDES: dist/src mirror + dev-harness pages)"
    - "Exceptions-file hygiene gate: empty line in a grep -vf pattern file silently disables the whole sweep — asserted against"

key-files:
  created:
    - docs/eden/TRADEMARK-MPL-CHECKLIST.md
    - scripts/eden/trademark-exceptions.txt
    - scripts/eden/verify-branding.sh
  modified:
    - .github/workflows/build.yml
    - browser/admin/adminIntegratorSettings.html.m4

key-decisions:
  - "Sweep scope = production ship-set: --exclude-dir=src + the 5 dev-harness pages, mirroring browser/Makefile.am's own non-debug install EXCLUDES (debug builds mirror all sources into dist/src — sweeping it floods on comments while its literals are all swept inside bundle.js anyway)"
  - "22 exception lines pre-seeded from a full source-literal inventory (not just the TRD's 8) — result: ZERO false survivors on first real-dist contact"
  - "adminIntegratorSettings.html.m4 title = genuine class-1 surface caught by the gate — fixed as isolated one-file upstream patch, NOT excepted"
  - "Engine tarball is a rolling upstream asset — its 18:02Z republish broke kit startup and blocks final green evidence; gate code exonerated and unchanged"

patterns-established:
  - "Every exceptions-file line pairs with a justifying checklist row; additions require both in the same commit (stated invariant in the checklist)"

requirements-completed: []

# Verification evidence
verification:
  gates_defined: 3
  gates_passed: 2
  auto_fix_cycles: 1
  tdd_evidence: false
  test_pairing: false

# Metrics
duration: ~2.5h (3 CI cycles + 1 exogenous-failure rerun)
completed: 2026-07-08 (Tasks 1-2 complete; Task 3 BLOCKED-EXOGENOUS, see below)
---

# Objective 2 TRD 05: Trademark Checklist + Mechanical Verification Gate — PARTIAL (2/3 tasks)

**BRAND-05's written trademark/MPL checklist now exists with a 6-class taxonomy, 13 per-surface rows, and 21 justified exception rows backing a 22-line fixed-string allowlist, and `scripts/eden/verify-branding.sh` enforces it mechanically in CI (per-occurrence minification-safe dist sweep + MPL byte-integrity + BRAND-06 no-phone-home asserts) — the gate's first real run caught a genuine missed trademark (integrator-settings title, patched as isolated commit `6f9e184b4ec`), but the final green-run evidence is BLOCKED by an exogenous upstream engine-tarball republish that SIGSEGVs kit startup.**

## Status: Tasks 1-2 COMPLETE, Task 3 BLOCKED-EXOGENOUS

Do **not** tick 02-05 in ROADMAP or mark BRAND-05 complete until the resume
action below lands a green run.

### Task 3 blocker — upstream engine flip (tree exonerated)

- Upstream Collabora republished the **rolling** asset
  `engine-main-assets.tar.gz` (release tag `for-code-assets`,
  CollaboraOnline/online) at **2026-07-08T18:02:24Z** — between our last
  smoke-green run and the next.
- Green run [28964158981](https://github.com/AO-Cyber-Systems/EdenDocs/actions/runs/28964158981)
  (fetched 17:58:45Z) used engine sha256
  `ea91ef33ba510eb1379568c58afe9ddc54aa05985f4ff963ab8e0a8cdfa6c59f` —
  build + smoke green (failed only at the sweep gate, correctly).
- Run [28964794754](https://github.com/AO-Cyber-Systems/EdenDocs/actions/runs/28964794754)
  attempts 1 AND 2 fetched the NEW engine sha256
  `076a41297c2ed659638617ed27cb6d4cff01f1558f36fb0e0b1abd6f4a14868e`
  (asset id 470521782, size 433537018) and failed identically:
  `SIGSEGV` in kit startup (`forkCOKit` → `lokit_main` → `startMainLoop`
  in `coolforkit-caps` / `libmergedlo.so`), coolwsd never accepting on 9980
  (`curl: (28)` after ~134s; smoke's 120-probe readiness loop exhausted).
- The one-line commit `6f9e184b4ec` between those runs edits an HTML title in
  an m4 template — physically incapable of a forkit SIGSEGV; the engine flip
  is the only variable, and the crash is in the engine-loading component.
- The old engine is unrecoverable (rolling asset overwritten; CI caches only
  ccache). This VALIDATES STATE.md's pre-existing unpinned-rolling-engine
  concern.

**Resume action:** when
`gh api repos/CollaboraOnline/online/releases/tags/for-code-assets --jq '.assets[] | select(.name=="engine-main-assets.tar.gz") | .updated_at'`
shows a value newer than `2026-07-08T18:02:24Z`, run
`gh run rerun 28964794754 --repo AO-Cyber-Systems/EdenDocs` and gate on it.
On green: fill checklist §8 + Evidence column (run URL +
`BRANDING VERIFICATION PASSED` line), commit
`docs(02-05): checklist evidence from green CI run (BRAND-05 verified)`,
tick 02-05 in ROADMAP, mark BRAND-05 complete in REQUIREMENTS.md.

## Accomplishments

- **Written checklist** (`docs/eden/TRADEMARK-MPL-CHECKLIST.md`): 6-class
  taxonomy, 13 per-surface rows (all of 02-01..02-04's surfaces + container
  LABELs verified-clean row + the new integrator-settings catch), 21
  exception rows with justification + revisit triggers, sweep-scope table,
  MPL preservation section (byte-identical legal files, MPL §3.2 source link
  https://github.com/AO-Cyber-Systems/EdenDocs), BRAND-06 section, divergence
  allowlist (now 5 upstream patch files + favicon overlay remark).
- **Exceptions file**: 22 comment-free fixed-string lines built from a FULL
  inventory of every Collabora string literal feeding the built dist
  (bundle.js/admin-bundle.js/css/html/svg) — proven exact: zero false
  survivors on the real Linux dist.
- **verify-branding.sh**: 8 gate groups (exceptions hygiene; BRAND-01
  artifacts + gold emblem; BRAND-02 titles/vendor; BRAND-03 brandProductName;
  BRAND-04 palette/welcome; BRAND-05 per-occurrence sweep; BRAND-05 legal
  byte-integrity vs git HEAD; BRAND-06 four negative flag greps + coolwsd.xml
  canary with local skip-warning). Static only — no server, no ports.
- **CI wiring**: single additive thin-orchestrator step after the smoke test;
  concurrency/cache blocks untouched; YAML-linted.
- **The gate works**: its first execution against a real dist surfaced
  exactly one genuine leak with self-diagnosing output, which became isolated
  upstream patch #5.

## Task Evidence

| Task | Verify command | Result |
|------|----------------|--------|
| 1: checklist + exceptions | TRD verify chain (`test -f` + 5 greps + line-count ≥6 + no-`#`), plus no-empty-line guard | PASS — `TASK 1 VERIFY PASSED` |
| 2: verify-branding.sh + build.yml step | TRD verify chain (`bash -n`, `-x`, 5 content greps, YAML parse, workflow wiring, no-8080) | PASS — `TASK 2 VERIFY PASSED`; sweep logic additionally proven on a synthetic fixture (excepted string filtered, real leak caught, src/ + debug.html excluded) |
| 3: drive gate green + evidence | `gh run list ... conclusion=success` + evidence grep + legal diff-clean | BLOCKED-EXOGENOUS — legal files diff-clean locally (PASS); latest run failure is the upstream engine SIGSEGV at the smoke step, before the branding gate |

## Task Commits

1. **Task 1: checklist + exceptions** — `0b95f6ae237` (docs)
2. **Task 2: verify script + CI step** — `3f2147f91c0` (feat)
3. **Task 3 cycle 1: gate-3 quote-agnostic fix** — `4771a289f99` (fix)
4. **Task 3 cycle 2: integrator-settings title (sweep catch)** — `6f9e184b4ec` (feat, isolated upstream patch)

## CI Cycles (3 of 4 budget used; + 1 exogenous rerun)

| # | Run | Outcome | Action |
|---|-----|---------|--------|
| 1 | [28963503473](https://github.com/AO-Cyber-Systems/EdenDocs/actions/runs/28963503473) | Gate 3 red: exact-string check assumed double quotes; `eden-branding/branding.js:17` uses `'EdenDocs'` | `4771a289f99` — quote-agnostic `grep -qE` |
| 2 | [28964158981](https://github.com/AO-Cyber-Systems/EdenDocs/actions/runs/28964158981) | Gate 5 red: ONE survivor — `<title>Collabora Online - Settings</title>` (`adminIntegratorSettings.html.m4:40`, class-1 user-facing) | `6f9e184b4ec` — isolated one-file title patch (fix, not exception) |
| 3 | [28964794754](https://github.com/AO-Cyber-Systems/EdenDocs/actions/runs/28964794754) ×2 attempts | Smoke red BEFORE the gate: upstream engine flip SIGSEGV (see blocker) | None — tree exonerated; rerun pending upstream republish |

## Validation Gate Results

| Gate | Command | Status |
|------|---------|--------|
| lint | `bash -n scripts/eden/verify-branding.sh && python3 -c "import yaml; yaml.safe_load(open('.github/workflows/build.yml'))"` | PASS |
| test | `! grep -rn '8080' scripts/eden/verify-branding.sh docs/eden/TRADEMARK-MPL-CHECKLIST.md \| grep -vi 'never\|banned\|prohibit\|do not' \| grep -q .` | PASS |
| build | latest build.yml run conclusion | BLOCKED-EXOGENOUS (28964794754 failure = upstream engine SIGSEGV at smoke, pre-gate; gate itself last executed in 28964158981 and behaved exactly as designed) |

## Post-TRD Verification

- All four artifacts exist and are wired (key_links hold: build.yml→verify-branding.sh, script→exceptions via `grep -viFf`, exceptions→checklist rows).
- Legal files diff-clean vs HEAD locally (`git diff --stat` empty).
- Gate behavior empirically proven on real dist (gates 1-4 PASS, gate 5 catch with self-diagnosing output) — final all-green log line pending the resume action.

## Deviations from Plan

- **Sweep scope excludes added** (include-globs are the TRD's sanctioned
  knob): `--exclude-dir=src` + the 5 dev-harness pages, mirroring upstream's
  own production install EXCLUDES. Reason: debug builds mirror every source
  file into `browser/dist/src` (tsc `--outDir`), which floods a dist sweep
  with comment hits carrying zero shipped-artifact meaning; all string
  literals remain swept inside bundle.js/admin-bundle.js. Documented in
  checklist §4 "Sweep scope".
- **Exceptions file has 22 lines, not the TRD's 8 pre-seeds**: a full source
  inventory (all string literals feeding dist) was done up front; every line
  has a justified checklist row. Result: zero tuning cycles wasted on
  known-in-advance survivors.
- **Gate-0 hygiene checks added** (empty-line / comment-line asserts on the
  exceptions file): an empty pattern line would silently disable the sweep
  via `grep -vf` — cheap permanent insurance.
- **Task 3 not finished** — exogenous upstream engine breakage (full
  diagnosis above); per orchestrator direction the gate was NOT weakened,
  the engine was NOT pinned/swapped, and completion state was NOT claimed.
- Metadata-commit push will trigger another run that fails at smoke — that
  is EXPECTED (same exogenous engine) and exonerated.

## Issues Encountered

- `df-tools` DevFlow ambient-mode gate blocks direct Write/Edit — resolved
  with `skill-active --start execute-objective` per task setup instructions.
- GitHub log for the smoke failure is dominated by benign cpio/systemplate
  noise; the real signal (`curl: (28)`, `SIGSEGV`, `Fatal signal`) is near
  the end — grep for it.

## User Setup Required

None.

## Next Objective Readiness

- Objective 5 (upstream sync) harvest list is consolidated in checklist §7:
  **5 isolated upstream patch files** — `browser/html/cool.html.m4`
  (`3aa5befe6c5`), `browser/admin/admintemplate.html` (`d27976178bb`),
  `browser/src/control/backstage/Sidebar.tsx` (`f13d4afc092`),
  `browser/admin/adminSettings.html` (`b2759f21f4a`),
  `browser/admin/adminIntegratorSettings.html.m4` (`6f9e184b4ec`) — plus the
  repo-root `favicon.ico` build-time-only overlay remark (never committed).
- Objective 4 (packaging) should decide whether to strip the dev-harness
  pages from shipped images (they are outside the sweep scope by upstream's
  own production convention).

---
*Objective: 02-edendocs-rebrand-web-ui*
*Tasks 1-2 completed: 2026-07-08; Task 3 blocked-exogenous (resume action documented above)*
