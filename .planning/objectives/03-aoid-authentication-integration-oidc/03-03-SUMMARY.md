---
objective: 03-aoid-authentication-integration-oidc
trd: "03"
subsystem: auth
tags: [go, wopi, proof-key, discovery, checkfileinfo, launch-page, coolwsd]

# Dependency graph
requires:
  - trd: "03-01"
    provides: "internal/session (Identity, WopiSession, MintWopiToken/LookupWopiToken) + internal/storage (Stat/Read/Write, ErrNotFound/ErrBadFileID) + injected-clock and opaque-token patterns"
provides:
  - "internal/proof: ParseProofKey + VerifyProof + VerifyEither — byte-for-byte mirror of wsd/ProofKey.cpp (len-prefixed token + UPPER(full URL) + .NET ticks, RSA-SHA256 PKCS1v15), rotation-tolerant, configurable timestamp window"
  - "internal/coolwsd: cached /hosting/discovery client — urlsrc(ext) lookup + current/old proof-key extraction, TTL cache + Invalidate"
  - "internal/wopi: CheckFileInfo/GetFile/PutFile handlers behind token-lookup + proof middleware; exact coolwsd field contract; structured log-line contract for 03-04 e2e greps"
  - "internal/launch: document list + launch page minting WOPI tokens, hidden auto-submitting POST-form iframe embed (token never in a GET URL)"
affects: [03-04-wiring-trust-e2e, 03-05-real-aoid-verification]

# Tech tracking
tech-stack:
  added: []  # stdlib + existing 03-01 deps only; go.mod/go.sum untouched (03-02 owns them this wave)
  patterns:
    - "Proof verification over wopiBaseURL + r.URL.RequestURI() — never r.Host (proxies/containers rewrite Host)"
    - ".NET-tick conversion via ticksToUnixEpoch=621355968000000000 offset — direct Duration math from year 1 overflows int64 nanoseconds"
    - "Identity injection: launch.New takes identityFrom func(*http.Request) (session.Identity, bool) — no OIDC RP import, wave-2 TRDs stay file-disjoint"
    - "Discovery actions collected by ext across ALL zones/apps, first hit wins — no zone-order assumption"
    - "CheckFileInfo via explicit struct marshal (UserExtraInfo omitempty + never populated -> key absent)"

key-files:
  created:
    - wopi-host/internal/proof/proof.go
    - wopi-host/internal/proof/proof_test.go
    - wopi-host/internal/coolwsd/discovery.go
    - wopi-host/internal/coolwsd/discovery_test.go
    - wopi-host/internal/coolwsd/testdata/discovery.xml
    - wopi-host/internal/wopi/wopi.go
    - wopi-host/internal/wopi/wopi_test.go
    - wopi-host/internal/launch/launch.go
    - wopi-host/internal/launch/launch_test.go
  modified: []

key-decisions:
  - "launch config type named launch.Config (not LaunchConfig) — avoids launch.LaunchConfig stutter; 03-04 wires launch.Config{WopiBaseURL, TokenTTL}"
  - "Timestamp window is a parameter (tests/wiring use 20 minutes, MS-WOPI convention) — research pseudocode's 20s is too tight for CI"
  - "Traversal/unknown fileIDs both map to 404 (storage guard surfaces before any authz-binding comparison)"
  - "UserFriendlyName fallback chain Name -> Email -> \"EdenDocs User\" — never empty (coolwsd LOG_ERRs + substitutes UnknownUser)"
  - "Generated proof_key in the live reference container (coolconfig generate-proof-key) — prerequisite for the captured fixture AND for 03-04's e2e signing (Pitfall 3: no proof_key file = coolwsd signs nothing)"

requirements-completed: []  # AUTH-02/03/04 carried but NOT marked — 03-04 (wiring + e2e) is the last contributing TRD for all three

# Verification evidence
verification:
  gates_defined: 3
  gates_passed: 3
  auto_fix_cycles: 1
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: ~30min
completed: 2026-07-09
---

# Objective 3 TRD 03: WOPI Protocol Surface Summary

**The WOPI host now speaks coolwsd's actual protocol: ProofKey.cpp-mirror signature verification on every /wopi/ request, a cached discovery client (urlsrc + proof keys from a hand-captured live fixture), CheckFileInfo/GetFile/PutFile with coolwsd's exact parsed field contract, and a launch page that hands the WOPI token to the editor iframe via hidden auto-submitting POST form — all proven by tests that sign and shape requests exactly as coolwsd does.**

## Performance

- **Duration:** ~30 min (across two executor turns in one session)
- **Completed:** 2026-07-09T02:40:00Z
- **Tasks:** 3 (all TDD RED→GREEN)
- **Files:** 9 created, all under wopi-host/internal/; go.mod/go.sum untouched

## Constructor Signatures (03-04 wires these)

### internal/wopi

```go
func New(sessions *session.Store, store *storage.Store, disco *coolwsd.Client,
         wopiBaseURL string, publicURL string, requireProof bool, window time.Duration) http.Handler
// Routes: GET /wopi/files/{id} | GET /wopi/files/{id}/contents | POST /wopi/files/{id}/contents
// wopiBaseURL: coolwsd's callback origin (host.docker.internal in docker mode) — proof URLs built from it
// publicURL:   browser origin -> CheckFileInfo PostMessageOrigin
// requireProof=false skips proof verification (dev only, WARN-logged once)
// window: X-WOPI-TimeStamp drift tolerance (use 20m — MS-WOPI convention)
```

### internal/launch

```go
type Config struct {
    WopiBaseURL string        // becomes WOPISrc origin in the form action
    TokenTTL    time.Duration // WOPI access_token lifetime per launch
}
func New(sessions *session.Store, store *storage.Store, disco *coolwsd.Client,
         cfg Config, identityFrom func(*http.Request) (session.Identity, bool)) http.Handler
// Routes: GET /{$} (document list) | GET /open?file=<id>
// NOTE: TRD sketched "LaunchConfig"; shipped as launch.Config (idiomatic, no stutter).
// identityFrom: 03-04 injects the OIDC RP package's request-context helper; unauthenticated -> 302 /login?return=<uri>
```

### internal/coolwsd

```go
func New(coolwsdURL string, cacheTTL time.Duration) *Client       // cacheTTL: 1h production default
func (c *Client) Get(ctx context.Context) (*Discovery, error)     // cached; Discovery{ProofKey, OldProofKey *rsa.PublicKey}
func (c *Client) URLSrc(ctx context.Context, ext string) (string, error) // ErrUnknownExtension typed error
func (c *Client) Invalidate()
```

### internal/proof

```go
func ParseProofKey(modulusB64, exponentB64 string) (*rsa.PublicKey, error)
func VerifyProof(pub *rsa.PublicKey, accessToken, uri, timestampHeader, proofB64 string, window time.Duration) error
func VerifyEither(current, old *rsa.PublicKey, accessToken, uri, timestampHeader,
                  proofB64, proofOldB64 string, window time.Duration) error
// Sentinels: ErrStaleProof, ErrNoProofKey, ErrNoMatch
```

## Log-Line Contract (verbatim — 03-04's e2e greps these)

```
wopi: CheckFileInfo file=<id> user=<sub> status=<code>
wopi: GetFile file=<id> status=<code>
wopi: PutFile file=<id> bytes=<n> status=<code>
proof: verified ok
proof: REJECTED reason=<...>
```

## Discovery Fixture Provenance

`wopi-host/internal/coolwsd/testdata/discovery.xml` was hand-captured (not invented) on 2026-07-09 via `curl -fsS http://127.0.0.1:9980/hosting/discovery` against the LIVE `edendocs-code` reference container running **coolwsd 26.04.2.1** — trimmed to the writer + calc apps plus the full `<proof-key>` element (4096-bit RSA, current==old moduli per coolwsd's own "TODO: implement proper rotation" duplication). The container initially had NO proof_key file (so discovery published no `<proof-key>`); it was generated in-container via `docker exec -u root edendocs-code coolconfig generate-proof-key` — coolwsd picked it up live, no restart needed. **This generated key persists in the container at /etc/coolwsd/proof_key and is the key coolwsd will sign with in 03-04's e2e.**

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: internal/proof (TDD) | `cd wopi-host && go test -race ./internal/proof/ -count=1 && go vet ./internal/proof/` | 0 | PASS |
| 2: internal/coolwsd (TDD) | `cd wopi-host && go test -race ./internal/coolwsd/ -count=1 && go vet ./internal/coolwsd/` | 0 | PASS |
| 3: internal/wopi + internal/launch (TDD) | `cd wopi-host && go test -race ./... -count=1 && go vet ./... && go build ./...` + no-oidc-RP-import grep + no-"is_admin" grep | 0 | PASS |

All 19 TRD test-list cases implemented and green under `-race`: wopi 1–9 (golden CheckFileInfo, byte-identical GetFile, persisting PutFile, 401s, proof rejection/rotation/stale/traversal), proof 10–12 (mirror-pair vectors, tamper, ParseProofKey), coolwsd 13–15 (urlsrc, proof keys, cache+Invalidate), launch 16–19 (login redirect, POST-form embed w/ ms-epoch ttl, token-not-in-URL regression guard, 404/400 paths).

## Task Commits

1. **Task 1: internal/proof** — `6f3bb7c6062` (test, RED) + `36e39ac6ce2` (feat, GREEN)
2. **Task 2: internal/coolwsd** — `aae510f03bb` (test + captured fixture, RED) + `1f1c3e035af` (feat, GREEN)
3. **Task 3: internal/wopi + internal/launch** — `fe043b9e335` (test, RED) + `149ab93978e` (feat, GREEN)

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| lint | `cd wopi-host && go vet ./...` | 0 | PASS |
| test | `cd wopi-host && go test -race ./... -count=1` | 0 | PASS |
| build | `cd wopi-host && go build ./...` | 0 | PASS |

Extra TRD greps: `oidcauth` absent from internal/launch/ + internal/wopi/ (imports AND text) — OK; `"is_admin"` absent from internal/wopi/ — OK; `gofmt -l internal/` clean; git shows zero changes to go.mod/go.sum and zero files outside wopi-host/ across all six commits.

## TDD Evidence

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| RED (proof) | `go test ./internal/proof/` | build failed: undefined VerifyProof/ParseProofKey/VerifyEither | FAIL (correct) |
| GREEN (proof) | `go test -race ./internal/proof/ -count=1` | 0 (6 tests incl. 3 tamper subtests) | PASS (correct) |
| RED (coolwsd) | `go test ./internal/coolwsd/` | build failed: undefined New/ErrUnknownExtension | FAIL (correct) |
| GREEN (coolwsd) | `go test -race ./internal/coolwsd/ -count=1` | 0 (4 tests incl. request-counted caching) | PASS (correct) |
| RED (wopi+launch) | `go test ./internal/wopi/ ./internal/launch/` | build failed: undefined New/Config | FAIL (correct) |
| GREEN (wopi+launch) | `go test -race ./internal/wopi/ ./internal/launch/ -count=1` | 0 (9 wopi + 6 launch tests) | PASS (correct) |

No REFACTOR-phase commits needed.

## Post-TRD Verification

- **Auto-fix cycles used:** 1 (the .NET-ticks overflow, fixed within Task 1 GREEN)
- **Must-haves verified:** 4/4 truths, 4/4 artifacts, 4/4 key links
  - CheckFileInfo golden test asserts the exact key set, `"UserExtraInfo"` ABSENT from raw body, top-level IsAdminUser false, PostMessageOrigin present
  - GetFile byte-identical + Content-Length; PutFile atomic (storage.Write temp+rename) + 200 + read-back equality
  - Proof middleware on all three /wopi/ routes: missing/invalid/stale rejected, ProofOld rotation accepted, `proof: REJECTED` logged
  - Launch page: method="post" form, action = urlsrc + "WOPISrc=" + urlencoded wopiSrc, hidden access_token (uuid, resolves in store), access_token_ttl == ExpiresAt.UnixMilli(), iframe + auto-submit; regression guard proves token appears in NO href/src/action URL
  - Artifact contains-checks: proof.go `VerifyPKCS1v15` ✓, discovery.go `hosting/discovery` ✓, wopi.go `IsAdminUser` ✓, launch.go `access_token_ttl` ✓
  - Key links: wopi.go→session.go `LookupWopiToken` ✓, wopi.go→proof.go `X-WOPI-Proof` ✓, launch.go→discovery.go `urlsrc` ✓, launch.go→session.go `MintWopiToken` ✓
- **Gate failures:** None final. Two intermediate: (a) tick-overflow test failures at first GREEN run, (b) gofmt drift in discovery.go — both fixed pre-commit
- **8080 discipline:** grep of new files shows zero occurrences; examples use 8091/9980

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Reference container had no proof_key — discovery published no `<proof-key>` element**
- **Found during:** Task 2 fixture capture (initial `curl` of /hosting/discovery had zero proof-key content)
- **Issue:** coolwsd only signs (and only publishes keys) if `{COOLWSD_CONFIGDIR}/proof_key` exists — exactly 3-RESEARCH.md Pitfall 3. Without it the fixture could not be honestly captured and 03-04's e2e could never verify proofs.
- **Fix:** `docker exec -u root edendocs-code coolconfig generate-proof-key` (ssh-keygen absent in image); coolwsd served the new `<proof-key>` immediately, no restart
- **Files modified:** none in-repo (container state); captured fixture committed in `aae510f03bb`

**2. [Rule 1 - Bug] Research pseudocode's .NET-tick conversion overflows time.Duration**
- **Found during:** Task 1 GREEN (round-trip test failed with a saturated 2562047h duration)
- **Issue:** `epoch.Add(time.Duration(ticks) * 100 * time.Nanosecond)` from the TRD/research snippet overflows — int64-nanosecond Duration spans ~292 years, far less than year-1→now
- **Fix:** convert via `ticksToUnixEpoch = 621355968000000000` (.NET's DateTime.UnixEpoch.Ticks): `time.Unix(0, (ticks-ticksToUnixEpoch)*100)`; test helper mirrors it
- **Files modified:** wopi-host/internal/proof/proof.go, proof_test.go
- **Commit:** `36e39ac6ce2` (part of Task 1 GREEN)

**3. [Rule 3 - Blocking] Executor worktree branched before the 03-01 merge**
- **Found during:** load_project_state (worktree had no wopi-host/ or .planning/)
- **Fix:** merged eden-main (f1e07bc091d, clean merge) before starting Task 1

**4. [Rule 1 - Bug] Doc comments contained the literal string "oidcauth", tripping the TRD's literal decoupling grep**
- **Found during:** Task 3 validation gates
- **Fix:** reworded three comments to "the OIDC RP package" — imports were never present; the gate is now honest for both text and imports
- **Commit:** `149ab93978e` (part of Task 3 GREEN)

---

**Total deviations:** 4 auto-fixed (2 bugs, 2 blocking). No scope creep; no Rule-4 escalations.

## Known Limitations (documented for the 03-05 runbook)

- `SupportsLocks: false` — deliberate v1 scope reduction (research Open Question 1: lock-conflict semantics untraced). coolwsd omits X-WOPI-Lock headers accordingly.
- `IsAdminUser` always false (AOID v1.0 has no consumable admin claim — Pitfall 1).
- `UserExtraInfo` omitted entirely (no avatar claim in AOID; browser falls back to built-in user.svg — Pitfall 5).

## Next TRD Readiness (03-04)

- Wire `wopi.New(sessions, store, disco, cfg.WopiBaseURL, cfg.PublicURL, cfg.RequireProof, 20*time.Minute)` and `launch.New(sessions, store, disco, launch.Config{WopiBaseURL: cfg.WopiBaseURL, TokenTTL: cfg.WopiTokenTTL}, oidc-RP identityFrom)` into cmd/wopi-host main
- The reference container's proof_key now exists (generated this TRD) — e2e proof signing will work out of the box; e2e log greps must use the log-line contract above verbatim
- Launch pages are browser-facing: mount launch and wopi handlers so proof middleware applies ONLY to /wopi/ routes

## Self-Check: PASSED

- All 9 created files exist on disk
- All 6 commit hashes present in git log (6f3bb7c6062, 36e39ac6ce2, aae510f03bb, 1f1c3e035af, fe043b9e335, 149ab93978e)
- Zero files outside wopi-host/ in any task commit; go.mod/go.sum byte-untouched
- Validation gates re-run at completion: lint 0 / test 0 / build 0

---
*Objective: 03-aoid-authentication-integration-oidc*
*Completed: 2026-07-09*
