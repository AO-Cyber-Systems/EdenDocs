# Objective 3: AOID Authentication Integration (OIDC) - Research

**Researched:** 2026-07-08
**Domain:** OIDC Authorization Code flow (Go relying party) + WOPI host protocol (Collabora Online / coolwsd)
**Confidence:** HIGH (coolwsd/wsd behavior and AOID claim shapes verified directly from source; MEDIUM on sdk.collaboraonline.com cross-check — official docs site was unreachable, see Sources)

<user_constraints>
## User Constraints (from CONTEXT.md)

`03-CONTEXT.md` contains no `## Decisions`, `## Claude's Discretion`, or `## Deferred Ideas` sections — no prior `/devflow:discuss-objective` session locked any choices for this objective. Everything below is Claude's discretion, informed by this research.

### Cross-Repo Considerations (verbatim from 03-CONTEXT.md)

#### Sibling repos
_(no matches)_

#### eden-libs candidates
_(no matches)_

#### Org Project overlap
_(no matches)_

_Misfiling check: no mismatch detected._

**Research note:** the automated cross-repo scan found no matches, but direct investigation (this research) DID find a highly relevant eden-libs candidate the scan missed: `eden-platform-go/platform/oidcrp` is the mandated, already-proven OIDC relying-party helper (see Don't Hand-Roll). Treat this as a correction to the cross-repo pass, not a contradiction of it.

### Hard platform constraints (from task context, not CONTEXT.md, but binding)
- NEVER use port 8080 anywhere. coolwsd native listens on 9980. Local web verification defaults to 8091.
- MPL-2.0 discipline: zero OIDC/OAuth code in `wsd/` (AUTH-04 explicit). Browser (`browser/src`) changes only if identity display genuinely requires them.
- Weekly upstream merges: new code must be additive/isolated, never edit tracked upstream files.
- Org backend language is Go (AOID, AOCore, eden-platform-go, aofamily backends all Go); don't hand-roll OIDC — use `eden-platform-go/platform/oidcrp`.
</user_constraints>

<phase_requirements>
## Objective Requirements

| ID | Description | Research Support |
|----|-------------|-------------------|
| AUTH-01 | A reference WOPI host service (new deployable, outside `wsd/`/`browser/`) authenticates users against AOID via OIDC Authorization Code flow | AOID's real `/oauth/authorize`, `/oauth/token` endpoints + PKCE requirement documented in Standard Stack/Architecture; `eden-platform-go/platform/oidcrp.BuildAuthURL`/`ExchangeAndVerify` is the mandated, already-proven implementation (Code Examples); AOID's own `internal/federation/connector/oidc.go` is a working reference of the exact same flow |
| AUTH-02 | WOPI host implements CheckFileInfo/GetFile/PutFile, mints short-lived WOPI `access_token`s, serves a launch page embedding the editor iframe | CheckFileInfo/GetFile/PutFile field contract verified against `wsd/wopi/WopiStorage.cpp` (Code Examples, Architecture Patterns); launch-page/iframe contract (`cool.html?WOPISrc=...` + `access_token` form field) verified against `wsd/FileServer.cpp` `UserRequestVars` (Architecture Patterns); WOPI access_token is a WOPI-host-minted opaque/JWT credential, NOT the AOID access token — see Pitfall 2 |
| AUTH-03 | AOID identity visible in editor UI via CheckFileInfo `UserFriendlyName`/avatar/`IsAdminUser` | Full CheckFileInfo↔AOID claim mapping table below; confirmed zero `browser/src` changes needed (`Control.UserList.ts`, `LOUtil.setUserImage` already render these fields with graceful avatar fallback to `user.svg`); `IsAdminUser` gap documented as Pitfall 1 with a firm v1 recommendation |
| AUTH-04 | coolwsd trusts WOPI host via `storage.wopi.alias_groups` (`mode="groups"`) + proof-key validation, zero OIDC code in `wsd/` | Exact `alias_groups` XML shape and `HostUtil.cpp` parsing semantics in Architecture Patterns; proof-key algorithm (coolwsd→WOPI-host direction only, no wsd toggle needed) fully documented in Code Examples/Pitfall 3 |
</phase_requirements>

## Summary

This objective wires a brand-new, standalone Go service — the **reference WOPI host** — in front of coolwsd. The WOPI host is the ONLY place OIDC code is allowed to live: it runs AOID's Authorization Code + PKCE flow (via the org's existing `eden-platform-go/platform/oidcrp` helper, not hand-rolled), then acts as a classic WOPI host for coolwsd, implementing `CheckFileInfo`/`GetFile`/`PutFile` and minting its own short-lived WOPI `access_token`s (unrelated to AOID's tokens — see Pitfall 2). coolwsd needs zero code changes for any of this: its WOPI-host trust model (`storage.wopi.alias_groups mode="groups"`) and its proof-key signing (`wsd/ProofKey.cpp`) are both already fully general and config/file-driven. The browser (`browser/src`) also needs zero changes — `UserFriendlyName`, `UserExtraInfo.avatar` and `IsAdminUser` are already fully parsed (`wsd/wopi/WopiStorage.cpp`) and rendered (`Control.UserList.ts`, `LOUtil.setUserImage`) by stock code, including a graceful fallback to a generic avatar icon when `avatar` is absent.

The one real technical gap is identity **richness**: AOID v1.0's OIDC surface (id_token, `/oauth/userinfo`, access-token claims) carries `sub`, `email`, `email_verified`, `name` — but no avatar/picture claim anywhere, and no admin/role claim usable by a generic OIDC RP without also hitting AOID's mTLS-gated introspection endpoint. The `ent[]` (entitlements) claim on AOID access tokens is the closest thing to a per-client admin flag, but it requires the WOPI host to locally decode the raw AOID access token (not just the id_token) — and AOID v1.0 hardcodes access-token `aud=["aoedge"]` for ALL user-flow tokens regardless of which OAuth client requested them, which breaks the default `oidc.Verifier` audience check if the WOPI host's own `client_id` is used as the expected audience. This is a concrete, must-handle pitfall (Pitfall 1), not a blocker — the recommendation below (skip client-id audience check on the WOPI host's own access-token decode, since AOID controls the issuer+signature already) resolves it cleanly.

**Primary recommendation:** Build the reference WOPI host as a new, additive, in-repo Go module at `wopi-host/` (own `go.mod`, isolated from coolwsd's C++ build and from `wsd/`/`browser/`), consuming `eden-platform-go/platform/oidcrp` for the OIDC RP flow, listening on port **8091** in dev/CI, registered with coolwsd purely via `coolwsd.xml`'s `storage.wopi.alias_groups mode="groups"` — no `wsd/` edits, no `browser/src` edits.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/aocybersystems/eden-platform-go/platform/oidcrp` | current (local dev via `replace` to `~/dev/eden-libs/eden-platform-go`, or `GOPRIVATE`+git `insteadOf` in CI) | OIDC RP flow: `BuildAuthURL`, `ExchangeAndVerify`, `ProviderCache`, `VerifierCache`, `State` (HMAC-signed state+nonce+PKCE), `ClaimMap` | Org-mandated "don't hand-roll" library; already used in production by AOID's own inbound federation connector and by all three AOFamily backends (`ai`/`browser`/`connect`) per the canonical `docs/sso-integration.md` contract |
| `github.com/coreos/go-oidc/v3` (v3.17.0, pulled transitively via oidcrp) | v3.17.0 | Discovery, JWKS, JWT verification underneath oidcrp | Same library AOID itself uses; keeps the whole org on one OIDC client stack |
| `golang.org/x/oauth2` (v0.36.0, transitive) | v0.36.0 | OAuth2 `Config`/`Endpoint`/code exchange underneath oidcrp | Standard Go OAuth2 client, wrapped by oidcrp |
| `golang-jwt/jwt/v5` (v5.3.1, transitive) | v5.3.1 | JWT parsing (also used to independently decode the AOID access token's `ent` claim — see Pitfall 1) | Same JWT lib AOID's own `internal/oauth/tokens.go` uses to mint tokens; symmetric library keeps claim-shape assumptions aligned |
| Go stdlib `net/http` | Go 1.22+ (org standard, matches aofamily's Go 1.26) | WOPI host HTTP server (CheckFileInfo/GetFile/PutFile handlers, launch page, OIDC callback) | Org convention: aofamily backends are plain `net/http` + Go 1.22+ pattern routing, no framework. No reason for the WOPI host to diverge. |
| Go stdlib `crypto/rsa` + `crypto/sha256` | stdlib | WOPI proof-key (`X-WOPI-Proof`) verification | coolwsd signs with RSA-SHA256 (PKCS#1 v1.5) via Poco's `RSADigestEngine`; Go's `rsa.VerifyPKCS1v15` with `crypto.SHA256` is the direct, dependency-free equivalent — no third-party WOPI SDK needed for this one primitive |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/google/uuid` | current | Generate WOPI `access_token` / session identifiers | Same lib AOID itself uses (`internal/oauth/tokens.go`) for `jti`; consistent org convention |
| In-memory map or Redis (discretionary) | n/a | WOPI `access_token` → (AOID identity, file, expiry) session store | An in-memory map is sufficient for a *reference* host (single instance, short-lived tokens); document Redis as the production-scale swap-in, don't build it now |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `eden-platform-go/platform/oidcrp` | Hand-rolled `go-oidc`+`oauth2` wiring | Explicitly forbidden by org convention (`aoid/CLAUDE.md`, `aofamily/docs/sso-integration.md` both mandate oidcrp/eden-platform-go for any OIDC RP in the org) and would re-implement state/nonce/PKCE/provider-caching bugs oidcrp already fixed (Pitfall 4 in `sso-integration.md`: never construct a verifier per-request) |
| In-repo `wopi-host/` Go module | Separate standalone repo (`eden-wopi-host` or similar) | Considered and rejected — see Architecture Patterns "WOPI host placement" for the full reasoning |
| Local decode of AOID access token for `ent[]` | Call AOID's `/oauth/introspect` | `/oauth/introspect` is `RequireMTLS`-gated (mTLS client cert), designed for AOEdge, not a generic OIDC RP; would require provisioning the WOPI host with an mTLS client cert purely to read one claim already present in the JWT it already holds — unnecessary complexity for v1 |

**Installation:**
```bash
cd wopi-host
go mod init github.com/ao-cyber-systems/edendocs-wopi-host
go get github.com/aocybersystems/eden-platform-go@latest   # or `replace` to local ~/dev/eden-libs/eden-platform-go in dev
go get github.com/google/uuid
```

## Architecture Patterns

### Recommended Project Structure
```
EdenDocs/                          # this repo (coolwsd/wsd, unchanged)
├── wsd/                           # ZERO changes for this objective
├── browser/                       # ZERO changes for this objective
├── coolwsd.xml.in                 # ONE new <alias_groups mode="groups"> block (config only, not code)
└── wopi-host/                     # NEW, additive, isolated Go module
    ├── go.mod                    # own module, own go.sum — does not entangle with any C++ build
    ├── cmd/wopi-host/main.go     # entrypoint; :8091 in dev/CI
    ├── internal/
    │   ├── oidcauth/             # AOID OIDC RP: BuildAuthURL, callback handler, ExchangeAndVerify
    │   ├── session/              # WOPI access_token mint + lookup (maps token -> AOID identity + file)
    │   ├── wopi/                 # CheckFileInfo / GetFile / PutFile handlers
    │   ├── proof/                # X-WOPI-Proof (RSA-SHA256) verification against coolwsd discovery pubkey
    │   └── launch/               # launch page: builds cool.html?WOPISrc=... POST form, embeds iframe
    └── testdata/                 # fixture documents for GetFile/PutFile round-trip tests
```

**Placement recommendation (firm):** in-repo `wopi-host/` directory, NOT a separate repo.
Reasoning:
1. **Mergeability is a non-issue either way** — it's a wholly new top-level directory that upstream Collabora will never create, so weekly `git merge upstream/master` can never conflict with it, in-repo or not.
2. **CI locality wins.** Objective 1 already established the CI dev-loop (`scripts/eden/build.sh`, `scripts/eden/smoke-test.sh`, native port 9980, no containers). Keeping the WOPI host in-repo lets one CI job build coolwsd AND the WOPI host, then run a true end-to-end smoke test (OIDC login → WOPI CheckFileInfo → editor loads) with zero cross-repo checkout, secret-sharing, or version-pinning complexity.
3. **A nested Go module keeps it fully isolated anyway** — `wopi-host/go.mod` means `go build ./...` at the repo root never touches it, and nothing in `wsd/`/`browser/` can accidentally import it. This gets the isolation benefit of a separate repo without the operational cost.
4. If the WOPI host later needs its own independent release cadence/versioning distinct from EdenDocs (coolwsd) releases, splitting it out is a pure `git subtree`/history-preserving extraction — cheap to defer, expensive to do prematurely.

**Port recommendation (firm): 8091.** Never 8080 (hard rule). Not 9980 (coolwsd's own native port — collision). 8091 is the project's already-designated "local web verification" port per the global hard rule, so using it for the WOPI host's dev/CI HTTP listener keeps exactly one blessed non-8080 port in play across the whole EdenDocs verification story, and CI's `smoke-test.sh` can curl both `127.0.0.1:9980` (coolwsd) and `127.0.0.1:8091` (WOPI host) with no ambiguity.

### Pattern 1: OIDC Authorization Code + PKCE against AOID (via oidcrp)
**What:** The WOPI host's launch page redirects the browser to AOID's `/oauth/authorize`; AOID redirects back to the WOPI host's callback with `code`+`state`; the WOPI host exchanges the code for tokens and verifies the id_token.
**When to use:** Every user session start (AUTH-01).
**Example (mirrors AOID's own federation connector, `internal/federation/connector/oidc.go`, and the org-normative `aofamily/docs/sso-integration.md` §2):**
```go
// Source: /Users/justin/dev/aoid/internal/federation/connector/oidc.go (verified pattern)
//         + /Users/justin/dev/eden-platform-go/platform/oidcrp/flow.go
provider, err := oidc.NewProvider(ctx, aoidIssuer) // e.g. https://id.aocyber.ai or tenant-scoped issuer
oauth2Cfg := &oauth2.Config{
    ClientID:     wopiHostClientID,      // registered via AOID admin RPC, see Pattern 3
    ClientSecret: wopiHostClientSecret,  // "confidential" client — server-side redirect flow
    Endpoint:     provider.Endpoint(),
    RedirectURL:  "https://wopi-host.example/callback",
    Scopes:       []string{"openid", "profile", "email"},
}
verifier := provider.Verifier(&oidc.Config{ClientID: wopiHostClientID}) // OK for id_token: aud == client_id there

// /login handler:
authURL, state, err := oidcrp.BuildAuthURL(oauth2Cfg, state, nonce, pkceVerifier, nil)

// /callback handler:
idToken, oauth2Token, claims, err := oidcrp.ExchangeAndVerify(ctx, oauth2Cfg, verifier, code, pkceVerifier, nonce)
// claims.Email, claims.Name now available (IDTokenClaims: email, email_verified, name — see claim table)
// oauth2Token.AccessToken is the RAW AOID access token — keep it if you need `ent[]` (Pitfall 1)
```
Provider/verifier caching MUST follow `oidcrp.ProviderCache`/`VerifierCache`, built ONCE at process startup, never per-request (org-normative rule, `sso-integration.md` §2, "Pitfall 4").

### Pattern 2: WOPI CheckFileInfo / GetFile / PutFile (coolwsd's actual parsing contract)
**What:** coolwsd calls `GET {WOPISrc}?access_token=...` for CheckFileInfo, `GET {WOPISrc}/contents?access_token=...` for GetFile, `POST {WOPISrc}/contents?access_token=...` for PutFile.
**When to use:** AUTH-02.
**Verified field contract (source: `wsd/wopi/WopiStorage.cpp` lines ~140-270, cross-checked against coolwsd's own test fixture `test/WopiTestServer.hpp`):**
```go
// Source: wsd/wopi/WopiStorage.cpp (field names/behavior verified directly from coolwsd source)
type CheckFileInfoResponse struct {
    // REQUIRED (coolwsd hard-errors / degrades without these)
    BaseFileName string `json:"BaseFileName"`
    Size         int64  `json:"Size"`
    OwnerId      string `json:"OwnerId"`

    // Effectively required: falls back to "UnknownUser"/"UnknownUser_<id>" + LOG_ERR if empty
    UserId           string `json:"UserId"`
    UserFriendlyName string `json:"UserFriendlyName"`

    UserCanWrite bool `json:"UserCanWrite"`

    // UserExtraInfo is a free-form JSON object; "avatar" is the recognized convention key
    // (see wsd/CollabSocketHandler.cpp: extra->optValue<std::string>("avatar", "")).
    // OMIT the key entirely if you have no avatar URL — browser falls back gracefully (see Pitfall 5).
    UserExtraInfo map[string]any `json:"UserExtraInfo,omitempty"`

    // Top-level boolean, NOT nested inside UserExtraInfo (the nested `is_admin` form is
    // explicitly deprecated and logged as such by coolwsd — always emit this key at top level).
    IsAdminUser bool `json:"IsAdminUser"`

    PostMessageOrigin string `json:"PostMessageOrigin,omitempty"` // auto-upgraded http->https if coolwsd SSL enabled
    WatermarkText     string `json:"WatermarkText,omitempty"`
    SupportsLocks     bool   `json:"SupportsLocks,omitempty"`
    SupportsRename    bool   `json:"SupportsRename,omitempty"`
    UserCanRename     bool   `json:"UserCanRename,omitempty"`
}
```
GetFile: stream the document bytes with `Content-Type` matching the file, HTTP 200.
PutFile: accept the raw body as the new document bytes, validate `X-WOPI-Lock`/`X-WOPI-Override` per standard WOPI semantics (not further investigated here — standard, not AOID/coolwsd-specific — flagged as Open Question 3).

### Pattern 3: AOID client registration (admin RPC, not static config)
**What:** Before the OIDC flow can work, the WOPI host must exist as an `aoid.oauth_clients` row.
**When to use:** One-time setup task, must precede AUTH-01 implementation work — a real task seam for the planner.
**Verified (source: `aoid/internal/oauth/admin/service.go` `CreateClient`, `aoid/migrations/0015_oauth_clients.up.sql`):** AOID has NO static-config client registration path — it is exclusively a tenant-scoped, admin-authenticated Connect-RPC call (`requireAdmin(ctx)` + `identity.HasTenantAccess`). The call:
- Sets `client_type = "confidential"` (server-side redirect flow, has a `client_secret`)
- Sets `token_endpoint_auth_method = "client_secret_basic"`
- Sets `grant_types` to include `authorization_code` (+ `refresh_token` if long sessions wanted)
- Sets `redirect_uris` to the WOPI host's `/callback` URL(s) — must be registered per environment (dev/CI/prod)
- The plaintext `client_secret` is emitted exactly once at creation — the plan must include "capture and store the WOPI host's client_secret" as an explicit step, not an afterthought
- Optionally sets `auto_consent = true` if the WOPI host is a first-party org service (recommended, avoids an extra consent-screen click on every login for an internal reference deployment)

### Pattern 4: coolwsd trust config — `storage.wopi.alias_groups mode="groups"`
**What:** The ONLY change coolwsd needs. Config-only, in `coolwsd.xml` (the generated/runtime file — never edit `coolwsd.xml.in` for a value like this beyond what's already parameterized; use `--o:` overrides or a post-`configure` injection like Objective 1's `build.sh` already does for branding).
**Verified (source: `wsd/HostUtil.cpp`, `wsd/HostUtil.hpp`):**
```xml
<!-- coolwsd.xml — storage/wopi section -->
<storage>
  <wopi>
    <alias_groups mode="groups">
      <group>
        <host allow="true">http://127.0.0.1:8091</host>
        <alias>http://localhost:8091</alias>
      </group>
    </alias_groups>
  </wopi>
</storage>
```
Critical semantics (source-verified, not assumed):
- If `alias_groups` is absent entirely → coolwsd falls back to legacy "compat" mode. If a `<group>` element exists but the `mode` attribute is not exactly `"groups"` → **the groups are silently rejected** (`LOG_ERR`, not a hard failure) — the plan MUST assert `mode="groups"` is present, this is a known footgun.
- Each `<host>`/`<alias>` entry is parsed by a regex (`parseAlias`): a plain hostname\[:port\] gets dot-escaped into a literal-match regex automatically; anything not matching that shape is treated as a raw regex — so `http://127.0.0.1:8091` is safe as literal config, no regex-escaping needed by the plan.
- Aliases are added to BOTH the alias-rewrite map AND the WOPI host allow-list (`WopiHosts`) — one config block does double duty (trust + URL rewriting for reverse-proxy scenarios).
- Multiple `|`-separated hosts in a single `<host>` element is deprecated (logged warning) — use multiple `<host>`/`<alias>` elements instead, one per origin.

### Pattern 5: Launch page → iframe → WebSocket handoff (the actual production contract)
**What:** How the WOPI host's launch page gets the user into the editor.
**Verified (source: `wsd/FileServer.cpp` `UserRequestVars`, `wsd/ClientRequestDispatcher.cpp` `allowedOrigin`):**
1. WOPI host resolves the coolwsd action URL from `GET {coolwsd}/hosting/discovery` (standard WOPI discovery XML; `urlsrc` template for the file's extension, per MS-WOPI convention — coolwsd's discovery XML also embeds the proof-key public modulus/exponent here, see Pattern 6).
2. WOPI host builds the iframe target URL: `{urlsrc}WOPISrc={urlencode(CheckFileInfoURL)}`.
3. WOPI host renders an HTML page containing an iframe (or a POST-submitted hidden form targeting the iframe — **recommended over query-string GET**, see Pitfall 6) carrying `access_token` and `access_token_ttl` — coolwsd's `FileServer.cpp` (`UserRequestVars::extractVariable`) reads `access_token`/`access_token_ttl`/`no_auth_header` from a Poco `HTMLForm`, which transparently accepts EITHER a GET query string OR a POST body — both work, POST is the safer convention (avoids the token landing in access logs / `Referer` headers).
4. Once loaded, `browser/src/main.js` reads the injected `accessToken` JS global and appends it as a **query-string parameter** when it opens the actual `/cool/<encoded-uri>/ws` WebSocket to coolwsd — confirmed in `wsd/ClientRequestDispatcher.cpp`'s `allowedOrigin()`, whose comment explicitly states "the access_token rides in the query string, not cookies" for the WS upgrade step (this is coolwsd's own documented anti-XSRF reasoning, not this research's inference).
5. From that point, coolwsd calls back to the WOPI host's CheckFileInfo/GetFile using that same `access_token` — this is a WOPI-host-minted token, see Pitfall 2.

### Pattern 6: WOPI proof-key verification (WOPI host implements this, coolwsd needs zero config)
**What:** Every outbound coolwsd→WOPI-host request carries `X-WOPI-TimeStamp`, `X-WOPI-Proof`, `X-WOPI-ProofOld`. The WOPI host verifies these to confirm the request genuinely came from the coolwsd instance it trusts (defense in depth beyond IP/hostname allow-listing).
**Verified (source: `wsd/ProofKey.cpp`, full algorithm):**
```go
// Source: wsd/ProofKey.cpp Proof::GetProof / SignProof — Go-side verification is the mirror operation.
// coolwsd's proof key is unconditional if COOLWSD_CONFIGDIR/proof_key exists — no coolwsd.xml toggle.
// The WOPI host fetches the public modulus/exponent from coolwsd's /hosting/discovery XML
// <proof-key value="..." modulus="..." exponent="..."/> attribute (base64).

func VerifyProof(pub *rsa.PublicKey, accessToken, uri string, timestampHeader, proofB64 string) error {
    ticks, err := strconv.ParseInt(timestampHeader, 10, 64)
    if err != nil { return err }
    // .NET ticks: 100ns units since 0001-01-01. Reject if too far from now (replay window).
    epoch := time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)
    ts := epoch.Add(time.Duration(ticks * 100) * time.Nanosecond)
    if time.Since(ts).Abs() > 20*time.Second { return errors.New("proof timestamp out of window") }

    uriUpper := strings.ToUpper(uri)
    buf := new(bytes.Buffer)
    writeLenPrefixed := func(s string) {
        binary.Write(buf, binary.BigEndian, int32(len(s)))
        buf.WriteString(s)
    }
    writeLenPrefixed(accessToken)
    writeLenPrefixed(uriUpper)
    tb := make([]byte, 8)
    binary.BigEndian.PutUint64(tb, uint64(ticks))
    binary.Write(buf, binary.BigEndian, int32(len(tb)))
    buf.Write(tb)

    sig, err := base64.StdEncoding.DecodeString(proofB64)
    if err != nil { return err }
    hash := sha256.Sum256(buf.Bytes())
    return rsa.VerifyPKCS1v15(pub, crypto.SHA256, hash[:], sig) // nil error == valid
}
```
Note: verify against BOTH `X-WOPI-Proof` and `X-WOPI-ProofOld` (coolwsd currently duplicates them, since it has no key-rotation implemented yet — `TODO: implement proper rotation` in `ProofKey.cpp` — accept a match against either).

### Anti-Patterns to Avoid
- **Editing `wsd/` or `browser/src` for any part of this objective:** Not needed anywhere — every requirement is satisfiable via config (`coolwsd.xml` `alias_groups`) + a wholly external WOPI host. If a task plan proposes a `wsd/` or `browser/src` diff, that is a signal something upstream was misunderstood — re-derive from `WopiStorage.cpp`/`Control.UserList.ts` first.
- **Treating the AOID access token as the WOPI `access_token`:** These are two unrelated credentials in two unrelated protocols — see Pitfall 2.
- **Constructing a new `oidc.Provider`/`Verifier` per HTTP request:** re-fetches discovery+JWKS every time, ignoring AOID's `max-age=300` cache directive (explicit org-wide anti-pattern, `sso-integration.md` "Pitfall 4").
- **Nesting `is_admin` inside `UserExtraInfo`:** deprecated, logged, and the browser's own `Control.ServerAuditDialog.ts` self-diagnostic will flag it — always use the top-level `IsAdminUser` boolean.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|--------------|-----|
| OIDC Authorization Code + PKCE client, state/nonce management, provider+JWKS caching | A custom `go-oidc`/`oauth2` wrapper | `eden-platform-go/platform/oidcrp` (`BuildAuthURL`, `ExchangeAndVerify`, `ProviderCache`, `VerifierCache`, `State`) | Org-mandated; already battle-tested by AOID's own federation connector and all three AOFamily backends; re-deriving state/nonce/PKCE handling risks exactly the caching and CSRF bugs `sso-integration.md` already documents fixes for |
| WOPI proof-key RSA verification | A generic "WOPI SDK" dependency (none exists for Go in the org) | The ~40-line `crypto/rsa`+`crypto/sha256` verifier in Pattern 6, mirroring coolwsd's own `ProofKey.cpp` byte-for-byte | The algorithm is small, fully specified by coolwsd's own source, and a third-party dependency would add supply-chain surface for something this narrow |
| WOPI CheckFileInfo/GetFile/PutFile server plumbing | A generic "WOPI framework" | Plain `net/http` handlers per Pattern 2 | Org convention (aofamily backends are plain `net/http`); the WOPI surface here is 3 endpoints, a framework is unwarranted weight |
| AOID client registration UI/API | A hand-written SQL insert into `aoid.oauth_clients` | AOID's Admin Connect-RPC `CreateClient` | Bypassing the RPC skips bcrypt-hashing the secret correctly, the CHECK constraints (`auth_method='none' ⟺ client_type='public'`), and audit logging (`ActionClientCreated`) — the RPC is the only sanctioned write path |

**Key insight:** everything this objective needs on the OIDC side is a thin, already-proven consumption of `eden-platform-go/platform/oidcrp` plus AOID's existing admin RPC — there is no green-field OIDC client code to design, only wiring.

## Common Pitfalls

### Pitfall 1: `IsAdminUser` has no clean source in AOID v1.0
**What goes wrong:** A plan that tries to source `IsAdminUser` from the id_token or `/oauth/userinfo` will find nothing — neither carries an admin/role claim. The only candidate is the access token's `ent[]` (entitlements) claim, resolved per `(account, tenant, client_id)` via `aoid.identity_memberships`.
**Why it happens:** AOID v1.0's `IDTokenClaims` (`internal/oauth/tokens.go`) is `{tnt, nonce, email, email_verified, name}` only. `/oauth/userinfo` is scope-gated to the same set. `IntrospectResp` (the one place `ent` might have surfaced generically) explicitly has no `ent` field. Additionally, `internal/oauth/service.go` hardcodes `accessTokenResourceAudience = "aoedge"` on ALL user-flow access tokens (with an explicit `// TODO(per-client): drive this from oauth_clients.audience` in AOID's own source) — so even decoding the raw access token client-side and checking `aud == wopiHostClientID` will fail, because `aud` is always `["aoedge"]` regardless of which client requested the token.
**How to avoid:** For v1, ship `IsAdminUser: false` unconditionally — this is safe, defensible, and avoids the audience footgun entirely. If admin-flag support becomes a hard requirement, the WOPI host must: (a) register itself as an AOID client, (b) have specific accounts granted an entitlement string (e.g. `"edendocs:admin"`) via AOID's admin RPC/`identity_memberships`, (c) locally decode+verify the raw AOID access token (not the id_token) using JWKS with `SkipClientIDCheck: true` in `oidc.Config` (since `aud` is meaningless here — verify only `iss`+signature+`exp`), then check for that string in `ent[]`. Document this as a v1 limitation and a natural AOID enhancement request (fix the `accessTokenResourceAudience` TODO) rather than working around it with more complexity in the WOPI host.
**Warning signs:** Any task/plan step that says "read `IsAdminUser` from AOID's id_token" — there is no such claim; re-derive from this pitfall.

### Pitfall 2: WOPI `access_token` and AOID's OIDC tokens are two unrelated credentials
**What goes wrong:** Treating the AOID access token (or id_token) AS the WOPI `access_token` that coolwsd uses, e.g. passing the AOID JWT straight through to `cool.html?...&access_token=<AOID JWT>`.
**Why it happens:** Both are called "access token" and both are opaque-ish bearer strings, inviting conflation.
**How to avoid:** The WOPI `access_token` is entirely the WOPI host's own invention — coolwsd never inspects its structure, only echoes it back on every WOPI callback (CheckFileInfo/GetFile/PutFile) so the WOPI host can look up which session/user/file it corresponds to. Mint a fresh, short-lived, WOPI-host-generated random token (e.g. `uuid.NewString()` or an HMAC-signed opaque token) at launch-page-render time, store it → `(AOID identity claims, file path, expiry)` in the session store (Supporting Stack), and use `coolwsd.xml`'s `storage.wopi.max_token_ttl` / the `access_token_ttl` form field to keep it short-lived. Keep the AOID tokens (id_token/access_token/refresh_token) entirely inside the WOPI host's own session — never forward them to coolwsd or the browser.
**Warning signs:** Any code path that base64-decodes or JWT-parses the WOPI `access_token` expecting AOID claim shapes.

### Pitfall 3: coolwsd's proof-key signing has no "enable" toggle — verification is 100% the WOPI host's job
**What goes wrong:** Looking for a `coolwsd.xml` setting like `<wopi><proof_key><enable>true</enable>` — it doesn't exist.
**Why it happens:** AUTH-04's phrasing ("proof-key validation enabled") reads like a coolwsd-side switch.
**How to avoid:** coolwsd unconditionally signs every outbound WOPI request with `X-WOPI-Proof`/`X-WOPI-TimeStamp`/`X-WOPI-ProofOld` IF a `proof_key` file exists at `{COOLWSD_CONFIGDIR}/proof_key` (generate via `coolconfig generate-proof-key` or `ssh-keygen -t rsa -N "" -m PEM`). There is nothing to "enable" on the coolwsd side beyond ensuring that file exists. "Proof-key validation enabled" for this objective means: the WOPI host implements the verification (Pattern 6) and REJECTS requests with missing/invalid proof headers — that's the only place "enabled" applies.
**Warning signs:** Any planned `wsd/` change to "add a proof-key config option" — not needed, the mechanism is already unconditional.

### Pitfall 4: `alias_groups mode="groups"` silent-rejection footgun
**What goes wrong:** Adding a `<group>` block under `<alias_groups>` without the `mode="groups"` attribute — coolwsd logs an error and silently ignores the entire group (WOPI host requests then fail as untrusted, with no obvious config-level error).
**Why it happens:** `HostUtil.cpp`'s `parseAliases` checks the mode attribute defensively but only `LOG_ERR`s, it does not refuse to start.
**How to avoid:** Verification step for AUTH-04 should grep the running coolwsd's resolved config (or its startup log) for confirmation the WOPI host's origin was accepted, not just that the XML "looks right" — e.g., `curl` a CheckFileInfo request through coolwsd and confirm it isn't rejected as an untrusted host, rather than only checking the XML file's presence.
**Warning signs:** WOPI requests from the reference host failing with an "not a trusted WOPI host" style error in coolwsd's log despite `alias_groups` looking correctly configured — check the `mode` attribute first.

### Pitfall 5: Avatar gap — resolved, not a blocker
**What goes wrong (would-be):** Assuming the browser requires `UserExtraInfo.avatar` and building a fallback-avatar-generation service in the WOPI host to compensate for AOID having no avatar claim.
**Why it's actually fine:** Verified directly in `browser/src/app/LOUtil.ts` (`setUserImage`): if `viewInfo.userextrainfo.avatar` is undefined, the browser uses a built-in default (`LOUtil.getImageURL('user.svg')`) automatically — and even attaches an `error` event listener so a broken/unreachable avatar URL ALSO falls back to the same default image. Simply omit the `avatar` key from `UserExtraInfo` in CheckFileInfo responses; no synthesis needed.
**Warning signs:** N/A — documenting so the planner doesn't invent unnecessary avatar-generation work.

### Pitfall 6: Passing `access_token` via GET query string is technically accepted but not the safe convention
**What goes wrong:** coolwsd's `FileServer.cpp` DOES accept `access_token` as a plain GET query parameter on the initial `cool.html` page load (Poco's `HTMLForm` parses query params regardless of method) — so a naive `<iframe src="https://coolwsd/browser/dist/cool.html?WOPISrc=...&access_token=SECRET">` will "work" in testing, but it leaks the token into browser history, any intermediate proxy access logs, and the `Referer` header of any resource the loaded page fetches cross-origin.
**How to avoid:** Have the WOPI host's launch page render a hidden auto-submitting `<form method="POST">` targeting the iframe (standard MS-WOPI UX pattern), carrying `access_token`/`access_token_ttl` as POST body fields — `UserRequestVars::extractVariable` reads the same field names from either source, so this is a drop-in swap with real security benefit.

## Code Examples

Verified patterns from primary sources (org code + coolwsd source — see Sources for exact files/lines):

### AOID OIDC discovery + supported flows (verified, `aoid/internal/oauth/http/discovery.go`)
```json
{
  "issuer": "https://<issuer>",
  "authorization_endpoint": "https://<issuer>/oauth/authorize",
  "token_endpoint": "https://<issuer>/oauth/token",
  "userinfo_endpoint": "https://<issuer>/oauth/userinfo",
  "jwks_uri": "https://<issuer>/.well-known/jwks.json",
  "response_types_supported": ["code"],
  "grant_types_supported": ["authorization_code", "refresh_token", "client_credentials", "urn:ietf:params:oauth:grant-type:device_code"],
  "code_challenge_methods_supported": ["S256"],
  "scopes_supported": ["openid", "profile", "email", "offline_access"],
  "claims_supported": ["sub", "tnt", "email", "email_verified", "name", "preferred_username"],
  "token_endpoint_auth_methods_supported": ["client_secret_basic", "private_key_jwt", "none"]
}
```
Confirms: Authorization Code + PKCE (S256-only, OAuth 2.1 strict — no implicit, no password grant), exactly matching AUTH-01's requirement.

### CheckFileInfo↔AOID Claim Mapping Table
| CheckFileInfo field | AOID source claim/endpoint | Confidence | Notes |
|---|---|---|---|
| `UserId` | `sub` (id_token / userinfo, AOID account UUID) | HIGH | Direct 1:1 |
| `UserFriendlyName` | `name` (id_token, if present) → else `preferred_username` (userinfo, `profile` scope, falls back to email) → else `email` | HIGH | AOID's own userinfo endpoint already implements this exact fallback chain (`internal/oauth/http/userinfo.go`) — the WOPI host can request `profile`+`email` scopes and use userinfo's `name`/`preferred_username` directly |
| `OwnerId` | Not an AOID identity concept — this is a WOPI/document-storage concept (who owns the file in the WOPI host's own storage backend) | N/A | Populate from the WOPI host's own file metadata, not AOID |
| `UserExtraInfo.avatar` | **No claim exists anywhere in AOID's OIDC surface** (id_token, userinfo, access token) | HIGH (confirmed absence) | Omit the key — browser falls back gracefully (Pitfall 5) |
| `IsAdminUser` | `ent[]` (access-token-only claim) containing a provisioned entitlement string, resolved per `(account, tenant, wopi-host-client-id)` | MEDIUM | See Pitfall 1 — recommend `false` for v1 rather than wiring this up, given the `aud` hardcoding gotcha |
| `UserCanWrite` | Not an AOID claim — a WOPI-host authorization decision (does this AOID-authenticated user have write access to this document, per the WOPI host's own access-control model) | N/A | Business logic in the WOPI host, informed by AOID identity but not sourced from a specific claim |
| `PostMessageOrigin` | N/A (static config: the WOPI host's own origin) | N/A | Set to the WOPI host's own base URL |
| `email` (not a CheckFileInfo field, but available if the WOPI host wants it internally) | `email`/`email_verified` (id_token / userinfo, `email` scope) | HIGH | Always boolean `true` per AOID's userinfo implementation (`email_verified` hardcoded true when `email` scope granted) |

### WOPI `access_token` minting (WOPI host's own, NOT an AOID token — see Pitfall 2)
```go
// Illustrative — not sourced from any existing file, this is new code the WOPI host owns.
token := uuid.NewString()
sessionStore.Put(token, Session{
    AOIDSubject: claims.Subject,
    Name:        claims.Name,      // from id_token/userinfo
    Email:       claims.Email,
    IsAdmin:     false,            // v1 default — see Pitfall 1
    FileID:      requestedFileID,
    ExpiresAt:   time.Now().Add(15 * time.Minute), // short-lived, per AUTH-02
})
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|-------------------|---------------|--------|
| Hand-rolled per-service OIDC RP code across the org | `eden-platform-go/platform/oidcrp` as the single shared RP helper | Adopted org-wide per `aofamily/docs/sso-integration.md` (Objective 2 of AOFamily, "Path B is LOCKED") | Any new OIDC-consuming service (including this WOPI host) is expected to consume oidcrp, not write its own `go-oidc` wiring |
| AOID static/manual client config | Admin Connect-RPC `CreateClient` (tenant-scoped, audited) | Established at AOID's OAuth objective (04-05/04-06 SUMMARYs) | The WOPI host's client registration is an operational/admin task, not a config-file edit — must be planned as an explicit step with secret capture |

**Deprecated/outdated:**
- `alias_groups` "compat" mode (no `mode="groups"` attribute, or absent entirely): legacy single-host WOPI trust model — still supported by coolwsd but not appropriate for a new integration; use `mode="groups"` from the start.
- Nesting `is_admin` inside `UserExtraInfo`: deprecated by coolwsd itself (logged), superseded by the top-level `IsAdminUser` boolean.

## Open Questions

1. **PutFile lock semantics (`X-WOPI-Lock`/`X-WOPI-Override`) depth**
   - What we know: coolwsd sends these headers per standard MS-WOPI PutFile/Lock semantics; `WopiStorage.cpp` confirms `SupportsLocks` is a CheckFileInfo field the WOPI host can advertise.
   - What's unclear: the exact lock-conflict response codes/bodies coolwsd expects (409 Conflict shape) were not traced line-by-line in this research pass (time-boxed).
   - Recommendation: the planner should scope a dedicated task to trace `wsd/wopi/WopiStorage.cpp`'s Lock/RefreshLock/UnlockFile call sites (not yet fully read in this pass) before implementing PutFile locking, or accept "no locking" (`SupportsLocks: false`) for a v1 reference host, which is a legitimate, simpler scope reduction.

2. **`sdk.collaboraonline.com` official docs were unreachable during this research pass** (the site is behind an Anubis anti-bot proof-of-work challenge that this research's automated fetch could not complete).
   - What we know: All CheckFileInfo/proof-key/alias_groups findings in this document are sourced directly from coolwsd's own C++ implementation (`wsd/wopi/WopiStorage.cpp`, `wsd/ProofKey.cpp`, `wsd/HostUtil.cpp`) and its own test fixtures (`test/WopiTestServer.hpp`) — arguably a MORE authoritative source than the docs site for this specific coolwsd fork, since it's ground truth for the exact code this objective integrates against.
   - What's unclear: whether the official docs describe any WOPI host best-practices (e.g., specific HTTP status codes, additional optional CheckFileInfo fields for richer UX like `BreadcrumbBrandName`/`FileSharingUrl`) beyond what source-reading surfaced.
   - Recommendation: a manual (human-browser) check of `sdk.collaboraonline.com/docs/advanced_integration.html` during implementation is worthwhile but not blocking — treat this document's field contract as authoritative since it's source-verified.

3. **Multi-tenant AOID issuer selection**
   - What we know: AOID supports both a global discovery document and per-tenant discovery (`/{tenant_slug}/.well-known/openid-configuration`).
   - What's unclear: whether the reference WOPI host should be single-tenant (one AOID tenant, simplest for a reference deployment) or tenant-aware (accepting a `?tenant=` param on the launch page and resolving the per-tenant issuer).
   - Recommendation: default to single-tenant for the reference host (simplicity matches "reference deployable" framing); document tenant-awareness as a documented extension point, not a v1 requirement — the planner should treat this as an explicit scope decision to confirm with the user if ambiguous.

4. **WOPI host's own storage backend for documents**
   - What we know: GetFile/PutFile need SOME persistence layer for the actual document bytes; AUTH-02 doesn't specify one.
   - What's unclear: whether this objective's "reference" scope includes real persistent storage (filesystem, S3-compatible) or an ephemeral/demo store is acceptable.
   - Recommendation: filesystem-backed storage under a `wopi-host/testdata/` or configurable data dir is sufficient for a reference deployment and keeps the objective's scope tight — flag as a planning decision, not a research gap requiring more investigation.

## Sources

### Primary (HIGH confidence — direct source code reading)
- `/Users/justin/dev/EdenDocs/wsd/wopi/WopiStorage.cpp` — CheckFileInfo field parsing (UserFriendlyName, UserExtraInfo, IsAdminUser, etc.)
- `/Users/justin/dev/EdenDocs/wsd/HostUtil.cpp` + `.hpp` — `alias_groups mode="groups"` semantics
- `/Users/justin/dev/EdenDocs/wsd/ProofKey.cpp` — proof-key signing algorithm (verification is the mirror, implemented Go-side)
- `/Users/justin/dev/EdenDocs/wsd/FileServer.cpp` — launch page `access_token`/`access_token_ttl` extraction (`UserRequestVars`), GET-or-POST acceptance
- `/Users/justin/dev/EdenDocs/wsd/ClientRequestDispatcher.cpp` — WS-upgrade `access_token` query-string convention, Origin-check bypass rationale
- `/Users/justin/dev/EdenDocs/browser/src/app/LOUtil.ts` — `setUserImage` avatar fallback behavior (Pitfall 5)
- `/Users/justin/dev/EdenDocs/browser/src/control/Control.UserList.ts` — stock rendering of `UserExtraInfo`/username, confirms zero browser changes needed
- `/Users/justin/dev/EdenDocs/coolwsd.xml.in` — `alias_groups`, `content_security_policy`/`frame_ancestors` config shape
- `/Users/justin/dev/EdenDocs/scripts/eden/build.sh`, `/Users/justin/dev/EdenDocs/scripts/eden/smoke-test.sh` — CI dev-loop conventions, port 9980, native (non-container) execution model
- `/Users/justin/dev/EdenDocs/test/WopiTestServer.hpp` — coolwsd's own minimal CheckFileInfo fixture (BaseFileName/Size/OwnerId/UserFriendlyName/UserCanWrite)
- `/Users/justin/dev/aoid/internal/oauth/tokens.go` — `AccessTokenClaims`/`IDTokenClaims` exact shapes, `MintAccessToken`/`MintIDToken` signatures
- `/Users/justin/dev/aoid/internal/oauth/http/discovery.go` — discovery document JSON shape, PKCE S256-only, scopes/grants supported
- `/Users/justin/dev/aoid/internal/oauth/http/userinfo.go` — scope-gated userinfo claim exposure, no `ent`/avatar ever exposed there
- `/Users/justin/dev/aoid/internal/oauth/service.go` — `accessTokenResourceAudience = "aoedge"` hardcoding (Pitfall 1), `entitlementsFor`, `IntrospectResp` shape
- `/Users/justin/dev/aoid/internal/oauth/admin/service.go` — `CreateClient` admin RPC (client registration model)
- `/Users/justin/dev/aoid/migrations/0015_oauth_clients.up.sql` — `oauth_clients` schema
- `/Users/justin/dev/aoid/internal/federation/connector/oidc.go` — working example of `oidcrp.BuildAuthURL`/`ExchangeAndVerify` usage
- `/Users/justin/dev/aoid/docs/identity-context-v1.md` — confirms `ent` is free-form `string[]`, no fixed entitlement-string convention
- `/Users/justin/dev/eden-platform-go/platform/oidcrp/{doc,flow,provider_cache,verifier_cache,state,claimmap,errors}.go` — full oidcrp API surface
- `/Users/justin/dev/aofamily/browser/go/internal/auth/oidc.go` + `/Users/justin/dev/aofamily/docs/sso-integration.md` — org-normative oidcrp consumption pattern, provider/verifier caching rule ("Pitfall 4")
- `/Users/justin/dev/EdenDocs/.planning/REQUIREMENTS.md` — exact AUTH-01..04 wording

### Secondary (MEDIUM confidence)
- None used beyond primary sources for this objective — every material claim was traced to source code rather than relying on training-data assumptions, per this research's explicit mandate.

### Tertiary (LOW confidence / unverified)
- General WOPI protocol knowledge (Lock/RefreshLock/UnlockFile response codes, MS-WOPI PutFile conflict semantics) — NOT independently re-verified against `wsd/wopi/WopiStorage.cpp`'s lock-handling code in this pass (see Open Question 1); flagged, not asserted as fact.
- `sdk.collaboraonline.com` official docs — inaccessible during this research (Anubis bot-block); WebSearch results returned only third-party forum/README hits, not fetched/read in full, so not cited as a claim source.

## Metadata

**Confidence breakdown:**
- Standard stack (oidcrp, go-oidc, oauth2, golang-jwt): HIGH — versions and APIs read directly from `go.mod`/source files, not assumed
- Architecture patterns (alias_groups, proof-key, launch contract, CheckFileInfo contract): HIGH — every claim traced to specific coolwsd source lines
- AOID claim mapping: HIGH for what AOID does emit (read directly from `tokens.go`/`discovery.go`/`userinfo.go`); MEDIUM for the `IsAdminUser`/`ent[]` recommendation specifically, since it depends on a judgment call about working around the `aud` hardcoding rather than a settled AOID API
- Pitfalls: HIGH — all six are source-verified, not speculative
- WOPI lock/PutFile-conflict semantics: LOW — flagged as an explicit open question, not fabricated

**Research date:** 2026-07-08
**Valid until:** 30 days (2026-08-07) for the coolwsd/wsd findings (stable, this repo's own fork); 14 days for the AOID claim-shape findings (AOID is under active development per its own `.planning/` — re-verify `internal/oauth/tokens.go` and `internal/oauth/service.go` if this research is consumed after AOID's next OAuth-related objective lands, in case the `accessTokenResourceAudience` TODO gets resolved).
