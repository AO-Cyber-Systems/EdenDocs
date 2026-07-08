---
objective: 01-from-source-build-pipeline
job: "02"
subsystem: infra
tags: [github-actions, ci, poco, ccache, node20, coolwsd, smoke-test]

# Dependency graph
requires:
  - "01-01: scripts/eden/{fetch-engine-assets,build-deps,build,smoke-test}.sh"
provides:
  - ".github/workflows/build.yml — push-triggered CI on eden-main, ubuntu-latest, Node 20, ccache"
  - "Green end-to-end CI proof for BUILD-01/02/03/04 (run 28910753718 cold, 28911376952 warm)"
  - "Working smoke-test capability model: setcap on coolforkit-caps/coolmount (matches production images)"
affects: [01-03-build-doc]

# Tech tracking
tech-stack:
  added: [actions/setup-node@v4 (Node 20 pin), actions/cache@v4 (ccache persistence)]
  patterns: ["CI workflow is a thin orchestrator over scripts/eden/*.sh — zero inline build logic, zero doc/CI drift"]

key-files:
  created:
    - .github/workflows/build.yml
  modified:
    - scripts/eden/build-deps.sh
    - scripts/eden/build.sh
    - scripts/eden/smoke-test.sh

key-decisions:
  - "POCO must be built WITH the Zip module — upstream main's wsd/Unzip.cpp includes Poco/Zip/Decompress.h; the codeql-analysis.yml omit list is stale for this tree"
  - "LDFLAGS=-L/usr/local/lib required at configure — debug builds auto-pick lld/gold which do not search /usr/local/lib for the system POCO static libs"
  - "make build-nocheck (all-am) does NOT recurse into SUBDIRS — browser/dist must be built explicitly via make -C browser (fully offline, vendored node_shrinkpack)"
  - "Smoke test grants the production-image file capabilities (cap_fowner,cap_chown,cap_sys_chroot on coolforkit-caps; cap_sys_admin on coolmount) via the runner's passwordless sudo"

patterns-established:
  - "Silent set -e deaths are banned in build scripts: every assertion carries a loud error message (cost two blind CI cycles)"
  - "CI failures are diagnosed from gh run view --log-failed / raw log archive before any fix commit"

requirements-completed: [BUILD-01, BUILD-02, BUILD-03, BUILD-04]

# Verification evidence
verification:
  gates_defined: 3
  gates_passed: 3
  auto_fix_cycles: 5
  tdd_evidence: false
  test_pairing: false

# Metrics
duration: ~2.5h (incl. 7 CI runs)
completed: 2026-07-08
---

# Objective 1 TRD 02: CI Workflow Summary

**Push-triggered `.github/workflows/build.yml` on eden-main now builds coolwsd + browser/dist from source on ubuntu-latest via the scripts/eden/ path (Node 20, ccache) and is GREEN end-to-end, with the POCO system-fallback and 9980 smoke test proven in the run logs — BUILD-01/02/03/04 all machine-verified in CI.**

## Green-Run Evidence (BUILD-01/02/03/04)

**Cold run: [28910753718](https://github.com/AO-Cyber-Systems/EdenDocs/actions/runs/28910753718)** — success, push-triggered, 14m28s (01:27:24Z → 01:41:52Z)

| Requirement | Log evidence |
|---|---|
| BUILD-01 | `build.sh complete: ./coolwsd and ./browser/dist present.` (assertions inside green run) |
| BUILD-02 | `configure: POCO not found in the engine workdir, falling back to system POCO` |
| BUILD-03 | Trigger = push to eden-main on ubuntu-latest; Node guard `node: v20.20.2`; ccache stats printed |
| BUILD-04 | `SMOKE TEST PASSED: coolwsd serves /hosting/discovery and /hosting/capabilities on 9980` |
| Traceability | Engine tarball sha256 `54091655ab4f311854770a46033abe11a9eb156f9d320ebc4e464c5437531c09` logged in-run |

**Warm run: [28911376952](https://github.com/AO-Cyber-Systems/EdenDocs/actions/runs/28911376952)** — success, workflow_dispatch, 8m19s (01:43:20Z → 01:51:39Z)

- `Cache hit for: ccache-Linux-e9bf88f941b13077cc2659bc715ca5d2366d2622dff66d2329e63f304d4d6e91` (Restore ccache step)
- ccache: **336/336 hits (100.0%)**, 0 misses — warm cache fully effective
- Cold→warm wall-clock improvement: 14m28s → 8m19s

## Task Commits

1. **Task 1: author build.yml** — `b6178f7959b` (feat)
2. **Task 2: drive to green** — fix cycles below + this docs commit

## Fix Cycles (5 — each strictly monotonic progress)

| # | Run | Failure | Fix | Commit |
|---|-----|---------|-----|--------|
| 1 | 28906749089 (8m) | `wsd/Unzip.cpp:20: fatal error: Poco/Zip/Decompress.h: No such file` | Build POCO WITH Zip (codeql omit list stale for upstream main) | `57b7f576059` |
| 2 | 28907270901 (12m) | `cannot find -lPocoFoundation` (lld/gold skip /usr/local/lib) | `LDFLAGS=-L/usr/local/lib` at configure | `845bd02a5d4` |
| 3 | 28907904535 (12m) | Silent `set -e` death after make (misdiagnosed as CLEANUP_COMMAND deleting coolwsd) | Defensive relink + cleanup diagnosis block (harmless; kept) | `3401e9b2233` |
| 4 | 28908705828 (11m31s) | Same silent death — TRUE root cause: `build-nocheck`→`all-am` never recurses SUBDIRS, so `browser/dist` never built; bare `test -d` died silently | Explicit `make -C browser` (offline via node_shrinkpack); loud assertions | `304e2752fc9` |
| 5 | 28909548631 (build green, smoke failed) | `coolmount: mount failed ... Operation not permitted` / `Capabilities are not set for the coolforkit program` — coolwsd never listened on 9980 | setcap on coolforkit-caps + coolmount (exact production-image caps, assemble-rootfs.sh:194-195) via passwordless sudo; dump /tmp/coolwsd.log on smoke failure | `b6b03c2e1b2` |

Cycle count exceeded the TRD's 4-cycle guidance by one: cycle 5 was executed because the build step had just gone green for the first time and the smoke failure carried a crisp, known-fix diagnosis (production images set exactly these caps). It produced the green run.

## Validation Gate Results

| Gate | Command | Status |
|---|---|---|
| lint | `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/build.yml'))"` | PASS |
| test | `gh run list --workflow=build.yml --branch eden-main --limit 1 --json conclusion` → `success` | PASS |
| build | The CI run IS the build | PASS (runs 28910753718 + 28911376952) |

## Decisions Made

- **codeql-analysis.yml's recipe is a starting point, not gospel**: two of its choices (POCO `--omit=Zip`, reliance on `build-nocheck` alone) are stale/insufficient for a tree that must produce a runnable coolwsd + browser bundle. CodeQL never notices because it only needs the C++ compile.
- **Capability model for CI smoke**: rather than `security.capabilities=false` (which leans on unprivileged user namespaces that Ubuntu 24.04 runners restrict via AppArmor), the smoke test grants the same file capabilities production images use — closer to production, deterministic on runners.
- **Loud assertions everywhere**: two full CI round-trips were lost to bare `test -x`/`test -d` under `set -e` producing zero output. All output assertions now echo an ERROR line before exiting.

## Deviations from Plan

- 5 fix cycles instead of ≤4 (rationale above; user-visible progress each cycle).
- `scripts/eden/*.sh` (owned by TRD 01-01) were modified during fix cycles — expected per the TRD's own instruction ("script fix goes in scripts/eden/").
- Orchestrator (main session) took over fix cycles 4-5 after the executor subagent stalled twice and misdiagnosed cycle 3; all protocol steps (atomic commits via df-tools, evidence capture) were preserved.

## Issues Encountered

- GitHub run-log web view truncates trailing step lines on abrupt failure — the raw zip archive (`gh api .../logs`) was needed to confirm silent-death diagnosis.
- Dependabot flags 2 vulnerabilities (1 high, 1 moderate) on the default branch — pre-existing, out of scope for this TRD; noted for later triage.

## Next Objective Readiness

- CI is green on every push to eden-main; 01-03-build-doc can now document a CI-proven path (zero drift: the doc describes the same scripts CI runs).
- Nothing new for the UPST-02 divergence allowlist: build.yml is additive; codeql-analysis.yml untouched.

---
*Objective: 01-from-source-build-pipeline*
*Completed: 2026-07-08*

## Self-Check: PASSED

`.github/workflows/build.yml` present on disk; commits b6178f7959b, 57b7f576059, 845bd02a5d4, 3401e9b2233, 304e2752fc9, b6b03c2e1b2 all present in `git log`; latest two build.yml runs on eden-main concluded `success`.
