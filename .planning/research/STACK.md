# Stack Research

**Domain:** Self-hosted collaborative office suite — Collabora Online (CODE) fork, brownfield/from-source build + rebrand + OIDC + container image pipeline
**Researched:** 2026-07-07
**Confidence:** HIGH (repo-grounded) for build toolchain and versions; MEDIUM for CI/runner sizing and OIDC integration pattern (community-sourced, not upstream-documented); LOW→flagged where upstream's own artifacts are internally inconsistent (see "What NOT to Use")

All findings below are grounded in the actual EdenDocs tree at `/Users/justin/dev/EdenDocs` (branch `eden-main`, forked from `upstream/main` at 0 commits behind, version `26.04.2.1` per `configure.ac:6` and `engine/configure.ac:21`) plus the live GitHub state of `CollaboraOnline/online` (the separate GH-Actions/release repo — see "Two Upstream Repos" below). File paths are cited so the roadmap/objective researchers can re-verify.

## Two Upstream Repos — Read This First

There are **two different upstream GitHub repos**, and confusing them is the single biggest way to waste time on this fork:

| Repo | Role | What lives there |
|------|------|-------------------|
| `CollaboraOnline/online.mirror` | Read-only mirror of the real source (Gerrit `gerrit.collaboraoffice.com`, project `online`). **This is what EdenDocs forked (`git remote -v` → `upstream`).** | Full monorepo: `wsd/`, `kit/`, `browser/`, `common/`, and — critically — `engine/` (LibreOffice core/LOKit, ~73K files, in-tree, not a submodule). `docker/from-source/` and `docker/from-packages/` live here. |
| `CollaboraOnline/online` | A **separate** GitHub-native repo used only for issue tracking + GitHub Actions artifacts (nightly Docker image, Helm chart, prebuilt `engine-*-assets.tar.gz` release blobs). PRs against it are for Helm/Docker only; source PRs are rejected (see `CONTRIBUTING.md:37-41`). | `.github/workflows/docker.yml` (nightly build), `.github/workflows/helm-release.yml`, `docker/from-source-gh-action/` (Dockerfile + build.sh), `kubernetes/helm/`. |

Confirmed by commit `4f41f82d7bc` ("Remove Helm chart and GitHub-action docker build") in the local tree (`git log --all -- docker/from-source-gh-action`), whose message states verbatim: *"docker/from-source-gh-action/ is built by its nightly docker workflow [...] these artifacts are now maintained and released exclusively from the GitHub repository at github.com/CollaboraOnline/online."* **`docker/from-source-gh-action/` does NOT exist in this tree** — do not go looking for it locally; it only exists in the sibling `CollaboraOnline/online` repo, and (see "What NOT to Use") its current contents there are stale relative to this tree's actual build.

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| GNU Autoconf/Automake | Autoconf ≥2.63 floor (`AC_PREREQ([2.63])`, root `configure.ac:4`), Automake ≥1.10 floor (`configure.ac:8`) | Build system for both `coolwsd` (root) and `engine/` (LibreOffice core) | The floors are ancient; any current Ubuntu 24.04/Debian 12 autoconf (2.71) and automake (1.16) satisfy them. Two separate `configure.ac`/`autogen.sh` pairs exist — root (coolwsd) and `engine/` (LO core) — and must be run in that order: engine first, then root (`README.md:112-125`). |
| GCC 12, or Clang 18 + libstdc++11 | Linux build baseline | C/C++ compiler for `engine/` and `wsd/`/`kit/`/`common/` | Stated explicitly in `engine/README.md:31-44` ("The Build Chain and Runtime Baselines... Linux: Build: either GCC 12; or Clang 18 with libstdc++ 11"). Root `configure.ac` additionally requires a C++20-capable compiler (comment at `configure.ac:96`, "the C++20 check"). Ubuntu 24.04 ships GCC 13, comfortably above floor — use the `docker/from-source/Ubuntu` base image (`ubuntu:24.04`) as the build/runtime OS for exactly this reason. |
| Node.js ≥ 20.0.0, npm ≥ 9.0.0 | `browser/` JS/TS build (esbuild-wasm, TypeScript, mocha tests, sass for branding) | Root `configure.ac:2295-2312` hard-fails configure if `node --version` < 20.0.0 or `npm -v` < 9.0.0. Upstream's own (separate-repo) nightly-build Dockerfile installs Node **20** via `deb.nodesource.com/setup_20.x` — confirms 20.x LTS, not 22.x, is what Collabora actually builds/tests against today. **Recommendation: pin Node 20 LTS in EdenDocs CI**, not a newer major, to match upstream's tested baseline exactly (avoids esbuild-wasm/TypeScript/browserify toolchain surprises that only upstream would have already hit). | 
| POCO C++ Libraries | **1.15.3**, built in-tree (`engine/external/poco/`) — NOT a system package | HTTP/WebSocket/JSON/crypto library used by `wsd/` | Confirmed via `engine/download.lst:562-563` (`POCO_TARBALL := poco-1.15.3-all.tar.bz2`) and `dev-notes/poco-build.md`. As of this fork's tree, POCO is **no longer** a separately-installed system dependency (`libpoco-dev` is obsolete) — it's compiled as one of the engine's third-party static libs (PocoFoundation/XML/JSON/Util/Net always; PocoCrypto/NetSSL on server platforms only) and discovered automatically via `--with-lo-builddir` (defaults to `engine`). `--with-poco-includes`/`--with-poco-libs` are accepted but silently ignored. **This is a major departure from "standard Collabora Online build" tutorials/blog posts you'll find online, which almost universally assume `apt install libpoco-dev` — those instructions are stale for this codebase.** |
| LibreOffice core / LOKit ("engine/") | Version-locked to coolwsd: **26.04.2.1** (`engine/configure.ac:21`, `AC_INIT([Collabora Office],[26.04.2.1]...)`) | Rendering engine coolwsd's `kit/` processes load via LOKit | Built via `cd engine && ./autogen.sh --with-distro=CPLinux-LOKit[-Dev] && make`, then root `./configure --with-lokit-path=.../engine/include --with-lo-path=.../engine/instdir`. See "Where Prebuilt Engine Assets Come From" below — do NOT assume you must compile this from scratch for every CI run. |
| Docker / BuildKit | `DOCKER_BUILDKIT=1` | Container image assembly | Required for the `--mount=type=secret` flow used by `docker/from-packages/Dockerfile` (Collabora Partner package builds — irrelevant to EdenDocs, which is from-source) and for `--mount=type=cache` in any custom Dockerfile you write for CI layer caching. `docker/README:11-15` documents this explicitly. |

### Where Prebuilt LibreOffice-Core Artifacts Come From

This is the highest-leverage finding for CI design. Upstream publishes prebuilt `engine/instdir` tarballs as GitHub Release assets on `CollaboraOnline/online` (the GH-native repo, NOT `.mirror`), under the permanent release tag `for-code-assets`:

```
https://github.com/CollaboraOnline/online/releases/download/for-code-assets/engine-main-assets.tar.gz
```

Verified live via GitHub API on 2026-07-07 (`api.github.com/repos/CollaboraOnline/online/releases/tags/for-code-assets`):

| Asset | Size | Last updated | Branch it tracks |
|-------|------|---------------|-------------------|
| `engine-main-assets.tar.gz` | ~434 MB | **2026-07-07T17:55:52Z** (same day as this research — actively, continuously rebuilt) | `main` |
| `core-co-25.04-assets.tar.gz` | ~445 MB | 2026-07-07T02:49:53Z | `distro/collabora/co-25.04` |
| `node_modules-26.04.1-1.tar.xz` | ~24 MB | 2026-06-01 | prebuilt `browser/node_modules` cache for the 26.04.1-1 line |
| `LibreOfficeKit-includes-co-25.04.tar.gz` | 31 KB | 2026-07-07 | headers-only, co-25.04 |

**Critical gap: there is no `engine-co-26.04-assets` (or similarly named) tarball.** Only `main` and `co-25.04` have prebuilt engine-instdir assets. Since EdenDocs's `eden-main` is forked directly from `upstream/main` (0 commits behind — verified via `git merge-base eden-main origin/main` == `origin/main` HEAD), and `main` is presently versioned `26.04.2.1` (same as this tree), **`engine-main-assets.tar.gz` is the correct, currently-fresh artifact to consume** — this is fortunate, not a coincidence to rely on going forward: if EdenDocs ever rebases onto `distro/collabora/co-26.04` instead (11 commits ahead of the fork point, per `git rev-list --count`), there is currently no matching prebuilt asset and engine would have to be built from source until Collabora publishes one.

**Constraint:** these prebuilt tarballs are glibc-built (per `docker/from-source/README.md:7-9`: *"musl needs the engine to be built from source — the prebuilt engine assets are glibc-only"*). If you use `ENGINE_ASSETS`, your runtime/build container **must** be glibc-based (Ubuntu/Debian/Fedora) — Alpine is out unless you compile the engine yourself via `build-alpine.sh`.

### CI Recipe — Proven Working, In-Tree Today

`.github/workflows/codeql-analysis.yml` (present in this exact repo, runs weekly + on dispatch) is the **only currently-working example in this tree** of building `coolwsd` in GitHub Actions against a prebuilt engine tarball, and it should be the starting template for EdenDocs's real build workflow:

```yaml
runs-on: ubuntu-latest
steps:
  - uses: actions/checkout@v3
  - run: |
      sudo apt install -y libunwind-dev build-essential git libcap-dev python3-polib npm \
        libpng-dev python3-lxml libpam-dev libzstd-dev libssl-dev libcppunit-dev wget cmake
      # POCO built standalone here — see "What NOT to Use" for why
      wget https://pocoproject.org/releases/poco-1.12.5p2/poco-1.12.5p2-all.tar.gz
      tar xf poco-1.12.5p2-all.tar.gz && cd poco-1.12.5p2-all
      ./configure --static --no-tests --no-samples --no-sharedlibs --cflags="-fPIC" \
        --omit=Zip,Data,Data/SQLite,Data/ODBC,Data/MySQL,MongoDB,PDF,CppParser,PageCompiler,Redis,Encodings,ActiveRecord,Prometheus,JWT
      make -j `nproc` && sudo make install
  - run: |
      wget --no-verbose https://github.com/CollaboraOnline/online/releases/download/for-code-assets/engine-main-assets.tar.gz
      tar xf engine-main-assets.tar.gz -C engine && rm engine-main-assets.tar.gz
      ./autogen.sh
      ./configure --enable-silent-rules \
        --with-lokit-path=${GITHUB_WORKSPACE}/engine/include \
        --with-lo-path=${GITHUB_WORKSPACE}/engine/instdir \
        --enable-debug
      cd browser && npm update
  - run: make -j `nproc` build-nocheck
```
(`.github/workflows/codeql-analysis.yml:46-74`, transcribed)

Note it uses `--with-lokit-path`/`--with-lo-path` (pointing directly at the extracted tarball's `include`/`instdir`), **not** `--with-lo-builddir` — see "What NOT to Use" for exactly why this matters and what breaks if you "fix" it naively.

### Supporting Libraries (bundled in `engine/`, built automatically — do not install separately)

| Library | Version | Source | When it matters to EdenDocs |
|---------|---------|--------|------------------------------|
| OpenSSL | 3.0.15 (per `cool-sbom-template.spdx.json:31-32`) | `engine/external/openssl` | `PocoNetSSL`/`PocoCrypto` link against this; only relevant if you touch TLS/coolwsd SSL termination config. |
| zstd | 1.5.5 | `engine/external` | Used for tile/asset compression; no action needed. |
| libpng | 1.6.44 | `engine/external` | No action needed. |
| expat | 2.5.0 | `engine/external` (bundled, `--without-system-expat` in `distro-configs/CPLinux-LOKit.conf`) | `PocoXML` links this bundled expat — see POCO note above; do not let a system `libexpat` leak into the link. |
| double-conversion, pcre2 | 3.3.0, 10.42 | `engine/external`, compiled into Poco static libs | No action needed. |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| `ccache` | Compiler cache for both `engine/` and root builds | Root `configure.ac:1994-2007` and `engine/configure.ac:3244-3330` both auto-detect and use ccache transparently if present on `PATH` — no flag needed for a normal Linux build (`--disable-ccache` to opt out). `distro-configs/CPLinux-LOKit-Dev.conf` additionally sets `--enable-ccache-shared` for cross-worktree cache reuse. **For CI, persist `~/.ccache` (or `CCACHE_DIR`) via `actions/cache` keyed on `engine/` + root source hash** — this is the single biggest lever for cutting rebuild time on incremental PRs, since a from-scratch engine build is 1-3 hours per `scripts/clone-online.md:34` ("1 to 3 hours of patience the first time you build"), ~30 GB disk. |
| `sass` (Dart Sass, via `npm install -g sass`) | Compiles Collabora's brand CSS overlay | Only invoked in `docker/from-source/build.sh:119-124` when the (proprietary, Collabora-internal) `online-branding` repo is present. EdenDocs cannot clone `git@gitlab.collabora.com:productivity/online-branding.git` (no access), but **the seam it targets is exactly what EdenDocs's own rebrand should hook**: `brand.sh $INSTDIR/opt/collaboraoffice $INSTDIR/usr/share/coolwsd/browser/dist CODE` — i.e. a post-build step that overlays compiled CSS/images/JS into `browser/dist` without touching `browser/src`. |
| `clang-format` / `clang-tidy` | Code style / static analysis for C++ | `.clang-format` (root) pins style (4-space indent, `AccessModifierOffset: -4`); `.clang-tidy` enables `clang-diagnostic-*,performance-*,unused-*,misc-*,readability-redundant-*` with `WarningsAsErrors: '*'`. Relevant if EdenDocs' merge workflow runs clang-tidy in CI on `wsd/`/`common/` changes. |
| `cosign` | Container image signing | `docker/cosign.pub` is Collabora's own public key for verifying **their** official images — not reusable for EdenDocs images, but establishes the pattern: **EdenDocs should generate its own cosign keypair and sign branded images** the same way, for parity with the upstream supply-chain story (relevant given AO Cyber's privacy/trust positioning). |
| `clone-online.sh` / `scripts/clone-online.md` | Guided first-time engine+online build | Not something EdenDocs's CI needs (it's an interactive onboarding script for new contributors), but useful as a reference for exact build command sequencing and system package hints (`engine/README.md` linked from it) when writing EdenDocs's own build docs. |

## Installation

```bash
# System build deps (Ubuntu 24.04, matches docker/from-source/Ubuntu base image)
sudo apt-get update && sudo apt-get install -y \
  build-essential git libtool pkg-config autoconf gperf nasm xsltproc flex bison \
  ccache zip libcap-dev libpam-dev libzstd-dev libpng-dev libcppunit-dev \
  python3-polib python3-lxml npm libssl-dev cmake

# Node 20 LTS (matches configure.ac floor + upstream's own CI)
curl -fsSL https://deb.nodesource.com/setup_20.x -o nodesource_setup.sh
sudo bash nodesource_setup.sh && sudo apt-get install -y nodejs

# Engine (LibreOffice core / LOKit) — build from source OR skip via ENGINE_ASSETS
cd engine && ./autogen.sh --with-distro=CPLinux-LOKit-Dev && make -j$(nproc)
cd ..

# coolwsd + browser (root)
./autogen.sh --enable-developer
make -j$(nproc)
```

```bash
# CI fast path — skip the engine build entirely, use upstream's prebuilt tarball
wget --no-verbose https://github.com/CollaboraOnline/online/releases/download/for-code-assets/engine-main-assets.tar.gz
tar xf engine-main-assets.tar.gz -C engine && rm engine-main-assets.tar.gz
./autogen.sh
./configure --enable-silent-rules \
  --with-lokit-path="$PWD"/engine/include \
  --with-lo-path="$PWD"/engine/instdir \
  --disable-tests
make -j$(nproc)
```

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|--------------------------|
| Build engine from source once, cache via `ccache` + Docker layer cache | Always fetch `ENGINE_ASSETS` prebuilt tarball, never build engine locally | Use the tarball for **CI/PR builds** (fast, no LOKit changes expected). Build engine from source only when actually bumping the LibreOffice core version, patching `engine/`, or when no matching prebuilt asset exists for your target branch (currently true for `co-26.04`, see above). |
| GitHub-hosted `ubuntu-latest` runners for the coolwsd+browser build (using prebuilt `ENGINE_ASSETS`) | Self-hosted runners / GitHub "larger runners" for a full from-scratch engine compile | A full `engine/` build needs ~30 GB disk and 1-3 hours (`scripts/clone-online.md:34`); GitHub-hosted Linux runners guarantee only 14 GB free disk (actual ~22-29 GB after preinstalled tooling, per GitHub's own runner-images discussion #9329, WebSearch, MEDIUM confidence). **Only reach for self-hosted/larger runners if EdenDocs starts patching `engine/` itself** — for a rebrand + OIDC + coolwsd-config fork, the tarball path keeps you comfortably on standard runners. |
| `docker/from-source/{build.sh,Ubuntu}` (in this tree) as the basis for EdenDocs's own Dockerfile | Copying `docker/from-source-gh-action/` from `CollaboraOnline/online` verbatim | See "What NOT to Use" — that directory's current contents are stale relative to this tree's POCO-in-engine restructuring. Only reuse it as a *reference* for the runtime `apt` package list and `ENTRYPOINT`, not for the build recipe. |
| `--with-lokit-path`/`--with-lo-path` pointed at the tarball's `include`/`instdir` (matches working `codeql-analysis.yml`) | `--with-lo-builddir=engine` (the "clean"/native path per `dev-notes/poco-build.md`) | `--with-lo-builddir` is correct **only** when you actually built `engine/` yourself in-tree (POCO 1.15.3 + its workdir will exist). It is unverified whether `engine-main-assets.tar.gz` currently ships `workdir/UnpackedTarball/poco/include` and `workdir/LinkTarget/StaticLibrary/libPoco*.a` — `dev-notes/poco-build.md:185-191` explicitly flags this as an open gap even for Collabora's own CodeQL workflow. Don't assume it works without testing it first. |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|--------------|
| `apt install libpoco-dev` / `--with-poco-includes` / `--with-poco-libs` | These configure flags are **accepted but silently ignored** in the current tree (`dev-notes/poco-build.md:54`: *"--with-poco-includes and --with-poco-libs are obsolete: they are accepted but ignored (with a warning). There is no longer any /opt/poco."*). Most "how to build Collabora Online" blog posts and even upstream's own (separate-repo, GH-Actions-only) `docker/from-source-gh-action/Dockerfile` **still do this** — confirmed via WebFetch on 2026-07-07: that Dockerfile installs `libpoco-dev` via apt AND separately builds standalone POCO **1.12.5p2** from source, both of which are stale relative to this tree's in-tree POCO **1.15.3** (`engine/download.lst:562-563`). **This is a genuine, currently-unresolved version-drift between Collabora's two repos** — don't copy that Dockerfile's POCO handling as-is. | Let `--with-lo-builddir` (default `engine`) or the `codeql-analysis.yml` fallback (system POCO via configure's fallback path) resolve POCO automatically; if you must install a system POCO for a tarball-only CI path, pin it to what `codeql-analysis.yml` actually uses (1.12.5p2) since that's the one *proven* to still satisfy configure's fallback — but treat this as a temporary shim, not the target state. |
| Copying `docker/from-source-gh-action/` from `CollaboraOnline/online` (the GH-native repo) into EdenDocs verbatim | That repo/directory is maintained independently of Gerrit/this monorepo specifically to avoid "divergence" (per the deletion commit message) — but per the POCO finding above, it has itself already diverged from the actual current build (still uses standalone POCO 1.12.5p2 + `libpoco-dev`, Ubuntu 24.04 base, `ENGINE_ASSETS=.../engine-main-assets.tar.gz` default, `ONLINE_EXTRA_BUILD_OPTIONS=--enable-experimental`). Treating it as authoritative risks baking in a build recipe upstream itself hasn't reconciled yet. | Base EdenDocs's Dockerfile on the **in-tree** `docker/from-source/{build.sh,Ubuntu}` (source of truth for this fork) plus the proven `codeql-analysis.yml` configure invocation, and track the POCO drift as an open upstream inconsistency to re-check periodically (WebSearch/WebFetch `CollaboraOnline/online` before each major rebase). |
| Alpine base image + prebuilt `ENGINE_ASSETS` | The prebuilt tarballs are glibc-built; Alpine (musl) will not link against them (`docker/from-source/README.md:7-9`). | Ubuntu 24.04 / Debian stable / Fedora (matches the existing `docker/from-source/{Ubuntu,Debian,Fedora}` runtime Dockerfiles) if using `ENGINE_ASSETS`. Alpine is only viable via `docker/from-source/build-alpine.sh`, which compiles the engine from source inside the container (self-contained multi-stage build, no shortcut). |
| Building coolwsd's own OIDC/OAuth support | It doesn't exist. Verified by exhaustive grep across `wsd/`, `net/`, `common/`, and `coolwsd.xml.in` for `OIDC`/`OAuth`/`OpenID`/`SAML` — zero hits. The only auth-adjacent config in `coolwsd.xml.in` is `auth_key` (Collabora license key, unrelated), `enable_pam`/`jwt_expiry_secs` (admin console only, `coolwsd.xml.in:258,335`), and the WOPI `storage.wopi` allow-list (`coolwsd.xml.in:292-322`, server-to-server, not end-user auth). | See "OIDC/Reverse-Proxy Integration Points" below — OIDC belongs in the **WOPI host** (a separate Eden app that fronts coolwsd), not in coolwsd/`wsd/` itself. Do not try to bolt OIDC middleware into `wsd/COOLWSD.cpp`. |

## Stack Patterns by Variant

**If building EdenDocs CI for PR/branch validation (no engine changes):**
- Use `ENGINE_ASSETS=https://github.com/CollaboraOnline/online/releases/download/for-code-assets/engine-main-assets.tar.gz` + the `codeql-analysis.yml`-style `--with-lokit-path`/`--with-lo-path` configure invocation.
- Because it fits comfortably in standard GitHub-hosted `ubuntu-latest` runners (no 30 GB engine-build disk pressure, no 1-3 hour compile), and the tarball is refreshed same-day against `main` (which this fork tracks).

**If EdenDocs starts patching `engine/` itself (e.g., LOKit-level changes):**
- Build `engine/` from source, cache via `ccache` + a persisted Docker layer / `actions/cache` on `engine/workdir`, and switch to `--with-lo-builddir=engine` so POCO 1.15.3 resolves natively from the engine workdir.
- Because the prebuilt tarball path has an unresolved gap (POCO workdir completeness unverified) and no `co-26.04`-specific tarball exists yet — self-building removes both risks.

**If producing the final branded release/customer image:**
- Runtime base = `docker/from-source/Ubuntu` (or Debian/Fedora) pattern: copy the built `instdir` into a minimal runtime image, `setcap` the two binaries, run as non-root numeric UID, `EXPOSE 9980`. Do not use the `docker/from-packages/` path (Dockerfile) — that's Collabora's proprietary partner-package flow (`--secret id=secret_key`) and doesn't apply to a from-source fork.
- Add an EdenDocs-authored `cosign` signing step (own keypair, not Collabora's `docker/cosign.pub`) to match upstream's supply-chain pattern.

**If integrating OIDC (AOID) with EdenDocs:**
- OIDC login + session issuance happens in a **separate WOPI-host service** (not this repo). That service authenticates the user via AOID/OIDC, then mints a per-user, per-document, unguessable WOPI access token and hands the browser a URL like `https://eden-docs.example/browser/dist/cool.html?WOPISrc=...&access_token=...`.
- A reverse proxy (nginx, per `etc/nginx/coolwsd.conf` in this tree) sits in front of coolwsd and must **NOT** gate `/browser`, `/hosting/discovery`, `/hosting/capabilities`, `/cool/`, or `/lool/` behind an OIDC challenge — those are either static assets or server-to-server WOPI callback paths authenticated by the WOPI token + proof-key (`/etc/coolwsd/proof_key`, published at `/hosting/discovery`), not browser session cookies. Gate the WOPI-host's own document-picker/UI routes with OIDC instead. (MEDIUM confidence — pattern corroborated by Collabora forum threads and SDK docs via WebSearch, not by an official "OIDC + Collabora" guide; no single authoritative doc exists, so validate against AOID's actual token-issuance model during the OIDC objective.)
- `coolwsd.xml.in:232` documents `ssl.termination` (`"SSL off-load can be done in a proxy, if so disable SSL, and enable termination below in production"`) — confirms coolwsd expects to run behind a TLS-terminating proxy in production, consistent with this architecture.

## Version Compatibility

| Package A | Compatible With | Notes |
|-----------|------------------|-------|
| `engine-main-assets.tar.gz` (fetched fresh) | `eden-main` (forked from `upstream/main`, 0 commits behind) | Safe today because both track the same `main` line at version 26.04.2.1. **Re-verify this alignment before any rebase** — if EdenDocs ever moves to `distro/collabora/co-26.04`, there is currently no matching prebuilt tarball (only `main` and `co-25.04` exist as of 2026-07-07). |
| POCO 1.15.3 (in-tree, `engine/external/poco`) | POCO 1.12.5p2 (system, used by `codeql-analysis.yml`'s fallback path) | These are **not** the same build and upstream has not reconciled them (see "What NOT to Use"). Fine as an unrelated CI-only fallback for CodeQL scanning; do not assume feature/API parity if EdenDocs code ever touches Poco APIs directly. |
| Node 20.x LTS | npm ≥ 9.0.0 (root `configure.ac` floor) | Node 20 ships npm 10.x, comfortably above floor. Don't jump to Node 22/24 without testing `esbuild-wasm@^0.28.1`, `browserify@16.5.1`, and `jscpd@3.5.10` (all pinned in `browser/package.json`) against it first — none of upstream's own tooling has been observed running on anything but Node 20. |
| GCC 13 (Ubuntu 24.04 default) | Baseline "GCC 12" (`engine/README.md:44`) | Comfortably above floor; no action needed. |

## Sources

- `/Users/justin/dev/EdenDocs/configure.ac` (root, coolwsd) — version, Node/npm floor, ccache detection, C++20 comment — HIGH
- `/Users/justin/dev/EdenDocs/engine/configure.ac` — engine version (26.04.2.1), ccache detection — HIGH
- `/Users/justin/dev/EdenDocs/engine/README.md` — compiler/OS/Java/Python baselines — HIGH
- `/Users/justin/dev/EdenDocs/engine/download.lst` — POCO 1.15.3 pin — HIGH
- `/Users/justin/dev/EdenDocs/dev-notes/poco-build.md` — full POCO-in-engine migration rationale, CI/packaging impact, explicit CodeQL-workflow caveat — HIGH
- `/Users/justin/dev/EdenDocs/docker/from-source/{README.md,build.sh,Ubuntu,Debian,Fedora}` — in-tree from-source Docker build — HIGH
- `/Users/justin/dev/EdenDocs/docker/README` — build-from-packages vs from-source usage — HIGH
- `/Users/justin/dev/EdenDocs/.github/workflows/codeql-analysis.yml` — only proven working GH Actions build recipe in this tree — HIGH
- `/Users/justin/dev/EdenDocs/coolwsd.xml.in` — no OIDC/OAuth config present; `ssl.termination`, `storage.wopi`, admin `jwt_expiry_secs`/`enable_pam` — HIGH (exhaustive grep)
- `/Users/justin/dev/EdenDocs/etc/nginx/coolwsd.conf` — reference reverse-proxy path map — HIGH
- `/Users/justin/dev/EdenDocs/README.md`, `CONTRIBUTING.md`, `.gitreview` — Gerrit-is-canonical, two-repo split — HIGH
- `git log --all -- docker/from-source-gh-action` (local clone history) — deletion commit `4f41f82d7bc` and its message — HIGH
- GitHub API `api.github.com/repos/CollaboraOnline/online/releases/tags/for-code-assets` (fetched 2026-07-07) — asset names/sizes/timestamps for prebuilt engine tarballs — HIGH (live API data)
- WebFetch of `github.com/CollaboraOnline/online/docker/from-source-gh-action/Dockerfile` and `.github/workflows/docker.yml` (fetched 2026-07-07) — MEDIUM (WebFetch summarization of live GitHub blob pages, not a raw diff; corroborated by the POCO version cross-check against `dev-notes/poco-build.md`, so treated as reliable, but re-verify directly before depending on exact line numbers)
- WebSearch: "Collabora Online CODE reverse proxy nginx OIDC WOPI host authentication pattern 2026" and "GitHub Actions ubuntu-latest runner disk space 2026 specs GB" — MEDIUM (community forum threads + GitHub's own runner-images discussion #9329; no single authoritative OIDC+Collabora doc exists)

---
*Stack research for: Collabora Online fork (EdenDocs) — build toolchain, container pipeline, OIDC integration points*
*Researched: 2026-07-07*
