---
objective: 02-edendocs-rebrand-web-ui
job: "04"
subsystem: infra
tags: [autotools, configure-flags, coolwsd-xml, branding, favicon, welcome, smoke-test, ci]

# Dependency graph
requires:
  - "02-01: title patches (cool.html.m4 / admintemplate.html)"
  - "02-02: trademark patches"
  - "02-03: eden-branding/ directory (branding.css, branding.js, favicon.ico, images/, welcome/)"
  - "01-02: green build.yml CI on eden-main (scripts/eden/*.sh path)"
provides:
  - "scripts/eden/build.sh — 4 branding configure flags (--with-app-branding/--with-app-name/--with-vendor/--with-help-url), brandProductName canary, coolwsd.xml brand-key injection (EdenDocs / https://aocyber.ai / eden-logo.svg), favicon + welcome build-time overlays, loud dist assertions"
  - "scripts/eden/smoke-test.sh — 6 runtime branding checks against coolwsd on native port 9980 (fetch-to-file pattern), incl. Basic-Auth admin-console BRAND-03 check"
  - "Green end-to-end CI proof for BRAND-01/02/03/04/06 (run 28960901457)"
affects: [02-05-verification]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Runtime branding checks live in smoke-test.sh against the SERVED pages on 9980, not just dist files on disk"
    - "curl fetch-to-file then grep — never `curl | grep -q` under pipefail (EPIPE/exit-23 false negative)"

key-files:
  created: []
  modified:
    - scripts/eden/build.sh
    - scripts/eden/smoke-test.sh

key-decisions:
  - "Zero upstream-file edits: all wiring lives in the two Eden-owned scripts (coolwsd.xml.in, configure.ac, browser/Makefile.am untouched)"
  - "brandProductName/brandProductURL/logoURL only exist in generated coolwsd.xml when NO welcome-url flag is passed (configure.ac else-branch, research Pitfall 1) — canary assertion guards this forever"
  - "Vendor string reaches served cool.html via automake's automatic `VENDOR = @VENDOR@` Makefile.in preamble for every AC_SUBST'd var → -DVENDOR at m4 time expands the bare VENDOR tokens in cool.html.m4; empirically confirmed by the green run's vendor check"
  - "browser/dist/welcome gets FULL replacement (rm -rf + cp -R), never partial cp — upstream slide*.png/welcome.js would survive a partial overlay"
  - "Favicon has no config override (hardcoded lookup in wsd/ClientRequestDispatcher.cpp) — repo-root favicon.ico is overlaid at build time only and NEVER committed"

patterns-established:
  - "Fetch-to-file smoke checks: grep -q closing the pipe gives curl EPIPE (exit 23) on responses larger than the pipe buffer; pipefail then fails a SUCCESSFUL match (proven in CI run 28959752077 on debug-mode cool.html)"

requirements-completed: [BRAND-01, BRAND-02, BRAND-03, BRAND-04, BRAND-06]

# Verification evidence
verification:
  gates_defined: 3
  gates_passed: 3
  auto_fix_cycles: 1
  tdd_evidence: false
  test_pairing: false

# Metrics
duration: ~3h (multi-session; 2 CI runs)
completed: 2026-07-08
---

# Objective 2 TRD 04: Build Wiring Summary

**The from-source pipeline now builds the fully-branded EdenDocs tree: configure gets the four branding flags, generated coolwsd.xml carries the EdenDocs brand keys (canary-guarded), favicon/welcome are overlaid at build time, and six runtime branding checks pass against the live coolwsd on its native port 9980 — CI GREEN with `BRANDING SMOKE PASSED` in the logs.**

## Green-Run Evidence

**Run: [28960901457](https://github.com/AO-Cyber-Systems/EdenDocs/actions/runs/28960901457)** — success, push-triggered (commit `2bb08f0350f`)

| Requirement | Log evidence |
|---|---|
| BUILD-02 (regression) | `configure: POCO not found in the engine workdir, falling back to system POCO` |
| BRAND-01/06 canary | assert-style (silent on success) — reaching `build.sh complete: ./coolwsd and ./browser/dist present.` under `set -euo pipefail` proves brandProductName existed and was injected |
| BUILD-04 (regression) | `SMOKE TEST PASSED: coolwsd serves /hosting/discovery and /hosting/capabilities on 9980` |
| BRAND-01/02/03/04 | `BRANDING SMOKE PASSED: EdenDocs branding served on 9980` — 6 checks: branding.css in served cool.html, `<title>EdenDocs</title>`, vendor `AO Cyber Systems` in served cool.html, favicon byte-identical to eden-branding/favicon.ico, eden-logo.svg served with gold `efb32c`, branding.js in Basic-Auth admin.html |

The vendor check passing empirically confirms the `--with-vendor` mechanism end-to-end: `AC_SUBST(VENDOR)` (configure.ac:1155) → automake auto-preamble `VENDOR = @VENDOR@` in Makefile.in → `-DVENDOR="$(VENDOR)"` in browser/Makefile.am's m4 recipe → bare `VENDOR` tokens in cool.html.m4 expand to the literal string in built cool.html.

## Task Commits

1. **Task 1: branding configure flags + coolwsd.xml brand keys + welcome-url canary** — `252f98ece5a` (feat)
2. **Task 2: favicon/welcome overlays + runtime branding smoke checks** — `60daa2727fc` (feat)
3. **Task 3 fix cycle 1: fetch-to-file smoke checks** — `2bb08f0350f` (fix)

## Fix Cycles (1 of 4 allowed)

| # | Run | Failure | Fix | Commit |
|---|-----|---------|-----|--------|
| 1 | [28959752077](https://github.com/AO-Cyber-Systems/EdenDocs/actions/runs/28959752077) | `curl: (23) Failure writing output to destination` → `SMOKE TEST FAILED: served cool.html does not reference branding.css` — but the match HAD succeeded: `grep -q` exits at first match and closes the pipe; debug-mode cool.html exceeds the pipe buffer so curl gets EPIPE (exit 23) and pipefail fails the pipeline. Build side fully green in the same run (`build.sh complete` reached; POCO line present). | All `curl \| grep` checks converted to fetch-once-to-mktemp-file then grep; favicon `\| cmp -s -` converted too for uniformity. Ports (9980), fail messages, and PASSED echoes unchanged. | `2bb08f0350f` |

Diagnosis was performed from the raw run log before the fix commit, per protocol. The failed run's log also independently proved the %BRANDING_JS% substitution working (`/browser/60daa2727f/branding.js` visible in the served page dump).

## Validation Gate Results

| Gate | Command | Status |
|---|---|---|
| lint | `bash -n scripts/eden/build.sh && bash -n scripts/eden/smoke-test.sh` | PASS |
| test | `! grep -rn '8080' scripts/eden/*.sh \| grep -vi 'never\|banned\|prohibit\|do not' \| grep -q .` | PASS |
| build | latest build.yml run on eden-main concluded `success` | PASS (run 28960901457) |

## Post-TRD Verification

- build.sh: 4 branding flags present; all 4 forbidden phone-home flags absent as literal `--flag` strings (BRAND-06); canary + injection assertions loud.
- smoke-test.sh: 6 runtime branding checks on 9980 using fail(), incl. the Basic-Auth admin-console BRAND-03 live check.
- CI green with `BRANDING SMOKE PASSED` in logs.
- `git log -- favicon.ico` shows only upstream commits (newest `bf46b8fe957`, upstream) — no Eden commit; working tree clean.

## Deviations from Plan

- **smoke-test.sh check plumbing deviates from the TRD's literal `curl | grep -q` text** (fix cycle 1, necessary bug fix — the TRD's own pattern is a pipefail false negative on large responses). Observable behavior preserved: same URLs, same port 9980, same fail messages, same `BRANDING SMOKE PASSED` / `SMOKE TEST PASSED` echoes. TRD 02-05's greps for check content still hold.
- **Self-introduced drift corrected pre-commit**: during a context-loss reconstruction the vendor check briefly targeted `/hosting/discovery`; restored to the TRD-specified `/browser/dist/cool.html` (with TRD-verbatim fail messages) before Task 2 was committed. Committed code always matched the TRD spec.
- Note: local ugrep resolves patterns oddly when `$PWD` contains the pattern text — validation greps were run with plain grep.

## REMARK — divergence allowlist (UPST-02)

**Repo-root `favicon.ico` is overwritten at BUILD time by scripts/eden/build.sh** (`cp eden-branding/favicon.ico ./favicon.ico`) because coolwsd's favicon route is a hardcoded filesystem lookup with no config override. This is a deliberate build-time overlay of a TRACKED upstream file: the working-tree change is expected after any local build and must be reverted with `git checkout -- favicon.ico`, never committed. Every commit in this TRD was guarded with an explicit `--files` list.

## Next Objective Readiness

- TRD 02-05 (verification) can run its mechanical gates: `! grep -- '--with-welcome-url' scripts/eden/build.sh` stays a plain grep (comments write flag names without leading dashes), and the runtime evidence is reproducible on every push.

---
*Objective: 02-edendocs-rebrand-web-ui*
*Completed: 2026-07-08*

## Self-Check: PASSED

`scripts/eden/build.sh` and `scripts/eden/smoke-test.sh` present on disk; commits 252f98ece5a, 60daa2727fc, 2bb08f0350f all present in `git log`; latest build.yml run on eden-main concluded `success` (28960901457).
