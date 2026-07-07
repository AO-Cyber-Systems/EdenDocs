---
objective: 01-from-source-build-pipeline
trd: "02"
type: standard
wave: 2
depends_on: ["01-01"]
files_modified:
  - .github/workflows/build.yml
autonomous: true
requirements: [BUILD-01, BUILD-02, BUILD-03, BUILD-04]

must_haves:
  truths:
    - "Every push to eden-main triggers .github/workflows/build.yml on ubuntu-latest, and the latest run is green: full autogen/configure/make build-nocheck via the scripts/eden/ path"
    - "The green run's logs contain 'POCO not found in the engine workdir, falling back to system POCO' (BUILD-02 proven end-to-end in CI) and 'SMOKE TEST PASSED' (BUILD-04: /hosting/discovery + /hosting/capabilities on 9980)"
    - "The run used Node 20 (node -v guard step passed) and ccache (ccache -s stats printed; cache persisted via actions/cache)"
  artifacts:
    - ".github/workflows/build.yml"
  key_links:
    - ".github/workflows/build.yml → scripts/eden/{build-deps,fetch-engine-assets,build,smoke-test}.sh (CI executes the same documented developer path — zero drift)"
    - "actions/setup-node@v4 → node-version '20' (ubuntu-latest no longer ships Node 20 in its toolcache as of 2026-05-26)"
    - "actions/cache@v4 → /home/runner/.ccache via job-level CCACHE_DIR env (root configure.ac has no Linux ccache auto-detect; build.sh wires CC/CXX)"
---

<objective>
Create a push-triggered GitHub Actions workflow (`.github/workflows/build.yml`) that builds EdenDocs on `ubuntu-latest` on every push to `eden-main` — Node 20 LTS pinned, ccache wired and cached — by invoking the Wave-1 `scripts/eden/` path verbatim, then push it and drive it to green. This single green run is the end-to-end proof of BUILD-01 (tarball fast-path build), BUILD-02 (system-POCO fallback fired and asserted), BUILD-03 (push-triggered CI with Node 20 + ccache), and BUILD-04 (live smoke test on 9980).

Purpose: The dev machine is macOS — GitHub Actions ubuntu-latest is the primary Linux proving ground for this fork. Modeled on the proven in-tree `codeql-analysis.yml` invocation, but as a NEW additive file (codeql-analysis.yml stays untouched: it is schedule-triggered by design and carries a deliberate registry-touching `npm update` step that must not leak into a build workflow).

Output: `.github/workflows/build.yml` committed and pushed, with the most recent run on eden-main green and its logs carrying the POCO-fallback and smoke-test evidence.
</objective>

<execution_context>
@~/.claude/devflow/workflows/execute-trd.md
@~/.claude/devflow/templates/summary.md
</execution_context>

<embedded_context>

<codebase_examples>
The only existing CI recipe, `.github/workflows/codeql-analysis.yml` — model, do not copy. Its trigger (schedule + dispatch, NOT push) and its `npm update` line are the two things that must NOT be carried over:

```yaml
on:
  schedule:
    - cron: '0 22 * * 3'
  workflow_dispatch:
# ...
    - if: matrix.language == 'cpp'
      name: Configure
      run: |
        wget --no-verbose https://github.com/CollaboraOnline/online/releases/download/for-code-assets/engine-main-assets.tar.gz
        tar xf engine-main-assets.tar.gz -C engine && rm engine-main-assets.tar.gz
        ./autogen.sh
        ./configure --enable-silent-rules --with-lokit-path=${GITHUB_WORKSPACE}/engine/include --with-lo-path=${GITHUB_WORKSPACE}/engine/instdir --enable-debug
        cd browser && npm update        # <-- CodeQL-only freshness step; NEVER copy this
```

All build logic already lives in Wave-1 scripts — the workflow is a thin orchestrator:
- `scripts/eden/build-deps.sh` — apt deps + source-built POCO 1.12.5p2 (+ ccache, autotools)
- `scripts/eden/fetch-engine-assets.sh` — tarball fetch/extract, sha256 logged
- `scripts/eden/build.sh` — autogen/configure/build; exports CC/CXX="ccache g++/gcc" when ccache present; hard-asserts the POCO fallback notice
- `scripts/eden/smoke-test.sh` — systemplate + coolwsd launch + --probe + curls on 9980; prints "SMOKE TEST PASSED"
</codebase_examples>

<anti_patterns>
- **No `npm update`, no node_modules caching, no registry mirror** — the vendored `browser/node_shrinkpack/` + `npm ci --offline` (Makefile.am:981) already make the browser build fully offline.
- **No unpinned Node** — ubuntu-latest defaults to Node 22.x and removed Node 20 from the toolcache entirely (2026-05-26). Relying on the runner default silently violates BUILD-03.
- **No hard-pinned tarball sha256** — `for-code-assets` is a rolling tag; fetch-engine-assets.sh logs the hash instead (research Pitfall 6).
- **No inline duplication of script logic in YAML** — the workflow must call the scripts, not re-implement them; duplication reintroduces the doc/CI drift this design eliminates.
- **NEVER reference port 8080** anywhere in the workflow — banned project-wide. coolwsd = 9980.
- **Do not edit `.github/workflows/codeql-analysis.yml`** — keep the new workflow purely additive for upstream mergeability.
</anti_patterns>

<error_recovery>
- **Configure fails "header Poco/Net/WebSocket.h not found"** → step order wrong: build-deps.sh (which installs POCO) must run before build.sh. Fix step order, push again.
- **autogen.sh fails on missing autotools** → research Open Question 2 resolved negative; build-deps.sh already installs autoconf/automake/libtool/pkg-config defensively, so this should not fire — if it does, check the apt step's log for install failures.
- **Smoke test times out (probe never ready)** → pull the coolwsd startup log from the run; common causes: missing systemplate (coolwsd-systemplate-setup failed → check engine/instdir extraction), or fatal config error (sys_template_path/child_root_path). The smoke script's stderr is in the step log.
- **ccache 0% hit rate on the second run** → cache not restored: check the "Restore ccache" step for a key miss, confirm job-level CCACHE_DIR matches the actions/cache path exactly (/home/runner/.ccache), and that `ccache -s` shows a nonzero cache size.
- **Disk pressure on the runner** (engine tarball ~434MB + extracted instdir + POCO build + objects): ubuntu-latest ships ~14GB free — expected fit; if exceeded, add a pre-step deleting unused preinstalled toolchains (e.g. /usr/local/lib/android) rather than shrinking the build.
- **Push rejected / auth failure on gh or git push** → surface as an auth gate; do not work around credentials.
</error_recovery>

</embedded_context>

<context>
@.planning/PROJECT.md
@.planning/ROADMAP.md
@.planning/STATE.md
@.planning/objectives/01-from-source-build-pipeline/1-RESEARCH.md
@.planning/objectives/01-from-source-build-pipeline/01-01-SUMMARY.md
@.github/workflows/codeql-analysis.yml
</context>

<research_context>
From 1-RESEARCH.md (HIGH confidence):
- Node 20 was removed from the ubuntu-latest toolcache entirely between 2026-05-19 and 2026-05-26 (Node 20 EOL 2026-04-30; default is now 22.x). `actions/setup-node@v4` with `node-version: '20'` is mandatory, and downloads on demand each run (acceptable cost).
- Root configure.ac has no Linux ccache auto-detection; build.sh already exports `CC="ccache gcc" CXX="ccache g++"` when ccache is on PATH — the workflow's jobs are (a) install ccache (build-deps.sh does it), (b) persist the cache dir across runs via actions/cache, (c) surface `ccache -s` stats.
- ccache ≥ 4.x defaults CCACHE_DIR to ~/.cache/ccache (XDG) — set an explicit job-level `CCACHE_DIR: /home/runner/.ccache` so the actions/cache path and ccache agree deterministically.
- Full build wall-clock on a 4-vCPU runner: expect roughly 30-60 min cold, less with a warm ccache. Set `timeout-minutes: 90`.
- The research doc's "Code Examples" section contains a full build.yml skeleton — use it as the starting shape, but route all run-steps through scripts/eden/*.sh instead of inline commands.
</research_context>

<gotchas>
- HARD RULE: never use port 8080 in the workflow, its comments, or any verification command. coolwsd = 9980; incidental local web servers = 8091.
- actions/cache restores at its step position and saves at job end — place the cache step BEFORE the build step; first run is a guaranteed miss (0% hit), which is normal.
- `hashFiles('configure.ac')` as the primary cache key + a `restore-keys: ccache-Linux-` prefix fallback gives warm partial hits after configure.ac changes.
- The workflow only fires from a branch that contains the file — the push that adds build.yml to eden-main is itself the first trigger.
- Trigger scope: `on.push.branches: [eden-main]` — every push to eden-main, per BUILD-03. Add `workflow_dispatch:` for manual re-runs (useful for the warm-cache verification).
</gotchas>

<tasks>

<task type="auto">
  <name>Task 1: Author .github/workflows/build.yml (push-triggered, Node 20, ccache, script-driven)</name>
  <files>.github/workflows/build.yml</files>
  <action>
Create `.github/workflows/build.yml` — new additive file. Structure:

```yaml
name: build
on:
  push:
    branches: [eden-main]
  workflow_dispatch:

concurrency:
  group: build-${{ github.ref }}
  cancel-in-progress: true

jobs:
  build:
    runs-on: ubuntu-latest
    timeout-minutes: 90
    env:
      CCACHE_DIR: /home/runner/.ccache
    steps:
      - uses: actions/checkout@v4

      # BUILD-03: Node 20 LTS pinned — ubuntu-latest no longer ships Node 20
      # in its toolcache (removed 2026-05-26; default is 22.x).
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
      - name: Verify Node 20
        run: node -v | grep -E '^v20\.'

      # BUILD-03: ccache persisted across runs. CCACHE_DIR is set job-wide so
      # ccache (XDG default would be ~/.cache/ccache) and this path agree.
      - name: Restore ccache
        uses: actions/cache@v4
        with:
          path: /home/runner/.ccache
          key: ccache-${{ runner.os }}-${{ hashFiles('configure.ac') }}
          restore-keys: |
            ccache-${{ runner.os }}-

      - name: Install build dependencies + system POCO 1.12.5p2
        run: ./scripts/eden/build-deps.sh

      - name: Configure ccache
        run: ccache -M 1G && ccache -z

      - name: Fetch prebuilt engine assets (BUILD-01 fast path, no engine/ compile)
        run: ./scripts/eden/fetch-engine-assets.sh

      - name: Configure and build (asserts system-POCO fallback — BUILD-02)
        run: ./scripts/eden/build.sh

      - name: ccache stats
        run: ccache -s

      # BUILD-04: native port 9980 — port 8080 is banned project-wide.
      - name: Smoke test /hosting/discovery + /hosting/capabilities on 9980
        run: ./scripts/eden/smoke-test.sh
```

Every build step is a one-line call into scripts/eden/ — the CI path IS the documented developer path. Do not add an `npm update` step, do not cache node_modules, do not touch codeql-analysis.yml.
  </action>
  <verify>
python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/build.yml'))" && grep -q "branches: \[eden-main\]" .github/workflows/build.yml && grep -q "node-version: '20'" .github/workflows/build.yml && grep -q 'CCACHE_DIR' .github/workflows/build.yml && grep -q 'actions/cache@v4' .github/workflows/build.yml && grep -q 'scripts/eden/build-deps.sh' .github/workflows/build.yml && grep -q 'scripts/eden/fetch-engine-assets.sh' .github/workflows/build.yml && grep -q 'scripts/eden/build.sh' .github/workflows/build.yml && grep -q 'scripts/eden/smoke-test.sh' .github/workflows/build.yml && ! grep -q 'npm update' .github/workflows/build.yml && test "$(grep -c '8080' .github/workflows/build.yml)" -le 1 && command -v actionlint >/dev/null && actionlint .github/workflows/build.yml || true
  </verify>
  <done>build.yml exists, parses as valid YAML (actionlint-clean if available), triggers on push to eden-main + workflow_dispatch, pins Node 20 with a runtime guard, wires CCACHE_DIR + actions/cache, calls all four scripts/eden/ scripts in dependency order, contains no npm update, and mentions 8080 at most once (the ban comment).</done>
  <recovery>YAML parse failure: fix indentation and re-verify. If actionlint flags expression errors, fix before pushing — a broken workflow file wastes a CI round trip.</recovery>
</task>

<task type="auto">
  <name>Task 2: Push to eden-main, drive the run to green, and capture BUILD-02/04 evidence from logs</name>
  <files>.github/workflows/build.yml</files>
  <action>
1. Commit build.yml (atomic commit) and push `eden-main` to origin — this push is itself the first trigger.
2. Watch the run: `gh run watch $(gh run list --workflow=build.yml --branch eden-main --limit 1 --json databaseId --jq '.[0].databaseId') --exit-status`.
3. If the run fails: pull the failing step's log (`gh run view <id> --log-failed`), diagnose against the error_recovery table, fix (script fix goes in scripts/eden/, workflow fix in build.yml), commit atomically, push, and watch again. Iterate until green. If more than 4 fix cycles are needed, stop and surface the failure pattern to the user instead of thrashing.
4. From the GREEN run, extract and record evidence for the SUMMARY:
   - `gh run view <id> --log | grep -F "POCO not found in the engine workdir, falling back to system POCO"` (BUILD-02 end-to-end)
   - `gh run view <id> --log | grep -F "SMOKE TEST PASSED"` (BUILD-04)
   - `gh run view <id> --log | grep -E "v20\."` (Node 20 guard)
   - the sha256 line logged by fetch-engine-assets.sh (tarball traceability)
   - ccache stats block from the "ccache stats" step
5. Warm-cache check: trigger one more run (`gh workflow run build.yml --ref eden-main`), let it complete, and confirm the "Restore ccache" step reports a cache hit and `ccache -s` shows a nonzero hit count. Record both runs' wall-clock times in the SUMMARY (cold vs warm).
  </action>
  <verify>
gh run list --workflow=build.yml --branch eden-main --limit 1 --json conclusion --jq '.[0].conclusion' | grep -x success && RUN_ID=$(gh run list --workflow=build.yml --branch eden-main --limit 1 --json databaseId --jq '.[0].databaseId') && gh run view "$RUN_ID" --log | grep -qF "POCO not found in the engine workdir, falling back to system POCO" && gh run view "$RUN_ID" --log | grep -qF "SMOKE TEST PASSED"
  </verify>
  <done>The most recent build.yml run on eden-main concluded success; its logs contain the POCO-fallback notice and SMOKE TEST PASSED; Node 20 guard passed; sha256 + ccache stats captured; a second (warm) run confirmed ccache restore with nonzero hits.</done>
  <recovery>Per-failure diagnosis via error_recovery table. On gh/git auth errors, raise an auth gate for the user — never embed or work around credentials. If GitHub Actions is disabled on the repo, surface to the user (Settings → Actions) as user_setup rather than attempting API workarounds. Hard stop after 4 failed fix cycles with a structured failure report.</recovery>
</task>

</tasks>

<validation_gates>
<lint>python3 -c "import yaml; yaml.safe_load(open('.github/workflows/build.yml'))"; command -v actionlint >/dev/null && actionlint .github/workflows/build.yml || true</lint>
<test>gh run list --workflow=build.yml --branch eden-main --limit 1 --json conclusion --jq '.[0].conclusion' | grep -x success</test>
<build>true # the CI run IS the build</build>
</validation_gates>

<verification>
1. Latest build.yml run on eden-main: conclusion=success, triggered by push (BUILD-03).
2. Run logs contain the POCO-fallback notice (BUILD-02) and SMOKE TEST PASSED with both 9980 curls (BUILD-04).
3. Run logs show `v20.` from the Node guard and a ccache stats block (BUILD-03); warm run shows cache restore + hits.
4. coolwsd binary and browser/dist asserted by build.sh inside the green run (BUILD-01).
5. `git log` on the workflow shows only additive commits; codeql-analysis.yml untouched.
</verification>

<success_criteria>
- Every push to eden-main now triggers a full configure+build+smoke CI run on ubuntu-latest, and the current one is green.
- BUILD-01/02/03/04 all have machine-checkable evidence in the green run's logs.
</success_criteria>

<output>
After completion, create `.planning/objectives/01-from-source-build-pipeline/01-02-SUMMARY.md` including: green run URL + id, cold vs warm wall-clock times, ccache hit stats, tarball sha256 from the run log, and any fix-cycle commits made along the way.
</output>
