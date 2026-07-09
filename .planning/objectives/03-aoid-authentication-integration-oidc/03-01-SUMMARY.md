---
objective: 03-aoid-authentication-integration-oidc
job: "01"
subsystem: auth
tags: [go, wopi, oidc, session, storage, uuid, eden-platform-go]

# Dependency graph
requires:
  - objective: 01-from-source-build-pipeline
    provides: "running coolwsd (native 9980) for the eventual end-to-end round trip; CI dev-loop conventions"
provides:
  - "Isolated additive Go module wopi-host/ (own go.mod, invisible to autotools/npm builds and upstream merges)"
  - "internal/config: Config + FromEnv() with EDENDOCS_WOPI_* env vars, 8080-ban validation, secret-redacting Stringer"
  - "internal/session: Identity, WopiSession, Store — web login sessions + opaque WOPI access_token mint/lookup with injected clock TTL"
  - "internal/storage: filesystem document store (List/Stat/Read/Write) with traversal guard + atomic writes, typed ErrNotFound/ErrBadFileID"
  - "cmd/wopi-host: minimal main serving GET /healthz on 127.0.0.1:8091"
  - "testdata/hello.odt: byte-copy of upstream test/data/hello.odt for tests and e2e seeding"
affects: [03-02-oidc-relying-party, 03-03-wopi-protocol-surface, 03-04-wiring-trust-e2e, 03-05-real-aoid-verification]

# Tech tracking
tech-stack:
  added:
    - "github.com/aocybersystems/eden-platform-go v0.0.0-20260708235425-c5fd1ee7cedb (PRIVATE org module; resolved via pre-configured git insteadOf + GOPRIVATE)"
    - "github.com/google/uuid v1.6.0"
  patterns:
    - "Nested Go module github.com/AO-Cyber-Systems/EdenDocs/wopi-host — module path matches repo path + subdirectory, stays go-gettable"
    - "Injected clock (now func() time.Time) for TTL tests — no time.Sleep in tests"
    - "Opaque WOPI access_token = bare uuid.NewString(), NEVER a JWT (Pitfall 2: two unrelated credential spaces)"
    - "Atomic writes: temp file inside DataDir + os.Rename (same filesystem)"
    - "MPL-2.0 header + AO Cyber Systems copyright on every Eden-authored .go file"

key-files:
  created:
    - wopi-host/go.mod
    - wopi-host/go.sum
    - wopi-host/cmd/wopi-host/main.go
    - wopi-host/internal/config/config.go
    - wopi-host/internal/session/session.go
    - wopi-host/internal/session/session_test.go
    - wopi-host/internal/storage/storage.go
    - wopi-host/internal/storage/storage_test.go
    - wopi-host/testdata/hello.odt
  modified: []

key-decisions:
  - "eden-platform-go pinned at resolved pseudo-version v0.0.0-20260708235425-c5fd1ee7cedb (from @latest, per TRD; no replace directive, no vendoring)"
  - "go.mod pins `go 1.26` (not the toolchain's 1.26.4) so CI's setup-go '1.26' matches"
  - "Config implements fmt.Stringer with always-redacted secrets — accidental %v prints can never leak ClientSecret/StateHMACKey"
  - "AUTH-01/AUTH-02 NOT marked complete in REQUIREMENTS.md: both requirements' full text (OIDC flow; CheckFileInfo/GetFile/PutFile + launch page) is delivered by later TRDs (03-02/03-03/03-04) that also carry these IDs — marking now would be premature"

patterns-established:
  - "Two-credential boundary: session.Store holds web sessions (cookie sid -> Identity) and WOPI tokens (opaque uuid -> WopiSession) as separate spaces; WopiSession.FileID is the ONLY join between an authenticated identity and a document"
  - "WopiSession.ExpiresAt time.Time exposed so 03-03 computes access_token_ttl = ExpiresAt.UnixMilli() (MS-WOPI convention)"
  - "storage fileID = exact base filename in DataDir; sanitize rejects empty, /, \\, .. across Read/Write/Stat"

requirements-completed: []  # AUTH-01, AUTH-02 targeted but intentionally NOT marked complete — see key-decisions (later TRDs 03-02/03-03/03-04 finish them)

# Verification evidence
verification:
  gates_defined: 3
  gates_passed: 3
  auto_fix_cycles: 1
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: ~25min
completed: 2026-07-09
---

# Objective 3 TRD 01: wopi-host Go Module Scaffold Summary

**Additive wopi-host/ Go module with tested session (opaque WOPI-token minting via uuid) and filesystem storage foundations, config with 8080-ban validation, and a live /healthz main on 127.0.0.1:8091 — zero footprint outside wopi-host/**

## Performance

- **Duration:** ~25 min (across two executor sessions)
- **Started:** 2026-07-09T01:53:00Z
- **Completed:** 2026-07-09T02:05:00Z
- **Tasks:** 3
- **Files modified:** 9 (all created, all under wopi-host/)

## Accomplishments

- Isolated nested Go module `github.com/AO-Cyber-Systems/EdenDocs/wopi-host` builds/vets/tests green — invisible to the C++/autotools build and weekly upstream merges (AUTH-04's zero-wsd discipline starts here)
- session package: mints opaque, short-lived WOPI access_tokens (bare `uuid.NewString()`, not JWTs) resolved to (Identity, FileID) until expiry via an injected clock; web login sessions round-trip with TTL + delete; race-free under `-race`
- storage package: filesystem document store with path-traversal rejection on every method, atomic temp-file+rename writes, typed `ErrNotFound`/`ErrBadFileID` sentinels for the WOPI HTTP layer
- Live functional evidence: `/healthz` returns HTTP 200 `{"status":"ok"}` on 127.0.0.1:8091; startup with `EDENDOCS_WOPI_LISTEN_ADDR=127.0.0.1:8080` refuses with "port 8080 is banned project-wide; use 8091" (exit 1)

## Resolved Dependency Versions

| Module | Version |
|---|---|
| github.com/aocybersystems/eden-platform-go | **v0.0.0-20260708235425-c5fd1ee7cedb** (resolved from `@latest` 2026-07-09) |
| github.com/google/uuid | v1.6.0 |

## Exported API Surfaces (03-02/03-03 build against these)

### internal/config

```go
type Config struct {
    ListenAddr   string        // EDENDOCS_WOPI_LISTEN_ADDR, default "127.0.0.1:8091"; ":8080" rejected
    PublicURL    string        // EDENDOCS_WOPI_PUBLIC_URL, default "http://"+ListenAddr (browser origin / PostMessageOrigin)
    WopiBaseURL  string        // EDENDOCS_WOPI_BASE_URL, default = PublicURL (coolwsd callback origin)
    AOIDIssuer   string        // EDENDOCS_WOPI_AOID_ISSUER (no default; required by 03-02)
    ClientID     string        // EDENDOCS_WOPI_OIDC_CLIENT_ID
    ClientSecret string        // EDENDOCS_WOPI_OIDC_CLIENT_SECRET
    StateHMACKey []byte        // EDENDOCS_WOPI_STATE_KEY (base64); ephemeral 32 random bytes + WARN if unset
    CoolwsdURL   string        // EDENDOCS_WOPI_COOLWSD_URL, default "http://127.0.0.1:9980"
    DataDir      string        // EDENDOCS_WOPI_DATA_DIR, default "./data"
    WopiTokenTTL time.Duration // EDENDOCS_WOPI_TOKEN_TTL, default 15m
    RequireProof bool          // EDENDOCS_WOPI_REQUIRE_PROOF, default true
}
func FromEnv() (Config, error)
func (c Config) Redacted() Config
func (c Config) String() string   // fmt.Stringer, secrets ALWAYS redacted
```

### internal/session

```go
type Identity struct {
    Subject string // AOID sub (account UUID) -> CheckFileInfo UserId
    Name    string // display name after fallback chain -> UserFriendlyName
    Email   string
    IsAdmin bool   // v1: always false (research Pitfall 1)
}
type WopiSession struct {
    Identity  Identity
    FileID    string
    ExpiresAt time.Time // 03-03: access_token_ttl = ExpiresAt.UnixMilli()
}
func NewStore(now func() time.Time) *Store // inject time.Now in production
func (s *Store) CreateWebSession(id Identity, ttl time.Duration) string
func (s *Store) LookupWebSession(sid string) (Identity, bool)
func (s *Store) DeleteWebSession(sid string)
func (s *Store) MintWopiToken(id Identity, fileID string, ttl time.Duration) (string, WopiSession)
func (s *Store) LookupWopiToken(token string) (WopiSession, bool)
```

### internal/storage

```go
type FileInfo struct { ID string; Size int64; ModTime time.Time }
var ErrNotFound = errors.New("storage: file not found")   // -> HTTP 404
var ErrBadFileID = errors.New("storage: invalid file id") // -> HTTP 400/404
func New(dataDir string) (*Store, error)                  // mkdir -p dataDir
func (s *Store) List() ([]FileInfo, error)
func (s *Store) Stat(fileID string) (FileInfo, error)
func (s *Store) Read(fileID string) ([]byte, FileInfo, error)
func (s *Store) Write(fileID string, r io.Reader) (FileInfo, error) // atomic: temp-in-DataDir + os.Rename
```

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: Scaffold module, config, minimal main | `cd wopi-host && go vet ./... && go build ./...` + zero-changes-outside-wopi-host git check + 8080 grep + `cmp testdata/hello.odt test/data/hello.odt` | 0 | PASS |
| 2: internal/session (TDD) | `cd wopi-host && go test -race ./internal/session/ -count=1 && go vet ./internal/session/` | 0 | PASS |
| 3: internal/storage (TDD) | `cd wopi-host && go test -race ./... -count=1 && go vet ./... && go build ./...` | 0 | PASS |

Additional live evidence (Task 1 main): `curl http://127.0.0.1:8091/healthz` → HTTP 200 `{"status":"ok"}`; `EDENDOCS_WOPI_LISTEN_ADDR=127.0.0.1:8080` → startup refused, exit 1, error "config: port 8080 is banned project-wide; use 8091"; startup log with a real `EDENDOCS_WOPI_OIDC_CLIENT_SECRET` set shows `[REDACTED]` and grep for the secret value in the log finds nothing.

## Task Commits

1. **Task 1: Scaffold the wopi-host module, config, and minimal main** — `3969cd83823` (feat)
2. **Task 2: internal/session** — `47238cc403b` (test, RED) + `296b31c256c` (feat, GREEN)
3. **Task 3: internal/storage** — `3410ccdb176` (test, RED) + `cb097a26985` (feat, GREEN)
4. **Deviation fix** — `3a1e63ce447` (fix: Config fmt.Stringer with always-redacted secrets)

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| lint | `cd wopi-host && go vet ./...` | 0 | PASS |
| test | `cd wopi-host && go test -race ./... -count=1` | 0 | PASS |
| build | `cd wopi-host && go build ./...` | 0 | PASS |

## TDD Evidence

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| RED (session) | `go test ./internal/session/` | 1 (build failed: undefined Identity/NewStore) | FAIL (correct) |
| GREEN (session) | `go test -race ./internal/session/ -count=1` | 0 (all 6 behavior cases pass) | PASS (correct) |
| RED (storage) | `go test ./internal/storage/` | 1 (build failed: undefined New/ErrBadFileID/ErrNotFound) | FAIL (correct) |
| GREEN (storage) | `go test -race ./... -count=1` | 0 (all 6 behavior cases pass) | PASS (correct) |

No REFACTOR-phase commits — implementations came out clean on GREEN.

## Post-TRD Verification

- **Auto-fix cycles used:** 1
- **Must-haves verified:** 4/4 truths, 4/4 artifacts, 2/2 key links
  - Additive module: `git log --name-only` across all 6 commits shows zero files outside `wopi-host/`
  - Opaque short-lived tokens: token-opacity test asserts no "." separators and no identity material; expiry test uses injected clock
  - Storage traversal guard: 7 bad-fileID cases rejected across Read/Write/Stat
  - `go vet ./...` and `go test -race ./... -count=1` both exit 0 inside wopi-host/
  - `wopi-host/go.mod` contains `github.com/aocybersystems/eden-platform-go`; `session.go` contains `MintWopiToken`; `main.go`→`config.go` link on `8091`; `session.go`→`storage.go` link on `FileID`
- **Gate failures:** None (final). One intermediate compile error during Task 3 GREEN, fixed in the same task (see Deviations)
- **hello.odt fixture:** `cmp` byte-identical to upstream `test/data/hello.odt`
- **8080 discipline:** grep of wopi-host/ shows the string only in the ban constant, its comment, and the validation error message

## Files Created/Modified

- `wopi-host/go.mod` / `go.sum` — nested module, go 1.26, eden-platform-go + uuid pins, private-module note
- `wopi-host/cmd/wopi-host/main.go` — config load, redacted-config log, `GET /healthz`, listen on ListenAddr (full wiring lands in TRD 03-04)
- `wopi-host/internal/config/config.go` — Config/FromEnv/Redacted/String, EDENDOCS_WOPI_* env parsing, 8080-ban validation, ephemeral state-key generation
- `wopi-host/internal/session/session.go` — Identity/WopiSession/Store, web sessions + opaque WOPI token mint/lookup, RWMutex-guarded maps, lazy expiry deletion
- `wopi-host/internal/session/session_test.go` — 6 behavior cases, fake clock, no sleeps
- `wopi-host/internal/storage/storage.go` — List/Stat/Read/Write, sanitize traversal guard, atomic writes, typed sentinels
- `wopi-host/internal/storage/storage_test.go` — 6 behavior cases, t.TempDir, hand-typed literals + hello.odt binary round-trip
- `wopi-host/testdata/hello.odt` — byte-copy of upstream `test/data/hello.odt`

## Decisions Made

- eden-platform-go pinned at `v0.0.0-20260708235425-c5fd1ee7cedb` (resolved from `@latest`); no replace directive, no vendoring — local git insteadOf + GOPRIVATE mapping used as-is
- `go 1.26` directive (not 1.26.4) so TRD 03-04's CI `setup-go: '1.26'` matches without a toolchain pull
- AUTH-01/AUTH-02 left unmarked in REQUIREMENTS.md: their full requirement text spans TRDs 03-02/03-03/03-04, which also carry those IDs — the last contributing TRD should mark them
- `Config.String()` implements fmt.Stringer with unconditional redaction (see Deviations)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed Read-arity compile error in storage traversal test**
- **Found during:** Task 3 (internal/storage GREEN phase)
- **Issue:** The RED test called `store.Read(id)` with 2 return values; the designed signature returns 3 (`[]byte, FileInfo, error`)
- **Fix:** Corrected to `_, _, err := store.Read(id)` in the traversal-rejection case
- **Files modified:** wopi-host/internal/storage/storage_test.go
- **Verification:** `go test -race ./... -count=1` exit 0
- **Committed in:** `cb097a26985` (part of Task 3 GREEN commit)

**2. [Rule 2 - Missing Critical] Config implements fmt.Stringer with always-redacted secrets**
- **Found during:** Post-task live verification of the healthz main
- **Issue:** `log.Printf("%+v", cfg.Redacted())` printed the redacted `[]byte` placeholder as byte numerals; more importantly, any future direct `%v` of a raw Config would dump the real StateHMACKey bytes and ClientSecret
- **Fix:** Added `func (c Config) String() string` routing every `%v`/`%+v` print through `Redacted()` — accidental logs can no longer leak secrets
- **Files modified:** wopi-host/internal/config/config.go
- **Verification:** Live run with `EDENDOCS_WOPI_OIDC_CLIENT_SECRET=super-secret-value`: log shows `[REDACTED]`, grep for the secret in the log output finds nothing
- **Committed in:** `3a1e63ce447`

---

**Total deviations:** 2 auto-fixed (1 bug, 1 missing critical)
**Impact on plan:** Both necessary for correctness/security. No scope creep; zero files outside wopi-host/ touched.

## Issues Encountered

- Executor session hit a turn limit mid-Task 3 (storage.go written but uncommitted); a continuation session verified state via commit hashes and completed GREEN + gates. No work lost or redone.

## User Setup Required

None - no external service configuration required. (Local dev relies on the ALREADY-CONFIGURED global git insteadOf + GOPRIVATE mapping for the private eden-platform-go module; CI setup for that lands in TRD 03-04.)

## Next Objective Readiness

- 03-02 (OIDC relying party, wave 2) can build against `config.Config` (AOIDIssuer/ClientID/ClientSecret/StateHMACKey) and `session.Store` (CreateWebSession/LookupWebSession)
- 03-03 (WOPI protocol surface, wave 2) can build against `session.MintWopiToken`/`LookupWopiToken` (ExpiresAt → access_token_ttl UnixMilli) and `storage.Store` (Read/Write/Stat + typed errors → 404/400 mapping)
- No blockers. Port discipline (8091 host / 8092 fake IdP / 9980 coolwsd) encoded in config defaults + validation.

## Self-Check: PASSED

- All 9 created files exist on disk
- All 6 commit hashes present in git log (`3969cd83823`, `47238cc403b`, `296b31c256c`, `3410ccdb176`, `cb097a26985`, `3a1e63ce447`)
- Validation gates re-run at completion: lint 0 / test 0 / build 0

---
*Objective: 03-aoid-authentication-integration-oidc*
*Completed: 2026-07-09*
