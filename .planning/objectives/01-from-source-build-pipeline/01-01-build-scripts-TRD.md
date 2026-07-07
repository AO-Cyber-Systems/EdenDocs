---
objective: 01-from-source-build-pipeline
trd: "01"
type: standard
wave: 1
depends_on: []
files_modified:
  - scripts/eden/fetch-engine-assets.sh
  - scripts/eden/build-deps.sh
  - scripts/eden/build.sh
  - scripts/eden/smoke-test.sh
autonomous: true
requirements: [BUILD-01, BUILD-02, BUILD-04]

must_haves:
  truths:
    - "The live engine-main-assets.tar.gz has been empirically re-confirmed to contain only instdir/ — zero POCO headers, zero workdir/ — proving system-POCO fallback is the required path (BUILD-02 validated in the earliest wave)"
    - "All four scripts/eden/*.sh are syntactically valid bash, executable, and contain zero references to port 8080"
    - "scripts/eden/build.sh hard-asserts the configure-log line 'POCO not found in the engine workdir, falling back to system POCO' so the POCO strategy is re-validated on every future build automatically"
  artifacts:
    - "scripts/eden/fetch-engine-assets.sh"
    - "scripts/eden/build-deps.sh"
    - "scripts/eden/build.sh"
    - "scripts/eden/smoke-test.sh"
  key_links:
    - "scripts/eden/fetch-engine-assets.sh → https://github.com/CollaboraOnline/online/releases/download/for-code-assets/engine-main-assets.tar.gz (extracted into engine/, sha256 logged)"
    - "scripts/eden/build.sh → configure output grep for the system-POCO fallback notice (BUILD-02 assertion)"
    - "scripts/eden/smoke-test.sh → http://127.0.0.1:9980/hosting/discovery and /hosting/capabilities (BUILD-04, native port 9980, never 8080)"
---

<objective>
Create the four canonical EdenDocs build scripts under a new, purely additive `scripts/eden/` directory, and empirically validate (in this earliest wave, per BUILD-02) that the prebuilt `engine-main-assets.tar.gz` ships no POCO/workdir content — making the system-POCO fallback the required, documented path.

Purpose: These scripts ARE the documented developer build path. The Wave-2 CI workflow calls them verbatim, so the path a developer runs locally and the path CI proves green are one and the same file — zero doc/CI drift by construction. Research (1-RESEARCH.md, HIGH confidence) already established the POCO auto-discovery failure as a deterministic negative; this TRD re-verifies it live and then encodes the fallback assertion into the build script permanently.

Output: `scripts/eden/{fetch-engine-assets,build-deps,build,smoke-test}.sh` — executable, shellcheck-clean where shellcheck is available, ready for the Wave-2 CI workflow to invoke.
</objective>

<file_tree>
scripts/
└── eden/                          ← CREATE (new dir, additive — no upstream file touched)
    ├── fetch-engine-assets.sh     ← CREATE
    ├── build-deps.sh              ← CREATE
    ├── build.sh                   ← CREATE
    └── smoke-test.sh              ← CREATE
</file_tree>

<execution_context>
@~/.claude/devflow/workflows/execute-trd.md
@~/.claude/devflow/templates/summary.md
</execution_context>

<embedded_context>

<codebase_examples>
The proven in-tree reference recipe — `.github/workflows/codeql-analysis.yml` (lines 47-74), the only CI recipe in this repo that builds against the ENGINE_ASSETS tarball today:

```yaml
sudo apt install -y libunwind-dev build-essential git libcap-dev python3-polib npm libpng-dev python3-lxml libpam-dev libzstd-dev libssl-dev libcppunit-dev wget cmake
wget https://pocoproject.org/releases/poco-1.12.5p2/poco-1.12.5p2-all.tar.gz
tar xf poco-1.12.5p2-all.tar.gz
cd poco-1.12.5p2-all
./configure --static --no-tests --no-samples --no-sharedlibs --cflags="-fPIC" --omit=Zip,Data,Data/SQLite,Data/ODBC,Data/MySQL,MongoDB,PDF,CppParser,PageCompiler,Redis,Encodings,ActiveRecord,Prometheus,JWT
make -j `nproc`
sudo make install
# ...then:
wget --no-verbose https://github.com/CollaboraOnline/online/releases/download/for-code-assets/engine-main-assets.tar.gz
tar xf engine-main-assets.tar.gz -C engine && rm engine-main-assets.tar.gz
./autogen.sh
./configure --enable-silent-rules --with-lokit-path=${GITHUB_WORKSPACE}/engine/include --with-lo-path=${GITHUB_WORKSPACE}/engine/instdir --enable-debug
make -j `nproc` build-nocheck
```

The smoke-test parameter set, from `Makefile.am` COMMON_PARAMS (line 726) and the `run:` target (line 745) — coolwsd runs as `./coolwsd` from the repo root (in-tree build):

```makefile
COMMON_PARAMS = \
	--o:sys_template_path="@SYSTEMPLATE_PATH@" \
	--o:child_root_path="@JAILS_PATH@" \
	--o:cache_files.path="@CACHE_PATH@" \
	--o:storage.filesystem[@allow]=true \
	--o:admin_console.username=admin --o:admin_console.password=admin
```

Systemplate bootstrap, from `Makefile.am:668`: `coolwsd-systemplate-setup "@SYSTEMPLATE_PATH@" "@LO_PATH@"` where LO_PATH = `engine/instdir`.

Response-content anchors for smoke assertions (verified in-tree):
- `discovery.xml:2` → root element `<wopi-discovery>` (served at `/hosting/discovery`, `wsd/ClientRequestDispatcher.cpp:2914`)
- `wsd/ClientRequestDispatcher.cpp:3037` → capabilities JSON always sets key `"convert-to"` (served at `/hosting/capabilities`)
- `common/Common.hpp:28` → `DEFAULT_CLIENT_PORT_NUMBER = 9980`
</codebase_examples>

<anti_patterns>
- **Do NOT pass `--with-poco-includes`/`--with-poco-libs`** — configure.ac:365-391 accepts them but silently ignores them with only a warning. They pin nothing.
- **Do NOT use apt `libpoco-dev`** — Ubuntu 24.04 ships POCO 1.11.0, below configure.ac's hard `>= 1.12.0` floor (configure.ac:2231-2239). It fails at configure, wasting a run.
- **Do NOT copy `docker/from-source/build.sh`'s tarball naming quirk** — that script saves the gzip tarball as `engine-assets.tar.xz` yet extracts with `-xzf` (build.sh:100). Works by accident; use `.tar.gz` naming consistently.
- **Do NOT add `npm update`, `actions/cache` for node_modules, or any npm-registry step** — `Makefile.am:981` runs `cd browser && UV_USE_IO_URING=0 npm ci --offline` automatically against the 866 git-tracked vendored packages in `browser/node_shrinkpack/`. Zero network needed.
- **Do NOT drop `--enable-debug`** — without it, a bad `--with-lo-path` is silently tolerated (configure.ac:1765-1774 only hard-errors the missing-versionrc check when ENABLE_DEBUG=true) and you get a quietly broken build.
- **NEVER reference port 8080 anywhere** — banned project-wide (permanently occupied on the dev machine). coolwsd native port is 9980; any incidental local web server uses 8091.
</anti_patterns>

<error_recovery>
- **Configure hard-errors "header Poco/Net/WebSocket.h not found"** → system POCO was not installed before `./configure` ran. Run `scripts/eden/build-deps.sh` first; POCO install must strictly precede configure (research Pitfall 1).
- **Configure log shows "POCO not found in the engine workdir, falling back to system POCO"** → this is SUCCESS, not an error. It is the expected, deterministic state for ENGINE_ASSETS builds; build.sh asserts its presence.
- **`grep` for the fallback notice FAILS in build.sh** → upstream changed the tarball to include workdir POCO, or the notice text changed. Do not silently loosen the grep — inspect the configure log, re-verify the tarball contents, and update the BUILD-02 documentation accordingly.
- **fetch script 404s** → the `for-code-assets` release tag names assets inconsistently across branches (`engine-main-assets.tar.gz` for main, `core-co-25.04-assets.tar.gz` for co-25.04). eden-main tracks main, so `engine-main-assets.tar.gz` is correct; a 404 means upstream renamed it — check https://api.github.com/repos/CollaboraOnline/online/releases/tags/for-code-assets.
- **`test -f engine/instdir/program/versionrc` fails after extraction** → tarball layout changed upstream; the `--with-lo-path` debug sanity check would also fail. Inspect `tar tzf` listing before proceeding.
</error_recovery>

</embedded_context>

<context>
@.planning/PROJECT.md
@.planning/ROADMAP.md
@.planning/STATE.md
@.planning/objectives/01-from-source-build-pipeline/1-RESEARCH.md
@.github/workflows/codeql-analysis.yml
@docker/from-source/build.sh
</context>

<research_context>
From 1-RESEARCH.md (HIGH confidence, empirically verified 2026-07-07):
- The live `engine-main-assets.tar.gz` (433,550,261 bytes, rebuilt 2026-07-07T17:55:52Z) contains ONLY `instdir/` — zero `.h` files, zero "poco" matches, zero `workdir/`. POCO auto-discovery via `--with-lo-builddir` (configure.ac:1795-1811) deterministically fails; configure then requires a system POCO >= 1.12.0 discoverable via default compiler paths (configure.ac:2202-2239), no flags.
- `engine/include` (~2680 headers incl. `COKit/COKit.h`) is git-tracked source, present regardless of the tarball — which is why `--with-lokit-path=$PWD/engine/include` resolves even though the tarball ships no headers.
- POCO 1.12.5p2 from source (exact codeql-analysis.yml recipe) is the only proven fallback. `sudo make install` puts it in /usr/local, where configure's `AC_CHECK_HEADERS` finds it with no flags.
- Root configure.ac has NO Linux ccache auto-detection (only Windows/CODA path, ~2758-2806). ccache must be wired externally via `CC="ccache gcc" CXX="ccache g++"`.
- `for-code-assets` is a rolling, unversioned release tag with no published checksum. Do not hard-pin a sha256 (fossilizes against a stale artifact); log the downloaded sha256 for traceability instead.
- Node/npm floors are hard-enforced at configure.ac:2291-2323 (node >= 20.0.0, npm >= 9.0.0). Node version pinning itself is handled in the Wave-2 CI TRD.
- `autoconf automake libtool pkg-config` are believed preinstalled on ubuntu-latest (MEDIUM confidence, research Open Question 2) — include them in the apt list defensively; it is a no-op when present.
</research_context>

<gotchas>
- HARD RULE: never use, bind, curl, or mention port 8080 in any script, comment, or verification. coolwsd = 9980; incidental local web servers = 8091.
- These scripts target Linux (Ubuntu 24.04). The dev machine is macOS — local verification in this TRD is syntax/content-level plus a network-only tarball inspection; the full execution proof lands in the Wave-2 CI run.
- `sha256sum` (Linux) vs `shasum -a 256` (macOS): scripts use `sha256sum` since they run on Linux; the Task-1 local tarball inspection avoids checksum tools entirely by streaming to `tar tzf -`.
- Upstream mergeability: everything in this TRD is additive (`scripts/eden/` is a new directory). No upstream file is touched — nothing for the future UPST-02 divergence allowlist.
</gotchas>

<tasks>

<task type="auto">
  <name>Task 1: Empirically validate tarball structure (BUILD-02, earliest wave) and create fetch-engine-assets.sh</name>
  <files>scripts/eden/fetch-engine-assets.sh</files>
  <action>
First, re-verify the BUILD-02 premise live (runs fine on macOS — network + tar only, no Linux build needed):

```bash
LISTING=$(mktemp)
curl -sL https://github.com/CollaboraOnline/online/releases/download/for-code-assets/engine-main-assets.tar.gz | tar tzf - > "$LISTING"
grep -c 'instdir/' "$LISTING"                    # expect > 0 (instdir tree present)
grep -Eic 'poco|workdir' "$LISTING" || true       # expect 0 (no POCO, no workdir → auto-discovery MUST fall back)
test -n "$(grep 'instdir/program/versionrc' "$LISTING")"   # versionrc present (needed by --enable-debug lo-path check)
```

Record the three results in the eventual SUMMARY (they are the BUILD-02 "validated early" evidence). If `poco|workdir` count is NOT zero, STOP: upstream changed the tarball shape — re-read 1-RESEARCH.md's Pattern 1/2 and reassess before writing any script.

Then create `scripts/eden/fetch-engine-assets.sh` (mode 0755, `#!/usr/bin/env bash`, `set -euo pipefail`):
1. `cd` to repo root (`cd "$(dirname "$0")/../.."`).
2. `ENGINE_ASSETS_URL="${ENGINE_ASSETS_URL:-https://github.com/CollaboraOnline/online/releases/download/for-code-assets/engine-main-assets.tar.gz}"` — overridable env var, defaulting to the main-branch asset (eden-main tracks upstream main).
3. Idempotence guard: if `engine/instdir` exists and `FORCE_REFETCH` != 1, print a skip message and `exit 0`.
4. `wget --no-verbose "$ENGINE_ASSETS_URL" -O engine-assets.tar.gz`
5. `sha256sum engine-assets.tar.gz` — log to stdout for traceability. Comment WHY there is no pinned checksum: `for-code-assets` is a rolling unversioned tag with no published sidecar hash; pinning would fossilize the build (research Pitfall 6).
6. `tar xzf engine-assets.tar.gz -C engine && rm engine-assets.tar.gz` (note: consistent .tar.gz naming — do not copy docker/from-source/build.sh's .tar.xz misnomer).
7. Post-extract assertion: `test -f engine/instdir/program/versionrc` (fails loudly if the tarball layout ever changes).
Include a header comment block: purpose, BUILD-01 fast path, "never port 8080" project rule pointer.
  </action>
  <verify>
LISTING=$(mktemp); curl -sL https://github.com/CollaboraOnline/online/releases/download/for-code-assets/engine-main-assets.tar.gz | tar tzf - > "$LISTING" && test "$(grep -c 'instdir/' "$LISTING")" -gt 0 && test "$(grep -Eic 'poco|workdir' "$LISTING" || true)" = "0"

bash -n scripts/eden/fetch-engine-assets.sh && test -x scripts/eden/fetch-engine-assets.sh && grep -q 'for-code-assets/engine-main-assets.tar.gz' scripts/eden/fetch-engine-assets.sh && grep -q 'sha256sum' scripts/eden/fetch-engine-assets.sh && grep -q 'versionrc' scripts/eden/fetch-engine-assets.sh && ! grep -q '8080' scripts/eden/fetch-engine-assets.sh
  </verify>
  <done>Live tarball listing empirically shows instdir-only content (0 poco/workdir entries) — BUILD-02 validated in Wave 1; fetch script exists, is executable, syntactically valid, logs sha256, asserts versionrc, and never mentions 8080.</done>
  <recovery>If the live listing shows poco/workdir entries, do NOT proceed — upstream tarball shape changed; halt and surface to user with the listing excerpt (plan premise invalidated). If the download stalls, retry once; GitHub release CDN blips are transient.</recovery>
</task>

<task type="auto">
  <name>Task 2: Create build-deps.sh (apt deps + source-built POCO 1.12.5p2) and build.sh (autogen/configure/make with POCO-fallback assertion)</name>
  <files>scripts/eden/build-deps.sh, scripts/eden/build.sh</files>
  <action>
Create `scripts/eden/build-deps.sh` (0755, bash, `set -euo pipefail`, `export DEBIAN_FRONTEND=noninteractive`):
1. `sudo apt-get update`
2. `sudo apt-get install -y libunwind-dev build-essential git libcap-dev python3-polib npm libpng-dev python3-lxml libpam-dev libzstd-dev libssl-dev libcppunit-dev wget cmake ccache autoconf automake libtool pkg-config` — the exact proven codeql-analysis.yml list, PLUS `ccache` (BUILD-03) and the defensive autotools quartet (research Open Question 2: MEDIUM-confidence preinstalled on ubuntu-latest; no-op if present).
3. POCO: guard with `if [ ! -f /usr/local/include/Poco/Net/WebSocket.h ]; then ... fi` (idempotent re-runs). Inside: work in `$(mktemp -d)`; wget `https://pocoproject.org/releases/poco-1.12.5p2/poco-1.12.5p2-all.tar.gz`; extract; `./configure --static --no-tests --no-samples --no-sharedlibs --cflags="-fPIC" --omit=Zip,Data,Data/SQLite,Data/ODBC,Data/MySQL,MongoDB,PDF,CppParser,PageCompiler,Redis,Encodings,ActiveRecord,Prometheus,JWT`; `make -j"$(nproc)"`; `sudo make install`. This recipe is copied VERBATIM from codeql-analysis.yml (lines 56-61) — it is the only proven-working POCO path for ENGINE_ASSETS builds.
4. Header comment: WHY source-built POCO is mandatory (tarball ships no workdir POCO — deterministic, empirically verified; apt libpoco-dev is 1.11.0 < configure's 1.12.0 floor, configure.ac:2231-2239; `--with-poco-*` flags are obsolete no-ops, configure.ac:365-391).

Create `scripts/eden/build.sh` (0755, bash, `set -euo pipefail`):
1. `cd` to repo root.
2. ccache wiring (root configure.ac has NO Linux ccache auto-detect — research Pitfall 4):
   ```bash
   if command -v ccache >/dev/null 2>&1 && [ -z "${CC:-}" ]; then
     export CC="ccache gcc" CXX="ccache g++"
   fi
   ```
3. `./autogen.sh`
4. Configure, teeing output to a temp log for the BUILD-02 assertion:
   ```bash
   CONFIGURE_LOG="$(mktemp)"
   ./configure --enable-silent-rules --enable-debug \
     --with-lokit-path="$PWD/engine/include" \
     --with-lo-path="$PWD/engine/instdir" 2>&1 | tee "$CONFIGURE_LOG"
   ```
   Keep `--enable-debug`: it arms the lo-path versionrc hard-check (configure.ac:1765-1774) and matches codeql-analysis.yml.
5. BUILD-02 permanent assertion (comment: "this notice is SUCCESS for ENGINE_ASSETS builds, not a warning to fix"):
   ```bash
   grep -q "POCO not found in the engine workdir, falling back to system POCO" "$CONFIGURE_LOG"
   ```
6. `make -j"$(nproc)" build-nocheck`
7. Output assertions: `test -x ./coolwsd` and `test -d ./browser/dist` (BUILD-01 artifacts).
No `npm update`, no registry access anywhere — `make` runs `npm ci --offline` itself via Makefile.am:981.
  </action>
  <verify>
bash -n scripts/eden/build-deps.sh && bash -n scripts/eden/build.sh && test -x scripts/eden/build-deps.sh && test -x scripts/eden/build.sh && grep -q 'poco-1.12.5p2-all.tar.gz' scripts/eden/build-deps.sh && grep -q -- '--omit=Zip,Data' scripts/eden/build-deps.sh && grep -q 'ccache' scripts/eden/build-deps.sh && grep -q 'falling back to system POCO' scripts/eden/build.sh && grep -q -- '--enable-debug' scripts/eden/build.sh && grep -q 'build-nocheck' scripts/eden/build.sh && grep -q 'browser/dist' scripts/eden/build.sh && ! grep -q -- '--with-poco' scripts/eden/build.sh && ! grep -q 'npm update' scripts/eden/build.sh && ! grep -q '8080' scripts/eden/build-deps.sh && ! grep -q '8080' scripts/eden/build.sh
  </verify>
  <done>Both scripts exist, executable and syntactically valid; build-deps.sh carries the verbatim POCO 1.12.5p2 recipe + ccache + defensive autotools; build.sh wires ccache via CC/CXX, keeps --enable-debug, hard-asserts the system-POCO fallback notice, builds via build-nocheck, and asserts coolwsd + browser/dist outputs; no obsolete POCO flags, no npm update, no 8080.</done>
  <recovery>Syntax errors: fix and re-run bash -n. If shellcheck (available via brew) flags issues, fix warnings-level findings; do not suppress errors. Full execution proof is deferred to the Wave-2 CI run by design.</recovery>
</task>

<task type="auto">
  <name>Task 3: Create smoke-test.sh — coolwsd serves /hosting/discovery and /hosting/capabilities on 9980</name>
  <files>scripts/eden/smoke-test.sh</files>
  <action>
Create `scripts/eden/smoke-test.sh` (0755, bash, `set -euo pipefail`), modeled on Makefile.am's `setup-wsd`/`run:` targets (lines 662-753) and wsd/COOLWSD.cpp's fatal-config requirements:

```bash
#!/usr/bin/env bash
# EdenDocs smoke test (BUILD-04): the built coolwsd must serve
# /hosting/discovery and /hosting/capabilities on its NATIVE port 9980.
# HARD PROJECT RULE: never use port 8080 for anything.
set -euo pipefail
cd "$(dirname "$0")/../.."

# systemplate bootstrap — canonical in-tree tool (Makefile.am:668)
./coolwsd-systemplate-setup ./systemplate "$PWD/engine/instdir"
mkdir -p ./jails ./cache

./coolwsd --disable-ssl \
  --o:sys_template_path="$PWD/systemplate" \
  --o:child_root_path="$PWD/jails" \
  --o:cache_files.path="$PWD/cache" \
  --o:storage.filesystem[@allow]=true \
  --o:admin_console.username=admin --o:admin_console.password=admin &
COOLWSD_PID=$!
trap 'kill "$COOLWSD_PID" 2>/dev/null || true' EXIT

# Readiness: coolwsd's built-in self-contained probe (wsd/COOLWSD.cpp,
# probeRunningServer) — primary signal, retried up to 120s for cold CI starts.
for _ in $(seq 1 120); do
  if ./coolwsd --probe >/dev/null 2>&1; then break; fi
  sleep 1
done

# Literal BUILD-04 checks on the native port 9980 (never 8080):
curl -fsS http://127.0.0.1:9980/hosting/discovery    | grep -q 'wopi-discovery'
curl -fsS http://127.0.0.1:9980/hosting/capabilities | grep -q 'convert-to'
echo "SMOKE TEST PASSED: coolwsd serves /hosting/discovery and /hosting/capabilities on 9980"
```

Content anchors are verified in-tree: `discovery.xml` root element is `<wopi-discovery>`; the capabilities JSON always contains the `convert-to` key (wsd/ClientRequestDispatcher.cpp:3037). Both routes are served directly by coolwsd — no WOPI host, auth, or storage backend needed. Run from repo root because the in-tree build drops `./coolwsd` and the generated `coolwsd.xml` there (matching the Makefile `run:` target's working directory).
  </action>
  <verify>
bash -n scripts/eden/smoke-test.sh && test -x scripts/eden/smoke-test.sh && grep -q 'coolwsd-systemplate-setup' scripts/eden/smoke-test.sh && grep -q -- '--probe' scripts/eden/smoke-test.sh && grep -q '127.0.0.1:9980/hosting/discovery' scripts/eden/smoke-test.sh && grep -q '127.0.0.1:9980/hosting/capabilities' scripts/eden/smoke-test.sh && grep -q 'wopi-discovery' scripts/eden/smoke-test.sh && grep -q 'convert-to' scripts/eden/smoke-test.sh && grep -q 'SMOKE TEST PASSED' scripts/eden/smoke-test.sh && test "$(grep -c '8080' scripts/eden/smoke-test.sh)" -le 1
  </verify>
  <done>smoke-test.sh exists, executable, syntactically valid; bootstraps systemplate via the canonical tool, launches coolwsd with the fatal-required config keys and --disable-ssl, gates readiness on the built-in --probe, curls both /hosting endpoints on 9980 with content assertions, cleans up via trap; 8080 appears only in the prohibition comment.</done>
  <recovery>If content-anchor doubts arise (wopi-discovery / convert-to), re-verify against discovery.xml:2 and wsd/ClientRequestDispatcher.cpp:3037 in-tree before weakening a grep to a plain HTTP-200 check (curl -f alone) — and note the weakening in the SUMMARY. Execution proof lands in the Wave-2 CI run.</recovery>
</task>

</tasks>

<validation_gates>
<lint>for f in scripts/eden/*.sh; do bash -n "$f"; done; command -v shellcheck >/dev/null && shellcheck scripts/eden/*.sh || true</lint>
<test>! grep -rn '8080' scripts/eden/ | grep -v 'never use port 8080\|never port 8080\|never use, bind\|never 8080'</test>
<build>true # Linux-only build; executed by the Wave-2 CI workflow</build>
</validation_gates>

<verification>
1. Live tarball listing captured and confirms instdir-only / zero poco / zero workdir (BUILD-02 validated in earliest wave — record evidence in SUMMARY).
2. All four scripts pass `bash -n`, are executable, and pass the per-task grep assertions above.
3. `git status` shows only additive files under `scripts/eden/` — no upstream file modified.
4. No script references port 8080 except the prohibition comment in smoke-test.sh.
</verification>

<success_criteria>
- scripts/eden/{fetch-engine-assets,build-deps,build,smoke-test}.sh exist, executable, syntax-clean, content-verified.
- BUILD-02's premise (deterministic POCO auto-discovery failure → system-POCO fallback required) is empirically re-confirmed and permanently asserted inside build.sh.
- The full developer build path is now encoded in scripts the Wave-2 CI workflow can call verbatim.
</success_criteria>

<output>
After completion, create `.planning/objectives/01-from-source-build-pipeline/01-01-SUMMARY.md` including the tarball-listing evidence (instdir count, poco/workdir count, versionrc presence) and the logged tarball size/date.
</output>
