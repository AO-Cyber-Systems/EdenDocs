# Architecture Research

**Domain:** Collabora Online (coolwsd C++ document server + LOKit + TS canvas frontend) — brownfield fork integration architecture
**Researched:** 2026-07-07
**Confidence:** HIGH (grounded directly in the EdenDocs source tree — `configure.ac`, `Makefile.am`, `browser/Makefile.am`, `coolwsd.xml.in`, `wsd/*`, `kit/*`, `docker/*` — not secondary docs)

## Standard Architecture

### System Overview — Runtime

```
┌───────────────────────────────────────────────────────────────────────────┐
│  BROWSER CLIENT (browser/src, TypeScript, leaflet-derived canvas)          │
│  cool.html.m4 → bundle.js/cool-src.js → Socket.ts (WebSocket) + AboutDialog│
└───────────────────────────┬─────────────────────────────────┬─────────────┘
        (1) GET/POST doc launch, static assets                │ (3) WS /cool/<docKey>/ws
        w/ WOPISrc + access_token in query/form                │  (protocol.txt tile/edit msgs)
                            ▼                                  ▼
┌───────────────────────────────────────────────────────────────────────────┐
│  coolwsd  (wsd/COOLWSD.cpp — Poco::Util::ServerApplication, port 9980)     │
│  ┌─────────────────┐ ┌───────────────────┐ ┌────────────────────────────┐│
│  │ClientRequestDisp-│ │RequestVettingStat- │ │ FileServer (static assets, ││
│  │atcher / FileServer│ │ion → wsd/wopi/    │ │ branding template subst.,  ││
│  │(HTTP entry point) │ │CheckFileInfo.cpp  │ │ /adminws/ JWT auth cookie) ││
│  └─────────┬────────┘ └─────────┬──────────┘ └────────────────────────────┘│
│            │                    │ (2) HTTPS CheckFileInfo/GetFile/PutFile  │
│  ┌─────────▼────────────────────▼──────┐   Authorization: Bearer <token>  │
│  │ DocumentBroker (wsd/DocumentBroker)  │──────────────────────────────┐  │
│  │  owns one doc's lifecycle, spawns    │                              │  │
│  │  Kit via ForKit, holds ClientSessions│                              │  │
│  └─────────┬─────────────────────────────┘                             │  │
│            │ Unix domain socket / pipe (KitWSDGlobals, WebSocketHandler)│  │
└────────────┼────────────────────────────────────────────────────────────┘  │
             ▼                                                               ▼
┌───────────────────────────────────┐                     ┌──────────────────────────────┐
│ forkit (kit/ForKit.cpp,            │                     │  WOPI HOST (identity owner)   │
│ forkit-main.cpp) — pre-spawns and  │                     │  - Authenticates user (OIDC   │
│ forks per-document `kit` processes │                     │    against AOID) BEFORE       │
│ into chroot jails built from       │                     │    embedding coolwsd          │
│ `systemplate` (coolwsd-systemplate-│                     │  - Mints WOPI access_token     │
│ setup, derived from the LOKit tree)│                     │  - Implements CheckFileInfo /  │
└─────────────┬───────────────────────┘                    │    GetFile / PutFile REST API  │
              ▼                                             │  - Serves launch page that      │
┌───────────────────────────────────┐                       │    POSTs WOPISrc+access_token   │
│ kit (kit/Kit.cpp, ChildSession)    │                       │    to coolwsd's cool.html       │
│ — one process per open document,   │                       │  - v1: no Eden Drive yet →      │
│ embeds LibreOfficeKit (LOKit) from  │                       │    build a minimal REFERENCE   │
│ the `engine/` (LibreOffice core)   │                       │    WOPI host for this role      │
│ build, renders tiles, applies edits│                       └───────────────┬──────────────┘
└─────────────────────────────────────┘                                       │ (0) OIDC login redirect
                                                                                ▼
                                                                     ┌──────────────────────┐
                                                                     │  AOID (Eden's OIDC    │
                                                                     │  Identity Provider)   │
                                                                     └──────────────────────┘
```

**coolwsd never talks to AOID.** The identity boundary is the WOPI host. This is the single most
important fact for objective sequencing (see Build Order below).

### Component Responsibilities

| Component | Responsibility | Grounded in |
|-----------|----------------|-------------|
| `wsd/` (coolwsd) | HTTP(S)/WebSocket server; request routing, per-document `DocumentBroker` lifecycle, WOPI client (calls out to the WOPI host), admin console, static file/branding template serving | `wsd/COOLWSD.hpp` ("Main server application class"), `wsd/ClientRequestDispatcher.cpp`, `wsd/DocumentBroker.hpp`, `wsd/FileServer.cpp` |
| `wsd/wopi/` | WOPI protocol **client** — `CheckFileInfo` (vet + fetch file metadata before creating a session), `WopiStorage` (download/upload document bytes, lock management), `WopiProxy` | `wsd/wopi/CheckFileInfo.{hpp,cpp}`, `wsd/wopi/WopiStorage.{hpp,cpp}`, `wsd/wopi/WopiProxy.cpp` |
| `wsd/ProofKey.cpp`, `wsd/HostUtil.{hpp,cpp}` | Server-to-server integrity: coolwsd signs its own outbound WOPI requests (RSA "proof key", published in `discovery.xml`) so the WOPI host can verify the request really came from coolwsd; `HostUtil` enforces the `storage.wopi.alias_groups`/host allow-list from `coolwsd.xml` | `wsd/ProofKey.cpp`, `wsd/HostUtil.hpp` |
| `common/Authorization.{hpp,cpp}` | Carries the WOPI `access_token`/`access_header` credential end-to-end (URL query param → `Authorization: Bearer` header on outbound WOPI calls); tracks TTL/expiry/refresh state | `common/Authorization.hpp` |
| `kit/` (`forkit-main.cpp`, `ForKit.cpp`) | Single-threaded supervisor that pre-spawns and chroot-jails child `kit` processes for fast per-document startup | `kit/forkit-main.cpp`, `kit/ForKit.cpp` |
| `kit/Kit.cpp`, `kit/ChildSession.cpp` | One sandboxed process per open document; embeds `LibreOfficeKit` (LOKit) from the built `engine/` tree, renders tiles, applies edit ops | `kit/Kit.cpp`, `kit/ChildSession.cpp` |
| `engine/` | Full LibreOffice core source, vendored in-tree (73k+ tracked files, not a submodule) as the LOKit provider. Built with its own `autogen.sh`/`configure`/`make`, multi-hour build. **Out of scope to fork per PROJECT.md** — EdenDocs consumes prebuilt LOKit assets instead of compiling this tree per build | `engine/` (top-level dir), `README.md` ("The engine (core) is built first, then online on top of it") |
| `browser/` | TypeScript/JS canvas frontend (leaflet-derived), admin console frontend (`browser/admin`), built via npm/tsc/m4/esbuild inside the autotools tree, output to `browser/dist` | `browser/Makefile.am`, `browser/src/`, `browser/admin/` |
| `common/` | Code shared by `wsd`, `kit`, `net` (config parsing, logging, file/proc utils, JailUtil for chroot setup) | `common/` listing |
| `net/` | Socket/HTTP/WebSocket/TLS primitives shared by wsd and kit | `net/` listing |
| `docker/from-packages/` | Installs upstream's **prebuilt `.deb` packages** (`coolwsd`, `collaboraofficebasis-*`) into a hardened distroless runtime — no source build at all. This is a repackaging/hardening pipeline, not a from-source build; matches the existing `edendocs-code` reference container | `docker/from-packages/Dockerfile` |
| `docker/from-source/` | Host-driven from-source build: builds `engine/` (or downloads prebuilt `ENGINE_ASSETS` tarball) + builds `coolwsd`/`browser` from the online monorepo source, then `docker build`s a distro-matched Dockerfile that just `COPY`s the resulting `instdir` | `docker/from-source/build.sh`, `docker/from-source/Ubuntu` |
| `coolwsd-systemplate-setup` | Builds the `systemplate` chroot tree (shared libs, LOKit assets) that `forkit` bind-mounts/copies into each `kit` jail; depends on the LOKit install path (`--with-lo-path`/`LO_PATH`), must be regenerated whenever that path's contents change | `coolwsd-systemplate-setup`, `Makefile.am:648-670` |
| Admin console (`wsd/Admin.cpp` + `browser/admin`) | Separate, self-contained auth system: HTTP Basic (optionally PAM) → `JWTAuth` (`common`/`wsd/Auth.hpp`) → JWT cookie scoped to `/adminws/`. **Not** related to WOPI/document auth or AOID | `wsd/Admin.cpp` (`enable_pam`, `JWTAuth authAgent("admin","admin","admin")`), `README.md` "Admin Panel" section |

## Rebrand Seams (grounded, config/additive-only — no upstream file edits required)

Collabora already ships first-class white-label hooks. EdenDocs should use these, not patch
upstream files directly, so the rebrand survives `upstream` merges.

1. **`coolwsd.xml` `<user_interface>` block — product name/URL/logo, config-only.**
   `configure.ac:1315-1328` injects `brandProductName`, `brandProductURL`, `logoURL` config keys
   into `coolwsd.xml.in` **only when `--with-welcome-url` is NOT passed** (the default path).
   `wsd/FileServer.cpp:1573-1581` reads these three keys at request time and substitutes
   `%PRODUCT_BRANDING_NAME%` / `%PRODUCT_BRANDING_URL%` / `%LOGO_URL%` into `cool.html`.
   `browser/src/control/Control.AboutDialog.ts:120-127` reads the resulting `brandProductName`
   JS var for the Help→About dialog, falling back to `"Collabora Online Development Edition
   (unbranded)"` if unset. **This is a runtime config change, zero source patches.**

2. **`branding.css` / `branding.js` — additive directory, injected at build time.**
   `browser/html/cool.html.m4:102,349` and `browser/admin/adminIntegratorSettings.html.m4:61-62`
   contain literal `<!--%BRANDING_CSS%--> <!-- add your logo here -->` / `<!--%BRANDING_JS%-->`
   hook comments — upstream's own placeholder for downstream branding. `wsd/FileServer.cpp:2624-2695`
   (`updateThemeResources`) serves `/browser/<version-hash>/branding.css` and `branding.js`
   (constant `BRANDING = "branding"`, `FileServer.cpp:1279`) from `browser/dist/`, optionally
   theme-prefixed (`browser/dist/<theme>/branding.css`) when a WOPI host passes a `theme` query
   param and `user_interface.use_integration_theme` is true.
   `browser/Makefile.am:1041-1047` (guarded by the `APP_HAS_BRANDING` automake conditional, driven
   by `--with-app-branding=<path>` in `configure.ac:222-226`) copies, post-build:
   - `<path>/branding*` → `browser/dist/` (i.e. `branding.css`, `branding.js`)
   - `<path>/images/*.svg` → `browser/dist/images/` (overwrite-by-filename icon replacement —
     also covers `discovery.xml` `favIconUrl` targets like `x-office-document.svg`, which live in
     the same `browser/images` namespace)
   - `<path>/images/toolbar-bg-logo.svg` → `browser/dist/images/toolbar-bg.svg`
   - `<path>/welcome/` → `browser/dist/welcome/` (macOS builds only)

   **This exactly mirrors Collabora's own internal practice**: `docker/from-source/build.sh:83-124`
   clones a separate proprietary `online-branding` repo and runs `./brand.sh <lo-instdir> <dist-dir>
   <theme-name>` for each of their own downstream brands (`CODE`, `NC-theme-community`). EdenDocs
   should create its own `eden-branding/` directory (in-repo, additive, or a small sibling repo) with
   the same `branding.css`/`branding.js`/`images/*.svg` shape and wire it in via
   `--with-app-branding=$(pwd)/eden-branding` — no upstream files touched, no merge conflicts.

3. **Custom welcome screen (opt-in, mutually exclusive with #1's simple fields).**
   If `--with-welcome-url` IS passed, `configure.ac` instead emits a `<welcome><enable>` config
   block and coolwsd expects a full custom `browser/dist/welcome/welcome.html`
   (`coolwsd.xml.in:1324`, `README` build docs). EdenDocs almost certainly wants the **default**
   (non-welcome-url) path — it's the simpler, config-only surface (#1) — unless a first-run
   onboarding screen is explicitly desired later.

4. **`discovery.xml`** — WOPI discovery document; `<app name favIconUrl>` entries reference
   `browser/images/*.svg`, so favicon rebranding is covered by seam #2's `images/*.svg` copy.
   The document does not otherwise carry product-name branding (it's a machine-readable
   action/extension map for WOPI hosts).

## Auth Architecture — Who Owns Identity

**Confirmed directly in source: coolwsd has no OIDC/OAuth/SAML/user-login code path at all.**
(`grep -i "openid|oauth|saml"` across `wsd/`, `common/`, `net/`, `coolwsd.xml.in` returns nothing
except test scaffolding — `test/UnitOAuth.cpp` only exercises `Poco::Net::OAuth20Credentials` for
the **coolwsd→WOPI-host** bearer-token call format, not end-user login.)

- **The WOPI host owns identity, full stop.** It authenticates the user however it wants (in
  Eden's case: OIDC against AOID), then decides what that user is allowed to do with a given
  document, and encodes that decision as an opaque **`access_token`** (format is entirely up to
  the WOPI host — JWT, opaque DB-backed string, whatever).
- **The token rides in the URL, not a cookie.** The WOPI host's own page embeds coolwsd via an
  iframe/POST to `.../browser/dist/cool.html` with `access_token` + `access_token_ttl` as
  hidden form fields (`wsd/FileServer.cpp:1341-1389`, `browser/src/main.js:26`). coolwsd's
  Origin/XSRF check is explicitly bypassed for this path because "WOPI auth uses no implicit
  browser credential" (`wsd/ClientRequestDispatcher.cpp:1052-1073`, verbatim comment).
- **coolwsd only ever relays the token.** `RequestVettingStation::handleRequest`
  (`wsd/RequestVettingStation.cpp:65-131`) validates the storage type and host allow-list, then
  calls `CheckFileInfo` (`wsd/wopi/CheckFileInfo.cpp`) which does an HTTPS GET to the WOPI host's
  `CheckFileInfo` endpoint with `Authorization: Bearer <access_token>`, and signs the request with
  its own RSA **proof key** (`wsd/ProofKey.cpp`) so the WOPI host can verify the call really came
  from this coolwsd instance. `GetFile`/`PutFile` (`wsd/wopi/WopiStorage.cpp`) follow the same
  pattern for content read/write. **coolwsd never validates the token's contents** — it treats
  CheckFileInfo's 200-with-JSON response as "authorized" and any 401/403 as "not."
- `common/Authorization.hpp` tracks token TTL and drives a refresh handshake
  (`postMessage App_TokenExpired` → browser gets a fresh token from the WOPI host → resent to
  coolwsd) when `access_token_ttl` is set.
- **Admin console auth is a completely separate system** (HTTP Basic/PAM → JWT cookie scoped to
  `/adminws/`, `wsd/Admin.cpp`) and is irrelevant to the document-session/OIDC requirement.

### What this means for the "AOID authentication integration (OIDC)" requirement

There is effectively **no coolwsd source change** required for OIDC. `coolwsd.xml`'s
`storage.wopi.alias_groups`/host allow-list needs to trust the WOPI host's origin, and SSL/proof-key
settings need to match — that's it. The actual work is building the thing that **is** the WOPI host:

- Since **Eden Drive doesn't exist yet** (per project context), v1's minimal integration target is
  a **small reference WOPI host** service (not part of this coolwsd fork's `wsd/`/`browser/`
  tree — a separate service/deployable) that:
  1. Performs OIDC Authorization Code flow against AOID for the end user.
  2. Implements the WOPI REST surface coolwsd calls: `CheckFileInfo` (GET, returns JSON with
     `BaseFileName`, `Size`, `OwnerId`, `UserId`, `UserFriendlyName`, permission flags, etc.),
     `GetFile` (GET, raw bytes), `PutFile` (POST, raw bytes + lock headers).
  3. Mints a short-lived `access_token` bound to the AOID-authenticated session + a target file,
     handles token refresh.
  4. Serves the launch page that POSTs `WOPISrc` + `access_token` + `access_token_ttl` to
     coolwsd's `cool.html` (mirrors the README's Nextcloud/ownCloud integration pattern, just
     pointed at AOID instead of NC/OC's own login).
  5. Optionally validates coolwsd's proof-key signature on inbound calls (can be relaxed for
     local dev, should be enforced for anything beyond localhost).
- This reference WOPI host is genuinely a **new, small service** — plan it as its own build unit,
  not a patch to `wsd/`. It can be a thin implementation (even a small Node/Go/Python service) since
  the WOPI contract is a handful of REST endpoints plus an OIDC client.

## Build Order — Autotools + Docker Composition (grounded)

### Local/from-source build DAG (as this fork's own tooling already expresses it)

```
engine/ (LibreOffice core, LOKit)          ──┐  MULTI-HOUR if built from source.
  ./autogen.sh --with-distro=CPLinux-LOKit   │  docker/from-source/build.sh already supports
  make                                        │  skipping this via $ENGINE_ASSETS (prebuilt
  → engine/instdir (LOKit + Office runtime)  ◄┘  tarball) — matches PROJECT.md's decision to
                                                  consume Collabora's prebuilt core, not fork it.
        │
        │ --with-lokit-path=<engine>/include  --with-lo-path=<engine-or-prebuilt>/instdir
        ▼
top-level ./autogen.sh && ./configure && make   (wsd/, kit/, common/, net/ C++ build via autotools)
        │
        ├─▶ browser/ npm build (wired into the SAME autotools tree via AC_CONFIG_LINKS +
        │   browser/Makefile.am: npm/node checked in configure.ac:2291-2323; tsc, m4 bundling
        │   (cool-src.js.m4, bundle.js.m4), esbuild minification, l10n JSON generation,
        │   output → browser/dist/) — `make` at top level drives this as a subdir target.
        │
        ├─▶ APP_HAS_BRANDING step (if --with-app-branding=<eden-branding>): copies
        │   branding.css/js + images/*.svg into browser/dist/ (see Rebrand Seams #2)
        │
        └─▶ coolwsd-systemplate-setup <SYSTEMPLATE_PATH> <LO_PATH>  (Makefile.am:648-670)
            builds the systemplate chroot tree FROM the LOKit install path — must be
            (re)run whenever LO_PATH's contents change (new engine build or new prebuilt drop).

`make run` / `setup-wsd` → launches coolwsd on 9980 using the above systemplate + browser/dist.
```

### Docker composition (two distinct, non-interchangeable pipelines)

- **`docker/from-packages/Dockerfile`** — apt-installs upstream's **prebuilt `coolwsd` .deb**
  (no EdenDocs source involved at all) into a hardened distroless runtime. This is what the
  current `edendocs-code` reference container already is: useful as a **behavioral baseline**
  only, not a template for "from-source build" or "branded images," since it never touches
  EdenDocs's own `wsd/`/`browser/` source.
- **`docker/from-source/{Ubuntu,Debian,Fedora,ArchLinux,openSUSE}`** — host builds
  engine+coolwsd+browser (per the DAG above) into `instdir`, then a thin distro Dockerfile just
  `COPY /instdir /` and runs `coolwsd-systemplate-setup` inside the image
  (`docker/from-source/Ubuntu:21-38`). **This is the correct base for "EdenDocs-branded container
  images"** — it already composes exactly "prebuilt core + built wsd" as the question asked,
  and already has the `ENGINE_ASSETS` escape hatch to avoid rebuilding `engine/` from source.
- **`docker/from-source/Alpine` + `build-alpine.sh`** — self-contained multi-stage build (musl
  requires building `engine/` from source; prebuilt engine assets are glibc-only) — not needed
  for v1 unless an Alpine/musl target is required later.

### Suggested objective build order for the five active requirements

```
1. From-source build pipeline (coolwsd + LOKit)         ── FOUNDATIONAL, build first
   └─ Use $ENGINE_ASSETS (prebuilt LOKit) not a full engine/ compile, per the
      "consume prebuilt core" decision. Verify against docker/from-source/build.sh's
      own documented pattern. Local dev on macOS via `make run` (9980); Linux container
      is the deliverable target.
        │
        ├──▶ 2. EdenDocs rebrand of the web UI            ── config + additive dir, LOW risk,
        │        needs (1)'s working build to visually verify. Delivers:
        │        coolwsd.xml user_interface.brandProductName/URL/logoURL +
        │        eden-branding/{branding.css,branding.js,images/*.svg}
        │        wired via --with-app-branding.
        │
        ├──▶ 3. AOID authentication integration (OIDC)     ── mostly INDEPENDENT of coolwsd
        │        source; builds the reference WOPI host as a new service. Needs (1)'s
        │        running coolwsd to test end-to-end (CheckFileInfo/GetFile/PutFile round
        │        trip + iframe launch), but the WOPI host's own code has no dependency on
        │        (2). Can run in parallel with (2) once (1) is stable.
        │
        └──▶ 4. EdenDocs-branded container images          ── DEPENDS on (1) [build outputs]
                 AND (2) [branding assets to embed]. Use docker/from-source/<distro>
                 pattern (prebuilt engine + built wsd/browser), not docker/from-packages
                 (pure upstream repackage). (3)'s WOPI host is a separate deployable and
                 does not need to be baked into this same image.

5. Upstream tracking workflow (merge cadence docs/tooling) ── define the RULE early
   ("branding lives in config + eden-branding/, never in upstream files"; "engine/ tracks
   upstream's engine/ 1:1, don't diff it") but finalize/validate it AFTER (2) proves the
   seams hold with zero upstream-file edits — the workflow doc should codify exactly the
   seams found above, not theoretical ones.
```

## Architectural Patterns

### Pattern 1: Config-driven identity, not code-driven identity

**What:** `coolwsd.xml`'s `user_interface.brandProductName/brandProductURL/logoURL` are read at
request time (`FileServer.cpp`), not baked in at compile time.
**When to use:** Any rebrand string that varies by deployment/tenant should go here first.
**Trade-offs:** Only covers product name/URL/logo text — does not cover CSS colors/toolbar art
(that's Pattern 2).

### Pattern 2: Additive branding directory injected post-npm-build

**What:** `--with-app-branding=<dir>` + `browser/Makefile.am`'s `APP_HAS_BRANDING` copy step.
**When to use:** Any visual asset (CSS, JS, SVG icons) that must survive upstream `browser/`
merges without ever diffing against an upstream file.
**Trade-offs:** Files are copied, not templated — CSS must use `!important`/high-specificity
selectors or CSS custom-property overrides to reliably beat upstream's shipped `cool.css`
cascade; verify visually after every upstream merge since upstream can rename DOM ids/classes.

### Pattern 3: Identity boundary at the WOPI host, not the app server

**What:** coolwsd deliberately has zero knowledge of *how* a user was authenticated — it only
trusts an opaque bearer token and the host that issued it (allow-listed by
`storage.wopi.alias_groups`).
**When to use:** This is why the OIDC requirement is scoped to "build/point-at a WOPI host,"
never "patch coolwsd's auth."
**Trade-offs:** Means EdenDocs must own and operate a WOPI host component even for a
minimal/testing deployment — there's no "just add an OIDC flag to coolwsd" shortcut.

## Anti-Patterns

### Anti-Pattern 1: Patching `browser/html/cool.html.m4` or `Control.AboutDialog.ts` directly for branding

**What people do:** Hardcode "EdenDocs" into the upstream HTML/TS files.
**Why it's wrong:** Every `upstream` merge will now conflict on these exact lines; defeats the
whole point of the fork's stated merge-survivability constraint.
**Do this instead:** Use the config keys (`user_interface.brandProductName` etc.) and the
`--with-app-branding` additive directory — both already exist for exactly this purpose.

### Anti-Pattern 2: Trying to make coolwsd do OIDC directly

**What people do:** Look for a "just add an OAuth provider" config flag in `coolwsd.xml`.
**Why it's wrong:** It doesn't exist and won't — the whole WOPI protocol is designed so coolwsd
never sees a password/OIDC token, only an opaque, host-issued `access_token`.
**Do this instead:** Build the reference WOPI host as the OIDC relying party; coolwsd config
only needs the host allow-listed.

### Anti-Pattern 3: Building `engine/` from source in CI/Docker on every run

**What people do:** `cd engine && ./autogen.sh ... && make` as step 1 of every image build.
**Why it's wrong:** Multi-hour build, contradicts PROJECT.md's explicit "consume prebuilt core"
decision, and engine/ is explicitly out of scope to fork.
**Do this instead:** Use `docker/from-source/build.sh`'s `$ENGINE_ASSETS` prebuilt-tarball path
(or Collabora's published `.deb`/CODE packages extracted to an instdir) and only build
`wsd`/`browser` from EdenDocs source.

### Anti-Pattern 4: Conflating admin-console auth with document-session auth

**What people do:** Assume "add OIDC" means changing `wsd/Admin.cpp`'s PAM/Basic+JWT flow.
**Why it's wrong:** That's a completely separate, smaller surface (the admin metrics console),
unrelated to the WOPI/document-session auth path end users hit.
**Do this instead:** Scope OIDC work to the WOPI host relationship first; admin-console SSO (if
wanted) is a separate, much smaller, later objective.

### Anti-Pattern 5: Skipping the WOPI proof-key check "for now" and forgetting to re-enable it

**What people do:** Disable proof-key validation in the reference WOPI host to unblock local dev
and never revisit it.
**Why it's wrong:** Proof-key validation is the only thing stopping an arbitrary third party from
impersonating coolwsd's server-to-server calls to the WOPI host.
**Do this instead:** Gate it behind an explicit dev-only flag, document it, and flag it in
PITFALLS for objective-level security review before anything beyond localhost.

## Integration Points

### External Services

| Service | Integration Pattern | Notes |
|---------|---------------------|-------|
| AOID (OIDC IdP) | OIDC Authorization Code flow, consumed by the **reference WOPI host**, never by coolwsd directly | coolwsd has no OIDC code path at all (confirmed by source grep) |
| Future Eden Drive (WOPI host) | Same WOPI contract as the reference host (`CheckFileInfo`/`GetFile`/`PutFile` + launch page) | v1 target is the reference WOPI host; Eden Drive can later implement the identical contract as a drop-in replacement |
| Collabora prebuilt LOKit/engine packages or `ENGINE_ASSETS` tarball | Consumed via `--with-lokit-path`/`--with-lo-path` at coolwsd configure time | Keeps EdenDocs's own build to "wsd + browser only," per PROJECT.md's out-of-scope decision |

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| Browser client ↔ coolwsd | HTTPS (static assets, `cool.html`) + WebSocket `/cool/<docKey>/ws` (protocol.txt tile/edit messages) | Origin/XSRF check bypassed specifically for the WOPI-token-authenticated doc-session WS path (`ClientRequestDispatcher.cpp`) |
| coolwsd (wsd) ↔ forkit/kit | Unix domain socket / pipe IPC; forkit pre-spawns chroot-jailed kit processes from `systemplate` | `kit/ForKit.cpp`, `KitWSDGlobals.cpp` |
| kit ↔ engine (LOKit) | In-process C API embedding (`COKit/COKit.hxx`, `KIT_USE_UNSTABLE_API`) | `kit/ForKit.cpp:39-40` |
| coolwsd ↔ WOPI host | HTTPS REST, `Authorization: Bearer <access_token>` + coolwsd RSA proof-key signature headers | `wsd/wopi/CheckFileInfo.cpp`, `wsd/wopi/WopiStorage.cpp`, `wsd/ProofKey.cpp` |
| Admin browser ↔ coolwsd | HTTP Basic/PAM login → JWT cookie scoped to `/adminws/` → WebSocket admin protocol | `wsd/Admin.cpp`, `README.md` "Admin Panel" |
| Build-time: `browser/` npm/tsc ↔ top-level autotools | `AC_CONFIG_LINKS` (`browser/package.json`, `.eslintrc`, etc.) + `browser/Makefile.am` invoked as an automake subdir target from the top `Makefile.am` | `configure.ac:2291-2354` |

## Sources

- Direct source inspection of `/Users/justin/dev/EdenDocs` (this fork): `configure.ac`,
  `Makefile.am`, `browser/Makefile.am`, `coolwsd.xml.in`, `discovery.xml`, `README.md`,
  `wsd/COOLWSD.hpp`, `wsd/ClientRequestDispatcher.cpp`, `wsd/RequestVettingStation.cpp`,
  `wsd/FileServer.cpp`, `wsd/HostUtil.{hpp,cpp}`, `wsd/ProofKey.cpp`, `wsd/Admin.cpp`,
  `wsd/wopi/CheckFileInfo.{hpp,cpp}`, `wsd/wopi/WopiStorage.cpp`, `common/Authorization.hpp`,
  `kit/ForKit.cpp`, `kit/forkit-main.cpp`, `browser/html/cool.html.m4`,
  `browser/admin/adminIntegratorSettings.html.m4`, `browser/src/control/Control.AboutDialog.ts`,
  `browser/src/main.js`, `docker/from-packages/Dockerfile`, `docker/from-source/build.sh`,
  `docker/from-source/Ubuntu`, `docker/from-source/README.md`, `test/UnitOAuth.cpp`,
  `.planning/PROJECT.md`.
- README's pointer to `https://sdk.collaboraonline.com/docs/architecture.html` and
  `wsd/protocol.txt` for the tile/edit WebSocket protocol — not fetched in this pass (fetch
  tool was blocked by an unrelated safety filter); source-code grounding above is authoritative
  and sufficient, but the SDK docs are worth a quick cross-read before the OIDC/WOPI-host
  objective starts, specifically for the exact `CheckFileInfo` JSON response schema.

---
*Architecture research for: Collabora Online fork (EdenDocs) — rebrand/auth/build integration*
*Researched: 2026-07-07*
