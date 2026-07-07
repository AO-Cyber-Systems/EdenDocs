# Project Research Summary

**Project:** EdenDocs
**Domain:** Brownfield fork of Collabora Online (coolwsd C++ document server + LOKit + TypeScript canvas frontend) — rebrand, OIDC integration, from-source build/container pipeline
**Researched:** 2026-07-07
**Confidence:** HIGH

## Executive Summary

EdenDocs is a fork of `CollaboraOnline/online.mirror`, not a greenfield build — every finding below is grounded directly in this exact repository tree (`configure.ac`, `wsd/`, `browser/`, `coolwsd.xml.in`, `docker/`), not generic Collabora tutorials (many of which are stale against this codebase, e.g. POCO handling). The four research passes converge on one central architectural fact that should drive the entire roadmap: **coolwsd has zero built-in concept of user identity or branding logic — both are first-class, already-built extension points (config keys + an additive `--with-app-branding` directory for branding; the WOPI `access_token`/`CheckFileInfo` contract for identity), and the correct engineering strategy is to use those seams rather than edit upstream source.** Rebranding and OIDC are therefore not "patch coolwsd" tasks — they're "configure coolwsd + build one small new service" tasks, and treating them as anything else (scattered `sed`-style rebranding, or bolting an OAuth client into `wsd/`) is the single biggest risk this project faces.

The recommended approach: build the from-source pipeline first, consuming Collabora's prebuilt LOKit/`engine` tarball (`engine-main-assets.tar.gz`) rather than compiling the 73k-file `engine/` tree from scratch — this keeps CI on standard GitHub-hosted runners and fits the project's explicit "don't fork core" scope decision. Once a working build exists, rebrand via the `--with-app-branding` directory + `coolwsd.xml` config keys (near-zero merge risk), and build OIDC as a small, separate WOPI-host reference service that authenticates against AOID and mints WOPI access tokens — coolwsd itself needs no source changes for auth, only a host allow-list entry. Container images and the upstream-tracking workflow both come after, since images need both the build and the branding assets, and the tracking workflow is best codified once the "config + additive files only" discipline has been proven to hold.

The key risks are: (1) the mirror moves fast (multiple commits/day across engine, wsd, and browser) — any team that lets the diff against `eden-main` grow (via scattered edits or a neglected merge cadence) will hit an effectively-unmergeable state within weeks; (2) trademark and MPL-2.0 obligations are legally distinct and commonly conflated — strip Collabora's marks, but preserve MPL headers/attribution, and never assume CODE's "10 document" limit or Collabora's SLA applies to a self-built fork (it doesn't — that's a `configure` flag Collabora sets low only in their own distributed binaries); (3) a build/branding/auth pass that only gets validated on macOS or via direct `127.0.0.1:9980` access will miss Linux-only sandboxing (`seccomp`, chroot, capability checks) and reverse-proxy/WebSocket failure modes that only appear in the real production topology. Mitigations for all three are well-documented and specific (see Critical Pitfalls below).

## Key Findings

### Recommended Stack

The build toolchain is fixed by upstream's own `configure.ac` floors, verified directly in-tree: GCC 12+/Clang 18 (Ubuntu 24.04's GCC 13 clears this comfortably), Node 20 LTS + npm ≥9 (hard-enforced by `configure.ac`; do not jump to Node 22/24 without testing `esbuild-wasm`/`browserify`/`jscpd` against it — nothing in upstream's own tooling has been observed running on anything newer), and Autoconf/Automake at ancient-but-easily-met floors. The most important, non-obvious finding: **POCO is no longer a separate system dependency** — it's built in-tree at `engine/external/poco/` (pinned 1.15.3) and `--with-poco-includes`/`--with-poco-libs` are silently ignored. Most "how to build Collabora Online" tutorials online (and even upstream's own separate GH-Actions-only Dockerfile) still assume `apt install libpoco-dev` — this is stale and must not be copied into EdenDocs's build recipe.

**Core technologies:**
- **GNU Autoconf/Automake + GCC 13 (Ubuntu 24.04)** — build system for both `coolwsd` (root) and `engine/`; floors comfortably met by current Ubuntu LTS.
- **Node 20 LTS / npm ≥9** — `browser/` TS/JS build; hard-enforced by `configure.ac`, matches what upstream's own CI actually uses.
- **POCO 1.15.3 (in-tree, `engine/external/poco/`)** — HTTP/WebSocket/JSON/crypto for `wsd/`; NOT a system package, auto-discovered via `--with-lo-builddir`.
- **Prebuilt LOKit/`engine` tarball (`engine-main-assets.tar.gz` from `CollaboraOnline/online`'s `for-code-assets` release)** — the highest-leverage CI decision: skips a 1-3 hour, ~30GB `engine/` compile entirely, matches the project's "consume prebuilt core" scope decision, and is confirmed fresh against `eden-main`'s current fork point (both track `main` @ 26.04.2.1). **Re-verify this alignment before any rebase** — no matching tarball exists yet for `distro/collabora/co-26.04`.
- **Docker/BuildKit** — container assembly; `docker/from-source/{Ubuntu,Debian,Fedora}` is the correct base for branded images (glibc required — the prebuilt tarball is glibc-only, Alpine is out unless engine is compiled from source).

Full detail, including the proven working CI recipe (`.github/workflows/codeql-analysis.yml`) and exact "what NOT to use" traps: `.planning/research/STACK.md`.

### Expected Features

This is a white-label-and-integrate project, not a feature-development project — "table stakes" here means "a credible rebrand," not "editor functionality" (that's inherited, already shipped). Every claim traces to a specific file/line in this tree.

**Must have (table stakes) — a credible v1 rebrand:**
- App-branding directory (`branding.css`, `branding.js`, logo SVGs, `toolbar-bg-logo.svg`) wired via `--with-app-branding` — the single highest-leverage item; drives menubar logo, toolbar watermark, and (via the shared `%BRANDING_JS%` placeholder) both the main app AND the admin console simultaneously.
- `brandProductName`/`brandProductURL`/`logoURL` set in `coolwsd.xml` (and confirm `--with-welcome-url` is NOT passed — it silently removes these three keys from the config if set).
- `--with-app-name="EdenDocs"` / `--with-vendor="AO Cyber Systems"` compiled fallbacks, favicon replacement, two small isolated `<title>` patches (`cool.html.m4`, `admintemplate.html` — no config hook exists for either), `help_url` repointed, color-palette CSS overridden via a branding-dir file (never edited in place).
- WOPI `CheckFileInfo` identity passthrough (`UserFriendlyName`, avatar, `IsAdminUser`) wired from whatever WOPI host stands in for AOID in v1 — this is what makes the OIDC integration visible in the UI at all.
- No-phone-home defaults verified (leave `INFOBAR_URL`/`FEEDBACK_URL`/`INFO_URL` unset — already the zero-effort default) and `--with-support-public-key` never passed (triggers an "unsupported" watermark).

**Should have (differentiators):**
- Per-tenant runtime `css_variables`/`theme` reskinning (Eden-Biz can reskin per customer at iframe-load time, zero rebuild) — mechanism already exists and is Collabora's own officially-documented pattern.
- Outbound `User-Agent` patch (`getAgentString()` in `net/Socket.cpp`) so Eden-Biz/AOCore logs show "EdenDocs," not "COOLWSD."
- Admin console PAM bridge as an interim non-static-password login.

**Defer (v2+):**
- Native OIDC login for the admin console itself (genuine new feature work, upstream has none today).
- Multiple named sub-brand themes (`?theme=`) — mechanism exists, no known customer demand yet.
- PWA manifest/icon set — upstream has none at all today.

Full detail, dependency graph, and the anti-features list (never fork `engine/` for branding, never reuse Collabora's proprietary `online-branding` repo/artwork, never touch the WOPI protocol surface for "brand consistency"): `.planning/research/FEATURES.md`.

### Architecture Approach

coolwsd is a C++ WebSocket document server (`wsd/`) that spawns sandboxed, chroot-jailed per-document `kit` processes (via `forkit`) embedding LibreOfficeKit from the vendored `engine/` tree; the TypeScript `browser/` frontend talks to it over HTTPS (static assets, `cool.html`) and a long-lived WebSocket (`/cool/<docKey>/ws`). The critical architectural fact: **coolwsd never talks to an identity provider — the WOPI host does.** The WOPI host authenticates the user (via AOID/OIDC in Eden's case), mints an opaque `access_token`, and coolwsd only ever validates that token against the host's `CheckFileInfo` endpoint over HTTPS with its own RSA proof-key signature. This confirms OIDC integration is entirely a new-service concern (a reference WOPI host), not a `wsd/` patch.

**Major components:**
1. **`wsd/` (coolwsd)** — HTTP(S)/WebSocket server, per-document `DocumentBroker` lifecycle, WOPI client (`wsd/wopi/CheckFileInfo.cpp`, `WopiStorage.cpp`), admin console, static/branding file serving (`FileServer.cpp`).
2. **`kit/` (forkit + kit)** — pre-spawns chroot-jailed per-document processes embedding LOKit from `engine/`; Linux-only sandboxing (seccomp, capabilities) that has no macOS equivalent.
3. **`engine/`** — vendored LibreOffice core/LOKit (73k+ files, out of scope to fork); consumed via prebuilt tarball or built once, never patched for branding.
4. **`browser/`** — TypeScript canvas frontend + admin console UI, built via npm/tsc inside the autotools tree, output to `browser/dist`; this is where the `--with-app-branding` copy step lands.
5. **WOPI host (new component, not part of this repo)** — owns identity: does the AOID OIDC dance, implements `CheckFileInfo`/`GetFile`/`PutFile`, mints/refreshes `access_token`, serves the launch page that embeds `cool.html`. Eden Drive can later implement the identical contract as a drop-in replacement.

Rebrand seams are config-driven (`coolwsd.xml` `<user_interface>` block, read at request time) plus an additive `--with-app-branding=<dir>` directory copied post-npm-build — both exist specifically so a downstream fork never needs to diff against upstream `browser/src`/`wsd/` files. Full component table, sequencing DAG, and grounded pattern/anti-pattern catalog: `.planning/research/ARCHITECTURE.md`.

### Critical Pitfalls

1. **Rebranding via scattered find-and-replace instead of the existing theming hooks** — 845+ files mention "Collabora" in `browser/` alone, but the overwhelming majority is the required MPL copyright header, not user-facing strings. Hand-editing source files instead of using `brandProductName`/`--with-app-branding` guarantees every future upstream merge conflicts on touched files. Avoid by building an additive `eden-branding/` directory + config keys, reserving actual source edits for the handful of genuinely-hardcoded fallback strings (tracked in a short allowlist).
2. **Underestimating upstream's merge velocity** — `online.mirror` publishes every merged Gerrit change directly to `main`, multiple commits/day across engine/wsd/browser simultaneously. A team that "merges upstream monthly" accumulates a multi-thousand-commit queue within a sprint. Decide explicitly whether `eden-main` tracks `upstream/main` (bleeding edge) or a stabilized `distro/collabora/co-XX.04` branch (batched, but still requires periodic large jumps), and schedule merges on a fixed, owned, weekly-minimum cadence as a first-class recurring objective.
3. **Trademark vs. MPL-2.0 conflation** — these are legally distinct regimes. Strip Collabora's name/logo/wordmark/help-URLs from user-facing surfaces and Docker `LABEL` metadata; but MPL headers, `COPYING`, `THIRDPARTYLICENSES`, and `AUTHORS` attribution must stay (removing them is itself an MPL violation). Never put "Collabora" in EdenDocs's own product/domain name without written permission.
4. **Trying to make coolwsd the identity provider** — coolwsd has no OIDC/OAuth/SAML code path at all (confirmed by exhaustive grep). Any PR that patches `wsd/` to add an OAuth client is fighting the architecture and creating unnecessary upstream-merge conflict surface. The correct integration point is a separate WOPI host service registered in `storage.wopi.alias_groups`.
5. **Validating only the macOS dev build or direct `127.0.0.1:9980` access, never the real Linux/container/reverse-proxy path** — Linux-specific sandboxing (chroot `systemplate`, seccomp, capability `verify-caps` build stage) and WebSocket-specific reverse-proxy config (timeouts, `Upgrade`/`Connection` headers, sticky sessions across replicas) are exactly the failure classes that "works on my Mac, direct-connect works" hides. CI must exercise the actual Linux/Docker target, and deployment verification must hold a document session open past 60 seconds through the real proxy.

Six more pitfalls (POCO/engine build coupling, CODE-limits/SLA misassumption, admin-console-vs-document-auth confusion, version skew in rolling upgrades, etc.), each with concrete warning signs and recovery steps: `.planning/research/PITFALLS.md`.

## Implications for Roadmap

Based on combined research, the five Active requirements in `.planning/PROJECT.md` map cleanly onto a dependency-ordered sequence. This ordering is corroborated independently by both ARCHITECTURE.md's build-order DAG and FEATURES.md's dependency graph.

### Objective 1: From-source build pipeline (coolwsd + LOKit)
**Rationale:** Foundational — nothing else (visual branding verification, container images, even OIDC end-to-end testing) can be validated without a working build. Every other objective either depends on this directly or needs it to test against.
**Delivers:** A working `coolwsd` + `browser/dist` build, using `ENGINE_ASSETS` (prebuilt `engine-main-assets.tar.gz`) rather than a from-scratch `engine/` compile, matching the project's "consume prebuilt core" scope decision. CI on standard GitHub-hosted `ubuntu-latest` runners.
**Uses:** Node 20 LTS, in-tree POCO 1.15.3 auto-discovery (no `--with-poco-*` flags), the proven `codeql-analysis.yml` configure invocation as a template.
**Avoids:** Pitfall 6 (POCO/engine coupling misconfiguration — do not pass `--with-poco-includes`/`--with-poco-libs`), Pitfall 5 (do not attempt to "unlock" `--with-max-connections`/`--with-max-documents` — there's no cap to unlock), Anti-Pattern 3 (never build `engine/` from source in CI on every run).

### Objective 2: EdenDocs rebrand of the web UI
**Rationale:** Config + additive-directory work, LOW technical risk, but needs Objective 1's working build to visually verify. Independent of the OIDC work below — can run in parallel with Objective 3 once Objective 1 is stable.
**Delivers:** `coolwsd.xml` `user_interface.brandProductName`/`brandProductURL`/`logoURL` set; a new `eden-branding/` directory (`branding.css`, `branding.js`, logo SVGs, `toolbar-bg-logo.svg`) wired via `--with-app-branding`; two small isolated source patches for the un-hookable `<title>` tags (`cool.html.m4`, `admintemplate.html`); favicon and `help_url` replaced; color-palette CSS overridden via branding-dir file, never edited in place.
**Addresses:** All P1 "table stakes" features from FEATURES.md — app-branding directory, product name/logo, About dialog, admin console branding via the shared `%BRANDING_JS%` hook.
**Avoids:** Pitfall 1 (scattered find-and-replace rebranding), Pitfall 3 (trademark under/over-removal — needs a pre-release checklist covering Docker `LABEL` metadata and help URLs too, not just the editor UI), Anti-Pattern 1 (patching `cool.html.m4`/`Control.AboutDialog.ts` directly instead of using config hooks).

### Objective 3: AOID authentication integration (OIDC)
**Rationale:** Mostly independent of coolwsd source — the actual deliverable is a new, small reference WOPI host service, not a `wsd/` patch. Needs Objective 1's running coolwsd to test the end-to-end round trip (`CheckFileInfo`/`GetFile`/`PutFile` + iframe launch), but has no dependency on Objective 2's branding work. Can run in parallel with Objective 2.
**Delivers:** A minimal reference WOPI host (new deployable, not part of this repo's `wsd/`/`browser/` tree) that performs OIDC Authorization Code flow against AOID, implements the WOPI REST surface, mints/refreshes short-lived `access_token`s, and serves the launch page. Registers itself in coolwsd's `storage.wopi.alias_groups` (switched to `mode="groups"`, not left at default `mode="first"`).
**Addresses:** WOPI `CheckFileInfo` identity passthrough differentiator from FEATURES.md — this is what makes AOID identity visible in the editor UI (`UserFriendlyName`, avatar, `IsAdminUser`).
**Avoids:** Pitfall 8 (trying to make coolwsd the identity provider), Pitfall 9 (confusing admin-console auth with document/WOPI auth — these are two independent systems and both need explicit hardening), Anti-Pattern 5 (skipping WOPI proof-key validation "for now" and forgetting to re-enable it before anything beyond localhost).

### Objective 4: EdenDocs-branded container images
**Rationale:** Depends on both Objective 1 (build outputs) and Objective 2 (branding assets to embed) — cannot start meaningfully before both exist. Objective 3's WOPI host is a separate deployable and does not need to be baked into this same image.
**Delivers:** Production container image(s) based on the `docker/from-source/{Ubuntu,Debian,Fedora}` pattern (prebuilt engine + built wsd/browser) — explicitly not `docker/from-packages/` (pure upstream repackage with no control point for branding or OIDC). Preserves the `verify-caps` capability-check build stage. Adds an EdenDocs-owned `cosign` signing step.
**Addresses:** Container image packaging requirement from PROJECT.md; sets up the deployment target that Objective 5's upstream-workflow and pitfall #10/#11's reverse-proxy/rolling-upgrade guidance apply to.
**Avoids:** Pitfall 7 (macOS dev build mistaken for the Linux production build — this objective is where that gap gets closed for real), Pitfall 10 (WebSocket/reverse-proxy misconfiguration — deployment verification must hold a session open past 60s through a real proxy, not just direct `127.0.0.1:9980` access), Pitfall 11 (version skew in rolling upgrades/clusters — requires session affinity if more than one replica is ever run).

### Objective 5: Upstream tracking workflow
**Rationale:** The *rule* ("branding lives in config + `eden-branding/`, never in upstream files"; fixed merge cadence) should be defined early so Objectives 2-4 are executed under it from day one — but the documented workflow itself is best finalized/validated after Objective 2 proves the rebrand seams hold with zero upstream-file edits, since the workflow doc should codify seams actually proven to work, not theoretical ones.
**Delivers:** A written, owned merge cadence (weekly minimum) for pulling `upstream/main` (or a chosen `distro/collabora/co-XX.04` branch) into `eden-main`; documentation of which files are allowed to diverge from upstream (the small `<title>`-tag allowlist, `getAgentString()` patch, etc.) and which must never be touched (`engine/`, WOPI protocol surface, `browser/src` beyond the allowlist).
**Addresses:** No specific FEATURES.md item — this is process/governance work that protects the investment made in Objectives 1-4.
**Avoids:** Pitfall 2 (upstream drift growing unmergeable — the single highest-cost pitfall to recover from per PITFALLS.md's Recovery Strategies table), and provides the review cadence that catches Pitfall 3 (trademark re-drift from new upstream strings) and Pitfall 11 (version-skew) on an ongoing basis, not just at initial ship.

### Objective Ordering Rationale

- **Build pipeline must come first** — it's a hard technical dependency for every other objective (branding needs `browser/dist` to exist to copy into; OIDC needs a running coolwsd to test against; images need build outputs).
- **Rebrand and OIDC can run in parallel** once the build is stable — ARCHITECTURE.md's dependency graph confirms the WOPI-host work has no code dependency on the branding-directory work, only a shared dependency on Objective 1.
- **Container images intentionally come after both** — this avoids baking an unbranded or unbuilt image and re-doing the image work twice.
- **Upstream-workflow is deliberately split**: define the *rule* early (before Objective 2 starts, so nobody scatters source edits), but finalize the *documented* workflow after Objective 2 validates the seams — this directly avoids Pitfall 1 (scattered rebrand) by having the rule in place from the first rebrand commit, while keeping the eventual documentation grounded in what was actually proven.
- This order also front-loads the highest-uncertainty item (OIDC/WOPI-host architecture, MEDIUM confidence — no single authoritative "OIDC + Collabora" doc exists) early enough that any surprises there don't block the build or rebrand tracks, since it runs in parallel rather than serially blocking them.

### Research Flags

Objectives likely needing deeper research during planning:
- **Objective 3 (AOID OIDC integration):** The reverse-proxy/OIDC-gating pattern (which routes must NOT be OIDC-challenged — `/browser`, `/hosting/discovery`, `/cool/`, `/lool/`) is MEDIUM confidence, corroborated by community forum threads and SDK docs rather than a single authoritative "OIDC + Collabora" guide. The exact `CheckFileInfo` JSON response schema should be cross-read against `sdk.collaboraonline.com/docs/architecture.html` (not fetched in this research pass — fetch tool was blocked) before implementation starts. Recommend `/devflow:research-objective` here specifically for the WOPI host's `CheckFileInfo`/`GetFile`/`PutFile` contract details and AOID's actual token-issuance model.
- **Objective 1 (build pipeline), engine-tarball POCO-completeness question:** It's unverified whether `engine-main-assets.tar.gz` ships a complete `workdir/UnpackedTarball/poco/include` + static libs sufficient for `--with-lo-builddir=engine` to resolve POCO natively — `dev-notes/poco-build.md` flags this as an open gap even in Collabora's own CodeQL workflow. Worth a quick spike/validation early in this objective rather than discovering it mid-CI-setup.

Objectives with standard, well-documented patterns (skip deep research-objective, research already sufficient):
- **Objective 2 (rebrand):** Every hook (`--with-app-branding`, `brandProductName`/`brandProductURL`/`logoURL`, the two `<title>` patch points) is directly grounded in this exact source tree with file:line citations. Proceed straight to planning.
- **Objective 4 (container images):** `docker/from-source/{Ubuntu,Debian,Fedora}` already provides a working reference pattern in-tree; the `verify-caps`/glibc-vs-musl considerations are well-documented in PITFALLS.md.
- **Objective 5 (upstream workflow):** This is a process/documentation objective, not a technical-integration one — no additional research needed, just execution discipline.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH for build toolchain/versions (repo-grounded: `configure.ac`, `download.lst`, live GitHub API for tarball freshness); MEDIUM for CI runner sizing and the OIDC/reverse-proxy integration pattern (community-sourced, not upstream-documented) |
| Features | HIGH — every table-stakes/differentiator claim traces to a specific file/line in this tree, corroborated by Collabora's own blog/FAQ and real-world Nextcloud-integration issue reports |
| Architecture | HIGH — component responsibilities, auth boundary, and build DAG all grounded in direct source inspection (`wsd/`, `kit/`, `browser/`, `coolwsd.xml.in`), not secondary docs. One noted gap: the SDK architecture doc (`sdk.collaboraonline.com`) was not fetched in this pass (tool blocked) — source-code grounding is treated as sufficient but the SDK doc is worth a quick cross-read before Objective 3 starts |
| Pitfalls | HIGH — repo-grounded (specific config keys, source files, build scripts, git log evidence of merge velocity) and cross-checked against Collabora's official trademark/MPL docs plus real GitHub issues/forum reports from operators who hit these exact failure modes |

**Overall confidence:** HIGH

### Gaps to Address

- **POCO workdir completeness in the prebuilt `engine-main-assets.tar.gz` tarball** (affects Objective 1): unverified whether `--with-lo-builddir=engine` resolves cleanly against the downloaded tarball, or whether a fallback system-POCO shim is needed for the tarball-only CI path. Validate early in the build-pipeline objective rather than assuming it works.
- **Exact `CheckFileInfo` JSON response schema and AOID's token-issuance model** (affects Objective 3): the WOPI contract shape is well-understood at the architectural level (this research cites the field names used: `BaseFileName`, `Size`, `OwnerId`, `UserId`, `UserCanWrite`, etc.) but the SDK's authoritative schema doc wasn't fetched, and AOID's specific OIDC claims/token format haven't been cross-referenced against what the reference WOPI host needs to produce. Flag for `/devflow:research-objective` before implementation.
- **No matching prebuilt engine tarball for `distro/collabora/co-26.04`** (affects the branch-tracking decision underlying Objective 5): only `main` and `co-25.04` currently have prebuilt `engine-*-assets.tar.gz` releases. If EdenDocs ever rebases off `main` onto a `co-26.04` distro branch, engine would need to be built from source until Collabora publishes a matching asset — factor this into the upstream-tracking strategy decision, don't assume the current `ENGINE_ASSETS` fast path survives every future rebase.
- **Eden brand tokens are TBD** (per PROJECT.md's own Constraints section, not fully resolved by this research pass): AO Cyber's master brand (dark+gold) is documented, but PROJECT.md explicitly notes "Eden products may carry their own palette" as an open decision. Objective 2 cannot finalize `eden-branding/`'s actual colors/logo until this is decided — treat as a blocking pre-req input to Objective 2, not a research gap this pass can close.

## Sources

### Primary (HIGH confidence)
- Direct source inspection of `/Users/justin/dev/EdenDocs` (this fork): `configure.ac` (root + `engine/`), `coolwsd.xml.in`, `discovery.xml`, `Makefile.am`, `browser/Makefile.am`, `wsd/*.{cpp,hpp}` (COOLWSD, FileServer, ClientRequestDispatcher, RequestVettingStation, HostUtil, ProofKey, Admin, Auth, wopi/CheckFileInfo, wopi/WopiStorage), `common/Authorization.hpp`, `common/ConfigUtil.cpp`, `kit/ForKit.cpp`, `kit/forkit-main.cpp`, `browser/html/cool.html.m4`, `browser/admin/admintemplate.html`, `browser/src/control/Control.AboutDialog.ts`, `browser/src/control/Control.Menubar.ts`, `browser/src/app/Socket.ts`, `browser/css/color-palette*.css`, `net/Socket.cpp`, `net/HttpRequest.hpp`, `docker/from-source/{README.md,build.sh,Ubuntu,Debian,Fedora}`, `docker/from-packages/Dockerfile`, `docker/README`, `.github/workflows/codeql-analysis.yml`, `engine/README.md`, `engine/download.lst`, `dev-notes/poco-build.md`, `dev-notes/dependency-issues.md`, `README.FILENOTICES.md`, `COPYING`, `browser/LICENSE`, `.planning/PROJECT.md`
- GitHub API `api.github.com/repos/CollaboraOnline/online/releases/tags/for-code-assets` (fetched live, 2026-07-07) — prebuilt engine tarball freshness/naming
- `git log`/`git diff` against `origin/main` and `origin/distro/collabora/co-26.04` — upstream velocity and branch-divergence measurements

### Secondary (MEDIUM confidence)
- [Theming of Collabora Online (official blog)](https://www.collaboraonline.com/blog/theming-of-collabora-online/) — `css_variables` per-tenant reskinning mechanism
- [Collabora Trademark Policy](https://www.collaboraonline.com/trademark-policy/) and [Branding Guidelines](https://www.collaboraonline.com/branding-guidelines/)
- [Collabora Online MPLv2 licensing terms](https://www.collaboraonline.com/terms/collabora-online-mplv2/)
- [Collabora Online SDK — How to integrate / Configuration](https://sdk.collaboraonline.com/docs/) and [Microsoft WOPI — Key concepts](https://learn.microsoft.com/en-us/microsoft-365/cloud-storage-partner-program/rest/concepts)
- WebFetch of `github.com/CollaboraOnline/online/docker/from-source-gh-action/Dockerfile` and `.github/workflows/docker.yml` (2026-07-07) — confirms stale POCO handling in the separate GH-native repo
- Community/operator reports: [Docker image is missing branding #4993](https://github.com/CollaboraOnline/online/issues/4993), [nextcloud/server theming issue #10264](https://github.com/nextcloud/server/issues/10264), [richdocuments version-detection issue #4267](https://github.com/nextcloud/richdocuments/issues/4267), [nginx proxy issue #10388](https://github.com/CollaboraOnline/online/issues/10388), Collabora community forum threads
- WebSearch: "Collabora Online CODE reverse proxy nginx OIDC WOPI host authentication pattern 2026", "GitHub Actions ubuntu-latest runner disk space 2026 specs GB"

### Tertiary (LOW confidence)
- None flagged — all findings either trace to primary source or are explicitly marked MEDIUM with corroboration above; no claims rest on training data alone.

---
*Research completed: 2026-07-07*
*Ready for roadmap: yes*
