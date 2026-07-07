# Objective 1: From-Source Build Pipeline - Research

**Researched:** 2026-07-07
**Domain:** Autoconf/Automake C++ build (coolwsd + POCO), prebuilt-artifact fast-path builds, GitHub Actions CI, offline npm/TypeScript frontend build (`browser/`)
**Confidence:** HIGH — grounded in direct repo inspection plus live empirical verification of the actual `engine-main-assets.tar.gz` artifact (downloaded and inspected 2026-07-07, same day as its last rebuild)

## Summary

EdenDocs (`online.mirror` fork) has two separate Autoconf projects: root `configure.ac` (coolwsd/wsd, package `coolwsd`) and `engine/configure.ac` (LibreOffice core / LOKit). BUILD-01 wants the fast path: skip building `engine/` entirely and instead unpack Collabora's prebuilt `engine-main-assets.tar.gz` into `engine/`, matching exactly what `docker/from-source/build.sh`'s `ENGINE_ASSETS` branch and `.github/workflows/codeql-analysis.yml`'s "Configure" step already do.

The critical finding, verified empirically rather than assumed: **`--with-lo-builddir` POCO auto-discovery will always fail against this tarball.** Root `configure.ac` looks for POCO at `$with_lo_builddir/workdir/UnpackedTarball/poco/include` and `.../workdir/LinkTarget/StaticLibrary` (configure.ac:1795-1811). I downloaded the live `engine-main-assets.tar.gz` (433,550,261 bytes, rebuilt 2026-07-07T17:55:52Z) and listed its full contents: it contains **only `instdir/`** (the compiled runtime tree — `program/`, `share/`, license files) — zero `.h` files, zero matches for "poco", zero `workdir/` directory of any kind. This is not a bug or an oversight to work around later; it is how the asset is built (see `docker/from-source/build.sh`'s own build path — the tarball is a packaging of `make instdir`, not `make`). So BUILD-02's "validate early" requirement isn't really a coin-flip risk to de-risk — it's a **known, deterministic negative** that should be treated as the default state, with the fallback wired in from the first CI run rather than discovered mid-setup.

The fallback is not `--with-poco-includes`/`--with-poco-libs` (confirmed obsolete: accepted but silently ignored with a warning, configure.ac:365-391). It's configure's built-in **system-POCO fallback**: when workdir POCO isn't found, configure emits `AC_MSG_NOTICE([POCO not found in the engine workdir, falling back to system POCO])` and then requires a system-discoverable POCO ≥1.12.0 (hard version floor, configure.ac:2231-2239) satisfying `AC_CHECK_HEADERS([Poco/Net/WebSocket.h])` (configure.ac:2206-2208) via default compiler search paths — no configure flags needed at all. Ubuntu 24.04's `libpoco-dev` apt package is POCO 1.11.0 — **too old, fails the version floor** — so the only proven-working fallback is building POCO from source, exactly as `codeql-analysis.yml` already does (POCO 1.12.5p2 from pocoproject.org, static, `--omit=...`, `sudo make install`). That recipe should be copied verbatim into the new BUILD-03 workflow, not merely referenced.

**Primary recommendation:** Build a new `.github/workflows/build.yml` (push-triggered, not schedule-only) that: (1) apt-installs the codeql-analysis.yml dependency list plus `ccache`; (2) builds+installs POCO 1.12.5p2 from source exactly as codeql-analysis.yml does — this is not optional, it is the only viable path against the ENGINE_ASSETS tarball; (3) downloads and extracts `engine-main-assets.tar.gz` into `engine/`; (4) pins Node 20 explicitly via `actions/setup-node@v4` (ubuntu-latest's toolcache no longer ships Node 20 at all as of 2026-05-26 — it must be fetched on demand); (5) wires ccache externally via `CC`/`CXX` env vars or the PATH-symlink trick, since root `configure.ac` has no ccache auto-detection outside the Windows/CODA path; (6) runs `./autogen.sh && ./configure --enable-silent-rules --enable-debug --with-lokit-path=$PWD/engine/include --with-lo-path=$PWD/engine/instdir && make -j$(nproc) build-nocheck`; (7) smoke-tests the result per BUILD-04 using coolwsd's built-in `--probe` flag plus a real curl against `/hosting/discovery`/`/hosting/capabilities` on port **9980** (never 8080).

<phase_requirements>
## Objective Requirements

| ID | Description | Research Support |
|----|-------------|-------------------|
| BUILD-01 | Developer can build coolwsd + browser/dist from EdenDocs source on Linux using the prebuilt engine assets fast path (`ENGINE_ASSETS` tarball, no from-scratch `engine/` compile) | `docker/from-source/build.sh`'s `ENGINE_ASSETS` branch is the proven-working reference implementation (Code Examples #1); exact configure invocation extracted from the same file and cross-checked against `codeql-analysis.yml`; confirmed `browser/dist` is produced directly by `make`/`make build-nocheck` without a `make install` step (Makefile.am:1079, browser/Makefile.am `DIST_FOLDER`) |
| BUILD-02 | Build validates early that the prebuilt engine tarball satisfies in-tree POCO auto-discovery (`--with-lo-builddir`), no `--with-poco-*` flags; documented fallback if it doesn't | Empirically confirmed (live tarball download+inspection) that auto-discovery **always fails** against `engine-main-assets.tar.gz`; exact configure.ac logic cited (lines 1795-1811, 2202-2239); documented, proven fallback recipe extracted verbatim from `codeql-analysis.yml` (POCO 1.12.5p2 from source); apt `libpoco-dev` pitfall (too old) documented so it isn't tried and discovered broken in CI |
| BUILD-03 | CI builds eden-main on GitHub-hosted ubuntu runners on every push, modeled on `codeql-analysis.yml` (Node 20 LTS, ccache) | Full `codeql-analysis.yml` recipe reverse-engineered line-by-line (Code Examples #2); gaps vs. BUILD-03 identified precisely (schedule-only trigger, no ccache, no Node pin, registry-touching `npm update` step); root-vs-engine ccache auto-detection distinction corrected; Node-20-removed-from-toolcache-2026-05-26 finding sourced live via WebSearch |
| BUILD-04 | Built coolwsd passes a smoke test — serves `/hosting/discovery` and `/hosting/capabilities` (never port 8080; native 9980) | `common/Common.hpp:28` confirms native port 9980; `wsd/ClientRequestDispatcher.cpp` confirms both routes are served directly by coolwsd (no WOPI host needed); `wsd/COOLWSD.cpp` fatal-config requirements (`sys_template_path`/`child_root_path`), `--disable-ssl`, and built-in `--probe` health-check flag documented with exact recipe (Makefile.am `run:`/`setup-wsd` targets, `coolwsd-systemplate-setup` script) |
</phase_requirements>

## Standard Stack

### Core

| Library/Tool | Version | Purpose | Why Standard |
|---|---|---|---|
| Autoconf/Automake/libtool | Ubuntu 24.04 defaults | Generates `./configure` from `configure.ac` via `./autogen.sh` | Root build system; `autogen.sh` calls `libtoolize`, `aclocal`, `autoheader`, `automake --add-missing`, `autoreconf` — standard GNU toolchain |
| POCO C++ Libraries | 1.12.5p2 (fallback, source build) | HTTP/WebSocket/JSON/crypto used by `wsd/` | Required because ENGINE_ASSETS tarball never populates `workdir/`; this is the exact version+recipe `codeql-analysis.yml` already proves works |
| Node.js | 20.x LTS (explicit pin) | Builds `browser/` (TS/JS via esbuild etc.) | Hard-enforced floor `>=20.0.0` in configure.ac:2291-2323; ubuntu-latest no longer ships Node 20 in its toolcache as of 2026-05-26, default is now 22.x — must be pinned explicitly, not inherited |
| npm | >=9.0.0 (ships with Node 20 LTS as ~10.x) | Package manager for `browser/` | Hard-enforced floor in configure.ac:2291-2323 |
| GCC (or Clang) with C++20 | Ubuntu 24.04 default (GCC 13) | Compiles `wsd/`, `common/`, `net/` | configure.ac:1658-1686 hard-errors if the compiler lacks C++20 support; GCC 13 (noble default) supports it |
| ccache | apt default | Speeds up repeat CI/dev builds | Required by BUILD-03 explicitly; root `configure.ac` has **no** auto-detection for Linux (only Windows/CODA `CODA_CCACHE`, ~lines 2758-2806) — must wire externally via `CC="ccache gcc" CXX="ccache g++"` or the `/usr/lib/ccache` PATH-symlink trick |

### Supporting

| Library | Purpose | When to Use |
|---|---|---|
| `libcap-dev`, `libpam-dev`, `libzstd-dev`, `libssl-dev` (>=3.0.0), `libcppunit-dev`, `libpng-dev`, `libunwind-dev` | wsd/ runtime + build dependencies | Always, for the root `configure`/`make` step — exact list proven by `codeql-analysis.yml`'s apt install line |
| `python3-polib`, `python3-lxml` | Translation/XML tooling used by the build | Always, part of the same proven apt list |
| `cmake`, `wget`, `build-essential`, `git` | General build tooling; `wget`/tarball fetch | Always |
| `browser/node_shrinkpack/` (866 git-tracked `.tar` packages) + `browser/npm-shrinkwrap.json.in` | Fully offline npm install | Automatic — root `Makefile.am:980-981` runs `cd browser && npm ci --offline` as part of plain `make`; requires zero npm-registry network access |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|---|---|---|
| Build POCO 1.12.5p2 from source | apt `libpoco-dev` on Ubuntu 24.04 | **Rejected** — noble ships POCO 1.11.0, below configure's hard `1.12.0` floor (configure.ac:2231-2239); configure will hard-error at the header-version check |
| System-POCO fallback (no flags) | `--with-poco-includes`/`--with-poco-libs` | **Rejected** — confirmed obsolete: accepted on the command line but silently ignored with only a warning (configure.ac:365-391); gives false confidence that POCO location was pinned |
| ENGINE_ASSETS prebuilt tarball | Build `engine/` from source (`engine/autogen.sh && make`) | Explicitly out of scope per REQUIREMENTS.md; 1-3 hours and ~30GB disk (`scripts/clone-online.md`) vs. a ~434MB download; only needed if/when no matching tarball exists for a target branch (currently true for a hypothetical `co-26.04`, not relevant to `main`) |
| Reuse `codeql-analysis.yml` unmodified | Add a new dedicated `build.yml` | **Recommended: new file.** codeql-analysis.yml is schedule/dispatch-triggered by design (CodeQL scans are not meant to run every push) and includes a deliberate `npm update` step for dependency-freshness scanning that would reintroduce registry-touching non-determinism into a normal build — don't repurpose it, model a new workflow on it |

**Installation (Ubuntu 24.04 CI runner):**
```bash
# System build dependencies (proven list, from .github/workflows/codeql-analysis.yml)
sudo apt-get update
sudo apt-get install -y \
  libunwind-dev build-essential git libcap-dev python3-polib npm \
  libpng-dev python3-lxml libpam-dev libzstd-dev libssl-dev \
  libcppunit-dev wget cmake ccache

# POCO fallback — required, not optional, when using ENGINE_ASSETS (see Pitfall 1)
wget https://pocoproject.org/releases/poco-1.12.5p2/poco-1.12.5p2-all.tar.gz
tar xf poco-1.12.5p2-all.tar.gz && cd poco-1.12.5p2-all
./configure --static --no-tests --no-samples --no-sharedlibs --cflags="-fPIC" \
  --omit=Zip,Data,Data/SQLite,Data/ODBC,Data/MySQL,MongoDB,PDF,CppParser,PageCompiler,Redis,Encodings,ActiveRecord,Prometheus,JWT
make -j "$(nproc)"
sudo make install
cd ..

# Node 20 LTS — explicit, ubuntu-latest's toolcache no longer defaults to it
# (use actions/setup-node@v4 with node-version: '20' in CI; nvm/n locally)
```

## Architecture Patterns

### Pattern 1: ENGINE_ASSETS fast path (proven, `docker/from-source/build.sh`)
**What:** Skip `engine/autogen.sh && make` entirely; download+extract a prebuilt tarball into `engine/`, then configure root against `engine/include` (git-tracked source headers) and `engine/instdir` (the tarball's compiled runtime).
**When to use:** Always, per BUILD-01 — this is the only in-scope build path.
**Example:**
```bash
# Source: docker/from-source/build.sh (verbatim structure)
ENGINE_ASSETS="https://github.com/CollaboraOnline/online/releases/download/for-code-assets/engine-main-assets.tar.gz"
( cd engine && wget "$ENGINE_ASSETS" -O engine-assets.tar.gz \
    && tar -xzf engine-assets.tar.gz && rm engine-assets.tar.gz )

./autogen.sh
./configure --enable-silent-rules --enable-debug \
  --with-lokit-path="$PWD/engine/include" \
  --with-lo-path="$PWD/engine/instdir"
make -j "$(nproc)" build-nocheck
```
Note: `engine/include` (containing `COKit/COKit.h` and ~2679 other headers) is **git-tracked source**, always present regardless of the tarball — confirmed via `git ls-files engine/include | wc -l`. The tarball supplies only the compiled `instdir/` runtime. This is why `--with-lokit-path` and `--with-lo-path` both resolve correctly even though the tarball contains zero headers.

### Pattern 2: System-POCO fallback (required for ENGINE_ASSETS, proven by `codeql-analysis.yml`)
**What:** Install a source-built POCO into default system search paths *before* running root `./configure`. No `--with-poco-*` flags needed — configure finds it automatically once the workdir lookup fails.
**When to use:** Always, when using the ENGINE_ASSETS tarball (BUILD-02) — this is not a rare-case fallback, it is the expected/only path for this build mode.
**Example:**
```bash
# Source: configure.ac:1795-1811 (the actual auto-discovery/fallback logic)
# AS_IF([test -z "$POCOINCLUDE" -a -n "$with_lo_builddir"],
#       [poco_engine_builddir=`readlink -f "$with_lo_builddir"`
#        POCOINCLUDE="$poco_engine_builddir/workdir/UnpackedTarball/poco/include"
#        POCOLIB="$poco_engine_builddir/workdir/LinkTarget/StaticLibrary"])
# AS_IF([test -n "$POCOINCLUDE" -a -f "$POCOINCLUDE/Poco/Net/WebSocket.h"],
#       [AC_MSG_NOTICE([using POCO built in the engine: $POCOLIB])],
#       [AC_MSG_NOTICE([POCO not found in the engine workdir, falling back to system POCO])
#        POCOINCLUDE= ; POCOLIB=])
#
# ...with ENGINE_ASSETS this ALWAYS falls back. Install system POCO first
# (see Installation block above), then just run configure with no POCO flags:
./configure --enable-silent-rules --enable-debug \
  --with-lokit-path="$PWD/engine/include" --with-lo-path="$PWD/engine/instdir"
# Expect "POCO not found in the engine workdir, falling back to system POCO"
# in the configure log — this is SUCCESS, not a warning to fix.
```

### Pattern 3: Offline `browser/` build via vendored `node_shrinkpack`
**What:** `browser/`'s entire npm dependency tree (866 packages) is vendored as git-tracked `.tar` files at `browser/node_shrinkpack/`, paired with an autoconf-templated `browser/npm-shrinkwrap.json.in` (rendered to `browser/npm-shrinkwrap.json` by `./configure`'s `AC_CONFIG_FILES`, configure.ac:2714).
**When to use:** Automatic — no action needed. Plain `make`/`make build-nocheck` triggers `Makefile.am:980-981`: `cd browser && UV_USE_IO_URING=0 npm ci --offline`. Zero npm-registry network calls.
**Example:**
```makefile
# Source: Makefile.am:980-981
browser/node_modules: browser/package.json browser/node_shrinkpack
	@cd browser && UV_USE_IO_URING=0 npm ci --offline
```
Don't add `actions/cache` for `node_modules` or a registry mirror step — the shrinkpack mechanism already solves CI npm reproducibility/speed; it needs no extra CI plumbing at all.

### Pattern 4: Smoke test (BUILD-04)
**What:** coolwsd needs `sys_template_path`+`child_root_path` configured (fatal error otherwise, `wsd/COOLWSD.cpp`); the in-tree `Makefile.am` `run:`/`setup-wsd` targets and `coolwsd-systemplate-setup` script are the proven recipe.
**When to use:** Post-build CI/local verification step.
**Example:**
```bash
# Source: Makefile.am (setup-wsd / run: targets, COMMON_PARAMS), adapted for a
# non-interactive smoke test. Native port is 9980 — never 8080.
./coolwsd-systemplate-setup ./systemplate "$PWD/engine/instdir"
mkdir -p ./jails ./cache

./coolwsd --disable-ssl \
  --o:sys_template_path="$PWD/systemplate" \
  --o:child_root_path="$PWD/jails" \
  --o:cache_files.path="$PWD/cache" \
  --o:storage.filesystem[@allow]=true &
COOLWSD_PID=$!

# Built-in health check (no curl needed) — described in-code as
# "a self-contained HEALTHCHECK that needs no curl" (wsd/COOLWSD.cpp)
./coolwsd --probe

# Literal BUILD-04 requirement — real HTTP check on the native port, never 8080
curl -fsS http://127.0.0.1:9980/hosting/discovery
curl -fsS http://127.0.0.1:9980/hosting/capabilities

kill "$COOLWSD_PID"
```
Both `/hosting/discovery` and `/hosting/capabilities` are served directly by `wsd/ClientRequestDispatcher.cpp` — no WOPI host, auth, or storage config needed to smoke-test them.

### Anti-Patterns to Avoid
- **Copying `codeql-analysis.yml` verbatim as the new CI workflow:** it's `schedule`/`workflow_dispatch`-triggered (not `push`), has no ccache, no explicit Node pin, and includes a registry-touching `npm update` step meant for CodeQL's dependency-freshness scanning — none of which belong in an on-push build-verification workflow. Model the new workflow on it; don't reuse the file.
- **Passing `--with-poco-includes`/`--with-poco-libs` "just to be safe":** silently ignored (with a warning) — gives false confidence that POCO's location has been pinned when it hasn't.
- **Assuming `--with-lo-builddir` auto-discovery "might" work with ENGINE_ASSETS:** it deterministically won't (empirically confirmed against the live tarball) — build the system-POCO fallback into the CI recipe from the start rather than treating it as a rare contingency to branch on.
- **Trying apt `libpoco-dev` as the fallback:** Ubuntu 24.04 (noble) ships POCO 1.11.0, below configure's hard `1.12.0` floor — wastes a CI run discovering this.
- **Relying on `ubuntu-latest`'s default Node version:** as of 2026-05-26 Node 20 was removed from the runner toolcache entirely (default is now 22.x) — must pin via `actions/setup-node@v4`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---|---|---|---|
| npm dependency fetching/caching in CI | A custom `actions/cache` step for `browser/node_modules`, or a registry mirror | The existing `browser/node_shrinkpack/` (866 vendored `.tar` files) + `npm ci --offline` | Already solves reproducibility and speed; git-tracked, zero network required, wired automatically by `Makefile.am` |
| POCO fallback build script | A custom POCO-detection/build wrapper or version-sniffing logic | `codeql-analysis.yml`'s exact `./configure --static --no-tests --no-samples --no-sharedlibs --cflags="-fPIC" --omit=...` recipe | Proven working today in a real CI job against the same tarball-based build mode |
| Health-check/smoke-test harness | A custom curl-retry-loop / port-wait script | coolwsd's built-in `--probe` CLI flag (`wsd/COOLWSD.cpp`, `probeRunningServer()`) | Self-contained, connects to coolwsd's own loopback socket and checks for HTTP 200 — described in-code as needing no curl; use it as the primary readiness signal, with a literal `/hosting/discovery` curl on top to satisfy BUILD-04's exact wording |
| systemplate/jail directory bootstrap | Hand-rolled `mkdir`/copy logic for the chroot jail template | `coolwsd-systemplate-setup` (repo-root script, `dist_bin_SCRIPTS`) + Makefile's `setup-wsd`/`$(SYSTEM_STAMP)` targets | Canonical tool already used by both the dev `make run` target and the Docker image's entrypoint/postinst |

**Key insight:** Nearly every piece of BUILD-01 through BUILD-04 already has a proven, in-tree reference implementation (`docker/from-source/build.sh`, `codeql-analysis.yml`, `Makefile.am`'s `run:` target). The job is assembling/adapting these into one push-triggered workflow, not inventing new build logic.

## Common Pitfalls

### Pitfall 1: POCO auto-discovery fails silently by design, but only if you install POCO in the right order
**What goes wrong:** If you run `./configure` before installing a system POCO, and the ENGINE_ASSETS tarball is in use, `AC_CHECK_HEADERS([Poco/Net/WebSocket.h], [], [AC_MSG_ERROR(...)])` (configure.ac:2206-2208) hard-fails the entire configure step with "header Poco/Net/WebSocket.h not found; build the engine first (POCO is built there) or install a system POCO".
**Why it happens:** The workdir lookup at configure.ac:1795-1811 always misses against this tarball (empirically confirmed — the tarball ships only `instdir/`, no `workdir/`), so configure unconditionally falls through to needing a system POCO.
**How to avoid:** Build+install system POCO (the exact `codeql-analysis.yml` recipe) as an earlier CI step, strictly before `./configure` runs.
**Warning signs:** Configure log shows `POCO not found in the engine workdir, falling back to system POCO` — this line is *expected/normal* for this build mode, not a problem to fix. A hard `AC_MSG_ERROR` about the missing header is the actual failure signal.

### Pitfall 2: apt's POCO is too old for the version floor
**What goes wrong:** `sudo apt-get install libpoco-dev` on Ubuntu 24.04 installs POCO 1.11.0; configure.ac's `#if POCO_VERSION < 0x010C0000` check (configure.ac:2231-2239) rejects anything below 1.12.0 with "POCO version is too old, 1.12.0 or newer required".
**Why it happens:** Debian/Ubuntu package POCO versions lag upstream; noble's `libpoco-dev` predates the 1.12.0 floor this fork requires.
**How to avoid:** Build POCO 1.12.5p2 from source (pocoproject.org tarball), exactly as `codeql-analysis.yml` does — don't reach for the apt package as a shortcut.
**Warning signs:** `checking POCO version... too old` type configure output, or a build that fails at the `AC_MSG_ERROR` version-check line specifically (not the header-not-found line from Pitfall 1).

### Pitfall 3: `ubuntu-latest` no longer ships Node 20 by default
**What goes wrong:** A workflow that runs `node`/`npm` without an explicit version pin silently uses whatever the runner image defaults to, which drifts over time and is no longer Node 20.
**Why it happens:** Per GitHub's `actions/runner-images` project, Node.js 20 reached end-of-life 2026-04-30 and was **removed entirely from the runner toolcache** between 2026-05-19 and 2026-05-26; the default is now Node 22.x (22.23.1 confirmed as of late June 2026), with Node 24 also available. This still satisfies configure's `>=20.0.0` floor (so the build won't hard-fail), but it violates BUILD-03's explicit "Node 20 LTS" requirement and risks subtle toolchain drift in the browser build's pinned dependency graph.
**How to avoid:** Add an explicit `actions/setup-node@v4` step with `node-version: '20'` before the configure/build steps. Since Node 20 is no longer pre-cached, expect a small on-demand download each run (acceptable one-time cost; consider `actions/setup-node`'s built-in cache option if this becomes a bottleneck).
**Warning signs:** CI logs showing `node -v` as 22.x or 24.x instead of 20.x.

### Pitfall 4: ccache has no auto-detection for the Linux/root build
**What goes wrong:** Assuming ccache "just works" because it's installed via apt, without wiring it into the compiler invocation.
**Why it happens:** `engine/configure.ac` (lines ~1994-2007, ~3244-3330) has full ccache auto-detection (`AC_ARG_ENABLE(ccache...)`, `AC_PATH_PROG([CCACHE],...)`) — but that logic belongs to the *engine's own* configure script, which is never invoked when using ENGINE_ASSETS. Root `configure.ac` (coolwsd) has no equivalent generic auto-detection; its only ccache-adjacent logic (`CODA_CCACHE`, ~lines 2758-2806) is scoped to the Windows/CODA clang-cl cross-build path and does not apply here. *(Correction to prior project-level research, which had misattributed the engine's ccache auto-detect line numbers to the root configure.ac.)*
**How to avoid:** Set `CC="ccache gcc"` and `CXX="ccache g++"` as environment variables before running `./configure` (or use the standard `/usr/lib/ccache` PATH-symlink technique), and add an `actions/cache` step for `~/.ccache` (or `CCACHE_DIR`) keyed on a stable identifier (e.g., a hash of `configure.ac`) so warm caches persist across CI runs.
**Warning signs:** `ccache -s` shows a 0% hit rate across repeated CI runs.

### Pitfall 5: Don't copy `codeql-analysis.yml`'s `npm update` step
**What goes wrong:** Naively porting the whole "Configure" step from `codeql-analysis.yml` (which ends with `cd browser && npm update`) into the new build workflow reintroduces npm-registry network calls and non-reproducible dependency resolution.
**Why it happens:** That step exists specifically so CodeQL scans dependency versions that are current (freshness for vulnerability scanning), not because it's required for a normal build. The standard build path doesn't need it: plain `make`/`make build-nocheck` already runs `npm ci --offline` automatically (Makefile.am:980-981) against the fully vendored `node_shrinkpack/` tree.
**How to avoid:** Omit the `npm update` line entirely from the new workflow; let the Makefile's own `browser/node_modules` rule handle it offline.
**Warning signs:** CI build times or flakiness increase due to npm registry calls; `package-lock`/shrinkwrap drift between runs.

### Pitfall 6: `for-code-assets` is a rolling, unversioned release tag — no official checksum
**What goes wrong:** Expecting to "pin by checksum" for long-term reproducibility the way you would a tagged release.
**Why it happens:** `engine-main-assets.tar.gz` lives under a single perpetually-updated GitHub Release tag (`for-code-assets`) that tracks upstream `main` — confirmed via the GitHub Releases API and a fresh `Last-Modified` timestamp of 2026-07-07T17:55:52Z (rebuilt same day as this research). No `.sha256`/`.asc` sidecar asset is published.
**How to avoid:** Don't hard-pin a fixed sha256 (it would fossilize the build against a stale artifact quickly and silently diverge from what "main" actually means going forward). Instead, log the downloaded artifact's sha256 in CI output for traceability/drift-detection, and rely on HTTPS + GitHub's own transport integrity. Revisit if/when Objective 5's branch-tracking decision needs a more deliberate pin.
**Warning signs:** N/A directly — this is a design decision, not a failure mode, but worth documenting so a future contributor doesn't "fix" this by adding a stale hard pin.

### Pitfall 7: Tarball naming is inconsistent across branches
**What goes wrong:** Assuming the `co-25.04` distro branch's asset follows the same `engine-*-assets.tar.gz` naming as `main`.
**Why it happens:** Live GitHub Releases API inspection (2026-07-07) of the `for-code-assets` tag shows the `main` asset is named `engine-main-assets.tar.gz`, but the `co-25.04` asset is named **`core-co-25.04-assets.tar.gz`** (prefix `core-`, not `engine-`) — a naming inconsistency, not a typo in prior notes. *(Correction to prior project-level research, which assumed `engine-co-25.04-assets.tar.gz`.)* No `co-26.04` asset of either naming pattern currently exists.
**How to avoid:** Not a blocker for Objective 1 (eden-main tracks `main`, so `engine-main-assets.tar.gz` is correct). Flag for Objective 5's branch-tracking decision: any future switch to a `co-XX.04` branch must re-verify the exact asset name and structure (not yet confirmed whether `core-co-25.04-assets.tar.gz` has the same instdir-only layout — see Open Questions).
**Warning signs:** A hardcoded URL using the wrong prefix would simply 404 — easy to catch, low risk, but worth getting right the first time.

### Pitfall 8: `--with-lo-path` sanity check is silently skipped without `--enable-debug`
**What goes wrong:** A misconfigured or empty `--with-lo-path` in a non-debug configure invocation doesn't fail configure at all — it silently sets `have_lo_path=false` and prints "no integration tests", producing a build that may be broken in ways not caught until `make run` or the smoke test.
**Why it happens:** configure.ac:1765-1774: the `AC_MSG_ERROR([bad --with-lo-path; file does not exist: $version_file])` hard-check only fires when `$ENABLE_DEBUG = true` (i.e., `--enable-debug` was passed) — the reasoning being that production/packaging builds may legitimately point `--with-lo-path` at a runtime path (e.g. `/opt/collaboraoffice`) that doesn't exist yet at configure time.
**How to avoid:** Keep `--enable-debug` in both the local dev and CI configure invocations (matches `codeql-analysis.yml`'s existing usage) so a broken `--with-lo-path` fails loudly at configure time. (Confirmed the tarball does provide the needed `instdir/program/versionrc` file, so this check passes cleanly with a correctly-set `--with-lo-path="$PWD/engine/instdir"`.)
**Warning signs:** Configure output says "no integration tests" instead of "test against $LO_PATH" when you expected the latter.

## Code Examples

### Recommended `.github/workflows/build.yml` skeleton (BUILD-03)
```yaml
# Modeled on the proven .github/workflows/codeql-analysis.yml recipe,
# adapted for push-triggered build verification per BUILD-03.
name: build
on:
  push:
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-node@v4
        with:
          node-version: '20'   # ubuntu-latest toolcache no longer ships Node 20 (Pitfall 3)

      - name: Install ccache and cache its directory
        run: sudo apt-get update && sudo apt-get install -y ccache
      - uses: actions/cache@v4
        with:
          path: ~/.ccache
          key: ccache-${{ runner.os }}-${{ hashFiles('configure.ac') }}

      - name: Install build dependencies (proven list from codeql-analysis.yml)
        run: |
          sudo apt-get install -y libunwind-dev build-essential git libcap-dev \
            python3-polib libpng-dev python3-lxml libpam-dev libzstd-dev \
            libssl-dev libcppunit-dev wget cmake

      - name: Build system POCO (required — ENGINE_ASSETS ships no workdir/, Pitfall 1)
        run: |
          wget https://pocoproject.org/releases/poco-1.12.5p2/poco-1.12.5p2-all.tar.gz
          tar xf poco-1.12.5p2-all.tar.gz && cd poco-1.12.5p2-all
          ./configure --static --no-tests --no-samples --no-sharedlibs --cflags="-fPIC" \
            --omit=Zip,Data,Data/SQLite,Data/ODBC,Data/MySQL,MongoDB,PDF,CppParser,PageCompiler,Redis,Encodings,ActiveRecord,Prometheus,JWT
          make -j "$(nproc)"
          sudo make install

      - name: Fetch prebuilt engine assets (BUILD-01 fast path)
        run: |
          wget --no-verbose https://github.com/CollaboraOnline/online/releases/download/for-code-assets/engine-main-assets.tar.gz
          tar xf engine-main-assets.tar.gz -C engine && rm engine-main-assets.tar.gz

      - name: Configure and build
        env:
          CC: ccache gcc
          CXX: ccache g++
        run: |
          ./autogen.sh
          ./configure --enable-silent-rules --enable-debug \
            --with-lokit-path="${GITHUB_WORKSPACE}/engine/include" \
            --with-lo-path="${GITHUB_WORKSPACE}/engine/instdir"
          make -j "$(nproc)" build-nocheck

      - name: Smoke test (BUILD-04 — port 9980, never 8080)
        run: |
          ./coolwsd-systemplate-setup ./systemplate "${GITHUB_WORKSPACE}/engine/instdir"
          mkdir -p ./jails ./cache
          ./coolwsd --disable-ssl \
            --o:sys_template_path="${GITHUB_WORKSPACE}/systemplate" \
            --o:child_root_path="${GITHUB_WORKSPACE}/jails" \
            --o:cache_files.path="${GITHUB_WORKSPACE}/cache" &
          sleep 3
          curl -fsS http://127.0.0.1:9980/hosting/discovery
          curl -fsS http://127.0.0.1:9980/hosting/capabilities
```
This is a documented starting skeleton, not gospel — the planner should verify each step name/exact flag against the live repo state at plan time, but every step above is sourced from an in-tree file (`codeql-analysis.yml`, `docker/from-source/build.sh`, `Makefile.am`, `wsd/COOLWSD.cpp`) rather than invented.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|---|---|---|---|
| POCO installed as a separate system dependency, discovered via `--with-poco-includes`/`--with-poco-libs` | POCO built in-engine at `engine/external/poco` (1.15.3), auto-discovered via `--with-lo-builddir`'s workdir path | This fork's migration, documented in `dev-notes/poco-build.md` (undated but clearly in-tree/recent) | `--with-poco-*` flags are now no-ops; any CI recipe written before this migration (like `codeql-analysis.yml`) that still installs a system POCO is *not* stale — it's relying on the still-fully-supported system-POCO fallback, which is exactly what ENGINE_ASSETS-based builds must also do |
| `ubuntu-latest` defaulted to Node 20 LTS | `ubuntu-latest` (ubuntu-24.04 image) defaults to Node 22.x; Node 20 fully removed from toolcache | Rolled out 2026-05-19 to 2026-05-26 (Node 20 EOL 2026-04-30) | Any workflow assuming "whatever ubuntu-latest ships" satisfies "Node 20 LTS" is now wrong; explicit `actions/setup-node@v4` pinning is mandatory, not defensive-programming overkill |
| `codeql-analysis.yml` as the only existing CI workflow | A new, separate push-triggered `build.yml` | N/A — no push-triggered build workflow exists yet in this fork | BUILD-03 requires net-new CI, not a repurposed security-scanning job; keep `codeql-analysis.yml` as-is (it serves a different purpose and cadence) |

## Open Questions

1. **Does `core-co-25.04-assets.tar.gz` share the same instdir-only structure as `engine-main-assets.tar.gz`?**
   - What we know: naming differs (`core-` vs `engine-` prefix) and file size is similar (445,246,847 vs 433,550,261 bytes), suggesting a similarly-shaped artifact, but this was not independently downloaded/inspected in this research pass.
   - What's unclear: whether it also omits `workdir/` entirely (very likely, given it's built the same way upstream), and whether its internal `instdir/program/versionrc`-equivalent path matches.
   - Recommendation: Not a blocker for Objective 1 (eden-main tracks `main`). Spike this before Objective 5 finalizes any `co-25.04` branch-tracking recommendation.

2. **Are `autoconf`/`automake`/`libtool`/`pkg-config` actually preinstalled on `ubuntu-latest`?**
   - What we know: `codeql-analysis.yml`'s apt install list never mentions them, yet the workflow is understood to currently work, and `./autogen.sh` (which calls `libtoolize`, `aclocal`, `automake`, `autoreconf`) and configure's `PKG_CHECK_MODULES` calls (zlib, cppunit, openssl) both depend on these tools being present.
   - What's unclear: this wasn't independently confirmed against GitHub's documented default runner toolset in this research pass.
   - Recommendation: MEDIUM confidence this holds (consistent with GitHub's documented general-purpose build toolset on `ubuntu-latest`). Make this the first thing the new workflow validates — if wrong, it's a one-line `apt-get install autoconf automake libtool pkg-config` fix, cheap to add defensively even if likely unnecessary.

3. **Checksum/reproducibility strategy for the ENGINE_ASSETS download.**
   - What we know: no `.sha256`/`.asc` sidecar asset is published under the `for-code-assets` release tag (confirmed via the GitHub Releases API); the artifact is rebuilt on a rolling basis (same-day rebuild observed).
   - What's unclear: whether the project wants any pinning at all beyond "trust HTTPS + GitHub," given the artifact is intentionally a moving target that tracks upstream `main`.
   - Recommendation: Don't hard-pin a fixed hash (would fossilize against a stale artifact). Log the downloaded sha256 in CI output for drift-visibility/corruption-detection instead.

4. **POCO version skew: engine's vendored 1.15.3 vs. the fallback system build's 1.12.5p2.**
   - What we know: both satisfy configure's `>=1.12.0` floor and expose the headers coolwsd needs (`Poco::Net::WebSocket`, etc.); this exact skew is already present and working in `codeql-analysis.yml` today.
   - What's unclear: whether any coolwsd code path relies on POCO behavior/APIs introduced between 1.12.5 and 1.15.3 that would only surface as a runtime bug, not a compile error.
   - Recommendation: LOW-MEDIUM confidence this is harmless (no divergent behavior documented or observed). Note for awareness; not a reason to block or delay BUILD-01/03.

## Sources

### Primary (HIGH confidence — direct repo inspection + live empirical artifact verification)
- `/Users/justin/dev/EdenDocs/configure.ac` — POCO auto-discovery (1795-1811), header check + version floor (2202-2239), `--with-poco-*` obsolete handling (365-391), Node/npm floor (2291-2323), C++20 requirement (1658-1686), `--with-lo-path` sanity check (1714-1793), `AC_CONFIG_FILES` for `browser/npm-shrinkwrap.json` (2714)
- `/Users/justin/dev/EdenDocs/.github/workflows/codeql-analysis.yml` — full file, the only proven-working CI recipe in-tree
- `/Users/justin/dev/EdenDocs/docker/from-source/build.sh`, `docker/from-source/README.md`, `docker/from-source/Ubuntu` — `ENGINE_ASSETS` mechanic, glibc-only caveat, runtime entrypoint/port
- `/Users/justin/dev/EdenDocs/dev-notes/poco-build.md` — POCO-in-engine migration writeup, explicitly documents `codeql-analysis.yml`'s system-POCO-fallback special case
- `/Users/justin/dev/EdenDocs/Makefile.am` — `browser/node_modules` npm-offline rule (980-981), `build-nocheck` target (854), `run:`/`setup-wsd`/`COMMON_PARAMS` targets (647-778)
- `/Users/justin/dev/EdenDocs/browser/Makefile.am`, `browser/package.json`, git-tracked `browser/node_shrinkpack/` (866 files, confirmed via `git ls-files`), `browser/npm-shrinkwrap.json.in`
- `/Users/justin/dev/EdenDocs/wsd/COOLWSD.cpp` — fatal `sys_template_path`/`child_root_path` checks, self-signed cert auto-gen, `--disable-ssl`, `--probe`/`probeRunningServer()`
- `/Users/justin/dev/EdenDocs/wsd/ClientRequestDispatcher.cpp` — `/hosting/discovery`, `/hosting/capabilities` routes
- `/Users/justin/dev/EdenDocs/common/Common.hpp:28` — `DEFAULT_CLIENT_PORT_NUMBER = 9980`
- `/Users/justin/dev/EdenDocs/coolwsd.xml.in` — `file_server_root_path` default, `ssl.enable` default
- `/Users/justin/dev/EdenDocs/autogen.sh` — root autotools toolchain invocation
- `/Users/justin/dev/EdenDocs/scripts/clone-online.md` — from-scratch engine build cost baseline (30GB, 1-3 hours)
- Empirical: live download + `tar tzv` listing of `https://github.com/CollaboraOnline/online/releases/download/for-code-assets/engine-main-assets.tar.gz` (2026-07-07, 433,550,261 bytes, Last-Modified 2026-07-07T17:55:52Z) — confirmed instdir-only structure, zero headers, zero `workdir/`
- Live GitHub Releases API query, `api.github.com/repos/CollaboraOnline/online/releases/tags/for-code-assets` (2026-07-07) — full current asset inventory, including the `core-co-25.04-assets.tar.gz` naming discrepancy

### Secondary (MEDIUM confidence — WebSearch, cross-verified with GitHub's own changelog/issue tracker)
- [actions/runner-images#14029 — Default Node.js version changed from 20 to 22, Node 20 removed from toolcache](https://github.com/actions/runner-images/issues/14029)
- [GitHub Changelog: Deprecation of Node 20 on GitHub Actions runners](https://github.blog/changelog/2025-09-19-deprecation-of-node-20-on-github-actions-runners/)
- [actions/runner-images — Ubuntu 24.04 readme](https://github.com/actions/runner-images/blob/main/images/ubuntu/Ubuntu2404-Readme.md)

### Tertiary (LOW confidence)
- None relied upon — all findings above trace to either in-tree source or a live, dated verification.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — every version floor and dependency is a hard-coded check in `configure.ac`, cross-checked against the proven `codeql-analysis.yml` recipe
- Architecture (ENGINE_ASSETS + POCO fallback): HIGH — verified empirically against the live tarball, not inferred from documentation alone
- CI/Node/ccache gaps: HIGH for the in-tree gaps (direct file inspection); MEDIUM-HIGH for the Node-20-removal timeline (WebSearch, cross-verified against GitHub's own issue tracker and changelog, dated within the last two months)
- Smoke test recipe: HIGH — every piece (port, routes, config requirements, `--probe`) traced to exact source lines

**Research date:** 2026-07-07
**Valid until:** ~2026-08-07 for the CI/runner-image specifics (fast-moving — GitHub Actions runner defaults have already changed once in 2026; re-verify Node version pinning if this research is used after that date). The in-tree architectural findings (POCO auto-discovery behavior, `ENGINE_ASSETS` mechanic, port/route facts) are stable and tied to this fork's own source, not external drift — valid until this fork's build system changes.
