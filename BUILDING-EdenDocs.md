# Building EdenDocs

This is EdenDocs' own build documentation — additive to upstream's
[README.md](README.md), [CONTRIBUTING.md](CONTRIBUTING.md), and
[docker/from-source/README.md](docker/from-source/README.md), none of which
are edited by this doc. It describes exactly one build: the path
`scripts/eden/*.sh` implement and `.github/workflows/build.yml` proves green
on every push to `eden-main`. If anything here ever disagrees with those
scripts, the scripts are right — file a fix against this doc, not the other
way around.

## 1. Overview

EdenDocs builds two artifacts from source:

- **`coolwsd`** — the C++ document server (this repository's root autotools
  project).
- **`browser/dist`** — the JS/CSS web client bundle.

**Fast-path philosophy.** EdenDocs (like upstream Collabora Online) embeds a
LibreOffice-derived rendering engine ("the engine", under `engine/`). Building
that engine from scratch takes **1–3 hours and ~30GB of disk** and is
explicitly **out of scope** for this pipeline. Instead, the documented build
downloads Collabora's own prebuilt engine tarball
(`engine-main-assets.tar.gz`) and links `coolwsd` against it. This is the
same trick `docker/from-source/build.sh` uses via its `ENGINE_ASSETS` env var
— see Section 3.

**Target platform.** These instructions target **Linux (Ubuntu 24.04, or a
container thereof)** — this is what `scripts/eden/build-deps.sh` installs
packages for and what `.github/workflows/build.yml` runs on
(`ubuntu-latest`). **macOS is dev-convenience only** — there is no macOS
build path documented here. macOS users should either build inside a Linux
container/VM or rely on CI (push to `eden-main` and read the Actions log).

**Port rule (hard project rule).** **Never use port 8080** for anything in
this build or its verification — it is permanently occupied by another
application on the primary dev machine and must never be bound, served,
curled, or referenced. `coolwsd` listens on its own **native port 9980**
(Section 7); if you need an incidental local web server for anything
unrelated to this pipeline, use **8091** instead. No script or command in
this document uses 8080.

## 2. Quick start

Run these four scripts, in order, from the repository root. This is the
canonical path — it is exactly what CI runs:

```bash
./scripts/eden/build-deps.sh           # apt deps + POCO 1.12.5p2 from source (sudo)
./scripts/eden/fetch-engine-assets.sh  # prebuilt engine tarball → engine/ (sha256 logged)
./scripts/eden/build.sh                # autogen + configure + make
./scripts/eden/smoke-test.sh           # coolwsd serves /hosting/* on 9980
```

**Prerequisite: Node.js 20 LTS.** The root `configure` hard-requires Node
`>= 20` to build `browser/`. Install it with `nvm` or `n` locally (e.g.
`nvm install 20 && nvm use 20`) before running `build.sh`. Do **not** rely on
Ubuntu's `apt install npm` — the `npm` package pulled in by
`build-deps.sh`'s apt list is a dependency-resolution convenience only and
typically drags in an older `nodejs`; it is not a substitute for Node 20.
CI pins Node 20 explicitly via `actions/setup-node@v4` (see
`.github/workflows/build.yml`) — match that locally.

Each script is idempotent and safe to re-run:
`fetch-engine-assets.sh` skips re-downloading if `engine/instdir` already
exists (`FORCE_REFETCH=1` to force), and `build-deps.sh` skips rebuilding
POCO if it detects a complete install already at `/usr/local`.

## 3. The engine tarball fast path (BUILD-01)

`scripts/eden/fetch-engine-assets.sh` downloads
`https://github.com/CollaboraOnline/online/releases/download/for-code-assets/engine-main-assets.tar.gz`
and extracts it into `engine/`. This is the **`for-code-assets`** release
tag — a **rolling, unversioned** tag that upstream rebuilds on every push to
its `main` branch. It is the same mechanism
`docker/from-source/build.sh`'s `ENGINE_ASSETS` env var documents ("URL of
prebuilt engine assets tarball (skips building engine from source)").

**What's in the tarball, and why `engine/include` still resolves.** The
tarball is a packaging of `make instdir` — it contains **only** a compiled
`instdir/` (binaries, libraries, resources), **not** the engine's source
tree or its `workdir/` build intermediates. `engine/include` still works as
a `--with-lokit-path` target because those headers are **git-tracked
source** already present in this repository's `engine/` submodule/subtree —
the tarball supplies the missing compiled half (`instdir/`), not the
headers.

**Configure flags used** (exact — see `scripts/eden/build.sh`):

```
--with-lokit-path="$PWD/engine/include" \
--with-lo-path="$PWD/engine/instdir" \
--enable-silent-rules --enable-debug
```

`--with-lokit-path` points at the git-tracked LOK headers; `--with-lo-path`
points at the fetched `instdir/`. `--enable-debug` is not cosmetic: it arms
`configure`'s `--with-lo-path` sanity check against `instdir/program/versionrc`
(the same file `fetch-engine-assets.sh` asserts exists post-extract) — without
`--enable-debug` a bad or empty `--with-lo-path` can pass configure silently.

**Checksum stance.** `fetch-engine-assets.sh` runs `sha256sum` on the
downloaded tarball and logs the digest — it does **not** verify against a
hardcoded value, and this document does not publish one either. `for-code-assets`
has no published sidecar checksum and is rebuilt continuously; pinning a hash
here would fossilize the build against a stale artifact and break every
future fetch. The logged digest exists purely for traceability (e.g. "which
exact engine build did this coolwsd link against" when debugging a report).
Example from a real run: tarball size 433,550,261 bytes,
`sha256 54091655ab4f311854770a46033abe11a9eb156f9d320ebc4e464c5437531c09` (CI
run [28910753718](https://github.com/AO-Cyber-Systems/EdenDocs/actions/runs/28910753718)
— expect a different digest on your own fetch, since upstream rebuilds this
tag continuously).

## 4. POCO strategy and fallback (BUILD-02)

This is the part of the build most likely to surprise a developer coming from
upstream's own `dev-notes/poco-build.md`, so read this section fully before
filing a bug.

**Upstream's normal (in-tree engine build) story.** Per
`dev-notes/poco-build.md`, when you build the engine from source, POCO is
built as one of the engine's own third-party libraries
(`engine/external/poco/`) and `configure` finds it automatically via
`--with-lo-builddir` (which defaults to `engine` for in-tree builds) — "a
normal in-tree build needs no POCO options at all."

**Why that auto-discovery cannot work here.** `--with-lo-builddir`'s POCO
auto-discovery looks for POCO's build output under the engine's `workdir/`
(headers at `workdir/UnpackedTarball/poco/include`, archives at
`workdir/LinkTarget/StaticLibrary/libPoco*.a`). The prebuilt
`engine-main-assets.tar.gz` ships **zero** `workdir/` — it is, as Section 3
describes, a packaging of `make instdir` only. This was empirically verified
by directly inspecting the live tarball (2026-07-07: 5,881 entries, 100%
under `instdir/`, zero `.h` files, zero `poco`/`workdir` matches
case-insensitive) — see `.planning/objectives/01-from-source-build-pipeline/01-01-SUMMARY.md`
for the full evidence table. Given this, **POCO auto-discovery against the
tarball fast path deterministically fails, every time, by construction** —
it is not flaky, not version-dependent, not something a retry fixes.

`scripts/eden/build.sh` asserts this as a **permanent, expected-SUCCESS
invariant** rather than treating it as an error: it greps the configure log
for the exact line

```
POCO not found in the engine workdir, falling back to system POCO
```

and hard-fails the script if that line is *absent* (i.e. if configure
*didn't* fall back the way this pipeline requires). **If you see that line in
your own build output, that is correct — do not "fix" it.**

**The required fallback: source-built POCO 1.12.5p2.** Since auto-discovery
cannot work against the tarball, `configure` falls back to looking for a
**system-installed** POCO, which `scripts/eden/build-deps.sh` builds from
source and installs to `/usr/local` (recipe adapted from
`.github/workflows/codeql-analysis.yml`, the only other in-tree CI job that
builds against `ENGINE_ASSETS`). Run `build-deps.sh` **before** `build.sh` —
POCO must be installed before `configure` runs, or you hit Pitfall 1 below.

**Pitfall 1 — apt's `libpoco-dev` is too old.** `configure.ac` (lines
2231–2239) hard-requires POCO **>= 1.12.0**. Ubuntu 24.04's apt package
`libpoco-dev` ships **1.11.0** — below the floor. `configure` will hard-error
if you try to satisfy the POCO dependency via `apt install libpoco-dev`
instead of the source build. This is why `build-deps.sh` always
source-builds POCO 1.12.5p2 rather than reaching for apt.

**Pitfall 2 — `--with-poco-includes`/`--with-poco-libs` are obsolete
no-ops.** Both `configure.ac` (lines 365–391) and upstream's own
`dev-notes/poco-build.md` confirm: these flags are accepted for backward
compatibility but **silently ignored** (warning only) — "there is no longer
any `/opt/poco`." Passing them does not pin a POCO location and cannot
substitute for a real system install at the paths the linker searches by
default (`/usr/local/include`, `/usr/local/lib`). Do not reach for these
flags when troubleshooting a POCO problem; they will not help.

**Zip module note.** `scripts/eden/build-deps.sh` builds POCO **with** the
`Zip` module (unlike the `codeql-analysis.yml` recipe it's adapted from,
which omits it). Upstream `main`'s `wsd/Unzip.cpp` includes
`Poco/Zip/Decompress.h`; omitting `Zip` fails the build at `wsd/Unzip.o`
(proven in CI run 28906749089). If you're comparing this build's POCO
`--omit=` list against `codeql-analysis.yml`'s, this is the one deliberate
divergence.

For background on why upstream migrated POCO into the engine build in the
first place (the in-tree-build story this fast path bypasses), see
`dev-notes/poco-build.md`.

## 5. ccache

The root `configure.ac` has **no Linux auto-detection for ccache** (its only
ccache-adjacent logic is a Windows/CODA path). `scripts/eden/build.sh` wires
it in externally: if `ccache` is installed and the caller hasn't already
exported `CC`/`CXX`, it sets

```bash
export CC="ccache gcc" CXX="ccache g++"
```

before running `autogen.sh`/`configure`. `build-deps.sh` installs `ccache`
via apt. Locally this just needs a warm `~/.ccache` across builds to pay off;
CI persists `/home/runner/.ccache` across runs via `actions/cache@v4` keyed
on `configure.ac`'s hash (see Section 8).

## 6. Browser build

`scripts/eden/build.sh` runs `make -C browser` explicitly after the main
`make build-nocheck` (see Section 8 for why that second `make` call is
needed at all). Browser's `node_modules` rule runs `npm ci --offline`
against **fully vendored** packages already checked into
`browser/node_shrinkpack/` (866 packages) — there is **no network/registry
access** during this step and none is needed. Do **not** run
`npm install` or `npm update` inside `browser/` — that would attempt to
reach the npm registry, is unnecessary, and can desync the vendored
shrinkpack.

## 7. Smoke test (BUILD-04)

`scripts/eden/smoke-test.sh` proves the built `coolwsd` actually runs and
serves requests, on its **native port 9980** — never 8080.

Steps, in order:

1. **Grant file capabilities.** `coolwsd`'s forkit process needs Linux file
   capabilities to chroot/bind-mount its jails. Without them the forkit dies
   silently and `coolwsd` never listens. The script grants exactly what
   production images grant
   (`docker/from-packages/scripts/assemble-rootfs.sh:194-195`):
   ```bash
   sudo setcap cap_fowner,cap_chown,cap_sys_chroot=ep ./coolforkit-caps
   sudo setcap cap_sys_admin=ep ./coolmount
   ```
   This requires `sudo` (passwordless on GitHub-hosted runners; you may need
   to enter your password locally). The step is skipped automatically if
   `setcap`/`sudo` aren't present.
2. **Bootstrap the systemplate** via the canonical in-tree tool
   `./coolwsd-systemplate-setup ./systemplate "$PWD/engine/instdir"`
   (the same tool `Makefile.am`'s `setup-wsd`/`run` targets use).
3. **Launch `coolwsd`** in the background with `--disable-ssl` and the
   fatal-required config keys (`sys_template_path`, `child_root_path`,
   `cache_files.path`, `storage.filesystem[@allow]=true`, admin
   credentials) — modeled on `Makefile.am`'s `COMMON_PARAMS`/`run` recipe,
   minus the SSL cert options since this smoke test disables SSL.
4. **Wait for readiness** using `coolwsd`'s own built-in self-check,
   `./coolwsd --probe`, retried for up to 120 seconds (cold CI starts can be
   slow).
5. **Curl the two BUILD-04 endpoints on port 9980**:
   ```bash
   curl -fsS http://127.0.0.1:9980/hosting/discovery    | grep -q 'wopi-discovery'
   curl -fsS http://127.0.0.1:9980/hosting/capabilities | grep -q 'convert-to'
   ```

**Expected output** on success:

```
SMOKE TEST PASSED: coolwsd serves /hosting/discovery and /hosting/capabilities on 9980
```

On failure, the script tails `/tmp/coolwsd.log` (dev/debug builds log there
per `coolwsd.xml`) before exiting non-zero, since the step's own stdout can
be blind to `coolwsd`'s internal startup errors.

## 8. CI (BUILD-03)

`.github/workflows/build.yml` runs this exact script path
(`build-deps.sh` → `fetch-engine-assets.sh` → `build.sh` → `smoke-test.sh`)
on every `push` to `eden-main` (and on manual `workflow_dispatch`), on
`ubuntu-latest`, with Node 20 pinned via `actions/setup-node@v4` and ccache
persisted via `actions/cache@v4` (`CCACHE_DIR=/home/runner/.ccache`, cache
key hashes `configure.ac`).

Real timings from the two green runs that proved this pipeline
(01-02-SUMMARY.md):

| Run | Trigger | ccache state | Wall clock |
|---|---|---|---|
| [28910753718](https://github.com/AO-Cyber-Systems/EdenDocs/actions/runs/28910753718) | push | cold (no cache hit) | 14m28s |
| [28911376952](https://github.com/AO-Cyber-Systems/EdenDocs/actions/runs/28911376952) | workflow_dispatch | warm (cache hit) | 8m19s — ccache 336/336 hits (100.0%) |

**`.github/workflows/codeql-analysis.yml` is a separate workflow** —
schedule-only security scanning (CodeQL), not this build. It happens to
share some of the same apt/POCO recipe ancestry (see Section 4), but it only
needs the C++ compile step to succeed for CodeQL's analysis; it never builds
`browser/dist` and never runs a smoke test. Do not confuse its green status
with this pipeline's.

**Why `make -C browser` is a separate, explicit step.** `make build-nocheck`
maps to automake's `all-am` target, which builds only the top-level
directory and its `all-local` hooks — it does **not** recurse into
`SUBDIRS`, so `browser/` (and therefore `browser/dist`) is never built by it
alone. `codeql-analysis.yml` never notices this because CodeQL only needs
the C++ compile. `scripts/eden/build.sh` runs `make -C browser` explicitly
right after `make build-nocheck` to produce `browser/dist`.

## 9. Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| `configure: error: header Poco/Net/WebSocket.h not found` | POCO not installed before `configure` ran | Run `scripts/eden/build-deps.sh` before `scripts/eden/build.sh` |
| `configure: error: ... POCO version is too old` (or similar version check failure) | `apt`'s `libpoco-dev` (1.11.0) was used instead of the source build | Do not `apt install libpoco-dev`; let `build-deps.sh` source-build POCO 1.12.5p2 to `/usr/local` |
| `wsd/Unzip.cpp:20: fatal error: Poco/Zip/Decompress.h: No such file or directory` | POCO was built with `Zip` omitted (e.g. a stale/manual recipe copied from `codeql-analysis.yml`) | Rebuild POCO via the current `scripts/eden/build-deps.sh`, which does **not** omit `Zip` |
| `cannot find -lPocoFoundation` (or other `-lPoco*`) at link time | `LDFLAGS=-L/usr/local/lib` missing at configure — debug builds auto-pick a fast linker (lld/gold) that, unlike `bfd` ld, does not search `/usr/local/lib` by default | Ensure `configure` is invoked with `LDFLAGS="-L/usr/local/lib"` (as `scripts/eden/build.sh` does) |
| `Capabilities are not set for the coolforkit program` | `coolforkit-caps`/`coolmount` lack the required Linux file capabilities, so the forkit dies at startup | Run `scripts/eden/smoke-test.sh`'s `setcap` step (needs `sudo`): `sudo setcap cap_fowner,cap_chown,cap_sys_chroot=ep ./coolforkit-caps && sudo setcap cap_sys_admin=ep ./coolmount` |
| `coolmount: mount failed ... Operation not permitted` | Same root cause as above — `coolmount` missing `cap_sys_admin` | Same `setcap` fix as above |
| Fetch of `engine-main-assets.tar.gz` returns 404 | Asset naming varies per branch (e.g. `engine-main-assets.tar.gz` for `main`-tracking branches vs. `core-co-25.04-assets.tar.gz`-style names for release branches) | Set `ENGINE_ASSETS_URL` to the correct asset for the branch you're building; see `scripts/eden/fetch-engine-assets.sh` |
| `smoke-test.sh` times out waiting for `--probe` | `coolwsd` failed to start or is stuck | Check `/tmp/coolwsd.log` — the script tails it automatically on failure |
| `browser/dist` missing after a build that otherwise looked successful | `make build-nocheck` alone (`all-am`) never recurses into `SUBDIRS`, so `browser/` is never built by it | Run `make -C browser` explicitly (already done by `scripts/eden/build.sh`) |
| `configure` fails on a Node version check, or `browser/` build fails with a Node-related error | Node < 20 active in the shell | Activate Node 20 LTS (`nvm use 20` / `n 20`) before running `scripts/eden/build.sh` |
