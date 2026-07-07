# Pitfalls Research: Forking / Rebranding Collabora Online (EdenDocs)

**Domain:** Brownfield fork of Collabora Online (`CollaboraOnline/online.mirror`) — rebrand, integrate, build-from-source
**Researched:** 2026-07-07
**Confidence:** HIGH (repo-grounded findings verified against live source in `/Users/justin/dev/EdenDocs`, cross-checked with Collabora's official SDK/trademark docs, forum, and GitHub issues)

This is not generic "how to fork a repo" advice. Every pitfall below is anchored to something concrete in this exact codebase (a config key, a source file, a build script, a branch) plus what the wider Collabora Online operator community has hit in production.

## Critical Pitfalls

### Pitfall 1: Rebranding by scattered find-and-replace instead of using the existing theming hooks

**What goes wrong:**
A team greps for `"Collabora"` (845+ files under `browser/` alone reference it), and starts hand-editing source files to swap in "EdenDocs." Every subsequent `git merge upstream/main` then conflicts on every one of those files, and conflict resolution becomes a full-time job within a few weeks because upstream commits daily (see Pitfall 2).

**Why it happens:**
The naive read of "845 files mention Collabora" is "845 files to rebrand." In reality the overwhelming majority of those hits are the MPL copyright header (`Copyright the Collabora Online contributors.`) required by `README.FILENOTICES.md` — not user-facing strings. The actual user-facing brand surface is small and already has first-party override points that Collabora itself uses to produce "CODE" vs "Nextcloud Office" themed builds from the same source:
- `user_interface.brandProductName` config key (`common/ConfigUtil.cpp:324`, consumed in `wsd/FileServer.cpp:1577-1581`) — replaces the `%PRODUCT_BRANDING_NAME%`, `%PRODUCT_BRANDING_URL%`, `%LOGO_URL%` placeholders in `browser/html/cool.html.m4`.
- The admin console has its own injection point: `<!--%BRANDING_JS%-->` → `brandJS` in `wsd/FileServer.cpp:2607`.
- `configure.ac` has a first-class `--with-app-branding=<dir>` flag (`APP_BRANDING_DIR`/`APP_HAS_BRANDING`, `configure.ac:2386-2397`) that stages a branding directory into the mobile app builds.
- Collabora's own proprietary `online-branding` repo (referenced in `docker/from-source/build.sh` as `git@gitlab.collabora.com:productivity/online-branding.git`, invoked via `./brand.sh $INSTDIR ... CODE` / `./brand.sh $INSTDIR ... NC-theme-community`) is exactly this pattern: an **out-of-tree** brand package applied as a post-build step, never a patch to `browser/src`.
- The default (no brand package installed) build literally falls back to the string `'Collabora Online Development Edition (unbranded)'` hardcoded in `browser/src/app/Socket.ts` (3 occurrences) — confirming Collabora's own default build ships *unbranded* and expects a brand layer on top, not in-tree edits.

**How to avoid:**
Build EdenDocs branding as an additive, out-of-tree layer, mirroring Collabora's own pattern:
1. Set `brandProductName`, a logo URL, and product branding URL via `coolwsd.xml` (config, not source).
2. Provide an `EdenDocs` equivalent of the `online-branding` repo/script: CSS overrides, logo assets, and the `%BRANDING_JS%`/admin-console injection, applied as a post-build/post-install step (own small repo or a `branding/` directory + script invoked from the Docker build), not as edits scattered through `browser/src`.
3. Reserve actual in-tree source edits for the handful of hardcoded fallback strings (e.g., the `Socket.ts` unbranded fallback) — patch those as small, isolated, easily-rebased diffs, and track them in a short allowlist file so every upstream merge has a known, minimal conflict surface.
4. Never touch `engine/` (the grafted LibreOffice core, 1.4GB of the repo) for branding — it has no user-facing product name and touching it multiplies merge pain for zero benefit.

**Warning signs:**
- `git diff upstream/main...eden-main -- browser/` shows brand-string edits spread across dozens/hundreds of files instead of a handful of config/branding files.
- Every `git merge upstream/...` produces conflicts in files that have nothing to do with branding logic (i.e., the diff noise itself, not the fix, is what conflicts).
- Someone says "we'll just sed s/Collabora/EdenDocs/g the whole repo."

**Objective to address:** Rebrand (establish the branding layer as the *first* rebrand task, before any other cosmetic change)

---

### Pitfall 2: Underestimating how fast the mirror moves, and merging main directly instead of choosing a tracking strategy

**What goes wrong:**
`CollaboraOnline/online.mirror` is not a slow, quarterly-cut project — `git log origin/main` shows multiple commits *per day* across engine, wsd, and browser simultaneously (verified: 20 commits sampled span 2026-07-01 through 2026-07-07, touching accessibility, Calc formula engine, browser UI, and desktop app packaging in the same week). A team that plans to "merge upstream monthly" will accumulate a multi-thousand-commit merge queue within a single sprint of neglect, at which point the merge is effectively a rewrite.

**Why it happens:**
Teams size the "upstream sync" objective based on a typical OSS dependency's release cadence (weekly/monthly tags), not realizing this mirror publishes every merged Gerrit change directly to `main` with no batching.

**How to avoid:**
- Decide explicitly whether `eden-main` tracks `upstream/main` (bleeding edge, matches this repo's default branch) or one of Collabora's own stabilized `distro/collabora/co-XX.04` branches. These branches exist precisely because Collabora's own downstream packaging doesn't track `main` directly — `distro/collabora/co-26.04` diverges from `main` by **5,567 files / ~99K insertions / ~35K deletions** in this repo, i.e., it's a point-in-time snapshot, not a rolling branch. Tracking a `distro/collabora/co-*` branch trades bleeding-edge features for a much smaller, batched merge surface, but means periodically jumping to the *next* `co-XX.04` branch when Collabora cuts one (that jump is itself a large, infrequent merge — plan for it, don't be surprised by it).
- Whichever is chosen, schedule upstream merges on a fixed cadence (weekly minimum) and treat "merge upstream" as a first-class recurring objective/task, not an occasional chore.
- Keep the diff between `eden-main` and its tracked upstream branch as small as possible (Pitfall 1) specifically so each merge stays mechanical.

**Warning signs:**
- No documented cadence for pulling `upstream` exists anywhere in the repo/planning docs.
- `git log upstream/main --since="1 week ago" --oneline | wc -l` returns dozens of commits and nobody has looked at them.
- A "quick rebrand" PR sits unmerged for weeks while upstream moves on.

**Objective to address:** Upstream workflow

---

### Pitfall 3: Trademark stripping done incompletely (or overzealously) — confusing "MPL-licensed code" with "free to use the Collabora name/logo"

**What goes wrong:**
Two failure modes, both common:
1. **Under-removal:** Team assumes MPL-2.0 licensing means the Collabora name/logo can stay because "it's open source anyway." Ships a public fork still displaying "Collabora Online" strings, the CODA/CollaboraOffice product descriptions, or Collabora's help/support links, while calling it "EdenDocs" elsewhere — creates exactly the "confusingly capitalizes on goodwill" / implied-endorsement problem Collabora's trademark policy exists to prevent.
2. **Over-removal:** Team strips *required* upstream copyright/license notices (the MPL header block, `THIRDPARTYLICENSES`, `CODA-THIRDPARTYLICENSES.html`, `AUTHORS`/`.mailmap`) thinking "no Collabora branding allowed anywhere," which is itself an MPL-2.0 violation (notices must be preserved even when trademarks are removed).

**Why it happens:**
Trademark and copyright/license are legally distinct regimes that get conflated. Collabora's own trademark policy (collaboraonline.com/trademark-policy) is explicit that they are separate: the MPL grants no trademark rights, and the license terms separately require that "if you distribute any open source component of the software, you must remove all Marks except those identifying Collabora's ownership or licensing of the component" — i.e., strip the *branding* but the copyright attribution stays.

**How to avoid:**
- Strip: product name/logo/wordmark in the UI (already largely config-driven, see Pitfall 1), the CODE/Collabora product descriptions in Docker `LABEL description=...` (see `docker/from-packages/Dockerfile:182`, hardcoded to Collabora's marketing copy), help URLs pointing at `collaboraoffice.com`/`collaboraonline.com`/`forum.collaboraonline.com` (`browser/src/errormessages.js:40` and others), and any domain-name use of "Collabora."
- Keep: the MPL header block per `README.FILENOTICES.md` ("Copyright the Collabora Online contributors" + SPDX-License-Identifier: MPL-2.0) on every file that carries it, `COPYING`, `THIRDPARTYLICENSES`, `CODA-THIRDPARTYLICENSES.html`, and `AUTHORS`/`.mailmap` attribution — these identify Collabora's/contributors' ownership and licensing, which the policy explicitly allows (and MPL-2.0 requires) to remain.
- Do not put "Collabora" in the `AO-Cyber-Systems/EdenDocs` product name, domain, or marketing copy beyond factual "based on Collabora Online" attribution — per policy, using a Mark in a product/app/service name requires written permission, and only narrow "integration"-style phrasing ("EdenDocs, built on Collabora Online technology") is safe without it.
- Treat this as a checklist item at every release, not a one-time pass — new upstream merges can reintroduce Collabora-branded strings (new help-URL error messages, new default product-name fallbacks) that need re-auditing.

**Warning signs:**
- `grep -ril "collaboraonline.com\|collaboraoffice.com" browser/ wsd/` still returns hits in a "final" build's user-visible surfaces (not just code comments/headers).
- Docker image `LABEL description` / `LABEL author` still reads "Collabora Productivity Ltd." verbatim in a distributed image.
- Marketing/legal has never been asked to review the trademark-policy checklist before a public release.

**Objective to address:** Rebrand

---

### Pitfall 4: Misreading MPL-2.0 as "just keep a LICENSE file" instead of file-level copyleft with source-availability obligations

**What goes wrong:**
MPL-2.0 is file-level copyleft, not project-level. Teams often import the wholesale "just don't open-source your whole product" mental model from GPL avoidance, and either (a) wrongly believe they must open-source unrelated EdenDocs/Eden platform code that merely *links against* or *deploys alongside* coolwsd (not required — MPL's "Larger Work" provision explicitly allows combining MPL files with proprietary code under a different license, as long as the MPL-covered files themselves stay MPL and available), or (b) the opposite mistake: modify MPL-covered files (e.g., `wsd/FileServer.cpp`, `browser/src/app/Socket.ts`) and fail to make the *modified source* of those specific files available to recipients of the distributed binary/image, which is the actual obligation.

**Why it happens:**
MPL-2.0 sits between permissive (MIT/Apache) and strong copyleft (GPL) licenses and is less commonly encountered, so teams apply whichever mental model they know instead of reading the actual terms.

**How to avoid:**
- Every file EdenDocs modifies under `browser/`, `wsd/`, `kit/`, `common/`, `net/` (all MPL-2.0, confirmed via `COPYING` + `README.FILENOTICES.md`) must keep its MPL header and remain source-available to anyone who receives the built/distributed EdenDocs image — this repo being public on GitHub (`AO-Cyber-Systems/EdenDocs`, per `.planning/PROJECT.md`) already satisfies that as long as `eden-main` stays public and in sync with what's actually shipped.
- New files EdenDocs adds (the branding layer, the OIDC/AOID integration glue, new Dockerfiles) are *not* required to be MPL — pick whatever license is appropriate — but don't accidentally slap MPL headers on genuinely new Eden-proprietary files, and don't accidentally strip MPL headers off files that started as Collabora source and were merely edited.
- `browser/LICENSE` (Leaflet-derived BSD-2-Clause code) is a *different* license living inside the MPL tree — don't blanket-relicense it as MPL, and preserve its notice separately.
- If EdenDocs ever ships without the full corresponding source (e.g., a stripped-down binary-only distribution to a specific customer), MPL-2.0 §3.2 source-availability obligations kick in explicitly — this is the actual trigger condition to watch for, not "did we publish a LICENSE file."

**Warning signs:**
- A modified MPL file loses its header during a rebrand pass (Pitfall 1's blunt sed approach is a common cause).
- Someone proposes making `eden-main` private "for competitive reasons" while still distributing built images — this breaks the source-availability obligation for the MPL-covered portions.
- No process exists to keep the publicly-visible `eden-main` branch in lockstep with what's actually deployed to customers.

**Objective to address:** Rebrand, Images (distribution)

---

### Pitfall 5: Assuming Collabora's CODE limits or SLA/support carry over to a self-built fork

**What goes wrong:**
Two related misconceptions: (a) believing the well-known "CODE is limited to 10 documents / 20 connections" restriction is baked into the open-source code, so it'll silently cap EdenDocs in production; (b) believing that because the fork is "basically Collabora Online," Collabora's paid support/SLA/security-patch cadence somehow applies to it.

**Why it happens:**
Both numbers are widely quoted in Collabora's own marketing/community content, so they feel like intrinsic code behavior rather than what they actually are.

**How to avoid:**
- The 10-document/20-connection cap is **not** in the MPL source used by this repo's from-source build. It's a `configure` flag: `configure.ac:1278-1294` defines `MAX_CONNECTIONS`/`MAX_DOCUMENTS` defaulting to **9999** unless overridden via `--with-max-connections`/`--with-max-documents`. That default-9999 ceiling only appears in Collabora's *distributed CODE packages/images*, built with those flags set low specifically to discourage production use of the free-tier binaries — it is not a property of the source. EdenDocs building from source with `docker/from-source/build.sh` (which does not pass those flags) inherits no such cap, and doesn't need to "unlock" anything.
- There is no Collabora SLA, security-patch guarantee, or support channel for a self-built, self-branded fork. EdenDocs (AO Cyber Systems) is now solely responsible for tracking security fixes landing on Gerrit/`online.mirror` and pulling them in on its own cadence (ties directly into Pitfall 2's upstream-tracking cadence) — there's no vendor safety net once the trademarks are removed and the images are self-published.
- Don't market or represent EdenDocs as having "the same SLA as Collabora Online" — it doesn't, unless AO Cyber Systems separately purchases and maintains a Collabora partner/support relationship, which is an independent business decision from the fork itself.

**Warning signs:**
- Build scripts or docs reference `--with-max-connections`/`--with-max-documents` being "unlocked" as if that were a meaningful feature (it was never locked in the source).
- Any customer-facing material implies Collabora provides support or SLA backing for EdenDocs.
- No internal owner is assigned to monitor Gerrit/`online.mirror` for security-relevant commits.

**Objective to address:** Build pipeline

---

### Pitfall 6: Build fragility from misunderstanding the engine/coolwsd/POCO coupling

**What goes wrong:**
Builds fail (or silently link against the wrong library) because of a deceptively simple detail: **POCO is no longer a separate system dependency.** It is built as part of `engine/external/poco/`, alongside zstd/libpng/openssl, as static libraries, and `configure` looks for it in the engine's build tree first. Two concrete failure patterns from `dev-notes/poco-build.md`:
1. `--with-poco-includes`/`--with-poco-libs` are **accepted but silently ignored with only a warning** — a team porting old build instructions (or an LLM-generated Dockerfile trained on older Collabora docs) that still pass these flags will get a build that *looks* configured correctly but is quietly picking up whatever POCO the engine tree produced, not what was requested.
2. Package-style builds (building `coolwsd` from a source tarball that does **not** include `engine/`, e.g., building against a separately-fetched core) must pass `--with-lo-builddir=<path-to-engine>` explicitly. Skipping this, or trying to inject POCO's paths via raw `CPPFLAGS`/`LDFLAGS` instead, leaves `POCOLIB` unset — `configure` won't add `-lexpat`, and the final link fails on `XML_Parse` (a confusing, low-level linker error far removed from the actual misconfiguration).

**Why it happens:**
This is a genuinely non-obvious coupling that changed relatively recently upstream (per the dev-notes, it used to be a separate dependency); build docs and muscle memory from older Collabora Online forks/tutorials lag behind.

**How to avoid:**
- For the from-source Docker path (`docker/from-source/build.sh`), build `engine/` first, in-tree, and let `configure` auto-detect POCO from the default `engine` build dir — don't add `--with-poco-*` flags at all.
- If EdenDocs ever builds `coolwsd` from an extracted/tarball source tree separate from a full engine checkout (e.g., a slimmer CI job), explicitly pass `--with-lo-builddir=<path>`, not raw compiler flags.
- Any `ENGINE_ASSETS` prebuilt-engine shortcut (`build.sh` supports downloading a prebuilt engine tarball to skip the multi-hour engine build) must ship `workdir/UnpackedTarball/poco/include` and `workdir/LinkTarget/StaticLibrary/libPoco*.a` inside that tarball, or the coolwsd build will fail to find POCO.
- Watch POCO version-specific bugs called out in `dev-notes/dependency-issues.md`: Poco 1.13.0 crashes on startup if `coolwsd.xml` log rotation is `"never"` (workaround: use `"monthly"`); Poco < 1.13.2 has a `seekg` bug; Poco between 1.12.5p2 and 1.14.2 (exclusive) has a logging perf regression. The engine currently bundles 1.15.3 — if EdenDocs ever overrides that version, re-check these.

**Warning signs:**
- Build logs show a POCO-related `configure` warning about ignored flags.
- Link errors mentioning `XML_Parse` or missing `-lexpat` with no obvious connection to POCO.
- A CI job builds `coolwsd` against a source tarball (not a full monorepo checkout) without `--with-lo-builddir`.

**Objective to address:** Build pipeline

---

### Pitfall 7: Treating the macOS dev build as equivalent to the Linux production build

**What goes wrong:**
Several coolwsd security/isolation mechanisms are Linux-specific: chroot-jailed kit processes built by `coolwsd-systemplate-setup` (copies `ld.so`, NSS libs, fonts, CA bundle into a chroot tree — logic branches on `glibc` vs `musl` via `ldd --version`), Linux `seccomp` syscall filtering (`security.seccomp` in `coolwsd.xml.in`, fatal by default if it can't be enabled), and Linux capabilities (`cap_sys_chroot`/`cap_sys_admin` on `coolforkit-caps`/`coolmount`, explicitly verified in a dedicated Docker build stage — see `docker/from-packages/Dockerfile`'s `verify-caps` stage, which **fails the build** if BuildKit doesn't preserve those capability xattrs across `COPY --from`). None of this exists in the same form on macOS. A team that validates "the build works" on a developer's Mac and assumes that translates to the Linux container target can ship a container that starts but runs with degraded/absent process isolation, or that flat out fails the `verify-caps` check in CI the first time someone touches the Dockerfile.

**Why it happens:**
`.planning/PROJECT.md` already correctly scopes macOS as "dev-convenience only," but this distinction is easy to lose once a build "just works" locally and the Linux/container path hasn't been exercised in a while.

**How to avoid:**
- Never treat a green macOS build as a signal that the Linux/Docker build path (the actual deliverable per `.planning/PROJECT.md`) is healthy — CI must build and run the real target (Linux containers, from-source) on every change that touches build config, `coolwsd.xml.in`, or the Dockerfiles.
- Preserve the `verify-caps` capability-check stage in any customized Dockerfile — it exists because BuildKit's handling of the `security.capability` xattr across `COPY --from` is not guaranteed on every builder/host, and losing it produces a jail that silently can't start (the whole point of the stage is "fail loud in CI, not silently in prod").
- If EdenDocs changes the base image (currently a ZenDiS/openCode distroless "hardened" base per the Dockerfile), re-verify glibc vs musl detection in `coolwsd-systemplate-setup` and re-run the capability check — both are base-image-sensitive.

**Warning signs:**
- CI only builds/tests on macOS runners, or the Linux Docker build is only exercised manually/occasionally.
- A Dockerfile change removes or skips the `verify-caps` stage "to speed up the build."
- `security.seccomp`/`security.capabilities` get flipped to `false` in `coolwsd.xml` to work around a local startup failure instead of fixing the underlying jail/capability setup.

**Objective to address:** Build pipeline, Images

---

### Pitfall 8: Trying to make coolwsd the identity provider instead of integrating OIDC at the WOPI host layer

**What goes wrong:**
Teams new to WOPI-based editors reach for coolwsd expecting it to behave like a normal web app with logins, sessions, and roles. It doesn't. `coolwsd.xml.in`'s `<storage><wopi>` section shows the actual trust model: coolwsd accepts an opaque `access_token` (with a TTL, `storage.wopi.access_token.default_lifetime_mins`) issued by whichever host is in the `alias_groups`/WOPI-host allowlist, and re-validates it against that host — coolwsd itself has **no concept of a user account, password, or OIDC client**. A team tries to bolt AOID's OIDC flow directly into coolwsd (patching `wsd/` to talk to an OIDC provider) instead of putting that logic in the WOPI host layer, and ends up fighting the architecture: duplicating identity logic that the WOPI host should own, and creating a large, hard-to-merge diff against `wsd/` that undermines Pitfall 1/2's "keep the diff small" goal.

**Why it happens:**
The WOPI split (storage/identity host vs. rendering/collab engine) is Microsoft Office Online's architecture, inherited by Collabora Online; it's not obvious from the outside that "the editor" and "the auth/storage system" are supposed to be two separate services communicating over a narrow, token-based protocol. Per Microsoft's WOPI spec (which Collabora's implementation follows): "the WOPI host that stores the file has the information about user permissions, not the WOPI client" — the access token is how the host tells coolwsd who's allowed to do what, not something coolwsd derives on its own.

**How to avoid:**
- Build (or reuse) a WOPI host component that sits in front of coolwsd, does the actual AOID OIDC login flow, and — after authenticating the user — mints a per-user/per-document `access_token`, implements `CheckFileInfo` (returning at minimum `BaseFileName`, `Size`, `OwnerId`, `UserId`, and `UserCanWrite`), and implements `GetFile`/`PutFile`. This is the actual integration surface, not `wsd/`.
- Register that WOPI host in coolwsd's `storage.wopi.alias_groups` (switch `mode="groups"` once there's more than one host/alias, per the config comments) — don't rely on the default `mode="first"` behavior in production.
- Keep coolwsd's role scoped to what it actually does: validate the token against the registered host, render/collaborate on the document, and expire/refresh the token per `access_token.refresh_timeout_secs`.
- If a fully server-mediated auth handoff between AOID and the WOPI host is needed (redirecting the WOPI request through a central login step per document open), that's what `indirection_endpoint` in `coolwsd.xml.in` exists for — it's a config-level integration point, not a code change.

**Warning signs:**
- Any PR touching `wsd/` to add OIDC client logic, session cookies, or password checks directly into coolwsd.
- Confusion in design docs about "where does the user log in" that doesn't clearly name a separate WOPI host component.
- `storage.wopi.alias_groups` left at default (`mode="first"`) in a production config meant to serve more than one integration.

**Objective to address:** Auth

---

### Pitfall 9: Confusing the Admin Console's auth with end-user/document auth

**What goes wrong:**
coolwsd *does* have one built-in auth system — but it's for the operational Admin Console only, and it's easy to mistake it for (or accidentally reuse it as) general application auth. Per `coolwsd.xml.in`'s `<admin_console>` block and `<security><jwt_expiry_secs>`: the admin console uses either PAM (`enable_pam`) or a single hardcoded `username`/`password` pair (explicitly marked `deprecated on most platforms` in favor of PAM or `coolconfig`), guards its websocket with a JWT that expires after 30 minutes by default, and is entirely separate from the WOPI/document access-token system in Pitfall 8. A team wiring AOID into "coolwsd auth" without distinguishing these two systems might end up either leaving the Admin Console on default/no credentials (since "auth is handled by AOID now"), or trying to make AOID gate the Admin Console via the same code path as document access, which doesn't exist.

**Why it happens:**
"Auth" reads as one concept; coolwsd actually has two independent, differently-shaped auth systems (operator console vs. document/WOPI) that share no code path.

**How to avoid:**
- Treat Admin Console access as a separate hardening task: enable PAM (or front it with AOID via a reverse-proxy auth gate, since coolwsd's own admin auth is basic/PAM-only) rather than leaving `admin_console.username`/`password` unset or default.
- Never assume "we did the AOID OIDC integration" (Pitfall 8) automatically secures `/adminws/` or the `admin.html` surfaces — verify both independently.
- If `enable_metrics_unauthenticated` is ever flipped to `true` for a monitoring integration, treat that as a deliberate, reviewed exception, not a default.

**Warning signs:**
- Admin console reachable on a production deployment with default/blank credentials.
- Security review checklist has one line item for "auth" instead of two (document/WOPI auth, admin console auth).

**Objective to address:** Auth

---

### Pitfall 10: WebSocket/reverse-proxy misconfiguration breaking collaborative editing in production

**What goes wrong:**
coolwsd's actual editing traffic rides a long-lived WebSocket (`/cool/<...>/ws`), and coolwsd's *own* client-side error message points straight at this class of bug: `browser/src/app/Socket.ts`'s failure handler literally says *"Failed to establish socket connection or socket connection closed unexpectedly. The reverse proxy might be misconfigured..."* and links to Collabora's proxy-settings docs. The concrete ways this bites in production, all confirmed against Collabora's own reference nginx config and real operator reports:
1. **Read-timeout too short:** nginx's default `proxy_read_timeout` is 60s; without an explicit long timeout (Collabora's own reference config uses `proxy_read_timeout 36000s` on the websocket + admin-websocket locations), idle-but-open editing sessions get silently dropped.
2. **Missing hop-by-hop headers:** `Upgrade`/`Connection` headers are hop-by-hop and are *not* forwarded automatically — they must be set explicitly (`proxy_set_header Upgrade $http_upgrade; proxy_set_header Connection "Upgrade";`), along with `proxy_http_version 1.1`.
3. **The websocket request falling through to the wrong location block:** `/cool/<...>/ws?WOPISrc=...` must match the websocket-specific proxy location; if a broader `/cool` regex catches it first, requests fail with "Invalid or unknown request," which surfaces to users as the same generic socket-failure message.
4. **SSL termination flag mismatch:** `coolwsd.xml`'s `ssl.enable`/`ssl.termination` must match reality — if a proxy terminates TLS and forwards plaintext, coolwsd needs `ssl.enable=false` + `ssl.termination=true` (or the equivalent `--o:ssl.enable=false --o:ssl.termination=true` env-var override used in the Docker entrypoint), otherwise scheme mismatches break WOPI callbacks and postMessage origin checks.
5. **Origin/port mismatches:** newer coolwsd versions do WebSocket origin checks; a proxy that doesn't strip the default `:443` from `Host`/origin headers consistently can produce "Rejecting WebSocket upgrade with: origin [...] expected [...]" failures.
6. **Sub-path deployments** (serving under e.g. `/edendocs/`) need `net.service_root` set to match, plus a proxy rewrite rule for the `/ws` suffix specifically — a generic `proxy_pass` rewrite that handles normal HTTP paths often mishandles the websocket path.

**Why it happens:**
Reverse-proxy config for long-lived WebSockets is a narrow, easy-to-get-wrong slice of nginx/Apache/Traefik/ingress-controller configuration that most teams only encounter once, and the failure mode (generic "socket closed") gives no hint which of the six causes above is the actual problem.

**How to avoid:**
- Start from Collabora's reference proxy config (linked from coolwsd's own error message: `sdk.collaboraonline.com/docs/installation/Proxy_settings.html`) and adapt it rather than writing proxy rules from scratch.
- Explicitly set `net.service_root` in `coolwsd.xml` if EdenDocs is served under a sub-path, and test the websocket path specifically (not just the static asset paths) after any proxy change.
- Add a smoke test that opens a document and holds the connection open past 60 seconds (past nginx's *default* timeout) as part of deployment verification — this is exactly the class of bug that "loads fine, breaks after a minute" symptoms indicate.
- Keep `ssl.enable`/`ssl.termination` and the proxy's actual TLS behavior in sync as a documented pairing, not something set once and forgotten.

**Warning signs:**
- Users report documents "randomly disconnect" or "stop syncing" after ~60 seconds of inactivity — classic default-timeout symptom.
- The exact `errormessages.js` "reverse proxy might be misconfigured" text appears in the browser console.
- Local dev testing only ever hits coolwsd directly on `127.0.0.1:9980` (as the current EdenDocs reference runtime does) and the reverse-proxy path has never been exercised before a "ready to deploy" milestone.

**Objective to address:** Images (deployment config), Upstream workflow (keep proxy docs/config current with upstream changes)

---

### Pitfall 11: Version skew between served browser assets and coolwsd itself, especially in clusters/rolling upgrades

**What goes wrong:**
coolwsd embeds a version hash into `/browser/dist/<hash>/...` asset URLs specifically to cache-bust after upgrades; when a client's cached page requests assets under a hash the server no longer recognizes, coolwsd logs (and can behave inconsistently around) a **"Client - server version mismatch, disabling browser cache"** warning (`wsd/FileServer.cpp:520` per the observed log format). In a single-node deployment this mostly self-heals on refresh. In a **load-balanced cluster** or a **rolling upgrade**, it doesn't self-heal the same way: if a client loads `cool.html` from a node running version A and a subsequent asset/websocket request lands on a node running version B (no session affinity), it repeatedly hits this mismatch, and editing sessions can break outright mid-upgrade. A related, separate failure: WOPI-integrating hosts (richdocuments/Nextcloud-style integrations, and by extension any custom EdenDocs WOPI host) can fail to notice a coolwsd version change at all and keep serving stale `discovery.xml`/capabilities data, producing confusing errors that look unrelated to the actual cause.

**Why it happens:**
Rolling upgrades and horizontal scaling are exactly the deployment shapes where this matters, and they're often not tested until production, after the single-node dev/staging setup (like the current `edendocs-code` container on `127.0.0.1:9980`) has already been signed off as "working."

**How to avoid:**
- Enable session/connection affinity (sticky sessions) at the load balancer for any multi-node deployment, so a given client's websocket and asset requests consistently hit the same coolwsd version during a rolling upgrade window.
- Roll upgrades node-by-node with affinity in place rather than a "kill all, start all new" deploy, or accept a brief editing-session interruption window and communicate it.
- Have the WOPI host (EdenDocs' own integration layer, per Pitfall 8) re-fetch `discovery.xml`/capabilities rather than caching it indefinitely, or provide an explicit "refresh" action — don't assume it auto-detects a coolwsd version bump.
- Verify all nodes report the same `coolwsd --version` / `/hosting/capabilities` output before considering a rolling upgrade complete.

**Warning signs:**
- The FileServer "version mismatch, disabling browser cache" log line appears in production logs outside of an active deploy window.
- Load balancer config has no session affinity and multiple coolwsd replicas are running different image tags simultaneously (normal mid-rollout, a bug if sustained).
- A WOPI integration keeps failing after a coolwsd upgrade until someone manually "re-saves" its server config (the known richdocuments workaround) — a sign the integration isn't refreshing cached version/capability info on its own.

**Objective to address:** Upstream workflow, Images (deployment/rollout process)

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|-----------------|------------------|
| Re-tag the upstream `collabora/code` Docker image as `edendocs` without a from-source build | Instant "branded" image, zero build pipeline work | No control point for OIDC/WOPI integration at the wsd layer, no license-compliance control over what's actually shipped, brand is skin-deep (admin console, error strings, help URLs stay Collabora's) | Acceptable only as the *current* interim state (matches `.planning/PROJECT.md`'s "local reference runtime" note) — never as the shipped deliverable |
| Hand-edit `browser/src` strings instead of using `brandProductName`/branding-layer hooks | Rebrand "looks done" fastest | Every upstream merge conflicts on touched files; drift compounds weekly given upstream's commit cadence (Pitfall 2) | Never — the hooks already exist upstream for this exact purpose |
| Patch `engine/` (LibreOffice core) directly for a quick fix instead of filing/waiting on an upstream Gerrit change | Unblocks a feature/bug fix immediately | 1.4GB, fastest-moving part of the tree; any direct edit multiplies merge pain and is explicitly out of scope per `.planning/PROJECT.md` ("core-level changes are upstream's domain") | Never for this project — engine is consumed prebuilt/vendored, not modified |
| Disable seccomp/capabilities checks to get a container to start locally | Unblocks local dev quickly | Ships (or normalizes shipping) a coolwsd process without its designed sandboxing; the `verify-caps` build stage exists specifically to prevent this reaching production | Only ever locally, with the change never committed to the production Dockerfile |
| Skip session affinity on the load balancer "for now" | Simpler LB config initially | Rolling upgrades and version-skew bugs (Pitfall 11) that are hard to reproduce and diagnose after the fact | Acceptable only for genuinely single-replica deployments |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|-----------------|-------------------|
| AOID (OIDC) | Trying to add OIDC login logic inside `wsd/` (coolwsd) directly | Implement OIDC + `access_token` minting in a separate WOPI host component; register it in `storage.wopi.alias_groups` |
| WOPI host generally | Leaving `alias_groups mode="first"` (default) in production with more than one integration/host | Switch to `mode="groups"` and explicitly list hosts/aliases once beyond a single trusted host |
| Reverse proxy (nginx/Apache/ingress) | Generic proxy config without websocket-specific `Upgrade`/`Connection` headers, long timeouts, and a dedicated `/cool/.../ws` location | Start from Collabora's reference proxy config; test the websocket path under sustained load, not just page load |
| Admin Console | Assuming AOID/OIDC integration secures it, or leaving default username/password | Enable PAM (or front with a separate auth gate) explicitly; verify independent of document-auth work |
| Docker base image swap (e.g. hardening/distroless base) | Swapping base image without re-verifying glibc/musl detection in `coolwsd-systemplate-setup` and the `verify-caps` capability stage | Re-run and re-verify both after any base image change |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|-----------------|
| No load-balancer session affinity across coolwsd replicas | Intermittent "version mismatch" cache-busting warnings, editing sessions that drop mid-collaboration during deploys | Enable sticky sessions; roll upgrades node-by-node | As soon as more than one replica exists and a deploy/scale event happens |
| Default `serverside_config.idle_timeout_secs`/`max_idle_subforkits` untuned for actual load | Kit-process churn (fork/chroot overhead repeated) under bursty usage, or excess idle memory held under low usage | Tune against real usage patterns once traffic profile is known; don't leave at defaults uncritically for a multi-tenant deployment | Noticeable once concurrent-document counts grow beyond a handful |
| Rebuilding `systemplate` chroot naively on every deploy instead of caching it | Slower container startup / image build times as the LO installation grows | `coolwsd-systemplate-setup` is designed to be run once per LO installation; cache/layer it appropriately in the image build | Becomes a real drag as build frequency increases with the upstream-tracking cadence (Pitfall 2) |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Leaving Admin Console on default/blank `username`/`password` (or PAM disabled) | Full operational control (killing documents, viewing metrics, live connections) exposed to anyone who finds `/browser/dist/admin/admin.html` | Enable PAM or `coolconfig`-managed credentials; never rely on the deprecated inline password field in production |
| Flipping `enable_metrics_unauthenticated` to `true` "for the monitoring dashboard" | Exposes internal operational metrics (`/cool/getMetrics`) without auth | Keep it `false`; use a scoped, authenticated scrape path or network-level restriction instead |
| Disabling `security.seccomp`/`security.capabilities` to work around a container startup issue | Removes the syscall-filtering and chroot-jail isolation that sandboxes per-document rendering processes — meaningful defense-in-depth for a multi-tenant document editor | Fix the underlying capability/base-image issue (Pitfall 7) instead of disabling the isolation |
| Disabling `ssl.ssl_verification` in production "because self-signed certs" | Storage/WOPI connections become vulnerable to MITM | Use a real CA chain (`ssl.ca_file_path`) even internally, or a properly trusted internal CA — reserve verification-disable for genuinely ephemeral test environments |
| Broadening `net.post_allow`/`storage.wopi.lok_allow` beyond what's needed (e.g. `0.0.0.0/0`-equivalent ranges) to "make an integration work" | Expands SSRF-style surface (external data sources reachable from inside edited documents via `=WEBSERVICE()`, image insertion, etc.) | Scope allow-lists tightly to actual WOPI host/storage addresses; treat broadening as a reviewed exception, not a quick fix |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-------------------|
| `brandProductName` left unset | Users see the literal fallback string `"Collabora Online Development Edition (unbranded)"` in error dialogs — a jarring, unprofessional, off-brand surface in an otherwise-rebranded product | Always set `user_interface.brandProductName` (and the logo/URL config) as part of the base deployment config, not an optional nice-to-have |
| Admin console left unbranded after the main editor UI is rebranded | Operators/admins see Collabora chrome in the one surface they use daily, undermining trust in the rebrand's completeness | Apply the `%BRANDING_JS%` admin-console hook (Pitfall 1) alongside the main UI branding, not as an afterthought |
| Help/error links still pointing at `forum.collaboraonline.com` / `collaboraoffice.com` | Users clicking "get help" land on Collabora's community forum for a product they don't recognize as "EdenDocs," confusing support flow and diluting brand | Point `help_url` and any hardcoded support links at Eden's own help surfaces |
| Watermark/default UI text untouched | Minor but visible brand inconsistency in exported/shared documents | Review `watermark`, welcome-message, and any other user-facing text config alongside the primary branding pass |

## "Looks Done But Isn't" Checklist

- [ ] **Rebrand:** Often missing the Admin Console (`%BRANDING_JS%` hook) and hardcoded fallback strings (e.g. `Socket.ts`'s "unbranded" text) — verify by hitting `/browser/dist/admin/admin.html` and forcing a socket-failure path, not just the main editor screen.
- [ ] **Trademark compliance:** Often missing Docker image `LABEL description`/`LABEL author` metadata and help/error URLs — verify with `grep -ri "collabora" browser/ wsd/ docker/` against the *shipped* build config, not just the editor UI.
- [ ] **Build pipeline:** Often "works on my Mac" but untested on the actual Linux/Docker target — verify by running the real from-source Docker build (`docker/from-source/build.sh` or the customized EdenDocs equivalent) in CI, including the `verify-caps` stage.
- [ ] **Auth integration:** Often covers the WOPI/document flow (AOID → WOPI host → coolwsd) but leaves the Admin Console on defaults — verify both paths independently.
- [ ] **Proxy/deployment config:** Often tested only via direct connection to coolwsd (e.g., today's `127.0.0.1:9980` reference setup) and never through the real reverse proxy with a sustained (>60s) websocket session — verify with a live document held open past a minute through the actual proxy path.
- [ ] **Upstream sync process:** Often exists as a one-time initial fork setup but has no scheduled recurring cadence documented anywhere — verify a merge cadence is written down and owned, not just that `upstream` remote exists.

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|-----------------|------------------|
| Rebrand done via scattered source edits, now blocking merges | HIGH | Revert the scattered edits; reimplement via `brandProductName`/branding-layer hooks (Pitfall 1); re-apply only the small, necessary hardcoded-fallback patches as an isolated, documented diff |
| Upstream drift grown too large to merge cleanly | HIGH | Snapshot current `eden-main` state, do the merge in a dedicated branch with extended time budget, prioritize `engine/` (usually resolves mechanically since EdenDocs shouldn't touch it) before the smaller, EdenDocs-owned `wsd/browser` diff; afterward, tighten the merge cadence (Pitfall 2) |
| Trademark violation found in an already-distributed image | MEDIUM | Patch and re-release immediately (branding config + Docker label fixes are fast); audit all previously-shipped image tags/versions for the same issue; add a pre-release trademark checklist so it's caught before shipping next time |
| Broken `systemplate`/chroot jail after a Docker base image change | LOW | Re-run `coolwsd-systemplate-setup` against the new base image; re-verify glibc/musl detection and the `verify-caps` capability stage before shipping |
| Version-skew incident during a cluster rolling upgrade | LOW–MEDIUM | Roll back to a single consistent version across all nodes; add load-balancer session affinity before retrying the rollout |

## Pitfall-to-Objective Mapping

| Pitfall | Prevention Objective | Verification |
|---------|------------------------|----------------|
| Scattered-edit rebrand causing merge conflicts | Rebrand | `git diff upstream/main...eden-main` touches only branding-layer files + a small documented allowlist, not hundreds of scattered files |
| Upstream drift from unclear tracking strategy/cadence | Upstream workflow | A written, owned merge cadence exists; `git log upstream/main --since=<cadence>` is reviewed on schedule |
| Trademark under/over-removal | Rebrand | Pre-release checklist covering UI strings, Docker labels, help URLs, and confirming MPL notices/attribution are intact |
| MPL-2.0 misunderstanding (source availability) | Rebrand, Images | Public `eden-main` stays in sync with what's actually built/shipped; modified MPL files keep headers |
| Assuming CODE limits/SLA carry over | Build pipeline | Build config reviewed to confirm no artificial `--with-max-connections`/`--with-max-documents` caps are inherited; support/SLA ownership documented internally |
| engine/coolwsd/POCO build coupling misconfigured | Build pipeline | CI from-source build succeeds without `--with-poco-*` flags on the in-tree path; package-style builds explicitly set `--with-lo-builddir` |
| macOS dev build mistaken for the production build | Build pipeline, Images | CI builds and smoke-tests the actual Linux/Docker target (including `verify-caps`) on every relevant change |
| coolwsd mistaken for the identity provider | Auth | WOPI host component (not `wsd/`) owns OIDC/AOID login and token minting; `wsd/` diff against upstream stays minimal |
| Admin Console auth confused with document auth | Auth | Admin Console PAM/credentials verified independently of the AOID/WOPI integration test |
| WebSocket/proxy misconfiguration | Images (deployment config) | Deployment verification includes a document session held open past 60s through the real reverse proxy, not direct coolwsd access |
| Browser-asset/wsd version skew in clusters | Upstream workflow, Images | Rolling-upgrade runbook requires session affinity and same-version confirmation across nodes before considering a rollout complete |

## Sources

- Repository (primary, ground truth): `/Users/justin/dev/EdenDocs` — `coolwsd.xml.in`, `configure.ac`, `wsd/FileServer.cpp`, `common/ConfigUtil.cpp`, `browser/src/app/Socket.ts`, `README.FILENOTICES.md`, `COPYING`, `browser/LICENSE`, `docker/from-source/build.sh`, `docker/from-packages/Dockerfile`, `docker/from-source/README.md`, `coolwsd-systemplate-setup`, `dev-notes/poco-build.md`, `dev-notes/dependency-issues.md`, `.gitreview`, `.git-hooks/commit-msg`, and `git log`/`git diff` against `origin/main` and `origin/distro/collabora/co-26.04`
- [Collabora Trademark Policy](https://www.collaboraonline.com/trademark-policy/)
- [Collabora Press Kit & Branding Guidelines](https://www.collaboraonline.com/branding-guidelines/)
- [Collabora Online MPLv2 licensing terms](https://www.collaboraonline.com/terms/collabora-online-mplv2/)
- [What's the difference between CODE and Collabora Online?](https://www.collaboraonline.com/case-studies/differences-between-code-and-collabora-online/)
- [Collabora Online SDK — How to integrate](https://sdk.collaboraonline.com/docs/How_to_integrate.html)
- [Collabora Online SDK — Configuration](https://sdk.collaboraonline.com/docs/installation/Configuration.html)
- [Microsoft WOPI — Key concepts](https://learn.microsoft.com/en-us/microsoft-365/cloud-storage-partner-program/rest/concepts)
- [GitHub: Opening admin console adds version mismatch warnings to logs · Issue #7650](https://github.com/CollaboraOnline/online/issues/7650)
- [nextcloud/richdocuments — Collabora updates are undetected · Issue #4267](https://github.com/nextcloud/richdocuments/issues/4267)
- [Possible nginx reverse-proxy issue: Bad URI syntax · Issue #10388](https://github.com/CollaboraOnline/online/issues/10388)
- [Collabora forum: "Failed to establish socket connection" / proxy misconfiguration reports](https://forum.collaboraonline.com/)
- Mozilla Public License 2.0, full text (`http://mozilla.org/MPL/2.0/`) — file-level copyleft, "Larger Work," and source-availability (§3.2) provisions

---
*Pitfalls research for: Collabora Online fork/rebrand (EdenDocs)*
*Researched: 2026-07-07*
