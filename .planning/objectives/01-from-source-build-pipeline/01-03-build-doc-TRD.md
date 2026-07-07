---
objective: 01-from-source-build-pipeline
trd: "03"
type: standard
wave: 3
depends_on: ["01-02"]
files_modified:
  - BUILDING-EdenDocs.md
autonomous: true
requirements: [BUILD-01, BUILD-02]

must_haves:
  truths:
    - "A developer reading BUILDING-EdenDocs.md can run the documented Linux build end-to-end (deps → POCO → tarball fetch → configure/build → smoke test) without consulting any other document"
    - "The doc explains WHY POCO auto-discovery deterministically fails against the prebuilt tarball and documents the system-POCO fallback as the required path, including the apt-libpoco-too-old and obsolete --with-poco-* pitfalls (BUILD-02 'documented fallback')"
    - "Every command in the doc matches the CI-proven scripts/eden/ path exactly — the doc, the scripts, and build.yml describe one and the same build"
  artifacts:
    - "BUILDING-EdenDocs.md"
  key_links:
    - "BUILDING-EdenDocs.md → scripts/eden/*.sh (scripts are the canonical commands; doc references them by path)"
    - "BUILDING-EdenDocs.md → .github/workflows/build.yml (CI is the continuous proof of the documented path)"
---

<objective>
Write `BUILDING-EdenDocs.md` (repo root, new additive file) — the developer build documentation that BUILD-01's "documented Linux build" and BUILD-02's "documented fallback" require, describing exactly the path the Wave-2 CI run proved green.

Purpose: Sequenced after the green CI run deliberately: the doc codifies a proven path (with actual timings, actual log lines, the actual sha256 mechanism), not a theoretical one. It is additive — upstream's README.md, CONTRIBUTING.md, and docker/from-source/README.md are not edited, preserving mergeability.

Output: BUILDING-EdenDocs.md at the repo root, cross-verified for consistency against scripts/eden/ and build.yml.
</objective>

<execution_context>
@~/.claude/devflow/workflows/execute-trd.md
@~/.claude/devflow/templates/summary.md
</execution_context>

<embedded_context>

<codebase_examples>
Existing repo-root docs are upstream files (README.md, CONTRIBUTING.md, SECURITY.md) — do not edit them. The new doc follows their plain-Markdown style. In-tree anchors the doc should cite by path (all verified):
- `scripts/eden/*.sh` — canonical build commands (Wave 1)
- `.github/workflows/build.yml` — CI proof (Wave 2)
- `docker/from-source/build.sh` — upstream's ENGINE_ASSETS reference implementation
- `dev-notes/poco-build.md` — upstream's own writeup of the POCO-in-engine migration (background for the fallback rationale)
- `Makefile.am` `run:`/`setup-wsd` targets — the upstream dev-run recipe the smoke test is modeled on
</codebase_examples>

<anti_patterns>
- **No command drift**: never paste a configure invocation or POCO recipe into the doc that differs from scripts/eden/ — quote the scripts or reference them by path. Duplicated-but-divergent commands are how build docs rot.
- **No editing upstream files**: no README.md pointer edit, no docker/from-source/README.md notes. Purely additive (UPST-02 discipline).
- **No hard-pinned tarball checksum in the doc** — document the rolling-tag reality and the sha256-logging traceability approach instead (research Pitfall 6).
- **NEVER present port 8080 as usable** — it appears in the doc only inside the explicit prohibition.
</anti_patterns>

<error_recovery>
- If cross-verification (Task 2) finds a doc/script mismatch, fix the DOC to match the scripts (the scripts are CI-proven); only fix a script if it is genuinely wrong, and then re-run the CI workflow to re-prove it before finishing this TRD.
</error_recovery>

</embedded_context>

<context>
@.planning/PROJECT.md
@.planning/ROADMAP.md
@.planning/STATE.md
@.planning/objectives/01-from-source-build-pipeline/1-RESEARCH.md
@.planning/objectives/01-from-source-build-pipeline/01-01-SUMMARY.md
@.planning/objectives/01-from-source-build-pipeline/01-02-SUMMARY.md
</context>

<research_context>
Content the doc must carry, all HIGH-confidence from 1-RESEARCH.md and proven by the Wave-2 CI run:
- Fast path: prebuilt `engine-main-assets.tar.gz` (rolling `for-code-assets` release tag, ~434MB) extracted into `engine/` → configure with `--with-lokit-path=$PWD/engine/include --with-lo-path=$PWD/engine/instdir --enable-silent-rules --enable-debug` → `make build-nocheck`. `engine/include` is git-tracked source; the tarball supplies only compiled `instdir/`. From-scratch engine builds (1-3h, ~30GB) are explicitly out of scope.
- POCO strategy (BUILD-02): the tarball ships zero `workdir/`, so `--with-lo-builddir` auto-discovery deterministically fails and configure falls back to system POCO — the log line "POCO not found in the engine workdir, falling back to system POCO" is SUCCESS. System POCO must be ≥ 1.12.0 (hard floor, configure.ac:2231-2239); Ubuntu 24.04's libpoco-dev (1.11.0) is too old; `--with-poco-includes/--with-poco-libs` are obsolete no-ops. The required path: build POCO 1.12.5p2 from source (scripts/eden/build-deps.sh) before configure.
- Node 20 LTS required explicitly (configure floor is >=20; CI pins via actions/setup-node; locally use nvm/n — apt's npm brings an older nodejs, don't rely on it).
- ccache: root configure.ac has no Linux auto-detect; build.sh exports CC/CXX="ccache gcc/g++" when ccache is present.
- browser/ deps are fully vendored (browser/node_shrinkpack, 866 packages); `make` runs `npm ci --offline` itself — no registry access, no extra steps.
- Smoke test: `scripts/eden/smoke-test.sh` — systemplate via `coolwsd-systemplate-setup`, coolwsd with `--disable-ssl` + fatal-required paths, readiness via built-in `--probe`, then curls of `/hosting/discovery` and `/hosting/capabilities` on native port 9980.
</research_context>

<gotchas>
- HARD RULE: port 8080 is banned project-wide (permanently occupied on the primary dev machine). The doc must state this prohibition explicitly: coolwsd uses its native 9980; any incidental local web server uses 8091.
- macOS is dev-convenience only — the doc's build instructions target Linux (Ubuntu 24.04 or a container thereof); say so up front and point macOS users at Docker or CI.
- Pull real numbers (cold/warm CI wall-clock, tarball size) from 01-02-SUMMARY.md rather than inventing them.
</gotchas>

<tasks>

<task type="auto">
  <name>Task 1: Write BUILDING-EdenDocs.md — tarball fast path, POCO fallback rationale, smoke test</name>
  <files>BUILDING-EdenDocs.md</files>
  <action>
Create `BUILDING-EdenDocs.md` at the repo root with these sections:

1. **Overview** — what gets built (coolwsd + browser/dist), the fast-path philosophy (consume Collabora's prebuilt engine tarball; never compile `engine/` from scratch — out of scope, 1-3h/~30GB), target platform (Linux / Ubuntu 24.04; macOS users: use a container or rely on CI), and the port rule: *never use port 8080 for anything — coolwsd's native port is 9980; incidental local web servers use 8091*.
2. **Quick start** — the four canonical commands, in order, presented as THE build path:
   ```bash
   ./scripts/eden/build-deps.sh           # apt deps + POCO 1.12.5p2 from source (sudo)
   ./scripts/eden/fetch-engine-assets.sh  # prebuilt engine tarball → engine/ (sha256 logged)
   ./scripts/eden/build.sh                # autogen + configure + make build-nocheck
   ./scripts/eden/smoke-test.sh           # coolwsd serves /hosting/* on 9980
   ```
   Note Node 20 LTS as a prerequisite (nvm/n locally; configure hard-requires >=20; CI pins '20').
3. **The engine tarball fast path (BUILD-01)** — what `engine-main-assets.tar.gz` is (rolling `for-code-assets` release tag, instdir-only, ~434MB), why `engine/include` still resolves (git-tracked headers), the exact configure flags used and why `--enable-debug` matters (arms the lo-path versionrc sanity check), and the checksum stance (rolling tag, no published hash → sha256 logged per fetch for traceability, deliberately not pinned).
4. **POCO strategy and fallback (BUILD-02)** — the heart of the doc. State plainly: against the prebuilt tarball, in-tree POCO auto-discovery (`--with-lo-builddir` → `workdir/UnpackedTarball/poco`) **always fails, by construction** — the tarball is a packaging of `make instdir` and ships no `workdir/`. This was empirically verified (2026-07-07, live tarball inspection) and is asserted on every build by `scripts/eden/build.sh`. The configure line "POCO not found in the engine workdir, falling back to system POCO" is expected SUCCESS output. The required fallback: source-built POCO 1.12.5p2 installed to /usr/local (exact recipe in build-deps.sh, lifted verbatim from the proven codeql-analysis.yml). Document both pitfalls: apt `libpoco-dev` is 1.11.0 < the 1.12.0 configure floor (hard error), and `--with-poco-includes/--with-poco-libs` are accepted-but-ignored no-ops. Cite `dev-notes/poco-build.md` for the upstream migration background.
5. **ccache** — root configure.ac has no Linux ccache auto-detection; build.sh wires `CC="ccache gcc" CXX="ccache g++"` automatically when ccache is installed; CI persists /home/runner/.ccache across runs.
6. **Browser build** — fully offline by design (`browser/node_shrinkpack/` vendored packages; `make` runs `npm ci --offline`); explicitly: no `npm install`/`npm update` needed or wanted.
7. **Smoke test (BUILD-04)** — what smoke-test.sh does (systemplate bootstrap, coolwsd launch flags, `--probe` readiness, the two `/hosting/` curls on **9980**) and expected output ("SMOKE TEST PASSED...").
8. **CI (BUILD-03)** — `.github/workflows/build.yml` runs this exact script path on every push to eden-main (ubuntu-latest, Node 20 pinned, ccache cached); include the real cold/warm run times from 01-02-SUMMARY.md; note codeql-analysis.yml is a separate, schedule-only security-scanning workflow — not the build.
9. **Troubleshooting** — table mapping the known failure signatures to causes/fixes: "header Poco/Net/WebSocket.h not found" → POCO not installed before configure; "POCO version is too old" → apt libpoco-dev was used; fetch 404 → asset naming varies per branch (`engine-main-` vs `core-co-25.04-`); smoke-test timeout → check systemplate/extraction; Node < 20 → configure floor.
  </action>
  <verify>
test -f BUILDING-EdenDocs.md && grep -q 'engine-main-assets.tar.gz' BUILDING-EdenDocs.md && grep -q 'falling back to system POCO' BUILDING-EdenDocs.md && grep -q '1.12.5p2' BUILDING-EdenDocs.md && grep -qi 'libpoco-dev' BUILDING-EdenDocs.md && grep -q -- '--with-poco' BUILDING-EdenDocs.md && grep -q 'scripts/eden/build-deps.sh' BUILDING-EdenDocs.md && grep -q 'scripts/eden/fetch-engine-assets.sh' BUILDING-EdenDocs.md && grep -q 'scripts/eden/build.sh' BUILDING-EdenDocs.md && grep -q 'scripts/eden/smoke-test.sh' BUILDING-EdenDocs.md && grep -q '9980' BUILDING-EdenDocs.md && grep -qi 'never use port 8080' BUILDING-EdenDocs.md && grep -q 'Node 20' BUILDING-EdenDocs.md && grep -q 'ccache' BUILDING-EdenDocs.md && grep -q 'build.yml' BUILDING-EdenDocs.md
  </verify>
  <done>BUILDING-EdenDocs.md exists at the repo root covering: quick start via the four scripts, tarball fast path + checksum stance, the full POCO deterministic-failure + fallback rationale with both pitfalls, ccache wiring, offline browser build, 9980 smoke test, CI section with real timings, troubleshooting table, and the explicit 8080 prohibition.</done>
  <recovery>If 01-02-SUMMARY.md lacks timing data, use the run durations from `gh run list --workflow=build.yml --json displayTitle,updatedAt,createdAt` instead; do not invent numbers.</recovery>
</task>

<task type="auto">
  <name>Task 2: Cross-verify doc ↔ scripts ↔ workflow consistency (zero drift)</name>
  <files>BUILDING-EdenDocs.md</files>
  <action>
Mechanically cross-check the three surfaces describe one build:
1. Every `scripts/eden/*.sh` path referenced in the doc exists on disk, and conversely all four scripts are referenced in the doc.
2. Every configure flag quoted in the doc appears verbatim in `scripts/eden/build.sh` (`--enable-silent-rules`, `--enable-debug`, `--with-lokit-path`, `--with-lo-path`) — and the doc quotes no flag the script doesn't use.
3. The POCO version and `--omit=` list quoted in the doc match `scripts/eden/build-deps.sh` exactly.
4. The workflow steps in `.github/workflows/build.yml` reference the same four scripts, and the doc's CI section names the right workflow file and trigger (push to eden-main).
5. Port audit: `grep -rn '8080' BUILDING-EdenDocs.md scripts/eden/ .github/workflows/build.yml` — every match must be inside an explicit prohibition statement, none in a runnable command.
Fix the doc for any mismatch (scripts are CI-proven ground truth). Record the cross-check results in the SUMMARY.
  </action>
  <verify>
for f in $(grep -o 'scripts/eden/[a-z-]*\.sh' BUILDING-EdenDocs.md | sort -u); do test -f "$f" || exit 1; done && for s in build-deps fetch-engine-assets build smoke-test; do grep -q "scripts/eden/$s.sh" BUILDING-EdenDocs.md || exit 1; done && grep -o -- '--omit=[^ "]*' BUILDING-EdenDocs.md | head -1 | grep -qxF "$(grep -o -- '--omit=[^ "]*' scripts/eden/build-deps.sh | head -1)" && ! (grep -n '8080' BUILDING-EdenDocs.md | grep -viv 'never\|banned\|prohibit\|do not\|don.t')
  </verify>
  <done>All script paths in the doc exist and all four are referenced; configure flags and the POCO --omit list match the scripts verbatim; the CI section matches build.yml's file name and trigger; every 8080 mention across doc/scripts/workflow is a prohibition, never a usable value.</done>
  <recovery>On mismatch, edit the doc to match the scripts. If a script itself is found wrong, fix it, push, re-verify CI green (gh run watch), then finish the doc — never document an unproven change.</recovery>
</task>

</tasks>

<validation_gates>
<lint>test -f BUILDING-EdenDocs.md</lint>
<test>! (grep -n '8080' BUILDING-EdenDocs.md | grep -viv 'never\|banned\|prohibit\|do not\|don.t')</test>
<build>true # documentation TRD</build>
</validation_gates>

<verification>
1. BUILDING-EdenDocs.md exists at repo root, additive (no upstream file modified — confirm via `git status`/`git diff --stat`).
2. All Task-1 content greps pass; all Task-2 consistency checks pass.
3. The doc alone is sufficient to reproduce the CI-proven build (mental walkthrough: deps → POCO → fetch → build → smoke, with Node 20 prerequisite stated).
</verification>

<success_criteria>
- BUILD-01's "documented" clause satisfied: a written Linux developer build doc for the tarball fast path.
- BUILD-02's "documented fallback" clause satisfied: deterministic auto-discovery failure + system-POCO fallback rationale, with both known pitfalls.
- Zero drift between doc, scripts, and CI workflow.
</success_criteria>

<output>
After completion, create `.planning/objectives/01-from-source-build-pipeline/01-03-SUMMARY.md` including the cross-verification results table.
</output>
