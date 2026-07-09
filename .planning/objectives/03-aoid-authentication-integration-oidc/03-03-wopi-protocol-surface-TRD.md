---
objective: 03-aoid-authentication-integration-oidc
trd: "03"
type: standard
wave: 2
depends_on: ["03-01"]
files_modified:
  - wopi-host/internal/proof/proof.go
  - wopi-host/internal/proof/proof_test.go
  - wopi-host/internal/coolwsd/discovery.go
  - wopi-host/internal/coolwsd/discovery_test.go
  - wopi-host/internal/wopi/wopi.go
  - wopi-host/internal/wopi/wopi_test.go
  - wopi-host/internal/launch/launch.go
  - wopi-host/internal/launch/launch_test.go
autonomous: true
requirements: [AUTH-02, AUTH-03, AUTH-04]

must_haves:
  truths:
    - "CheckFileInfo emits the exact coolwsd-parsed contract: BaseFileName/Size/OwnerId/UserId/UserFriendlyName/UserCanWrite, top-level IsAdminUser, PostMessageOrigin — with UserExtraInfo.avatar OMITTED entirely (no avatar claim exists in AOID; browser falls back gracefully)"
    - "GetFile streams exact document bytes; PutFile persists the request body atomically and returns 200 — both only with a valid, unexpired WOPI access_token (401 otherwise)"
    - "Every /wopi/ request is verified against coolwsd's X-WOPI-Proof/X-WOPI-ProofOld RSA-SHA256 signatures (mirror of wsd/ProofKey.cpp) and rejected when missing/invalid/stale — proof-key validation is the WOPI host's job, there is no coolwsd-side toggle"
    - "The launch page renders a hidden auto-submitting POST form (never GET query-string) carrying access_token + access_token_ttl into the editor iframe, with the action URL built from coolwsd's /hosting/discovery urlsrc"
  artifacts:
    - path: "wopi-host/internal/proof/proof.go"
      provides: "ParseProofKey (discovery modulus/exponent → rsa.PublicKey) + VerifyProof implementing coolwsd's ProofKey.cpp byte construction (len-prefixed token + UPPER(url) + .NET ticks), accepting current OR old proof"
      contains: "VerifyPKCS1v15"
    - path: "wopi-host/internal/coolwsd/discovery.go"
      provides: "Cached /hosting/discovery client: urlsrc lookup by file extension + proof-key extraction (modulus/exponent/oldmodulus/oldexponent)"
      contains: "hosting/discovery"
    - path: "wopi-host/internal/wopi/wopi.go"
      provides: "CheckFileInfo/GetFile/PutFile handlers guarded by token lookup + proof middleware; structured log line per request (the e2e log-grep contract)"
      contains: "IsAdminUser"
    - path: "wopi-host/internal/launch/launch.go"
      provides: "Document list + launch page minting WOPI tokens and rendering the POST-form iframe embed"
      contains: "access_token_ttl"
  key_links:
    - from: "wopi-host/internal/wopi/wopi.go"
      to: "wopi-host/internal/session/session.go"
      via: "access_token query param → Store.LookupWopiToken → identity fields into CheckFileInfo JSON"
      pattern: "LookupWopiToken"
    - from: "wopi-host/internal/wopi/wopi.go"
      to: "wopi-host/internal/proof/proof.go"
      via: "proof middleware verifies X-WOPI-Proof(/Old) + X-WOPI-TimeStamp against the discovery-published key before any WOPI handler runs"
      pattern: "X-WOPI-Proof"
    - from: "wopi-host/internal/launch/launch.go"
      to: "wopi-host/internal/coolwsd/discovery.go"
      via: "urlsrc(ext) from cached discovery builds the iframe form action: {urlsrc}WOPISrc={urlencode(WopiBaseURL + /wopi/files/{id})}"
      pattern: "urlsrc"
    - from: "wopi-host/internal/launch/launch.go"
      to: "wopi-host/internal/session/session.go"
      via: "MintWopiToken at launch-render time; access_token_ttl = ExpiresAt.UnixMilli() (MS-WOPI ms-epoch expiry convention)"
      pattern: "MintWopiToken"
---

<objective>
Implement the WOPI protocol surface the editor integration stands on:
proof-key verification (the AUTH-04 "validation enabled" half that lives in
the WOPI host), the coolwsd discovery client, the three WOPI endpoints
(CheckFileInfo / GetFile / PutFile) with the exact field contract coolwsd
parses, and the launch page that embeds the editor iframe via an
auto-submitting POST form.

Purpose: AUTH-02's protocol surface and AUTH-03's identity pass-through become
real, testable code; AUTH-04's proof-key validation is enforced on every WOPI
callback. Runs parallel to TRD 03-02 (disjoint files; the only shared contract
is session/storage from 03-01 — identity extraction is INJECTED, see gotchas).

Output: proof, coolwsd, wopi, launch packages with integration tests driving
coolwsd-shaped signed requests.
</objective>

<file_tree>
wopi-host/internal/
├── proof/proof.go              ← CREATE
├── proof/proof_test.go         ← CREATE
├── coolwsd/discovery.go        ← CREATE
├── coolwsd/discovery_test.go   ← CREATE
├── wopi/wopi.go                ← CREATE
├── wopi/wopi_test.go           ← CREATE
├── launch/launch.go            ← CREATE
└── launch/launch_test.go       ← CREATE
</file_tree>

<execution_context>
@~/.claude/devflow/workflows/execute-trd.md
@~/.claude/devflow/templates/summary.md
</execution_context>

<embedded_context>

<codebase_examples>
CheckFileInfo response — coolwsd's ACTUAL parsing contract (verified from
wsd/wopi/WopiStorage.cpp; field names are exact and case-sensitive):

```go
type CheckFileInfo struct {
    BaseFileName      string         `json:"BaseFileName"`      // required
    Size              int64          `json:"Size"`              // required
    OwnerId           string         `json:"OwnerId"`           // required — WOPI-host storage concept, use "edendocs"
    UserId            string         `json:"UserId"`            // ← Identity.Subject (AOID sub)
    UserFriendlyName  string         `json:"UserFriendlyName"`  // ← Identity.Name (falls back to "UnknownUser" + LOG_ERR if empty — never send empty)
    UserCanWrite      bool           `json:"UserCanWrite"`      // true for the reference host
    UserExtraInfo     map[string]any `json:"UserExtraInfo,omitempty"` // OMIT entirely in v1 — no avatar claim in AOID (Pitfall 5)
    IsAdminUser       bool           `json:"IsAdminUser"`       // TOP-LEVEL — nested UserExtraInfo.is_admin is deprecated+logged by coolwsd
    PostMessageOrigin string         `json:"PostMessageOrigin,omitempty"` // cfg.PublicURL
    SupportsLocks     bool           `json:"SupportsLocks"`     // false — v1 scope reduction (research Open Question 1)
}
```

Proof verification — mirror of wsd/ProofKey.cpp GetProof/SignProof (the
byte-for-byte algorithm, from 3-RESEARCH.md Pattern 6):

```go
func VerifyProof(pub *rsa.PublicKey, accessToken, uri, timestampHeader, proofB64 string, window time.Duration) error {
    ticks, err := strconv.ParseInt(timestampHeader, 10, 64) // .NET ticks: 100ns since 0001-01-01 UTC
    if err != nil { return err }
    epoch := time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)
    ts := epoch.Add(time.Duration(ticks) * 100 * time.Nanosecond)
    if d := time.Since(ts); d < -window || d > window { return ErrStaleProof }

    buf := new(bytes.Buffer)
    write := func(b []byte) { binary.Write(buf, binary.BigEndian, int32(len(b))); buf.Write(b) }
    write([]byte(accessToken))
    write([]byte(strings.ToUpper(uri)))       // FULL absolute request URL, uppercased
    tb := make([]byte, 8); binary.BigEndian.PutUint64(tb, uint64(ticks))
    write(tb)

    sig, err := base64.StdEncoding.DecodeString(proofB64)
    if err != nil { return err }
    h := sha256.Sum256(buf.Bytes())
    return rsa.VerifyPKCS1v15(pub, crypto.SHA256, h[:], sig)
}
```

Discovery proof-key XML shape (served by coolwsd at /hosting/discovery):
`<proof-key exponent="..." modulus="..." oldexponent="..." oldmodulus="..."
value="..." oldvalue=".."/>` — use the base64 `modulus`/`exponent` attributes
(big-endian bytes) with math/big + rsa.PublicKey; IGNORE the CAPI-blob
`value` attribute. Action lookup: `<net-zone><app name="writer"><action
ext="odt" urlsrc="http://.../browser/dist/cool.html?"/>...` — match by `ext`
across all apps/zones, first hit wins.

Hand-capture the discovery test fixture from the LIVE reference container
(running on this machine): `curl -fsS http://127.0.0.1:9980/hosting/discovery`
— trim to a few representative actions + the proof-key element and commit as a
Go string literal or testdata XML. Hand-captured real output, not invented.
</codebase_examples>

<anti_patterns>
- Do NOT edit wsd/ or browser/ to make any of this work — if something seems
  to require it, the WOPI-side understanding is wrong; re-derive from
  wsd/wopi/WopiStorage.cpp / wsd/ProofKey.cpp (research anti-pattern).
- Do NOT nest is_admin inside UserExtraInfo (deprecated, coolwsd logs it) —
  top-level IsAdminUser only.
- Do NOT emit a UserExtraInfo.avatar key with an empty/placeholder URL and do
  NOT build avatar synthesis — omit the key; browser/src/app/LOUtil.ts
  setUserImage falls back to the built-in user.svg (verified).
- Do NOT put access_token in the iframe src query string (Pitfall 6: leaks to
  history/logs/Referer) — hidden auto-submitting POST form only. coolwsd's
  UserRequestVars reads the same field names from either; POST is the
  convention here.
- Do NOT import internal/oidcauth from launch (would couple the two wave-2
  TRDs) — identity extraction is an injected dependency (see gotchas).
- Do NOT skip proof verification in tests by disabling the middleware —
  tests sign requests with a test RSA key exactly as coolwsd would.
- No JWT parsing of the WOPI access_token anywhere (Pitfall 2).
</anti_patterns>

<error_recovery>
- Proof self-test failing: the three most common mistakes are (a) hashing the
  path instead of the FULL absolute URL, (b) forgetting ToUpper on the URL,
  (c) little-endian length prefixes. The test task builds a signer that is
  the exact mirror of the verifier — if coolwsd later rejects in e2e (03-04),
  compare against wsd/ProofKey.cpp line by line before touching anything
  else.
- coolwsd duplicates Proof and ProofOld today ("TODO: implement proper
  rotation" in ProofKey.cpp) — verify against current key first, then old
  key, then oldProof header against both; accept ANY single match.
- If the timestamp window trips in slower environments: window is
  configurable (default 5 minutes — MS-WOPI convention is 20 minutes; 20s in
  the research pseudocode is too tight for CI).
- Discovery XML parse errors against the live container: re-capture the
  fixture; upstream occasionally reorders zones. Match actions by ext
  anywhere in the document rather than assuming zone order.
</error_recovery>

</embedded_context>

<context>
@.planning/PROJECT.md
@.planning/objectives/03-aoid-authentication-integration-oidc/3-RESEARCH.md
@.planning/objectives/03-aoid-authentication-integration-oidc/03-01-SUMMARY.md
</context>

<research_context>
- Launch/iframe contract (research Pattern 5, verified in wsd/FileServer.cpp
  UserRequestVars): coolwsd's cool.html accepts access_token /
  access_token_ttl from GET or POST transparently; the WS upgrade carries
  access_token in the query string by coolwsd's own design. The launch page
  form action is `{urlsrc}WOPISrc={urlencode(wopiSrc)}` where wopiSrc =
  cfg.WopiBaseURL + "/wopi/files/" + fileID (WopiBaseURL differs from
  PublicURL in docker mode — host.docker.internal).
- access_token_ttl = expiry as Unix MILLISECONDS (MS-WOPI convention);
  compute from WopiSession.ExpiresAt.UnixMilli().
- WOPI routes coolwsd calls (WopiStorage.cpp): GET {WOPISrc}?access_token=…
  (CheckFileInfo), GET {WOPISrc}/contents?access_token=… (GetFile),
  POST {WOPISrc}/contents?access_token=… (PutFile, headers X-WOPI-Override:
  PUT; lock headers absent since SupportsLocks:false).
- SupportsLocks:false is the deliberate v1 scope reduction for research Open
  Question 1 (lock-conflict semantics untraced) — record in code comment and
  the 03-05 runbook as a documented limitation.
</research_context>

<gotchas>
- HARD RULE: no port 8080 anywhere, including test fixtures and doc comments.
  Tests use httptest-assigned ports; examples use 8091/9980.
- Identity injection: launch.New takes
  `identityFrom func(*http.Request) (session.Identity, bool)` — TRD 03-04
  wires oidcauth's context helper into it. Launch tests inject a stub. This
  keeps 03-02/03-03 file-disjoint and independently executable.
- The proof middleware needs the FULL external request URL as coolwsd signed
  it (scheme://host:port/path?query). Reconstruct from cfg.WopiBaseURL +
  r.URL.RequestURI() — NOT from r.Host (proxies/containers rewrite Host).
- PutFile: read body with a size cap (e.g. 100 MiB), write via storage's
  atomic Write, respond 200 with `{}` body. On unknown fileID respond 404;
  coolwsd treats non-200 as save failure and surfaces it in the UI.
- Structured log lines are a CONTRACT for 03-04's e2e greps — emit exactly:
  `wopi: CheckFileInfo file=<id> user=<sub> status=<code>`,
  `wopi: GetFile file=<id> status=<code>`,
  `wopi: PutFile file=<id> bytes=<n> status=<code>`,
  `proof: verified ok` / `proof: REJECTED reason=<...>`.
- This TRD must NOT touch go.mod/go.sum (03-02 owns them in wave 2). All
  imports here are stdlib + 03-01 packages + google/uuid (already required).
- MPL-2.0 headers on all new .go files.
</gotchas>

## Test list

Outside-in (coolwsd-shaped signed HTTP requests through the real mux first,
then narrower units):

**internal/wopi (integration through mux, requests signed like coolwsd):**
1. CheckFileInfo with valid token + valid proof → 200; golden JSON contains
   exact keys BaseFileName/Size/OwnerId/UserId/UserFriendlyName/UserCanWrite/
   IsAdminUser/PostMessageOrigin; `"UserExtraInfo"` ABSENT; IsAdminUser top
   level and false; UserFriendlyName == identity name.
2. GetFile with valid token+proof → 200, body byte-identical to stored file,
   Content-Length correct.
3. PutFile (POST /contents, X-WOPI-Override: PUT) with valid token+proof →
   200 and storage read-back returns the new bytes.
4. Unknown/expired access_token → 401 for all three routes (expired via
   injected clock).
5. Missing proof headers → 401/500-class rejection; log line
   `proof: REJECTED`.
6. Invalid proof signature (signed with a different key) → rejected.
7. Stale X-WOPI-TimeStamp (outside window) → rejected.
8. Valid X-WOPI-ProofOld with garbage X-WOPI-Proof → accepted (rotation
   tolerance).
9. Path-traversal fileID in URL → 404 (storage guard surfaces, no disk
   access outside data dir).

**internal/proof (unit):**
10. Self-signed vector round-trip: sign with test RSA key using the exact
    byte construction → VerifyProof nil error.
11. Tampered token/url/ticks each → verification error.
12. ParseProofKey builds the same key from modulus/exponent that signed (use
    the test key's own big-endian bytes b64-encoded).

**internal/coolwsd (unit, httptest):**
13. urlsrc("odt") resolves from the hand-captured discovery fixture;
    unknown extension → typed error.
14. Proof keys (current + old) parsed from the fixture's proof-key element.
15. Discovery result cached (second call served without a second HTTP hit —
    count requests in the httptest handler); Invalidate forces refetch.

**internal/launch (integration, stub identity):**
16. GET / without identity → 302 to /login?return=… (uses injected
    identityFrom returning false).
17. GET /open?file=hello.odt with identity → 200 HTML containing: a form
    with method="post", action = urlsrc + "WOPISrc=" + urlencoded wopiSrc,
    hidden access_token input (uuid value), access_token_ttl input equal to
    mint expiry UnixMilli, an iframe target, and auto-submit script.
18. access_token appears NOWHERE in any href/src/GET URL in the rendered
    page (Pitfall 6 regression guard).
19. /open with unknown file → 404; /open without file param → 400.

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: internal/proof — coolwsd proof-key verification (ProofKey.cpp mirror)</name>
  <files>wopi-host/internal/proof/proof.go, wopi-host/internal/proof/proof_test.go</files>
  <action>
RED (test-list 10-12) then GREEN. Implement ParseProofKey(modulusB64,
exponentB64 string) (*rsa.PublicKey, error) and VerifyProof per the
codebase_examples snippet, plus a convenience
`VerifyEither(current, old *rsa.PublicKey, accessToken, uri, ts, proof,
proofOld string, window time.Duration) error` that accepts any single valid
combination (proof/proofOld × current/old keys — coolwsd rotation tolerance).
The test signer implements the SAME byte construction with
rsa.SignPKCS1v15 — a genuine mirror pair, hand-built vectors (no recorded
coolwsd traffic needed at this layer; e2e in 03-04 proves it against the
real signer).
Commits: `test(03-03): proof verification vectors (RED)` /
`feat(03-03): X-WOPI-Proof RSA-SHA256 verification, rotation-tolerant (GREEN)`.
  </action>
  <verify>cd wopi-host && go test -race ./internal/proof/ -count=1 && go vet ./internal/proof/</verify>
  <done>Self-signed vectors verify; each tamper case fails; ProofOld and old-key combinations accepted; window enforced on .NET ticks.</done>
  <recovery>See error_recovery — the three canonical byte-construction mistakes. Never loosen the tamper tests to pass.</recovery>
</task>

<task type="auto" tdd="true">
  <name>Task 2: internal/coolwsd — discovery client (urlsrc + proof keys, cached)</name>
  <files>wopi-host/internal/coolwsd/discovery.go, wopi-host/internal/coolwsd/discovery_test.go</files>
  <action>
First hand-capture the fixture: `curl -fsS http://127.0.0.1:9980/hosting/discovery`
against the running edendocs-code reference container; trim to writer/calc
apps + proof-key element; commit as testdata (real captured output — satisfies
no_llm_test_data). Then RED (test-list 13-15), then GREEN:

```go
type Client struct { baseURL string; ttl time.Duration; mu; cached *Discovery; fetchedAt time.Time }
type Discovery struct { ProofKey, OldProofKey *rsa.PublicKey; actions map[string]string /* ext → urlsrc */ }
func New(coolwsdURL string, cacheTTL time.Duration) *Client   // TTL default 1h
func (c *Client) Get(ctx) (*Discovery, error)                  // cached
func (c *Client) URLSrc(ctx, ext string) (string, error)
func (c *Client) Invalidate()
```
Parse with encoding/xml; collect every `<action ext=... urlsrc=...>`
regardless of zone/app nesting; parse proof-key via internal/proof.
Commits: `test(03-03): discovery client behavior + captured fixture (RED)` /
`feat(03-03): cached /hosting/discovery client (GREEN)`.
  </action>
  <verify>cd wopi-host && go test -race ./internal/coolwsd/ -count=1 && go vet ./internal/coolwsd/</verify>
  <done>urlsrc lookup + both proof keys parse from the real captured fixture; caching proven by request counting; Invalidate refetches.</done>
  <recovery>If the reference container is not running for capture: `docker start edendocs-code` (exists on this machine, PROJECT.md "Local reference runtime"); as a last resort capture from any coolwsd 26.x discovery output — but real capture is required, do not hand-write the fixture from memory.</recovery>
</task>

<task type="auto" tdd="true">
  <name>Task 3: internal/wopi endpoints + internal/launch page</name>
  <files>wopi-host/internal/wopi/wopi.go, wopi-host/internal/wopi/wopi_test.go, wopi-host/internal/launch/launch.go, wopi-host/internal/launch/launch_test.go</files>
  <action>
RED (test-list 1-9 and 16-19) then GREEN.

internal/wopi:
```go
func New(sessions *session.Store, store *storage.Store, disco *coolwsd.Client,
         wopiBaseURL string, publicURL string, requireProof bool, window time.Duration) http.Handler
```
Routes (Go 1.22 mux): `GET /wopi/files/{id}`, `GET /wopi/files/{id}/contents`,
`POST /wopi/files/{id}/contents`. Shared middleware order: (1) access_token
lookup → 401 + log; (2) proof verification via disco proof keys +
proof.VerifyEither over wopiBaseURL + r.URL.RequestURI() (skip only when
requireProof=false, log a WARN once) → reject + `proof: REJECTED` log;
(3) handler. CheckFileInfo per the exact struct in codebase_examples
(OwnerId "edendocs", UserCanWrite true, SupportsLocks false, IsAdminUser
false from identity, PostMessageOrigin publicURL). Emit the structured log
lines from gotchas verbatim — 03-04's e2e greps them.

internal/launch:
```go
func New(sessions *session.Store, store *storage.Store, disco *coolwsd.Client,
         cfg LaunchConfig, identityFrom func(*http.Request) (session.Identity, bool)) http.Handler
```
- `GET /` — document list (names + open links) when authenticated, else 302
  /login?return=/.
- `GET /open?file=<id>` — RequireLogin via identityFrom; storage.Stat; mint
  token (cfg.TokenTTL); disco.URLSrc(ext from filename); render
  html/template page: hidden POST form (action = urlsrc + "WOPISrc=" +
  url.QueryEscape(wopiSrc), target = iframe name) with access_token +
  access_token_ttl (ExpiresAt.UnixMilli()) inputs, full-viewport iframe,
  `<script>document.forms[0].submit()</script>`. No token in any GET URL.
Commits: `test(03-03): WOPI endpoints + launch page behavior (RED)` /
`feat(03-03): CheckFileInfo/GetFile/PutFile + POST-form launch page (GREEN)`.
  </action>
  <verify>cd wopi-host && go test -race ./... -count=1 && go vet ./... && go build ./... && ! grep -rn "oidcauth" internal/launch/ internal/wopi/ | grep -q . && ! grep -rn '"is_admin"' internal/wopi/ | grep -q .</verify>
  <done>All 19 test-list cases green under -race; launch/wopi import no oidcauth (decoupling held); golden CheckFileInfo JSON matches coolwsd's parsed contract; token never in a GET URL.</done>
  <recovery>If the golden JSON test fights omitempty semantics, marshal via an explicit struct (no map) and assert with a raw JSON string compare. If proof middleware breaks the launch tests, remember launch pages are BROWSER-facing (no proof headers) — proof applies to /wopi/ routes only.</recovery>
</task>

</tasks>

<validation_gates>
<lint>cd wopi-host && go vet ./...</lint>
<test>cd wopi-host && go test -race ./... -count=1</test>
<build>cd wopi-host && go build ./...</build>
</validation_gates>

<verification>
- Whole wopi-host module green under -race; zero changes outside wopi-host/;
  go.mod/go.sum untouched by this TRD.
- CheckFileInfo golden JSON: required keys present, UserExtraInfo absent,
  IsAdminUser top-level.
- Proof middleware enforced on all /wopi/ routes with rotation + window
  tolerance; launch page emits POST-form embed with ms-epoch
  access_token_ttl.
- Structured log-line contract present verbatim (grep the source for
  `wopi: CheckFileInfo` / `wopi: PutFile` / `proof: REJECTED`).
</verification>

<success_criteria>
The WOPI host speaks coolwsd's actual protocol: signed-request verification
mirroring ProofKey.cpp, the exact CheckFileInfo identity contract for AUTH-03,
byte-faithful GetFile/PutFile, and a launch page that safely hands the
short-lived WOPI token to the editor iframe — all proven by tests that sign
and shape requests exactly as coolwsd does.
</success_criteria>

<output>
After completion, create `.planning/objectives/03-aoid-authentication-integration-oidc/03-03-SUMMARY.md`
recording: constructor signatures for wopi/launch (03-04 wires them), the
log-line contract, and the captured discovery fixture provenance (container
version).
</output>
