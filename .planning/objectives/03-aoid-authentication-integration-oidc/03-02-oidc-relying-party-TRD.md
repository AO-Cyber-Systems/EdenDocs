---
objective: 03-aoid-authentication-integration-oidc
trd: "02"
type: standard
wave: 2
depends_on: ["03-01"]
files_modified:
  - wopi-host/internal/oidcauth/oidcauth.go
  - wopi-host/internal/oidcauth/oidcauth_test.go
  - wopi-host/internal/oidctest/idp.go
  - wopi-host/cmd/fake-aoid/main.go
  - wopi-host/go.mod
  - wopi-host/go.sum
autonomous: true
requirements: [AUTH-01, AUTH-03]

must_haves:
  truths:
    - "An unauthenticated request to a protected page 302s into an OIDC Authorization Code + PKCE flow (S256) and, after the IdP round trip, lands back with a server-side web session — implemented via eden-platform-go/platform/oidcrp, not hand-rolled"
    - "The callback maps OIDC claims to session.Identity per the research claim table: UserId=sub, UserFriendlyName=name→preferred_username→email fallback, IsAdmin=false (v1)"
    - "Tampered state, mismatched nonce, expired state, and missing id_token are all rejected (no session created)"
    - "A standalone fake-aoid binary serves an AOID-shaped OIDC issuer on 127.0.0.1:8092 for tests and the TRD 03-04 e2e — production code paths are identical for fake and real AOID"
  artifacts:
    - path: "wopi-host/internal/oidcauth/oidcauth.go"
      provides: "Authenticator with LoginHandler/CallbackHandler/LogoutHandler + RequireLogin middleware, oidcrp ProviderCache/VerifierCache built once at construction"
      contains: "oidcrp.ExchangeAndVerify"
    - path: "wopi-host/internal/oidctest/idp.go"
      provides: "In-process fake AOID IdP: discovery, JWKS, /oauth/authorize (auto-approve), /oauth/token (RS256 id_token with nonce), /oauth/userinfo"
      contains: "openid-configuration"
    - path: "wopi-host/cmd/fake-aoid/main.go"
      provides: "Standalone fake IdP binary (default 127.0.0.1:8092) reused by scripts/eden/wopi-e2e.sh in TRD 03-04"
      contains: "8092"
  key_links:
    - from: "wopi-host/internal/oidcauth/oidcauth.go"
      to: "github.com/aocybersystems/eden-platform-go/platform/oidcrp"
      via: "BuildAuthURL / ExchangeAndVerify / SignState+VerifyState / InMemoryInFlightStore / ProviderCache / VerifierCache — the org-mandated RP surface"
      pattern: "oidcrp\\."
    - from: "wopi-host/internal/oidcauth/oidcauth.go"
      to: "wopi-host/internal/session/session.go"
      via: "CallbackHandler maps verified claims → session.Identity → CreateWebSession cookie; the launch page (03-03) reads this via RequireLogin"
      pattern: "CreateWebSession"
    - from: "wopi-host/internal/oidcauth/oidcauth_test.go"
      to: "wopi-host/internal/oidctest/idp.go"
      via: "full-flow integration tests drive login→authorize→callback against the in-process fake IdP with a cookie jar"
      pattern: "oidctest\\."
---

<objective>
Implement the AOID OIDC relying-party surface of the WOPI host — /login,
/callback, /logout, and a RequireLogin middleware — as a thin consumption of
`eden-platform-go/platform/oidcrp` (org mandate: never hand-roll OIDC), plus
the fake-AOID test IdP (in-process for Go tests, standalone binary for the
e2e script) that mirrors AOID's real discovery/token/userinfo shapes.

Purpose: AUTH-01's Authorization Code flow becomes real, and AUTH-03's
identity mapping (claims → CheckFileInfo fields) is decided and tested here so
03-03 only consumes session.Identity.

Output: oidcauth + oidctest packages, fake-aoid binary, integration tests
covering the full flow and its failure modes.
</objective>

<file_tree>
wopi-host/
├── internal/oidcauth/oidcauth.go        ← CREATE
├── internal/oidcauth/oidcauth_test.go   ← CREATE
├── internal/oidctest/idp.go             ← CREATE (test infrastructure, also linked into fake-aoid)
├── cmd/fake-aoid/main.go                ← CREATE
├── go.mod                               ← MODIFY (go mod tidy fallout: go-oidc/oauth2 become direct)
└── go.sum                               ← MODIFY
</file_tree>

<execution_context>
@~/.claude/devflow/workflows/execute-trd.md
@~/.claude/devflow/templates/summary.md
</execution_context>

<embedded_context>

<codebase_examples>
Verified oidcrp API (read 2026-07-08 from
/Users/justin/dev/eden-platform-go/platform/oidcrp/ — signatures are exact):

```go
// flow.go
func BuildAuthURL(cfg *oauth2.Config, state, nonce, pkceVerifier string,
    extra []oauth2.AuthCodeOption) (string, error)
    // pkceVerifier from oauth2.GenerateVerifier() (43 chars min, S256 enforced)
func ExchangeAndVerify(ctx context.Context, cfg *oauth2.Config,
    verifier *oidc.IDTokenVerifier, code, pkceVerifier, storedNonce string,
) (*oidc.IDToken, *oauth2.Token, map[string]any, error)
    // returns ErrNonceMismatch / ErrMissingIDToken; verifier MUST have
    // SkipIssuerCheck and SkipClientIDCheck false (fine for id_tokens —
    // aud == client_id there)

// state.go — HMAC-signed state: base64url(json).base64url(hmac-sha256)
type State struct { Tenant, Idp, ReturnURL, Nonce string; CreatedAt int64 }
func SignState(key []byte, s State) string
func VerifyState(key []byte, raw string, maxAge time.Duration) (State, error)
    // enforces non-empty Tenant/Idp/Nonce — set Tenant:"default", Idp:"aoid"
    // for the single-tenant reference host (empty values FAIL verification)

// in_flight.go — keyed by NONCE (stable from /login through /callback)
type InFlightPayload struct { PKCEVerifier, StoredNonce, TenantID, IdpID, ReturnURL string; CreatedAt time.Time }
func NewInMemoryInFlightStore() *InMemoryInFlightStore
    // Put(ctx, key, payload, ttl); Pop(ctx, key) is single-use

// provider_cache.go / verifier_cache.go — build ONCE at startup, never per-request
func NewProviderCache() *ProviderCache
func (c *ProviderCache) Get(ctx context.Context, key, issuer string) (*oidc.Provider, error)
func NewVerifierCache() *VerifierCache
func (c *VerifierCache) Get(key string, provider *oidc.Provider, clientID string, supportedAlgs []string) *oidc.IDTokenVerifier
```

Working consumption reference:
/Users/justin/dev/aoid/internal/federation/connector/oidc.go and the
org-normative /Users/justin/dev/aofamily/docs/sso-integration.md §2.

AOID's real discovery document shape to mirror in oidctest (verified from
/Users/justin/dev/aoid/internal/oauth/http/discovery.go):
```json
{
  "issuer": "<issuer>",
  "authorization_endpoint": "<issuer>/oauth/authorize",
  "token_endpoint": "<issuer>/oauth/token",
  "userinfo_endpoint": "<issuer>/oauth/userinfo",
  "jwks_uri": "<issuer>/.well-known/jwks.json",
  "response_types_supported": ["code"],
  "code_challenge_methods_supported": ["S256"],
  "scopes_supported": ["openid", "profile", "email", "offline_access"],
  "claims_supported": ["sub", "tnt", "email", "email_verified", "name", "preferred_username"],
  "token_endpoint_auth_methods_supported": ["client_secret_basic", "private_key_jwt", "none"]
}
```
</codebase_examples>

<anti_patterns>
- NEVER construct `oidc.NewProvider` / `provider.Verifier` per request
  (sso-integration.md "Pitfall 4" — ignores AOID's max-age=300 JWKS caching).
  Build ProviderCache/VerifierCache once in `NewAuthenticator`.
- NEVER hand-roll state/nonce/PKCE handling with raw go-oidc/oauth2 — use
  oidcrp end to end (org mandate; the whole point of this dependency).
- NEVER forward AOID tokens (id_token / access_token / refresh_token) to
  coolwsd or the browser (research Pitfall 2). They live and die inside the
  WOPI host's server-side session.
- NEVER redirect to a raw ReturnURL from state — oidcrp docs warn ReturnURL
  is signature-verified but NOT allowlisted. Allowlist to same-origin paths
  only (must start with "/" and not "//").
- Do NOT source IsAdminUser from the id_token or userinfo — no such claim
  exists in AOID v1.0 (research Pitfall 1). Hard-code `IsAdmin: false` with a
  comment referencing the pitfall and the ent[]/aud=aoedge v2 path.
- No .feature files, no property-based tests, no LLM-generated claim
  fixtures — the fake IdP's canned identity is hand-written
  (sub "aoid-e2e-user-1", name "Eden Test User", email "e2e@aocyber.ai").
</anti_patterns>

<error_recovery>
- ErrNonceMismatch or ErrMissingIDToken from ExchangeAndVerify in tests:
  check the fake IdP embeds the EXACT nonce from the authorize request into
  the id_token, and that the token response JSON field is `id_token`.
- `oidc: id token issued by a different provider` — the fake IdP's `iss`
  claim must equal the httptest server URL exactly (build the IdP so its
  issuer is set AFTER httptest.NewServer assigns the URL, or use
  httptest.NewUnstartedServer + custom Listener on a fixed 127.0.0.1 port).
- `failed to verify signature` — JWKS must serve the SAME RSA public key that
  signs id_tokens, with matching `kid` in JWK and JWS header.
- If VerifyState fails with ErrStateInvalid on seemingly-good state: check
  Tenant/Idp/Nonce are all non-empty (zero-value defense in oidcrp) and the
  same HMAC key is used to sign and verify.
</error_recovery>

</embedded_context>

<context>
@.planning/PROJECT.md
@.planning/objectives/03-aoid-authentication-integration-oidc/3-RESEARCH.md
@.planning/objectives/03-aoid-authentication-integration-oidc/03-01-SUMMARY.md
</context>

<research_context>
- Flow pattern (3-RESEARCH.md Pattern 1): oauth2.Config with ClientID/Secret
  (confidential client, client_secret_basic), Scopes
  ["openid","profile","email"], RedirectURL = PublicURL + "/callback".
- Claim mapping (research "CheckFileInfo↔AOID Claim Mapping Table"):
  UserId ← sub; UserFriendlyName ← name, falling back to preferred_username,
  falling back to email (AOID's own userinfo implements this chain — the RP
  replicates it over id_token claims + optional userinfo call; implement over
  id_token claims map first, only call userinfo if `name` absent);
  email/email_verified available; NO avatar claim exists anywhere (omit —
  Pitfall 5); IsAdminUser ← false in v1 (Pitfall 1).
- AOID is PKCE S256-only, OAuth 2.1 strict (no implicit) — BuildAuthURL
  always attaches PKCE, matching.
- Fake IdP port: 8092 (8080 is banned; 8091 is the WOPI host; 9980 is
  coolwsd).
</research_context>

<gotchas>
- HARD RULE: no port 8080 anywhere. Fake IdP standalone default
  127.0.0.1:8092 (override EDENDOCS_FAKE_AOID_ADDR).
- Session cookie: Name "edendocs_wopi_sid", HttpOnly, SameSite=Lax,
  Path "/", Secure only when PublicURL is https (reference host runs http
  locally). SameSite=Lax is REQUIRED for the OIDC redirect to carry the
  cookie back on the top-level GET /callback.
- In-flight TTL: 10 minutes; state maxAge: 10 minutes. Login handler
  generates nonce via uuid and pkce via oauth2.GenerateVerifier().
- go.mod: `go mod tidy` here will promote go-oidc/v3 and golang.org/x/oauth2
  to direct requires — expected; commit go.mod+go.sum in this TRD (TRD 03-03
  deliberately does not touch go.mod to keep wave-2 files disjoint).
- MPL-2.0 headers on all new .go files.
</gotchas>

## Test list

Outside-in (HTTP integration through the real handler mux first, then
narrower units):

1. Full happy path: GET protected page → 302 /login → 302 to fake IdP
   /oauth/authorize (assert query carries code_challenge, S256,
   nonce, signed state) → IdP 302s back to /callback with code+state →
   callback sets session cookie → original page now 200 with identity
   (cookie-jar http.Client against httptest mux).
2. Identity mapping: id_token with name → Identity.Name == name; without
   name but with preferred_username → that; with neither → email.
   IsAdmin always false. Subject == sub.
3. Callback with tampered state (flip one byte) → 4xx, no session cookie.
4. Callback whose id_token nonce ≠ in-flight nonce → 4xx (ErrNonceMismatch
   path), no session.
5. Callback replay: second GET /callback with the same code/state → 4xx
   (in-flight Pop is single-use).
6. Expired state (CreatedAt older than maxAge; inject clock or short maxAge)
   → 4xx.
7. Token response missing id_token → 4xx (ErrMissingIDToken path).
8. ReturnURL allowlist: state ReturnURL "https://evil.example" or "//evil"
   → redirect falls back to "/" (never off-origin).
9. RequireLogin without cookie → 302 /login?return=<original path>.
10. Logout clears the session: subsequent protected GET → 302 /login.

<tasks>

<task type="auto">
  <name>Task 1: internal/oidctest fake AOID IdP + cmd/fake-aoid binary</name>
  <files>wopi-host/internal/oidctest/idp.go, wopi-host/cmd/fake-aoid/main.go, wopi-host/go.mod, wopi-host/go.sum</files>
  <action>
Build the hand-written fake IdP mirroring AOID's real endpoint shapes (see
codebase_examples discovery JSON). This is test infrastructure with
deterministic behavior — no LLM-generated data, no randomness beyond key
generation.

```go
package oidctest

type IdP struct { /* rsa key, issuer, mu, issued codes map[code]authRequest */ }
func New() *IdP                       // generates RSA-2048 key, kid "fake-aoid-1"
func (p *IdP) Handler() http.Handler  // mounts all routes; issuer set via SetIssuer
func (p *IdP) SetIssuer(url string)   // call after httptest.NewServer / net.Listen
```

Routes (Go 1.22 pattern mux):
- GET /.well-known/openid-configuration — discovery JSON exactly mirroring
  AOID's field set, endpoints under the issuer.
- GET /.well-known/jwks.json — the RSA public key as a JWK (kty RSA, use sig,
  alg RS256, kid, n/e base64url).
- GET /oauth/authorize — auto-approve: record {state, nonce, code_challenge,
  redirect_uri, client_id}, mint a one-time code, 302 to
  redirect_uri?code=...&state=... (no login UI — deterministic).
- POST /oauth/token — validate grant_type=authorization_code, code known and
  unused, code_verifier present and S256(code_verifier)==code_challenge;
  return {access_token: "fake-aoid-opaque-<uuid>", token_type: "Bearer",
  expires_in: 300, id_token: <RS256 JWT>} where the id_token claims are
  {iss, aud: client_id, sub: "aoid-e2e-user-1", email: "e2e@aocyber.ai",
  email_verified: true, name: "Eden Test User", nonce: <from authorize>,
  iat, exp: +5m}. Support a WithClaims option to override claims per-test
  (e.g. drop `name` for fallback-chain tests, omit id_token for case 7).
- GET /oauth/userinfo — bearer-token gated, returns the same claim set.

cmd/fake-aoid/main.go: net.Listen on EDENDOCS_FAKE_AOID_ADDR (default
"127.0.0.1:8092"), SetIssuer("http://"+addr), serve, log the issuer URL.
Used by scripts/eden/wopi-e2e.sh in TRD 03-04.

Sign JWTs with golang-jwt/jwt/v5 (already in the dependency graph via
eden-platform-go — same lib AOID itself mints with).
Commit: `feat(03-02): fake AOID IdP (in-process + standalone on 8092) for tests/e2e`
  </action>
  <verify>cd wopi-host && go vet ./internal/oidctest/ ./cmd/fake-aoid/ && go build ./... && (EDENDOCS_FAKE_AOID_ADDR=127.0.0.1:8092 ./bin-test-run() { :; }; go run ./cmd/fake-aoid & FPID=$!; sleep 1; curl -fsS http://127.0.0.1:8092/.well-known/openid-configuration | grep -q '"authorization_endpoint"'; RC=$?; kill $FPID; exit $RC)</verify>
  <done>fake-aoid serves an AOID-shaped discovery document on 8092; JWKS + authorize + token endpoints implemented with S256 PKCE validation and one-time codes.</done>
  <recovery>If the inline run-check is flaky in the sandbox, verify via a Go test in oidctest that spins the handler on httptest and asserts discovery/jwks/token behavior instead — the standalone binary is smoke-tested again in 03-04's e2e.</recovery>
</task>

<task type="auto" tdd="true">
  <name>Task 2: internal/oidcauth — login/callback/logout + RequireLogin via oidcrp</name>
  <files>wopi-host/internal/oidcauth/oidcauth.go, wopi-host/internal/oidcauth/oidcauth_test.go, wopi-host/go.mod, wopi-host/go.sum</files>
  <action>
RED first: write the 10 test-list cases as integration tests — httptest mux
mounting Authenticator handlers + a trivial protected page behind
RequireLogin, fake IdP from Task 1 as issuer, cookie-jar client. Then GREEN.

Design:

```go
type Authenticator struct {
    cfg       *oauth2.Config          // built once from config + provider.Endpoint()
    verifier  *oidc.IDTokenVerifier   // from VerifierCache.Get — once
    stateKey  []byte
    inflight  oidcrp.InFlightStore    // NewInMemoryInFlightStore()
    sessions  *session.Store
    publicURL string
}
func NewAuthenticator(ctx context.Context, cfg config.Config, sessions *session.Store) (*Authenticator, error)
    // ProviderCache.Get(ctx, "aoid", cfg.AOIDIssuer) → provider
    // VerifierCache.Get("aoid", provider, cfg.ClientID, nil) → verifier
```

- LoginHandler (GET /login?return=/path):
  nonce := uuid.NewString(); pkce := oauth2.GenerateVerifier()
  state := oidcrp.SignState(a.stateKey, oidcrp.State{Tenant: "default",
  Idp: "aoid", ReturnURL: sanitizeReturn(r), Nonce: nonce,
  CreatedAt: time.Now().Unix()})
  a.inflight.Put(ctx, nonce, oidcrp.InFlightPayload{PKCEVerifier: pkce,
  StoredNonce: nonce, TenantID: "default", IdpID: "aoid",
  ReturnURL: ..., CreatedAt: ...}, 10*time.Minute)
  url, _ := oidcrp.BuildAuthURL(a.cfg, state, nonce, pkce, nil); 302.
- CallbackHandler (GET /callback?code&state):
  st, err := oidcrp.VerifyState(a.stateKey, state, 10*time.Minute) → 400 on err
  payload, err := a.inflight.Pop(ctx, st.Nonce) → 400 on err (single-use)
  _, _, claims, err := oidcrp.ExchangeAndVerify(ctx, a.cfg, a.verifier,
  code, payload.PKCEVerifier, payload.StoredNonce) → 401 on err
  identity := mapClaims(claims) — the AUTH-03 mapping:
    Subject=claims["sub"]; Name = first non-empty of name,
    preferred_username, email; Email=claims["email"]; IsAdmin=false
    // v1: no admin claim in AOID id_tokens — research Pitfall 1
  sid := a.sessions.CreateWebSession(identity, 12*time.Hour); set cookie
  (edendocs_wopi_sid, HttpOnly, SameSite=Lax); 302 to allowlisted ReturnURL.
- LogoutHandler: delete session + expire cookie; 302 "/".
- RequireLogin(next http.Handler): cookie → LookupWebSession → inject
  Identity into context (typed key + FromContext helper for 03-03); else
  302 /login?return=<url-escaped r.URL.RequestURI()>.
- sanitizeReturn: must start with "/", must not start with "//" → else "/".

One behavior per test case, descriptive names. Failure-path tests use the
fake IdP's WithClaims/misbehavior options.
Commits: `test(03-02): OIDC RP flow behavior against fake AOID (RED)` then
`feat(03-02): AOID OIDC relying party via oidcrp (GREEN)` then
`chore(03-02): go mod tidy (go-oidc/oauth2 direct)`.
  </action>
  <verify>cd wopi-host && go test -race ./internal/oidcauth/ -count=1 -v 2>&1 | tail -25 && go vet ./... && go build ./... && ! grep -rn "NewProvider\|provider.Verifier(" internal/oidcauth/oidcauth.go | grep -v "Cache" | grep -q .</verify>
  <done>All 10 flow cases green under -race; provider/verifier constructed only via oidcrp caches at NewAuthenticator time; claims mapping implements the exact AUTH-03 fallback chain; IsAdmin hard-false with pitfall comment.</done>
  <recovery>See error_recovery for the three canonical go-oidc verification failures. If SameSite handling breaks the cookie-jar test client, assert the Set-Cookie attributes directly instead of relying on jar semantics.</recovery>
</task>

</tasks>

<validation_gates>
<lint>cd wopi-host && go vet ./...</lint>
<test>cd wopi-host && go test -race ./... -count=1</test>
<build>cd wopi-host && go build ./...</build>
</validation_gates>

<verification>
- Full-flow integration test green: login → fake IdP → callback → session.
- All failure modes (tampered state, nonce mismatch, replay, expiry, missing
  id_token, open-redirect attempt) rejected without sessions.
- `grep -rn "oidcrp\." wopi-host/internal/oidcauth/` shows BuildAuthURL,
  ExchangeAndVerify, SignState, VerifyState, InFlight usage (no hand-rolled
  equivalents).
- fake-aoid binary builds and serves discovery on 8092.
- Zero changes outside wopi-host/.
</verification>

<success_criteria>
The WOPI host authenticates users via AOID-shaped OIDC Authorization Code +
PKCE end to end in tests, producing a session.Identity whose fields are
exactly what CheckFileInfo needs (AUTH-03 mapping), through the org-mandated
oidcrp library with every canonical failure mode covered.
</success_criteria>

<output>
After completion, create `.planning/objectives/03-aoid-authentication-integration-oidc/03-02-SUMMARY.md`
recording: the Identity context-helper API (03-03 consumes it), fake-aoid
usage (addr env var, canned identity values), and any oidcrp behaviors that
surprised (for the 03-05 runbook).
</output>
