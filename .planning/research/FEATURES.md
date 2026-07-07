# Feature Research

**Domain:** White-label/rebrand + platform integration of a Collabora Online (coolwsd + browser/) fork
**Researched:** 2026-07-07
**Confidence:** HIGH — grounded directly in the EdenDocs source tree (`configure.ac`, `coolwsd.xml.in`, `wsd/FileServer*.cpp`, `browser/`), corroborated by Collabora's own blog/FAQ and community reports (Nextcloud, CODE deployments) via WebSearch.

## Grounding Method

All "table stakes" and "differentiator" claims below trace to a specific file/line in `/Users/justin/dev/EdenDocs` (the actual `eden-main` checkout). Where a claim relies only on WebSearch (no local source hit), it is marked MEDIUM confidence and cited. Nothing here is asserted from training data alone.

---

## Feature Landscape

### Table Stakes (A Credible Rebrand Must Have These)

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **App-branding directory** (`branding.css`, `branding.js`, `images/*.svg`, `toolbar-bg-logo.svg`) wired via `--with-app-branding=<path>` | This *is* Collabora's own supported white-label mechanism — every derivative (Nextcloud's `code-brand`/`collabora-online-brand` packages, CODE itself) uses it | LOW–MEDIUM | `configure.ac:223-228` (`AC_ARG_WITH([app-branding]...)`), `configure.ac:2386-2397` (`APP_BRANDING_DIR`/`APP_HAS_BRANDING`), `browser/Makefile.am:1042-1046` — build copies `$(APP_BRANDING_DIR)/branding*` → `browser/dist/`, `$(APP_BRANDING_DIR)/images/*.svg` → `browser/dist/images/`, and `toolbar-bg-logo.svg` → `browser/dist/images/toolbar-bg.svg`. Runtime injection point: `wsd/FileServer.cpp:1279-1280` (`BRANDING = "branding"`), `:2635-2690` (`updateThemeResources()` links `browser/<hash>/branding.css` + `<script src=".../branding.js">` into **every** served page via the `<!--%BRANDING_JS%-->` placeholder). Zero upstream source edits required — purely additive files. |
| **Product name** — compiled fallback + runtime override | Users/admins expect the app to say "EdenDocs," not "Collabora Online Development Edition" | LOW | Compiled: `--with-app-name` → `APP_NAME` (`configure.ac:1123-1132`, default `"Collabora Online Development Edition"`). Runtime, no rebuild: `user_interface.brandProductName` in `coolwsd.xml.in` (populated via `CONFIG_INTERFACE_FRAGMENT`, `configure.ac:1329`) is read by `browser/src/control/Control.AboutDialog.ts:119-128` for the About dialog `#product-name`, and by `wsd/FileServer.cpp:1576-1580` for the welcome-popup template. Also settable via an undocumented-but-functional `<product_name>` key read by `ConfigUtil::getString("product_name", APP_NAME)` in `wsd/ClientRequestDispatcher.cpp:3051` (feeds the `/hosting/capabilities` API's `productName` field) and `wsd/COOLWSD.cpp:3839` (update-check query param). |
| **Logo** — menubar logo, toolbar background, About-dialog logo | Every white-label deployment replaces the corner logo | LOW–MEDIUM | Menubar logo: `user_interface.logoURL` config, consumed in `browser/src/control/Control.Menubar.ts:2689-2700` (`window.logoURL`, set `"none"` to hide entirely). Toolbar watermark: `toolbar-bg-logo.svg` in the app-branding dir → `browser/dist/images/toolbar-bg.svg` (`browser/Makefile.am:1044`). About dialog: empty `<fig id="integrator-logo">` / `<fig id="product-logo">` placeholders in `browser/html/cool.html.m4:244-249` — populated by CSS/JS from the branding bundle, not by `Control.AboutDialog.ts` itself (that file only fills text, per its source). |
| **Favicon** | Browser tab / bookmark identity | LOW | `wsd/ClientRequestDispatcher.cpp:1351,1771-1775` — serves `favicon.ico` from the app's working directory, falling back to `COOLWSD::FileServerRoot + "/favicon.ico"` (repo root, currently the stock `favicon.ico`). Simple drop-in file replace; no PWA icon set (`manifest.json`, `apple-touch-icon`) exists upstream today — `browser/html/cool.html.m4` has no such tags, so a PWA icon suite is a differentiator to *add*, not a table-stakes gap to fill. |
| **About dialog product name / vendor / copyright** | Standard "Help → About" expectation | LOW–MEDIUM | `browser/src/control/Control.AboutDialog.ts:119-128` — falls back to literal string `"Collabora Online Development Edition (unbranded)"` if `brandProductName` is unset (so setting `brandProductName` is mandatory, not optional). Copyright line at `:204-210` reads `window.vendor`/`window.copyrightYear`, sourced from `--with-vendor` (`configure.ac:1134-1152`, default is the *build machine's Unix username* if unset — must be explicitly set). Git-hash link at `:144-150` is hardcoded to `gerrit.collaboraoffice.com` — leave as-is (see Anti-Features; it correctly points at upstream's own source). |
| **Window `<title>` tag** | Browser tab text | LOW effort, but a genuine merge-risk source edit | `browser/html/cool.html.m4:42` hardcodes `<title>Online Editor</title>`. No config/branding hook exists for this specific tag (confirmed by grep — no `%PRODUCT_TITLE%`-style placeholder). Must be a literal, isolated one-line patch to a file upstream actively edits — flag for a dedicated small commit that's easy to re-apply on merge conflicts. |
| **Admin console title/branding** | `/upload/wopi/admin` is a real surface ops staff see daily | MEDIUM | `browser/admin/admintemplate.html:12` hardcodes `<title>Collabora Online - Admin console</title>` — same "no config hook" situation as the app title. However, colors/logo in the admin console **are** branding-dir-driven: `admintemplate.html:62` has the same `<!--%BRANDING_JS%-->` placeholder, filled by `updateThemeResources()` (`wsd/FileServer.cpp:2485`). Admin console already uses `Montserrat` (`admintemplate.html:25-28`), which incidentally matches the AO Cyber heading font. |
| **Welcome/onboarding screens** | First-run impression | MEDIUM | Assets at `browser/welcome/{welcome.html,welcome.css,welcome.js,slide1-3.png,welcome-slideshow.odp}`. Gated by `welcome.enable` in `coolwsd.xml.in` (`WELCOME_CONFIG_FRAGMENT`, `configure.ac:1323-1327`), whose own `desc` text says: *"your custom welcome screen must be created at `/usr/share/coolwsd/browser/dist/welcome/welcome.html`."* **Important nuance:** the app-branding Makefile rule only auto-copies `$(APP_BRANDING_DIR)/welcome` → `$(DIST_FOLDER)/welcome` `if ENABLE_MACOS` (`browser/Makefile.am:1044-1046`) — on the Linux/server build EdenDocs actually ships, this copy does **not** happen automatically; a custom welcome screen needs its own `COPY` step in the Docker build. |
| **Help URL** | "Help" menu / F1 | LOW | `--with-help-url` / `help_url` in `coolwsd.xml.in:413`, default `https://help.collaboraoffice.com/help.html?`. Point at Eden's own docs, or set empty to hide Help buttons entirely (per the config `desc`). |
| **CSS color tokens (editor chrome)** | Toolbar/menu accent colors should read as Eden, not Collabora blue | LOW | `browser/css/color-palette.css` (light) and `color-palette-dark.css` (dark) define ~30 CSS custom properties (`--color-primary: #0b87e7`, `--color-main-background`, etc.). Override via a `browser/dist/<theme>/branding.css` file that redeclares the same `:root` variables (loaded after `cool.css`, so it wins) — do **not** edit `color-palette*.css` directly (guaranteed upstream merge conflicts). |
| **Vendor string** | Legal/copyright line, packaging metadata | LOW | `--with-vendor="AO Cyber Systems"` (`configure.ac:1134-1152`); also flows into `coolwsd.spec.in`/`debian/` packaging `Vendor:` fields (currently `Collabora Productivity Ltd.`) if RPM/DEB packaging is used instead of/alongside containers. |

### Differentiators (Eden-Specific Value-Adds)

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Per-tenant runtime CSS reskinning via `css_variables`/`theme` iframe-embed params** | Eden-Biz can reskin EdenDocs per customer at *load time*, with zero server config change or redeploy — no branding-directory rebuild per tenant | LOW (consumer-side only; mechanism already exists) | `wsd/FileServer.cpp:1380` — `extractVariablePlain(form, "css_variables", CSS_VARS)`; `:1384` — `extractVariable(form, "theme", BRANDING_THEME)`. Format (`wsd/FileServerUtil.cpp:381-414`, `cssVarsToStyle()`): semicolon-separated `--css-var-name=value` pairs, URL-form-encoded, injected as an inline `:root { ... }` `<style>` block. Officially documented by Collabora's own blog, "Theming of Collabora Online" (WebSearch, MEDIUM-HIGH confidence — corroborates the source finding): pass a hidden form field named `css_variables` alongside `access_token` in the COOL iframe load. `theme=<name>` separately selects a whole `browser/dist/<name>/branding.{css,js}` bundle if `browser/dist/<name>` exists (`wsd/FileServer.cpp:2635-2649`, `hasIntegrationTheme` check) — this is exactly how Collabora's own `docker/from-source/build.sh` builds two themes into one image today (`./brand.sh ... CODE` and `./brand.sh ... NC-theme-community`, `docker/from-source/build.sh:118-121`), proving multi-theme-per-build is an established, working pattern. |
| **WOPI `CheckFileInfo` identity passthrough as the AOID/SSO integration seam** | Confirms *where* OIDC integration actually belongs — not inside coolwsd | MEDIUM | coolwsd has **no native OIDC/OAuth client** for end-user login (`wsd/Auth.hpp`/`Auth.cpp` only implement JWT signing/verification for the *admin console* session, unrelated to end-user identity). End-user auth is entirely the WOPI host's job: the host (Eden-Biz, fronted by AOID) does the OIDC dance, then mints an opaque `access_token` and returns identity via WOPI `CheckFileInfo` fields (`UserFriendlyName`, `UserExtraInfo` [avatar URL — `wsd/CollabSocketHandler.hpp:90`], `UserId`, `IsAdminUser` [`wsd/ClientSession.hpp:480`], `UserCanWrite`, `HideUserList`, `WatermarkText`, `BreadcrumbDocName`, `EnableInsertRemoteImage`). "AOID OIDC auth integration" for EdenDocs = wiring these WOPI response fields from AOID-authenticated Eden-Biz sessions, not adding an OIDC client to coolwsd. This reframes that milestone's scope significantly. |
| **Outbound HTTP identification patch** | Eden-Biz/AOCore logs show "EdenDocs" instead of "COOLWSD" when coolwsd calls out (WOPI PutFile, LanguageTool, AI backend) | LOW | `net/Socket.cpp:2001` — `getAgentString()` returns hardcoded `"COOLWSD HTTP Agent " COOLWSD_VERSION`, set as the `User-Agent` header on every outbound request (`net/HttpRequest.hpp:1814`). No config hook; a genuine (small, isolated) one-line source patch. `Server:` response header stays a single space by default (`security.server_signature` defaults `false`, `coolwsd.xml.in`) — i.e. inbound responses already don't self-identify unless explicitly turned on; leave that default alone. |
| **Multiple named sub-brand themes in one image** | Reseller / multi-brand scenarios (if ever needed) without separate builds | LOW (given the theme mechanism above already exists) | `?theme=<name>` param + multiple `browser/dist/<name>/` directories baked into the same image, exactly as Collabora's own build script demonstrates for CODE + Nextcloud themes. |
| **Admin console PAM bridge** | Interim non-static-password admin login until/if an OIDC-aware admin login exists | MEDIUM | `admin_console.enable_pam` (`coolwsd.xml.in:333-344`) dlopen's `libpam.so.0` at runtime (`wsd/FileServer.cpp` PAM block, ~L121-204) — optional, no link-time dependency, degrades gracefully if PAM isn't present in a hardened container. There is **no** native OIDC login for the admin console itself — flag as v2+, not v1. |
| **Explicit "no phone-home" posture** | Matches AO Cyber's privacy-first positioning; nothing to disable because nothing is on by default | LOW (verify + document, not build) | `INFOBAR_URL`/`FEEDBACK_URL`/`INFO_URL` are compile-time constants that default to **empty** unless `--with-infobar-url`/`--with-feedback-url`/`--with-info-url` are explicitly passed (`configure.ac:1290-1302`). `COOLWSD::processFetchUpdate()` (`wsd/COOLWSD.cpp:3827-3831`) returns immediately, no network call, when `INFOBAR_URL` is empty — so simply *not* passing those flags is already a correct, zero-effort privacy default. Worth stating explicitly in a security/privacy note so nobody "helpfully" wires them up later. |

### Anti-Features (Commonly Tempting, Deliberately Avoid)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|------------------|-------------|
| **Forking/patching LibreOffice core (engine/) for branding** | "True" white-label including native splash/about assets inside the office suite itself | Multi-hour separate build; explicitly Out of Scope per `.planning/PROJECT.md`; engine's own branding hooks (`engine/desktop/Package_branding.mk`, `engine/icon-themes/*/brand*`, `engine/vcl/source/app/brand.cxx`, `engine/setup_native/.../packinfo_brand.txt`) are native-desktop-installer artifacts users of the browser-delivered product never see | Consume Collabora's prebuilt core/LOKit; brand only `wsd/` + `browser/` — the layer 100% of EdenDocs users actually interact with |
| **Reusing Collabora's private `online-branding` repo/tooling** | `docker/from-source/build.sh:84-90,118-121` shows Collabora itself uses a polished `git@gitlab.collabora.com:productivity/online-branding.git` repo + `brand.sh` tool to theme CODE and Nextcloud Office | That repo is proprietary — EdenDocs has no credentials, and Collabora's own community FAQ states branding/theming CSS is one of their "(small) pieces of proprietary" IP they intend to open "over time" (WebSearch, `collaboraonline.github.io/post/faq/`) | Build an Eden-owned `branding.css`/`branding.js`/logo set from scratch using the same *public, open* `--with-app-branding` mechanism — the extension point is free; Collabora's specific artwork is not |
| **Reusing the Collabora/CODE name, wordmark, or logo anywhere** | Saves design time; "looks legitimate" out of the box | Collabora's product names/marks are trademarked separately from the MPL-2.0 source license (`collaboraonline.com/branding-guidelines/`); shipping them under Eden branding is a trademark issue, independent of the (fully permitted) source fork itself | Original Eden wordmark/emblem; keep only what MPL-2.0 actually requires (copyright headers, `COPYING`, `THIRDPARTYLICENSES`, `CODA-THIRDPARTYLICENSES.html`) |
| **Editing the WOPI protocol surface for "brand consistency"** (`discovery.xml` action names, `X-WOPI-*` headers, `CheckFileInfo` JSON shape, `access_token` flow) | Wanting everything, including protocol internals, to "feel like Eden" | `discovery.xml` (`wsd/` generated, templated from `discovery.xml`) and the WOPI request/response contract are the interoperability surface every WOPI host — including Eden-Biz — is coded against; renaming actions or reshaping `CheckFileInfo` breaks any WOPI client and violates the spec | Leave the protocol alone; cosmetic-only touches (e.g. a `favIconUrl` swap in `discovery.xml`) are fine, structural ones are not |
| **`--with-support-public-key` / `ENABLE_SUPPORT_KEY`** | Looks like it "unlocks" a more official/supported build path | This is Collabora's own commercial support-subscription gate. An unverified/expired key causes coolwsd to inject a `branding-unsupported` watermark CSS/JS into *every* served page (`wsd/FileServer.cpp:1279-1280`, `:2585-2593`, `:2654-2668`, `SUPPORT_KEY_BRANDING_UNSUPPORTED`) — the opposite of what a clean rebrand wants. EdenDocs is not a Collabora reseller and has no such key | Simply never pass `--with-support-public-key`; `ENABLE_SUPPORT_KEY` compiles to `0` and the entire watermark code path is compiled out (`configure.ac:2063-2078`) |
| **`--with-welcome-url`** | Quick built-in "what's new" onboarding pointing at a release-notes URL | Mutually exclusive at compile time with `brandProductName`/`brandProductURL`/`logoURL` — when `--with-welcome-url` is set, `configure.ac:1311-1329` forces `CONFIG_INTERFACE_FRAGMENT` to stay empty, meaning **those three config keys never even appear in `coolwsd.xml.in`** | Use plain `welcome.enable` (the `else` branch of the same configure.ac block) plus a custom `browser/dist/welcome/welcome.html`; this keeps `brandProductName`/`brandProductURL`/`logoURL` available |

---

## Feature Dependencies

```
[From-source build pipeline]
    └──requires──> [Prebuilt/consumed LibreOffice core (LOKit) — engine/, NOT forked]
    └──produces──> [browser/dist build output (webpack/browserify bundle target for --with-app-branding copy)]
                       └──enables──> [App-branding directory injection via --with-app-branding]
                                        └──produces──> [browser/<hash>/branding.css + branding.js served on every page]
                                                          └──enables──> [?theme=<name> selection of alternate browser/dist/<name>/ bundles]

[EdenDocs-branded container images]
    └──requires──> [From-source build pipeline]
    └──requires──> [App-branding directory injection]
    └──requires (Linux/server build)──> [Manual Docker COPY of custom welcome/ — NOT auto-copied like the macOS app target]

[AOID OIDC auth integration]
    └──is-owned-by──> [Eden-Biz / AOID as the WOPI host — NOT coolwsd]
    └──surfaces-through──> [WOPI CheckFileInfo identity fields (UserFriendlyName, UserExtraInfo avatar, IsAdminUser, ...) + opaque access_token]
    └──does-not-require──> [Any OIDC client code inside wsd/Auth.cpp — that class only does admin-console JWT]

[--with-welcome-url]
    └──conflicts──> [brandProductName / brandProductURL / logoURL runtime config keys]
       (configure.ac forces the latter three out of coolwsd.xml.in entirely when the former is set)

[--with-app-name (compiled APP_NAME)]
    └──is-fallback-for──> [user_interface.brandProductName runtime override]
       (brandProductName wins when non-empty; APP_NAME/compiled default only shows if it's left blank)

[Runtime css_variables / theme URL params]
    └──requires──> [CSS custom properties already declared in browser/css/color-palette.css + color-palette-dark.css]
    └──enhances──> [App-branding directory theming] (per-tenant override on top of the baked-in default brand)

[Upstream tracking workflow]
    └──constrains──> [All branding should land in additive branding-dir files or browser/dist output, not browser/src/ or wsd/ edits]
    └──unavoidably-conflicts-with──> [Window <title> edit (cool.html.m4:42), Admin console <title> edit (admintemplate.html:12)]
       (no config hook exists for either; keep both as small, isolated, easily-rebased commits)

[--with-support-public-key / ENABLE_SUPPORT_KEY]
    └──conflicts──> [Clean rebrand] (triggers "branding-unsupported" watermark injection if key is absent/invalid)
    └──recommended: never enable
```

### Dependency Notes

- **App-branding directory requires a completed build pipeline first:** `browser/dist/` doesn't exist until `make build-cool` (or equivalent) runs; branding can't be verified end-to-end until the from-source build objective is done. This is a hard ordering constraint for roadmap sequencing.
- **`--with-welcome-url` conflicts with `brandProductName`/`brandProductURL`/`logoURL`:** this is a real, easy-to-hit trap — whoever writes the `./configure` invocation for EdenDocs must NOT pass `--with-welcome-url`, or three of the table-stakes branding config keys silently disappear from `coolwsd.xml.in`.
- **AOID OIDC integration doesn't touch coolwsd's auth code:** the milestone-context phrasing ("OIDC/IdP auth integration") could mislead someone into looking for an OAuth client inside `wsd/Auth.cpp`. There isn't one, and there shouldn't be — the correct scope is Eden-Biz-side WOPI host work (minting `access_token` + populating `CheckFileInfo` identity fields), which is a different codebase/objective than EdenDocs itself.
- **Two source edits are unavoidable and low-conflict-risk in practice:** the `<title>` tag in `cool.html.m4` and the admin console `<title>` in `admintemplate.html` have no config hook. Both are single-line, well-isolated diffs — keep them as their own small commits so upstream merges only need a one-line re-application, not a re-review of a larger diff.
- **Support-key and welcome-url flags are both "don't touch" build flags** — neither should ever appear in EdenDocs's `./configure` invocation; document this explicitly in the build pipeline objective so nobody adds them "to be thorough."

---

## MVP Definition

### Launch With (v1)

- [ ] App-branding directory (`branding.css`, `branding.js`, logo SVGs, `toolbar-bg-logo.svg`) wired via `--with-app-branding` — this is the single highest-leverage item; it drives menubar logo, toolbar watermark, and (via the `%BRANDING_JS%` placeholder) both the main app and the admin console simultaneously
- [ ] `brandProductName` / `brandProductURL` / `logoURL` set in `coolwsd.xml.in` (confirm `--with-welcome-url` is NOT passed)
- [ ] `--with-app-name="EdenDocs"` and `--with-vendor="AO Cyber Systems"` compiled in as fallbacks
- [ ] Favicon replaced
- [ ] `<title>` patched in `cool.html.m4` and `admintemplate.html` (two small isolated commits)
- [ ] `help_url` pointed at Eden's own docs or emptied
- [ ] `browser/css/color-palette.css`/`color-palette-dark.css` custom properties overridden via a branding-dir CSS file (not edited in place)
- [ ] WOPI `CheckFileInfo` identity fields (`UserFriendlyName`, `UserExtraInfo` avatar, `IsAdminUser`) wired from whatever WOPI host stands in for Eden-Biz in v1 — needed for the AOID integration to be visible in the UI at all
- [ ] `INFOBAR_URL`/`FEEDBACK_URL`/`INFO_URL` left unset; `--with-support-public-key` never passed
- [ ] From-source build pipeline producing a working `coolwsd` + `browser/dist`
- [ ] EdenDocs-branded container image build/push

### Add After Validation (v1.x)

- [ ] Runtime `css_variables`/`theme` per-tenant reskinning wired into Eden-Biz's actual iframe embed URL — trigger: first multi-tenant Eden-Biz customer that wants a different accent color than the default Eden theme
- [ ] Admin console PAM bridge to system accounts — trigger: ops wants per-admin accountability instead of one shared static password
- [ ] `getAgentString()` outbound `User-Agent` patch ("EdenDocs HTTP Agent") — trigger: Eden-Biz/AOCore log correlation work needs it
- [ ] Fully custom `browser/dist/welcome/welcome.html` content (currently just needs the Docker `COPY` step, not authored content) — trigger: product wants a real onboarding narrative, not just "welcome disabled"

### Future Consideration (v2+)

- [ ] Native OIDC login for the admin console itself — upstream has none today; this is genuine new feature work, not a rebrand/integration task, and should not be pulled into v1 scope
- [ ] Multiple named sub-brand themes (`?theme=`) for reseller/white-label-of-white-label scenarios — no known customer need yet
- [ ] PWA manifest + `apple-touch-icon` set — upstream doesn't have one at all; only worth adding if EdenDocs wants installable-PWA behavior, which is not currently a stated requirement

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| App-branding directory (logo, colors, toolbar) | HIGH | LOW–MEDIUM | P1 |
| brandProductName/brandProductURL/logoURL config | HIGH | LOW | P1 |
| `<title>` tags (app + admin) | MEDIUM | LOW | P1 |
| Favicon | MEDIUM | LOW | P1 |
| Vendor/copyright/About dialog | MEDIUM | LOW | P1 |
| Help URL | LOW–MEDIUM | LOW | P1 |
| WOPI CheckFileInfo identity passthrough (AOID surface) | HIGH | MEDIUM | P1 |
| No-phone-home defaults (leave INFOBAR/FEEDBACK/INFO URLs unset) | MEDIUM (trust/privacy positioning) | LOW (verification only) | P1 |
| From-source build + branded container images | HIGH (blocks everything else) | HIGH | P1 |
| Per-tenant runtime `css_variables`/`theme` reskinning | HIGH (multi-tenant differentiator) | LOW | P2 |
| Admin console PAM bridge | MEDIUM | MEDIUM | P2 |
| Outbound `User-Agent` patch | LOW–MEDIUM | LOW | P2 |
| Custom welcome screen content | LOW–MEDIUM | MEDIUM | P2 |
| Native OIDC admin console login | MEDIUM | HIGH (new feature, not rebrand) | P3 |
| Multi-sub-brand `?theme=` support | LOW (no known demand) | LOW (mechanism exists) | P3 |
| PWA manifest/icons | LOW | MEDIUM | P3 |

**Priority key:** P1 = must have for a credible v1 rebrand; P2 = should have, add once v1 is validated; P3 = defer until real demand exists.

---

## Competitor / Derivative Feature Analysis

| Feature | Nextcloud Office (richdocuments integration) | CODE / generic self-hosted Collabora white-label | EdenDocs approach |
|---------|-----------------------------------------------|----------------------------------------------------|--------------------|
| Editor UI branding | Ships its own `nextcloud/branding.css`+`.js` under `browser/dist/nextcloud/`, selected via `theme=nextcloud`; Nextcloud's own admin **Theming app does NOT propagate into the editor** — a still-open upstream Nextcloud issue (`nextcloud/server#10264`) | `code-brand`/`collabora-online-brand` APT packages install `browser/dist/branding.*` alongside `coolwsd` | Same mechanism: Eden's own `branding.css`/`.js` via `--with-app-branding`, packaged into the EdenDocs image build (no separate APT package needed since it's baked into the container) |
| Per-tenant color overrides | Uses the documented `css_variables` hidden form field on the COOL iframe (Collabora's own officially blogged mechanism) | Same, when integrators want it | Same — Eden-Biz passes `css_variables`/`theme` on the iframe embed URL for multi-tenant reskinning |
| Identity/SSO | Nextcloud authenticates the user, then acts as WOPI host, populating `CheckFileInfo` from the Nextcloud session — Collabora never sees Nextcloud's login flow | Whatever the self-hosting WOPI host implements (ownCloud, Seafile, custom apps) — same pattern | Eden-Biz (fronted by AOID/OIDC) does the same: authenticates the user, then acts as WOPI host, passing identity through `CheckFileInfo` |
| Admin console | Left mostly stock, PAM optional; not typically customer-facing in a Nextcloud deployment | Left mostly stock | Rebranded chrome (title/colors) but functionally stock; PAM bridge as v1.x interim auth |
| Trademark/branding IP | Nextcloud doesn't reuse the Collabora mark; ships its own "Nextcloud Office" name | Resellers use the free `code-brand` open mechanism but their own artwork; the polished Collabora-authored theme (`online-branding` repo) is proprietary/partner-only | EdenDocs builds original Eden artwork against the same open mechanism — no Collabora trademark reuse |

---

## Sources

**Local source tree (HIGH confidence, primary evidence):**
- `/Users/justin/dev/EdenDocs/configure.ac` — `--with-app-branding`, `--with-app-name`, `--with-vendor`, `--with-help-url`, `--with-welcome-url`, `--with-feedback-url`, `--with-infobar-url`, `--with-info-url`, `--with-support-public-key`
- `/Users/justin/dev/EdenDocs/coolwsd.xml.in` — `admin_console`, `help_url`, `user_interface`, `WELCOME_CONFIG_FRAGMENT`/`CONFIG_INTERFACE_FRAGMENT`
- `/Users/justin/dev/EdenDocs/wsd/FileServer.cpp`, `wsd/FileServerUtil.cpp`, `wsd/ClientRequestDispatcher.cpp`, `wsd/COOLWSD.cpp`, `wsd/Auth.cpp`/`Auth.hpp`, `wsd/CollabSocketHandler.hpp`, `wsd/ClientSession.hpp`
- `/Users/justin/dev/EdenDocs/browser/Makefile.am`, `browser/html/cool.html.m4`, `browser/admin/admintemplate.html`, `browser/src/control/Control.AboutDialog.ts`, `browser/src/control/Control.Menubar.ts`, `browser/css/color-palette.css`, `browser/welcome/`
- `/Users/justin/dev/EdenDocs/docker/from-source/build.sh`, `docker/from-packages/Dockerfile`
- `/Users/justin/dev/EdenDocs/net/Socket.cpp`, `net/HttpRequest.hpp`
- `/Users/justin/dev/EdenDocs/discovery.xml`, `coolwsd.spec.in`
- `/Users/justin/dev/EdenDocs/.planning/PROJECT.md` — scope/constraints

**WebSearch, verified/corroborating (MEDIUM-HIGH confidence):**
- [Theming of Collabora Online (official blog)](https://www.collaboraonline.com/blog/theming-of-collabora-online/) — confirms the `css_variables` hidden-form-field mechanism
- [Press Kit & Branding Guidelines - Collabora Online](https://www.collaboraonline.com/branding-guidelines/) — trademark/logo usage policy
- [Docker image is missing branding · Issue #4993 · CollaboraOnline/online](https://github.com/CollaboraOnline/online/issues/4993) — confirms `branding-desktop.css`/`branding-mobile.css`/`branding-tablet.css`/`nextcloud/` layout convention
- [FAQ | Collabora Office - Community Page](https://collaboraonline.github.io/post/faq/) — "we will work to more explicitly open up our (small) pieces of proprietary theming/branding/CSS over time"
- [Use Collabora with native theme · nextcloud/all-in-one Discussion #6475](https://github.com/nextcloud/all-in-one/discussions/6475) and [Editing coolwsd.xml - Nextcloud community](https://help.nextcloud.com/t/editing-coolwsd-xml/170615) — confirms `use_integration_theme` real-world usage
- [[FEATURE] Make NC theme logo and color translate to Collabora's Logo and color theme · Issue #10264 · nextcloud/server](https://github.com/nextcloud/server/issues/10264) — confirms host-app theming does not auto-propagate into the editor, even for Nextcloud
- [How can I set a custom logo for light mode? - Collabora Online forum](https://forum.collaboraonline.com/t/how-can-i-set-a-custom-logo-for-light-mode/2702) — confirms `document-logo` CSS class as a real-world logo override target
- [online/configure.ac at master · CollaboraOnline/online (GitHub mirror)](https://github.com/CollaboraOnline/online/blob/master/configure.ac) — confirms current upstream `configure.ac` still exposes the same options as of this research date, and that active development is on Gerrit (matches `.planning/PROJECT.md` context)

---
*Feature research for: Collabora Online white-label/rebrand + platform integration (EdenDocs)*
*Researched: 2026-07-07*
