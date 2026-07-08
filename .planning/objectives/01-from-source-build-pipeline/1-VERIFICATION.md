---
objective: 01-from-source-build-pipeline
verified: 2026-07-08T02:05:41Z
status: passed
score: 4/4 must-haves verified
notes:
  - kind: bookkeeping
    path: .planning/ROADMAP.md
    note: "The per-job checkbox list under 'Objective Details > Objective 1 > Jobs:' (lines 52-54) still shows [ ] for 01-01/01-02/01-03, even though the summary table above it (line 148) correctly shows 3/3 Complete and the top-level objective checkbox (line 32) shows [x]. Cosmetic staleness only — does not affect the deliverable."
  - kind: bookkeeping
    path: .planning/REQUIREMENTS.md
    note: "BUILD-01..04 checkboxes and the requirements-coverage table still show '[ ]' / 'Pending' despite all four being satisfied with evidence. Cosmetic staleness only."
---

# Objective 1: From-Source Build Pipeline Verification Report

**Objective Goal:** A developer can build `coolwsd` + `browser/dist` from EdenDocs source on Linux using Collabora's prebuilt engine/LOKit tarball (no from-scratch `engine/` compile), and every push is verified by CI on GitHub-hosted runners.
**Verified:** 2026-07-08T02:05:41Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Live `engine-main-assets.tar.gz` empirically re-confirmed to contain only `instdir/` — zero POCO headers, zero `workdir/` | ✓ VERIFIED | 01-01-SUMMARY.md: 5,881 entries, 100% under `instdir/`, 0 POCO/workdir matches, 0 `.h` files, `versionrc` present. Encoded permanently as a `grep` assertion in `build.sh`. |
| 2 | All four `scripts/eden/*.sh` are syntactically valid bash, executable, zero port-8080 references | ✓ VERIFIED | `bash -n` clean on all 4; all `-rwxr-xr-x`; only 8080 hit is the prohibition comment in `smoke-test.sh:4`. |
| 3 | `build.sh` hard-asserts the POCO fallback configure-log line | ✓ VERIFIED | `build.sh:51` — `grep -q "POCO not found in the engine workdir, falling back to system POCO" "$CONFIGURE_LOG"`. Fired and passed in both CI runs (below). |
| 4 | Every push to `eden-main` triggers `build.yml` on `ubuntu-latest`; latest run is green with full autogen/configure/make via `scripts/eden/` | ✓ VERIFIED | `build.yml`: `on.push.branches: [eden-main]`, `runs-on: ubuntu-latest`, steps call all 4 `scripts/eden/*.sh` verbatim. Latest push run (28910753718) and latest overall run (28911376952, workflow_dispatch, same head SHA) both `conclusion: success`. |
| 5 | Green run's logs contain the POCO-fallback line and `SMOKE TEST PASSED` | ✓ VERIFIED | Confirmed via `gh run view --log` on both runs (see Functional Verification table). |
| 6 | Run used Node 20 (guard passed) and ccache (stats printed, cache persisted via `actions/cache@v4`) | ✓ VERIFIED | Both runs: `node: v20.20.2`, `Verify Node 20` step passed. Warm run ccache: 336/336 hits (100%). Cold run: 129/336 partial hits (cross-run cache restore working as designed). |
| 7 | A developer reading `BUILDING-EdenDocs.md` can run the documented build end-to-end without another document | ✓ VERIFIED | Doc's Quick Start (Section 2) lists the exact 4-script sequence; deps/configure flags/setcap/9980 all spot-checked below and match scripts verbatim. |
| 8 | Doc explains WHY POCO auto-discovery fails and documents system-POCO fallback incl. `apt-libpoco-too-old` and obsolete `--with-poco-*` pitfalls | ✓ VERIFIED | `BUILDING-EdenDocs.md` Troubleshooting/POCO sections cover both pitfalls explicitly (verified via grep + read). |
| 9 | Every command in the doc matches the CI-proven `scripts/eden/` path exactly | ✓ VERIFIED | Cross-checked `--omit=` list, `--with-lokit-path`/`--with-lo-path`, `--enable-debug`, `LDFLAGS`, setcap commands, port 9980 curls — all byte-identical to the scripts. |

**Score:** 9/9 truths verified (rolled up to 4/4 requirement-level must-haves: BUILD-01, BUILD-02, BUILD-03, BUILD-04)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `scripts/eden/fetch-engine-assets.sh` | Fetch prebuilt tarball into `engine/`, sha256 logged | ✓ VERIFIED | Executable, downloads `for-code-assets/engine-main-assets.tar.gz`, `sha256sum` logged (not pinned, documented why), `test -f engine/instdir/program/versionrc` sanity check. |
| `scripts/eden/build-deps.sh` | apt deps + system POCO 1.12.5p2 | ✓ VERIFIED | Executable, apt list incl. `ccache`, idempotent source-build of POCO 1.12.5p2 with Zip module included (fix for CI run 28906749089), guarded re-run. |
| `scripts/eden/build.sh` | autogen/configure/make + BUILD-02 assertion | ✓ VERIFIED | Executable, wires ccache CC/CXX, `--with-lokit-path`/`--with-lo-path`/`--enable-debug`/`LDFLAGS=-L/usr/local/lib`, hard-asserts POCO fallback line, builds `browser/dist` explicitly, `test -x ./coolwsd` / `test -d ./browser/dist` final assertions. |
| `scripts/eden/smoke-test.sh` | BUILD-04 live smoke test on 9980 | ✓ VERIFIED | Executable, setcap grants, starts coolwsd, probes readiness up to 120s, curls `/hosting/discovery` + `/hosting/capabilities` on `127.0.0.1:9980`, fails loud with log tail on failure. |
| `.github/workflows/build.yml` | Push-triggered CI, Node 20, ccache | ✓ VERIFIED | `on.push.branches:[eden-main]`, `ubuntu-latest`, `actions/setup-node@v4` pinned `'20'` + guard, `actions/cache@v4` on `/home/runner/.ccache`, calls all 4 scripts in order, `ccache -s` step. |
| `BUILDING-EdenDocs.md` | Developer build doc | ✓ VERIFIED | 17.8KB, root-level, additive (does not edit README.md/CONTRIBUTING.md/docker/from-source/README.md), matches scripts + CI verbatim. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `fetch-engine-assets.sh` | `for-code-assets/engine-main-assets.tar.gz` | `wget` + `tar xzf` into `engine/`, sha256 logged | ✓ WIRED | Confirmed in script body and live CI logs. |
| `build.sh` | configure log system-POCO fallback notice | `grep -q` hard assertion | ✓ WIRED | Fired and passed in both CI runs. |
| `smoke-test.sh` | `http://127.0.0.1:9980/hosting/{discovery,capabilities}` | `curl -fsS \| grep -q` | ✓ WIRED | Both endpoints confirmed served; CI logged `SMOKE TEST PASSED`. Never 8080. |
| `build.yml` | `scripts/eden/{build-deps,fetch-engine-assets,build,smoke-test}.sh` | `run:` steps, verbatim order | ✓ WIRED | All 4 invoked in correct dependency order (deps → fetch → build → smoke). |
| `actions/setup-node@v4` | `node-version: '20'` | workflow step + `node -v \| grep '^v20\.'` guard | ✓ WIRED | Guard passed both runs; `v20.20.2` observed. |
| `actions/cache@v4` | `/home/runner/.ccache` | job-level `CCACHE_DIR` env + cache key on `configure.ac` hash | ✓ WIRED | Warm run shows 336/336 (100%) ccache hits — cache round-trip proven functional, not just configured. |
| `BUILDING-EdenDocs.md` | `scripts/eden/*.sh` | doc references scripts by exact path/flags | ✓ WIRED | Quick Start + Section 3/7 content byte-matches script internals. |
| `BUILDING-EdenDocs.md` | `.github/workflows/build.yml` | doc states CI proves the documented path green | ✓ WIRED | 6 references to `build.yml` across the doc, including explicit "runs this exact script path" statement. |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|--------------|--------|----------|
| BUILD-01 | 01-01, 01-02 | Build coolwsd + browser/dist from EdenDocs source via prebuilt `ENGINE_ASSETS` tarball, no from-scratch `engine/` compile | ✓ SATISFIED | `fetch-engine-assets.sh` never compiles the engine (tarball extract only); `build.sh` asserts `./coolwsd` and `./browser/dist` exist post-build; CI green end-to-end. |
| BUILD-02 | 01-01, 01-02, 01-03 | Build validates POCO auto-discovery early; documented fallback if it fails | ✓ SATISFIED | Empirical tarball inspection (01-01-SUMMARY.md) + permanent `grep` assertion in `build.sh` + CI log line present on both runs + `BUILDING-EdenDocs.md` documents WHY/pitfalls. |
| BUILD-03 | 01-02 | CI builds eden-main on GitHub-hosted ubuntu runners on every push, Node 20 LTS + ccache | ✓ SATISFIED | `build.yml` push-triggered on `eden-main`, `ubuntu-latest`, Node 20 pinned + guarded, ccache wired + cache-persisted (100% hit warm run). |
| BUILD-04 | 01-01, 01-02 | Built coolwsd passes smoke test on native port 9980, never 8080 | ✓ SATISFIED | `smoke-test.sh` curls both endpoints on 9980; `SMOKE TEST PASSED` in both CI runs; zero non-comment 8080 references anywhere in the objective's files. |

No orphaned requirements — REQUIREMENTS.md maps only BUILD-01..04 to Objective 1, and all four appear in TRD `requirements:` frontmatter across 01-01/01-02/01-03.

### Anti-Patterns Found

None. Scanned `scripts/eden/*.sh`, `.github/workflows/build.yml`, and `BUILDING-EdenDocs.md` for `TODO|FIXME|XXX|HACK|PLACEHOLDER` and `placeholder|coming soon|will be here` — zero matches.

### Functional Verification (CI logs, no live server started per HARD RULE)

| Check | Run | Result | Evidence |
|-------|-----|--------|----------|
| Latest push-triggered run on `eden-main` | 28910753718 (cold, push) | ✓ PASS | `conclusion: success`, head SHA `b6b03c2e1b2...` |
| Latest overall run, same head SHA | 28911376952 (warm, workflow_dispatch) | ✓ PASS | `conclusion: success` |
| POCO fallback line (cold) | 28910753718 | ✓ PRESENT | `configure: POCO not found in the engine workdir, falling back to system POCO` |
| POCO fallback line (warm) | 28911376952 | ✓ PRESENT | same line, step "Configure and build (asserts system-POCO fallback — BUILD-02)" |
| `SMOKE TEST PASSED` (cold) | 28910753718 | ✓ PRESENT | `SMOKE TEST PASSED: coolwsd serves /hosting/discovery and /hosting/capabilities on 9980` |
| `SMOKE TEST PASSED` (warm) | 28911376952 | ✓ PRESENT | same line |
| Node 20 guard (cold) | 28910753718 | ✓ PRESENT | `node: v20.20.2` |
| Node 20 guard (warm) | 28911376952 | ✓ PRESENT | `node: v20.20.2`, `Verify Node 20` step output `v20.20.2` |
| ccache stats (warm) | 28911376952 | ✓ PRESENT | `Hits: 336 / 336 (100.0%)` — full cache reuse proven |
| ccache stats (cold) | 28910753718 | ✓ PRESENT | `Hits: 129 / 336 (38.39%)` — partial cross-run carryover, consistent with a "cold" run restoring from `restore-keys` prefix |

_No browser/Maestro functional pass performed — this objective is CLI/CI build tooling, not a UI surface. Per HARD RULE, no server was started for this verification; all evidence is sourced from the recorded, immutable CI run logs via `gh run view --log`._

### Human Verification Required

None. Every truth is machine-verifiable via static file inspection or immutable CI log evidence, and all checks passed.

### Gaps Summary

No gaps. All 4 requirements (BUILD-01..04) are satisfied with direct, reproducible evidence: the four `scripts/eden/` scripts exist, are executable, and are free of stub/anti-patterns; `build.yml` is push-triggered on `eden-main`, pins Node 20, wires ccache, and calls the scripts verbatim; the two most recent CI runs (cold run 28910753718 and warm run 28911376952, same head SHA) are both green and their logs contain the required POCO-fallback and smoke-test evidence lines; `BUILDING-EdenDocs.md` is present, additive, and byte-matches the scripts/CI for every spot-checked command (`--omit=` list, configure flags, setcap step, port 9980); and `git log --name-only` confirms every commit in this objective touched only `scripts/eden/`, `.github/workflows/build.yml`, `BUILDING-EdenDocs.md`, or `.planning/` — no upstream file was modified.

Two purely cosmetic bookkeeping staleness items were found (ROADMAP.md's per-job `Jobs:` checkbox list and REQUIREMENTS.md's `[ ]`/"Pending" markers for BUILD-01..04 were not flipped even though the objective-level and summary-table rollups were updated correctly). These do not affect the deliverable and are recorded as `notes:` in the frontmatter, not gaps.

---

*Verified: 2026-07-08T02:05:41Z*
*Verifier: Claude (verifier)*
