---
objective: 02-edendocs-rebrand-web-ui
verified: 2026-07-08T20:02:27Z
status: gaps_found
score: 4/5 must-haves verified
gaps:
  - truth: "No Collabora trademarks remain on any user-facing surface, verified against the written trademark/MPL checklist (final all-green verify-branding.sh CI evidence)"
    status: partial
    reason: >-
      EXOGENOUS-BLOCKED, not a code defect. All implementation is complete and
      independently verified: the checklist exists and is substantive, all 5
      upstream trademark patches are applied and isolated, MPL legal files are
      byte-intact, and the verify-branding.sh gate empirically works (run
      28964158981 log confirms gates 0-4 passed on the real Linux dist and
      gate 5 caught the sole genuine leak — adminIntegratorSettings title —
      which was patched as isolated commit 6f9e184b4ec). The FINAL all-green
      "BRANDING VERIFICATION PASSED" run is blocked because upstream
      republished the rolling engine-main-assets.tar.gz at
      2026-07-08T18:02:24Z and the new engine SIGSEGVs kit startup
      (independently confirmed in run 28968503099 log: SIGSEGV in
      coolforkit-caps + curl (28) on 9980, failing at the smoke step BEFORE
      the branding gate; the only source delta vs the run that reached gate 5
      is the one-line HTML title patch — physically incapable of a forkit
      SIGSEGV).
    artifacts:
      - path: "docs/eden/TRADEMARK-MPL-CHECKLIST.md"
        issue: "§8 Green-run evidence column not yet filled with an all-green run URL + BRANDING VERIFICATION PASSED line"
      - path: ".planning/ROADMAP.md"
        issue: "02-05 correctly left unticked pending the green run"
      - path: ".planning/REQUIREMENTS.md"
        issue: "BRAND-05 correctly left unmarked pending the green run"
    missing:
      - "Execute the documented resume action (02-05-SUMMARY.md): poll `gh api repos/CollaboraOnline/online/releases/tags/for-code-assets --jq '.assets[] | select(.name==\"engine-main-assets.tar.gz\") | .updated_at'` for a value newer than 2026-07-08T18:02:24Z, then `gh run rerun 28964794754 --repo AO-Cyber-Systems/EdenDocs`"
      - "On green: fill checklist §8 evidence, commit `docs(02-05): checklist evidence from green CI run (BRAND-05 verified)`, tick 02-05 in ROADMAP.md, mark BRAND-05 complete in REQUIREMENTS.md"
---

# Objective 2: EdenDocs Rebrand of the Web UI — Verification Report

**Objective Goal:** Every user-facing surface of the editor and admin console — name, logo, colors, favicon, About dialog, help links — shows EdenDocs branding, delivered through config keys and an additive `eden-branding/` directory rather than scattered source edits, with Collabora trademarks removed and MPL/legal attribution preserved.

**Verified:** 2026-07-08T20:02:27Z
**Status:** gaps_found (single gap, exogenous-blocked; zero code defects found)
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (the 5 OBJECTIVE.md / ROADMAP Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Editor shows EdenDocs name/logo via `coolwsd.xml` brand keys + additive `eden-branding/` wired with `--with-app-branding`, no upstream edits for the mechanism | ✓ VERIFIED | All 11 `eden-branding/` files present; real AO emblem 76398 bytes with gradient stops `fce88d`/`efb32c`; `build.sh:63` `--with-app-branding="$PWD/eden-branding"`; `build.sh:91-98` injects + loudly asserts `brandProductName`=EdenDocs, `brandProductURL`=https://aocyber.ai, `logoURL`=images/eden-logo.svg; TRD 02-03 commit range purely additive (3117 insertions, 0 deletions, 0 upstream files). Runtime: green run 28960901457 log shows served cool.html references branding.css and eden-logo.svg served with `efb32c` |
| 2 | Tab titles + About read "EdenDocs" via two isolated `<title>` patches + compiled `--with-app-name`/`--with-vendor` fallbacks | ✓ VERIFIED | `cool.html.m4:42` = `<title>EdenDocs</title>`; `admintemplate.html:12` = `<title>EdenDocs - Admin console</title>` (0 collabora hits); commits `3aa5befe6c5`/`d27976178bb` each exactly 1 file, 1 line; `build.sh:64-65` `--with-app-name="EdenDocs"` / `--with-vendor="AO Cyber Systems"`; green-run log confirms served title + vendor string checks passed |
| 3 | Admin console branded via shared `%BRANDING_JS%` hook; favicon replaced; palette via branding-dir CSS, upstream `color-palette.css` never edited | ✓ VERIFIED | `branding.js:17-18` sole producer of `window.brandProductName`; green run's live Basic-Auth admin.html check passed (`branding.js` referenced); `build.sh:116` favicon overlay + smoke `cmp -s` byte-identity check passed; `branding.css` 9 `!important` gold overrides incl. `--color-primary: #D4A853`; **0 commits touch `browser/css/color-palette.css`** in the objective range |
| 4 | No Collabora trademarks on user-facing surfaces, verified against written trademark/MPL checklist; MPL headers/COPYING/THIRDPARTYLICENSES intact | ⚠ PARTIAL | Implementation 100% verified (see below); final all-green CI evidence blocked by exogenous upstream engine republish. This is the single gap |
| 5 | `INFOBAR_URL`/`FEEDBACK_URL`/`INFO_URL` unset; `--with-support-public-key`/`--with-welcome-url` never passed | ✓ VERIFIED | 0 literal forbidden flags in `build.sh`; `verify-branding.sh:119-124` asserts all 4 flags absent; `build.sh:81-82` + `verify-branding.sh:125-130` brandProductName canary (the key only survives configure when NO welcome URL is set — mechanical proof); green run reached `build.sh complete` under `set -euo pipefail`, proving the canary held |

**Score:** 4/5 truths fully verified; truth 4 partial (implementation verified, terminal CI evidence pending exogenous unblock)

### Required Artifacts (Levels 1-3: exists, substantive, wired)

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `eden-branding/` (11 files) | Additive brand overlay | ✓ VERIFIED | All present: branding.css/js, favicon.ico (3-layer), 4 emblem SVGs (all carry `efb32c`), welcome/ (3 files, 0 collabora hits, MPL source link present), README.md |
| `eden-branding/images/eden-emblem.svg` | Real AO emblem, 76398 bytes, gradient intact | ✓ VERIFIED | Exactly 76398 bytes; both `fce88d` and `efb32c` present; never recolored |
| `browser/html/cool.html.m4` | Isolated title patch | ✓ VERIFIED | Commit `3aa5befe6c5`: 1 file, +1/-1 |
| `browser/admin/admintemplate.html` | Isolated title patch | ✓ VERIFIED | Commit `d27976178bb`: 1 file, +1/-1 |
| `browser/src/control/backstage/Sidebar.tsx` | Isolated backstage-header patch, MPL header untouched | ✓ VERIFIED | Commit `f13d4afc092`: 1 file, +1/-1; line 72 reads "EdenDocs"; MPL header attribution intact |
| `browser/admin/adminSettings.html` | Isolated Version-tab patch | ✓ VERIFIED | Commit `b2759f21f4a`: 1 file, +2/-2; "EdenDocs (coolwsd)" / "LibreOffice Core Engine"; element IDs untouched |
| `browser/admin/adminIntegratorSettings.html.m4` | Isolated title patch (gate-5 catch) | ✓ VERIFIED | Commit `6f9e184b4ec`: 1 file, +1/-1; line 40 = `<title>EdenDocs - Settings</title>` |
| `scripts/eden/build.sh` | 4 branding flags, brand-key injection, canary, favicon/welcome overlays | ✓ VERIFIED | All present with loud assertions; full welcome replacement (`rm -rf` + `cp -R`); post-overlay collabora re-check at line 154 |
| `scripts/eden/smoke-test.sh` | 6 runtime branding checks on native 9980 | ✓ VERIFIED | Lines 68-85: branding.css, title, vendor, favicon byte-cmp, eden-logo gold, Basic-Auth admin branding.js; fetch-to-file pattern (no pipefail EPIPE trap); no 8080 anywhere |
| `scripts/eden/verify-branding.sh` | Mechanical BRAND-01..06 gate | ✓ VERIFIED | 132 lines, 8 gate groups: exceptions hygiene, dist artifacts + emblem, titles/vendor, brandProductName, palette/welcome, per-occurrence minification-safe sweep, MPL byte-integrity vs HEAD, 4 negative flag greps + canary. Static-only, portless |
| `scripts/eden/trademark-exceptions.txt` | Justified fixed-string allowlist | ✓ VERIFIED | 22 lines, 0 empty, 0 comments (hygiene-gated); spot-checked lines all have checklist rows |
| `docs/eden/TRADEMARK-MPL-CHECKLIST.md` | Written BRAND-05 checklist | ✓ VERIFIED (content) / ⚠ §8 pending | 177 lines, 8 sections: taxonomy, 13 per-surface rows with commit+evidence columns, justified exceptions with revisit triggers, sweep-scope table, MPL section with §3.2 source link, BRAND-06 section, divergence allowlist naming all 5 patch commits. §8 green-run evidence awaits the unblocked run |
| `.github/workflows/build.yml` | Branding gate step after smoke | ✓ VERIFIED | Lines 59-60: "Verify EdenDocs branding + trademark/MPL gates" → `./scripts/eden/verify-branding.sh`; file is Eden-owned (created Objective 1, `b6178f7959b`) — not upstream divergence |
| `COPYING`, `THIRDPARTYLICENSES`, `CODA-THIRDPARTYLICENSES.html` | Byte-intact MPL attribution | ✓ VERIFIED | `git diff --stat HEAD` empty; 0 commits touch them in the objective range; gate 6 enforces byte-identity forever |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| build.yml | verify-branding.sh | CI step after smoke | ✓ WIRED | build.yml:59-60 |
| build.yml | build.sh / smoke-test.sh | CI steps | ✓ WIRED | build.yml:48, 55 |
| configure | eden-branding/ | `--with-app-branding="$PWD/eden-branding"` | ✓ WIRED | build.sh:63; dist artifacts asserted by gates 1-4, proven on real dist in run 28964158981 |
| coolwsd.xml | brand keys | build.sh injection + assertions | ✓ WIRED | build.sh:91-98; canary at :81-82 |
| admin console | branding.js | `%BRANDING_JS%` hook | ✓ WIRED | Runtime-proven: green run's Basic-Auth admin.html fetch found `branding.js` |
| Backstage glyph | AO emblem | filename collision `collabora-office-white.svg` | ✓ WIRED | Collision file carries `efb32c`; gate 1 asserts overwrite landed in dist |
| verify-branding.sh | trademark-exceptions.txt | `grep -viFf` per-occurrence filter | ✓ WIRED | verify-branding.sh:94-96; hygiene gate 0 protects the mechanism |
| exceptions lines | checklist rows | stated invariant | ✓ WIRED | Spot-check: 5/5 sampled lines found verbatim in checklist §4 |

### Requirements Coverage

| Requirement | Source TRD(s) | Status | Evidence |
|-------------|---------------|--------|----------|
| BRAND-01 | 02-03, 02-04 | ✓ SATISFIED | Assets + wiring verified; runtime proof run 28960901457 |
| BRAND-02 | 02-01, 02-04 | ✓ SATISFIED | Both title patches + app-name/vendor flags; runtime title+vendor checks green |
| BRAND-03 | 02-03, 02-04 | ✓ SATISFIED | branding.js global + live Basic-Auth admin check green |
| BRAND-04 | 02-03, 02-04 | ✓ SATISFIED | Favicon byte-cmp green; gold palette in branding-dir CSS; color-palette.css untouched |
| BRAND-05 | 02-02, 02-05 | ⚠ PARTIAL | All patches applied, checklist written, MPL intact, gate proven-working; final green-run evidence exogenous-blocked (see gap) |
| BRAND-06 | 02-04, 02-05 | ✓ SATISFIED | Omit list + canary + gate 7 negative asserts; green run proves canary held |

No orphaned requirements: REQUIREMENTS.md maps exactly BRAND-01..06 to Objective 2; all six are claimed by TRD frontmatter. REQUIREMENTS.md state (01/02/03/04/06 `[x]`, 05 `[ ]`) and ROADMAP (02-05 unticked) are honest and consistent with the blocked state — completion was correctly NOT claimed.

### Mergeability Discipline (spot-check requested by orchestrator)

22 commits in `3aa5befe6c5^..7b0ec91787c`. Outside Eden-owned paths (`.planning/`, `eden-branding/`, `scripts/eden/`, `docs/eden/`), exactly 6 files were touched:

- 5 upstream files, each in exactly one isolated commit (1 file, 1-2 lines): `cool.html.m4` (`3aa5befe6c5`), `admintemplate.html` (`d27976178bb`), `Sidebar.tsx` (`f13d4afc092`), `adminSettings.html` (`b2759f21f4a`), `adminIntegratorSettings.html.m4` (`6f9e184b4ec`) — matching the documented divergence-allowlist candidates exactly.
- `.github/workflows/build.yml` (`3f2147f91c0`) — Eden-owned file created in Objective 1; not upstream divergence.

Repo-root `favicon.ico`: last commit is upstream (`bf46b8fe957`); working tree clean — the build-time overlay was never committed, as documented.

### Anti-Patterns Found

None. Zero TODO/FIXME/placeholder hits in `eden-branding/`, `scripts/eden/`, `docs/eden/`. Zero port-8080 references outside prohibition context.

### Functional Verification

Local build/run is impossible on this macOS host (dist requires the Linux CI path). Functional Level-4 evidence taken from CI logs, independently re-fetched during this verification:

| Check | Run | Log evidence (verbatim) | Status |
|-------|-----|--------------------------|--------|
| Full build + brand-key canary | 28960901457 (success) | `build.sh complete: ./coolwsd and ./browser/dist present.` | ✓ |
| 6 runtime branding checks on live coolwsd:9980 | 28960901457 (success) | `BRANDING SMOKE PASSED: EdenDocs branding served on 9980` + `SMOKE TEST PASSED` | ✓ |
| verify-branding gates 0-4 on real Linux dist | 28964158981 | Gate 5 reached under `set -euo pipefail` ⇒ gates 0-4 passed | ✓ |
| Gate-5 sweep efficacy (catches real leaks) | 28964158981 | `ERROR: unexplained Collabora trademark occurrences` → `<title>Collabora Online - Settings</title>` in `browser/dist/adminIntegratorSettings.html` — exactly one survivor, subsequently patched (`6f9e184b4ec`) | ✓ (gate works as designed) |
| Latest-run failure exogeneity | 28968503099 (failure) | `SIG Fatal signal received: SIGSEGV` in `coolforkit-caps` + `curl: (28) Failed to connect to 127.0.0.1 port 9980` at the SMOKE step — before the branding gate; head commit `7b0ec91787c` is docs-only | ✓ EXONERATED |

### Human Verification Required

None blocking. Optional (aesthetic-only, cannot affect the mechanical success criteria): eyeball the gold palette/logo lockup rendering in a live session once the container image (Objective 4) exists.

### Gaps Summary

The objective's implementation is complete and defect-free: every artifact exists, is substantive, and is wired; all five upstream edits are isolated single-file commits matching the documented allowlist; MPL/legal attribution is byte-intact and permanently gated; the trademark sweep gate demonstrably works (its first real run caught the one genuine leak, which was fixed, not excepted). The single gap is terminal evidence, not code: the final all-green `BRANDING VERIFICATION PASSED` CI run — and its three bookkeeping consequences (checklist §8, ROADMAP 02-05 tick, BRAND-05 mark) — is blocked by an exogenous upstream republish of the rolling `engine-main-assets.tar.gz` that SIGSEGVs kit startup before the branding gate executes. The tree is exonerated (only source delta vs the last gate-reaching run is a one-line HTML title). The resume action is fully documented in 02-05-SUMMARY.md and requires no re-planning — this gap should be closed by executing the pending rerun when upstream republishes, not by `/devflow:plan-objective --gaps`.

---

_Verified: 2026-07-08T20:02:27Z_
_Verifier: Claude (devflow-verifier)_
