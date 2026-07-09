<!--
Copyright 2026 AO Cyber Systems

SPDX-License-Identifier: MPL-2.0

This Source Code Form is subject to the terms of the Mozilla Public
License, v. 2.0. If a copy of the MPL was not distributed with this
file, You can obtain one at http://mozilla.org/MPL/2.0/.
-->

# EdenDocs Reference WOPI Host (AUTH-01..04)

Operator/integrator runbook for `wopi-host/` — the standalone Go service
that authenticates users against AOID via OIDC and fronts `coolwsd`
(EdenDocs' Collabora Online fork engine) as a WOPI host. Written for someone
who has never seen Objective 3: it covers architecture, quickstart, every
config env var, the AOID admin steps required before anything works, the
production trust config, and the documented v1 scope limits.

For the research this doc is condensed from — full source citations, claim
tables, and pitfall write-ups — see
`.planning/objectives/03-aoid-authentication-integration-oidc/3-RESEARCH.md`.
This document states what EXISTS (commands, env vars, config blocks); it
does not re-derive the reasoning.

## 1. What this is

`wopi-host/` is a brand-new, additive, isolated Go module (its own
`go.mod`) living outside `wsd/` and `browser/`. It is the ONLY place OIDC
code exists anywhere in this repo — `coolwsd` itself needs zero code
changes for any part of this objective (config only, see §5).

At a session level it does two unrelated jobs, and keeping them unrelated
is the single most important thing to understand about this service:

1. **OIDC relying party (browser <-> wopi-host <-> AOID).** The WOPI host
   runs the AOID Authorization Code + PKCE flow (`/login` -> AOID
   `/oauth/authorize` -> AOID redirects back to `/callback` -> the host
   exchanges the code for AOID's id_token/access_token). These AOID
   tokens live in the WOPI host's own server-side session **and go no
   further** — never forwarded to the browser, never forwarded to coolwsd.
2. **WOPI host (coolwsd <-> wopi-host).** Once a browser session exists,
   `/open` mints a completely separate, WOPI-host-invented, short-lived
   **opaque token** (a `uuid.NewString()` — never a JWT, never derived from
   the AOID token) and hands it to coolwsd via the launch page. coolwsd
   never inspects this token's structure; it just echoes it back on every
   `CheckFileInfo`/`GetFile`/`PutFile` callback so the WOPI host can look up
   which session/user/file it maps to.

```
 Browser                     wopi-host                        AOID
   |  GET /open?file=...        |                                |
   |--------------------------->|                                |
   |                            |  (no session) redirect /login   |
   |                            |------------------------------->|
   |                            |         AOID login UI          |
   |<---------------------------+--------------------------------|
   |  code, state (callback)    |                                |
   |--------------------------->|  exchange code -> id_token,    |
   |                            |  access_token (kept SERVER-SIDE)|
   |                            |------------------------------->|
   |  launch page (POST form:   |                                |
   |  access_token=<uuid>,      |                                |
   |  access_token_ttl=...)     |                                |
   |<---------------------------|                                |
   |                                                              |
   |        iframe loads coolwsd cool.html?WOPISrc=...            |
   |------------------------------------------------------------>|
   |               coolwsd -> wopi-host CheckFileInfo/GetFile/PutFile
   |               (WOPI opaque uuid token, X-WOPI-Proof signed)  |
```

**Never conflate the two credentials.** Treating the AOID access token (or
id_token) as the WOPI `access_token` — e.g. passing the raw AOID JWT
straight into `cool.html?...&access_token=<AOID JWT>` — is the single most
likely integration mistake; see §6 item 1 for why it also can't work
(the AOID JWT means nothing to coolwsd's proof-key expectations).

Route map (`wopi-host/cmd/wopi-host/main.go`):

| Route | Purpose | Auth |
|---|---|---|
| `GET /healthz` | liveness probe | none |
| `GET /login` | starts AOID Authorization Code + PKCE | none |
| `GET /callback` | AOID redirect target; mints the browser session | none (validates `state`) |
| `GET /logout` | clears the browser session | cookie |
| `GET /` | document list | cookie (`RequireLogin`) |
| `GET /open?file=...` | launch page; mints a WOPI access_token | cookie (`RequireLogin`) |
| `GET /wopi/files/{id}` | CheckFileInfo | WOPI access_token query param |
| `GET /wopi/files/{id}/contents` | GetFile | WOPI access_token query param |
| `POST /wopi/files/{id}/contents` | PutFile | WOPI access_token query param |

The `/wopi/*` routes are deliberately NOT wrapped in the browser's
`RequireLogin` cookie guard — coolwsd calls them server-to-server, with no
browser cookie jar, authenticating purely via the opaque token in the query
string (verified further by the `X-WOPI-Proof` signature, §5).

## 2. Quickstart (fake-aoid, local dev)

Building `wopi-host` requires org GitHub access even though EdenDocs itself
is a public repo — `eden-platform-go/platform/oidcrp` (the org-mandated
OIDC relying-party helper; never hand-rolled, see 3-RESEARCH.md "Don't
Hand-Roll") is a private module at `github.com/AO-Cyber-Systems/eden-platform-go`,
imported as `github.com/aocybersystems/eden-platform-go`. Two commands,
once, before the first `go build`/`go mod download` in this tree:

```bash
git config --global url."https://<YOUR_GITHUB_TOKEN>@github.com/AO-Cyber-Systems/".insteadOf "https://github.com/aocybersystems/"
go env -w GOPRIVATE=github.com/aocybersystems/*
```

(CI uses the same two commands, driven by the `GITOPS_PAT` repo secret —
see `.github/workflows/wopi-host.yml`.)

Build all three binaries (`wopi-host`, `fake-aoid`, `e2e-probe`):

```bash
cd wopi-host
go build ./cmd/...
```

Run against **fake-aoid** — a minimal in-repo OIDC IdP
(`wopi-host/cmd/fake-aoid`) that fabricates a working discovery document +
token endpoint, used for headless dev/CI so no real AOID instance is
required to exercise the OIDC plumbing:

```bash
# terminal 1 — fake AOID on 8092
EDENDOCS_FAKE_AOID_ADDR=127.0.0.1:8092 ./fake-aoid

# terminal 2 — coolwsd must already be running and configured to trust
# this host (see §5) — e.g. via the objective-1 dev loop on native port 9980

# terminal 3 — wopi-host on 8091
EDENDOCS_WOPI_AOID_ISSUER=http://127.0.0.1:8092 \
EDENDOCS_WOPI_OIDC_CLIENT_ID=edendocs-e2e \
EDENDOCS_WOPI_OIDC_CLIENT_SECRET=e2e-secret \
EDENDOCS_WOPI_COOLWSD_URL=http://127.0.0.1:9980 \
./wopi-host
```

Then visit `http://127.0.0.1:8091/open?file=hello.odt` in a browser.

**Run the full behavioral e2e instead of doing this by hand** — one
deterministic command builds all three binaries, brings up coolwsd
(native or docker mode), fake-aoid, and wopi-host in the correct order,
drives a headless WebSocket client through the whole
OIDC-login-to-PutFile chain, and asserts BOTH positive trust (a registered
WOPI host is accepted) and negative trust (an unregistered host is
rejected):

```bash
# native mode (CI): builds/runs ./coolwsd from the repo root, port 9980
COOLWSD_MODE=native ./wopi-host/scripts/wopi-e2e.sh

# docker mode (local, no repo-root coolwsd build needed): ephemeral
# collabora/code container on host port 9981 — never touches a
# long-running reference container that may already own 9980
COOLWSD_MODE=docker ./wopi-host/scripts/wopi-e2e.sh
```

`COOLWSD_MODE=auto` (the default) picks native if `./coolwsd` exists at the
repo root, docker otherwise. See `wopi-host/scripts/wopi-e2e.sh` for the
full 10-assertion list and `03-04-SUMMARY.md` for a captured passing
transcript.

## 3. Env reference

Every setting is `EDENDOCS_WOPI_`-prefixed (source of truth:
`wopi-host/internal/config/config.go`).

| Var | Default | Notes |
|---|---|---|
| `EDENDOCS_WOPI_LISTEN_ADDR` | `127.0.0.1:8091` | HTTP bind address. Port **8080 is permanently banned project-wide** — `config.validate()` refuses to start if this contains `:8080`. |
| `EDENDOCS_WOPI_PUBLIC_URL` | `http://<ListenAddr>` | Origin the **browser** uses to reach this host; also the `PostMessageOrigin` value coolwsd's editor iframe posts messages against. |
| `EDENDOCS_WOPI_BASE_URL` | same as `PublicURL` | Origin **coolwsd** uses to call back into this host. Differs from `PublicURL` in docker mode, where coolwsd runs in a container and can't reach `127.0.0.1` on the host — e.g. `http://host.docker.internal:8091`. This split exists specifically for that docker-vs-native asymmetry; leave it equal to `PublicURL` for a plain native/native deployment. |
| `EDENDOCS_WOPI_AOID_ISSUER` | *(none — required)* | The AOID OIDC issuer URL (e.g. a deployed instance, or a local dev instance from `~/dev/aoid`). Empty is only valid before the OIDC relying party is exercised; `main.go` `log.Fatal`s at startup if it can't complete discovery against this issuer. |
| `EDENDOCS_WOPI_OIDC_CLIENT_ID` | *(none — required)* | Output of AOID admin `CreateClient` — see §4. |
| `EDENDOCS_WOPI_OIDC_CLIENT_SECRET` | *(none — required)* | Output of AOID admin `CreateClient` — the plaintext secret is emitted **exactly once**, at creation time; see §4's capture warning. |
| `EDENDOCS_WOPI_STATE_KEY` | ephemeral (32 random bytes, generated at startup) | Base64-encoded HMAC key signing the OIDC `state`/session cookies. Unset means sessions do not survive a process restart — fine for a reference/dev deployment, never appropriate for production (a startup `WARN` log line says so explicitly). |
| `EDENDOCS_WOPI_COOLWSD_URL` | `http://127.0.0.1:9980` | Origin of the coolwsd instance this host fronts (used for `/hosting/discovery` fetches). |
| `EDENDOCS_WOPI_DATA_DIR` | `./data` | Filesystem root the storage package reads/writes documents from — a reference/demo store, not S3 or similar (3-RESEARCH.md Open Question 4; a documented, deliberate v1 scope reduction, not an oversight). |
| `EDENDOCS_WOPI_TOKEN_TTL` | `15m` (Go duration syntax) | Lifetime of a minted WOPI `access_token`. |
| `EDENDOCS_WOPI_REQUIRE_PROOF` | `true` | Whether `X-WOPI-Proof` signature verification (§5) is enforced. Only ever set `false` for isolated local debugging — never in CI or production. |
| `EDENDOCS_WOPI_LOG_FILE` | *(none — stdout only)* | When set, logs are ALSO appended to this file (in addition to stdout) — the structured log-line contract `wopi-host/scripts/wopi-e2e.sh` greps (`wopi: CheckFileInfo...status=200`, `proof: verified ok`, etc.). |

## 4. AOID client registration runbook

**AOID has no static-config client registration path.** Registration is
exclusively a tenant-scoped, admin-authenticated Connect-RPC call
(`CreateClient`, gated by `requireAdmin` + `HasTenantAccess` —
`aoid/internal/oauth/admin/service.go`). A hand-written SQL `INSERT` into
`aoid.oauth_clients` is forbidden: it bypasses bcrypt secret hashing, the
schema's `CHECK` constraints (e.g. `auth_method='none' <=> client_type='public'`),
and audit logging (`ActionClientCreated`). The RPC is the only sanctioned
write path.

**Steps:**

1. Authenticate as an AOID admin with access to the target tenant.
2. Call the AOID admin `CreateClient` RPC with these parameters:
   - `client_type = "confidential"` — server-side redirect flow, has a
     `client_secret` (the WOPI host never runs in a public/native-app
     context).
   - `token_endpoint_auth_method = "client_secret_basic"`
   - `grant_types = ["authorization_code"]` (add `"refresh_token"` only if
     long-lived sessions are wanted; not required for v1).
   - `redirect_uris = ["http://127.0.0.1:8091/callback"]` for a local dev
     run — **register per environment**: dev, CI (fake-aoid does not need
     a real AOID registration, only real deployments do), and each
     production/staging deployment need their own `redirect_uris` entry
     matching that environment's public origin + `/callback`.
   - `auto_consent = true` — recommended for this WOPI host: it is a
     first-party org service, so skipping the consent-screen click on
     every login is appropriate (not appropriate for a third-party
     integration).
3. **Capture the response's `client_secret` immediately — it is emitted
   exactly once, at creation time, and is never retrievable again.** Store
   it in whatever secret manager the deployment target uses; it becomes
   `EDENDOCS_WOPI_OIDC_CLIENT_SECRET`.
4. Set the three env vars from §3 (`EDENDOCS_WOPI_OIDC_CLIENT_ID`,
   `EDENDOCS_WOPI_OIDC_CLIENT_SECRET`, `EDENDOCS_WOPI_AOID_ISSUER`) on the
   wopi-host process for that environment.

This is a one-time, per-environment operational task — it must precede any
real-AOID run of the WOPI host (it is not needed for the fake-aoid
quickstart in §2, which fabricates its own discovery/token endpoints and
needs no AOID-side registration at all).

## 5. coolwsd trust (production)

The ONLY change coolwsd needs for this entire objective is config, not
code: `storage.wopi.alias_groups` in the deployed `coolwsd.xml` (the
generated/runtime config file — never hand-edit `coolwsd.xml.in` for a
per-deployment value like this).

```xml
<!-- coolwsd.xml — storage/wopi section -->
<storage>
  <wopi allow="true">
    <alias_groups mode="groups">
      <group>
        <host allow="true">https://wopi.example.aocyber.ai</host>
        <alias>https://wopi-alt.example.aocyber.ai</alias>
      </group>
    </alias_groups>
  </wopi>
</storage>
```

One `<group>` per trusted WOPI host origin; `<alias>` entries are optional
alternate origins for the same host (e.g. a reverse-proxy hostname).

**`mode="groups"` is mandatory and its absence fails silently.** If a
`<group>` element exists under `<alias_groups>` but the `mode` attribute is
not exactly the literal string `"groups"` — or `alias_groups` is omitted
entirely — coolwsd does not refuse to start; it just `LOG_ERR`s and
silently ignores the group (`wsd/HostUtil.cpp` `parseAliases`). The
symptom is a WOPI request that fails as "untrusted" with no obvious
config-level error. Verify a deployment's trust config **behaviorally**,
not by reading the XML: request a document through the real flow (or run
`wopi-host/scripts/wopi-e2e.sh`, which asserts both the positive-trust
accept AND a negative-trust reject against a deliberately unregistered
host) and confirm coolwsd's log/response shows the host was accepted.

**Proof-key: there is no "enable proof-key" toggle on the coolwsd side.**
coolwsd unconditionally signs every outbound WOPI request
(`X-WOPI-Proof`/`X-WOPI-TimeStamp`/`X-WOPI-ProofOld`) **whenever a
`proof_key` file exists** at `{COOLWSD_CONFIGDIR}/proof_key` (generate via
`coolconfig generate-proof-key`, or `ssh-keygen -t rsa -N "" -m PEM`).
"Proof-key validation enabled" for this objective means the WOPI host
**verifies** those headers and rejects unproven requests
(`EDENDOCS_WOPI_REQUIRE_PROOF=true`, the default, §3) — that is the only
place "enabled" applies; there is nothing to flip in `coolwsd.xml` beyond
ensuring the key file exists.

**`frame_ancestors` knob.** If the editor iframe renders blank in a
browser, check coolwsd's `net.frame_ancestors` — it must include the WOPI
host's own origin (e.g. `--o:net.frame_ancestors=http://127.0.0.1:8091` for
a local dev run) or the browser's frame-ancestors CSP will refuse to embed
`cool.html` inside the WOPI host's launch page.

## 6. v1 limitations & decisions

Each item below is a deliberate, research-backed v1 decision, not a silent
gap. All four are source-verified against AOID's actual OIDC surface and
coolwsd's actual CheckFileInfo parsing — see 3-RESEARCH.md for the full
claim tables and pitfall write-ups.

1. **`IsAdminUser` is hard-coded `false`.** AOID v1.0's id_token
   (`{tnt, nonce, email, email_verified, name}`) and `/oauth/userinfo`
   carry no admin/role claim reachable by a generic OIDC relying party.
   The only candidate, the access token's `ent[]` (entitlements) claim,
   is unusable as-is: AOID hardcodes `aud=["aoedge"]` on every user-flow
   access token regardless of which client requested it (AOID's own
   `// TODO(per-client): drive this from oauth_clients.audience`), which
   breaks a standard audience check against the WOPI host's own
   `client_id`. Shipping `IsAdminUser: false` unconditionally is safe and
   avoids that audience footgun. **v2 path:** register the WOPI host as an
   AOID client, provision specific accounts with an entitlement string
   (e.g. `"edendocs:admin"`) via AOID's admin RPC, then locally
   decode+verify the raw AOID access token (not the id_token) with
   `oidc.Config{SkipClientIDCheck: true}` (verify only `iss` + signature +
   `exp`, since `aud` is meaningless here) and check for that string in
   `ent[]`.
2. **No avatar claim, ever.** AOID's OIDC surface (id_token, userinfo,
   access token) has no `picture`/avatar claim anywhere. `UserExtraInfo`
   deliberately omits the `avatar` key entirely — the browser's own
   `LOUtil.setUserImage` already falls back gracefully to a built-in
   `user.svg` (and also falls back on a broken/unreachable avatar URL via
   an `error` event listener), so no synthetic-avatar service was built to
   compensate. **v2 path:** none needed unless AOID itself ever adds a
   picture claim.
3. **`SupportsLocks: false`.** Full MS-WOPI lock-conflict semantics
   (`X-WOPI-Lock`/`X-WOPI-Override`, 409-conflict response shapes) were
   deliberately descoped for a v1 reference host — a legitimate scope
   reduction, not an omission (3-RESEARCH.md Open Question 1). **v2 path:**
   trace `wsd/wopi/WopiStorage.cpp`'s Lock/RefreshLock/UnlockFile call
   sites and implement standard WOPI locking before advertising
   `SupportsLocks: true`.
4. **Single-tenant issuer.** The reference host resolves exactly one
   `EDENDOCS_WOPI_AOID_ISSUER` at startup; AOID also supports per-tenant
   discovery (`/{tenant_slug}/.well-known/openid-configuration`), which
   this v1 does not consume (3-RESEARCH.md Open Question 3). **v2 path:**
   accept a `?tenant=` param on the launch page and resolve the per-tenant
   issuer/provider/verifier per request, still built once and cached per
   tenant (never per-request — see the org-wide `oidcrp` caching rule).

## 7. Port policy + public-repo/private-dependency note

**Ports in play for this objective:**

| Port | Service |
|---|---|
| 8091 | wopi-host (dev/CI default listen port) |
| 8092 | fake-aoid (dev/CI headless OIDC IdP fixture) |
| 9980 | coolwsd, native mode |
| 9981 | coolwsd, ephemeral docker container used only by `wopi-host/scripts/wopi-e2e.sh`'s docker mode — never the long-running reference container that already owns `127.0.0.1:9980` |

**Port 8080 is permanently banned project-wide** — never bind, serve, curl, or reference it for anything in this service, enforced in code by `config.validate()` (§3).

**Public-repo / private-dependency note.** EdenDocs is a public repository;
`eden-platform-go` (home of the org-mandated `oidcrp` OIDC helper this
service consumes) is private. Anyone building `wopi-host` from a fresh
clone of EdenDocs therefore needs org GitHub access first — the exact two
commands are in §2's Quickstart. There is no way around this for v1: the
org's "don't hand-roll OIDC" mandate applies to public and private repos
alike.
