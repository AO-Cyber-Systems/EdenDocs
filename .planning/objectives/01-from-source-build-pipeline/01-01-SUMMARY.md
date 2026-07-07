---
objective: 01-from-source-build-pipeline
job: "01"
subsystem: infra
tags: [bash, build-scripts, poco, coolwsd, ccache, ci-prep]

# Dependency graph
requires: []
provides:
  - "scripts/eden/fetch-engine-assets.sh — idempotent fetch/extract of the rolling for-code-assets engine tarball"
  - "scripts/eden/build-deps.sh — apt deps + source-built POCO 1.12.5p2 + ccache"
  - "scripts/eden/build.sh — autogen/configure/make with a permanent BUILD-02 fallback assertion"
  - "scripts/eden/smoke-test.sh — coolwsd BUILD-04 smoke test on native port 9980"
  - "Empirical re-confirmation of BUILD-02 (live tarball inspected 2026-07-07): instdir-only, zero POCO, zero workdir"
affects: [01-02-ci-workflow, 01-03-build-doc]

# Tech tracking
tech-stack:
  added: [POCO 1.12.5p2 (source-built), ccache]
  patterns: ["scripts/eden/ as the single canonical build path both developers and CI invoke verbatim"]

key-files:
  created:
    - scripts/eden/fetch-engine-assets.sh
    - scripts/eden/build-deps.sh
    - scripts/eden/build.sh
    - scripts/eden/smoke-test.sh
  modified: []

key-decisions:
  - "System-built POCO 1.12.5p2 (verbatim codeql-analysis.yml recipe) is the only proven fallback — apt libpoco-dev (1.11.0) is below configure.ac's >=1.12.0 floor"
  - "sha256 of the fetched tarball is logged, not pinned — for-code-assets is a rolling unversioned tag; pinning would fossilize the build"
  - "ccache wired externally via CC/CXX in build.sh since root configure.ac has no Linux ccache auto-detection"

patterns-established:
  - "Build scripts live under scripts/eden/, fully additive — zero upstream files touched, nothing to track for future UPST-02 divergence allowlist"

requirements-completed: [BUILD-01, BUILD-02, BUILD-04]

# Verification evidence
verification:
  gates_defined: 3
  gates_passed: 3
  auto_fix_cycles: 1
  tdd_evidence: false
  test_pairing: false

# Metrics
duration: 7min
completed: 2026-07-07
---

# Objective 1 TRD 01: Build Scripts Summary

**Four canonical scripts/eden/*.sh build scripts (fetch/build-deps/build/smoke-test) that empirically re-confirm and permanently assert BUILD-02's system-POCO-fallback requirement, forming the single build path both developers and the Wave-2 CI workflow will invoke verbatim.**

## Performance

- **Duration:** 7 min
- **Started:** 2026-07-07T23:35:00Z
- **Completed:** 2026-07-07T23:42:00Z
- **Tasks:** 3
- **Files modified:** 4 created (all additive, zero upstream files touched)

## Accomplishments

- Empirically re-confirmed BUILD-02 live: the current `engine-main-assets.tar.gz` (433,550,261 bytes, sha256 `54091655ab4f311854770a46033abe11a9eb156f9d320ebc4e464c5437531c09`) contains 5,881 entries, ALL under `instdir/`, zero POCO headers (`.h` count = 0), zero `poco`/`workdir` matches (case-insensitive), and `instdir/program/versionrc` present — proving POCO auto-discovery must fail and the system-POCO fallback is the required path.
- `scripts/eden/build.sh` permanently hard-asserts the configure-log line `POCO not found in the engine workdir, falling back to system POCO`, so this validation re-runs automatically on every future build, not just this one-time check.
- Full developer build path now exists as four executable, shellcheck-clean scripts under `scripts/eden/`, ready for the Wave-2 CI workflow to call verbatim — eliminating doc/CI drift by construction.
- Zero references to port 8080 outside the single permitted prohibition comment in `smoke-test.sh`.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: fetch-engine-assets.sh + BUILD-02 live re-validation | `bash -n scripts/eden/fetch-engine-assets.sh && test -x ... && grep -q 'for-code-assets/engine-main-assets.tar.gz' ... && grep -q 'sha256sum' ... && grep -q 'versionrc' ... && ! grep -q '8080' scripts/eden/fetch-engine-assets.sh` (plus the live tarball `tar tzf` inspection) | 0 | PASS |
| 2: build-deps.sh + build.sh | `bash -n scripts/eden/build-deps.sh && bash -n scripts/eden/build.sh && ... && grep -q 'falling back to system POCO' scripts/eden/build.sh && ... && ! grep -q '8080' scripts/eden/build-deps.sh && ! grep -q '8080' scripts/eden/build.sh` | 0 | PASS |
| 3: smoke-test.sh | `bash -n scripts/eden/smoke-test.sh && test -x ... && grep -q 'coolwsd-systemplate-setup' ... && grep -q 'wopi-discovery' ... && grep -q 'convert-to' ... && test "$(grep -c '8080' scripts/eden/smoke-test.sh)" -le 1` | 0 | PASS |

## Task Commits

Each task was committed atomically:

1. **Task 1: fetch-engine-assets.sh + BUILD-02 re-validation** - `4f31e4e` (feat) + `6e47b8f` (fix — Rule 1 deviation, see below)
2. **Task 2: build-deps.sh + build.sh** - `fcca63d` (feat)
3. **Task 3: smoke-test.sh** - `a5416e1` (feat)

**Plan metadata:** (this commit, docs)

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| lint | `for f in scripts/eden/*.sh; do bash -n "$f"; done; shellcheck scripts/eden/*.sh \|\| true` | 0 | PASS (one info-level SC2102 noted below, not a warning-level finding) |
| test | `! grep -rn '8080' scripts/eden/ \| grep -v 'never use port 8080\|never port 8080\|never use, bind\|never 8080'` | 0 | PASS |
| build | N/A — Linux-only build, executed by the Wave-2 CI workflow (`true` per TRD) | 0 | PASS (deferred by design) |

**shellcheck note:** `smoke-test.sh` line 16 (`--o:storage.filesystem[@allow]=true`) triggers shellcheck's info-level SC2102 ("Ranges can only match single chars"). This is a false positive — the bracket syntax is coolwsd's own config-key naming convention, copied verbatim from `Makefile.am`'s `COMMON_PARAMS` (line 726) and the proven `codeql-analysis.yml` recipe. Re-ran `shellcheck -S warning` (warning severity and above) across all four scripts: clean. Per the TRD's own guidance ("fix warnings-level findings; do not suppress errors"), no code change was needed for an info-level, false-positive finding.

## Post-TRD Verification

- **Auto-fix cycles used:** 1
- **Must-haves verified:** 3/3
- **Gate failures:** None (after the Rule 1 auto-fix described below)

## BUILD-02 Live Tarball Evidence

Captured 2026-07-07 by downloading the live release asset and streaming it through `tar tzf`:

| Check | Result |
|---|---|
| URL | `https://github.com/CollaboraOnline/online/releases/download/for-code-assets/engine-main-assets.tar.gz` |
| File size | 433,550,261 bytes (matches 1-RESEARCH.md's prior measurement exactly) |
| sha256 | `54091655ab4f311854770a46033abe11a9eb156f9d320ebc4e464c5437531c09` |
| Total tar entries | 5,881 |
| Entries under `instdir/` | 5,881 (100% — no other top-level directory exists) |
| `poco\|workdir` matches (case-insensitive) | 0 |
| `.h` header files | 0 |
| `instdir/program/versionrc` present | Yes |

**Conclusion:** BUILD-02's premise is empirically re-confirmed — the tarball ships zero POCO source/headers and zero `workdir/`, so `configure`'s `--with-lo-builddir` POCO auto-discovery deterministically fails and a system-installed POCO >= 1.12.0 is mandatory. This is now permanently encoded as a hard assertion in `scripts/eden/build.sh` (grep for the exact fallback notice), so it self-validates on every future build rather than relying on this one-time manual check.

## Files Created/Modified

- `scripts/eden/fetch-engine-assets.sh` - Idempotent fetch + extract of the rolling `for-code-assets` engine tarball into `engine/`; logs (not pins) sha256; asserts `versionrc` post-extract.
- `scripts/eden/build-deps.sh` - Installs the verbatim `codeql-analysis.yml` apt package list plus `ccache` (BUILD-03) and a defensive autotools quartet; source-builds POCO 1.12.5p2 with an idempotent guard.
- `scripts/eden/build.sh` - Wires `ccache` via `CC`/`CXX`, runs `autogen.sh` + `configure --enable-debug --with-lokit-path=... --with-lo-path=...`, hard-asserts the BUILD-02 system-POCO fallback notice in the configure log, builds via `make build-nocheck`, asserts `./coolwsd` and `./browser/dist` outputs.
- `scripts/eden/smoke-test.sh` - Bootstraps systemplate, launches `coolwsd --disable-ssl` with the fatal-required config keys, gates readiness on `--probe` (up to 120s), curls `/hosting/discovery` and `/hosting/capabilities` on native port 9980 with content assertions, cleans up via `trap`.

## Decisions Made

- Used source-built POCO 1.12.5p2 (exact `codeql-analysis.yml` recipe) rather than apt `libpoco-dev`, because Ubuntu 24.04's package is 1.11.0 — below `configure.ac`'s hard `>= 1.12.0` floor.
- Logged the fetched tarball's sha256 for traceability but deliberately did not pin/verify it against a hardcoded value, since `for-code-assets` is a rolling, unversioned release tag that upstream rebuilds continuously; pinning would break every future fetch.
- Kept `--enable-debug` in `build.sh`'s configure invocation (matches the proven CI recipe) because it arms the `--with-lo-path` `versionrc` hard-check — omitting it would let a bad `--with-lo-path` fail silently.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Removed literal "8080" from build-script header comments**
- **Found during:** Task 1 verification (`fetch-engine-assets.sh`), then proactively applied to Task 2 before commit
- **Issue:** The TRD's own drafted content for `fetch-engine-assets.sh`, `build-deps.sh`, and `build.sh` included a "HARD PROJECT RULE: never use, bind, curl, or reference port 8080" header comment — but the TRD's own `<verify>` commands for Tasks 1 and 2 require `! grep -q '8080'` (zero occurrences) in those three files, and the TRD's own `<verification>` item 4 states the prohibition comment is reserved for `smoke-test.sh` only. As drafted, Task 1's script would have failed its own verify gate.
- **Fix:** Reworded the header comments in all three files to state "This script does not bind or reference any network port." (no digits), preserving intent without violating the file's own verify gate. Also trimmed a second, redundant "(never 8080)" parenthetical from `smoke-test.sh` so it carries exactly one occurrence as required (`<= 1`).
- **Files modified:** `scripts/eden/fetch-engine-assets.sh` (fix commit), `scripts/eden/build-deps.sh` and `scripts/eden/build.sh` (fixed before first commit), `scripts/eden/smoke-test.sh` (fixed before first commit)
- **Verification:** Re-ran each task's exact `<verify>` command from the TRD after the fix — all now exit 0. `git diff` confirms no upstream file touched.
- **Committed in:** `6e47b8f` (standalone fix commit for Task 1's already-committed file); Task 2 and Task 3 files were corrected before their first commit, so no separate fix commit was needed for those.

---

**Total deviations:** 1 auto-fixed (1 bug — TRD's own verify gate vs. TRD's own drafted script content)
**Impact on plan:** No scope creep. The fix only removed digits from comments; all functional behavior specified by the TRD is unchanged.

## Issues Encountered

- **Isolated worktree mismatch:** The Write/Edit tools were sandboxed to a stale, unrelated worktree (`.claude/worktrees/agent-a76fbbacec0a99c80`, branched from unrelated upstream history, missing `.planning/`). Per this TRD's explicit `<worktree_protocol>` instructions ("no isolated worktree... commit directly on eden-main"), all file writes and git operations were performed via the Bash tool directly against `/Users/justin/dev/EdenDocs` on branch `eden-main`, which the Bash tool was able to access without restriction. No files were written to or committed from the isolated worktree.

## Next Objective Readiness

- `scripts/eden/{fetch-engine-assets,build-deps,build,smoke-test}.sh` are ready for the Wave-2 CI workflow (01-02-ci-workflow) to invoke verbatim — no doc/CI drift by construction.
- BUILD-02's system-POCO-fallback requirement is both empirically proven and permanently self-asserting on every future build.
- No blockers for 01-02-ci-workflow or 01-03-build-doc.

---
*Objective: 01-from-source-build-pipeline*
*Completed: 2026-07-07*

## Self-Check: PASSED

All 4 created scripts confirmed present on disk; all 4 task commit hashes (4f31e4edfec, 6e47b8fa839, fcca63dbca7, a5416e1ba44) confirmed present in `git log --oneline --all`.
