---
objective: 03-aoid-authentication-integration-oidc
trd: "01"
type: standard
wave: 1
depends_on: []
files_modified:
  - wopi-host/go.mod
  - wopi-host/go.sum
  - wopi-host/cmd/wopi-host/main.go
  - wopi-host/internal/config/config.go
  - wopi-host/internal/session/session.go
  - wopi-host/internal/session/session_test.go
  - wopi-host/internal/storage/storage.go
  - wopi-host/internal/storage/storage_test.go
  - wopi-host/testdata/hello.odt
autonomous: true
requirements: [AUTH-01, AUTH-02]

must_haves:
  truths:
    - "A new, fully additive Go module exists at wopi-host/ with its own go.mod — repo-root builds (autotools, browser npm) are untouched and `git status` shows zero changes outside wopi-host/"
    - "The session package mints opaque, short-lived WOPI access_tokens (uuid, NOT AOID JWTs) and looks them up to (identity, fileID) until expiry — expired tokens are not returned"
    - "The storage package serves and persists document bytes from a configurable data dir, rejecting path-traversal fileIDs"
    - "go vet ./... and go test -race ./... -count=1 pass inside wopi-host/"
  artifacts:
    - path: "wopi-host/go.mod"
      provides: "Isolated nested Go module github.com/AO-Cyber-Systems/EdenDocs/wopi-host, go 1.26, requires eden-platform-go + google/uuid"
      contains: "github.com/aocybersystems/eden-platform-go"
    - path: "wopi-host/internal/session/session.go"
      provides: "Identity struct + web-session store + WOPI token mint/lookup with TTL"
      contains: "MintWopiToken"
    - path: "wopi-host/internal/storage/storage.go"
      provides: "Filesystem-backed document store (List/Stat/Read/Write) with fileID sanitization and atomic writes"
      contains: "func"
    - path: "wopi-host/testdata/hello.odt"
      provides: "Hand-built ODT fixture (byte-copy of upstream test/data/hello.odt) for tests and e2e seeding"
  key_links:
    - from: "wopi-host/internal/session/session.go"
      to: "wopi-host/internal/storage/storage.go"
      via: "WopiSession.FileID keys into storage fileIDs — the WOPI token is the ONLY join between an authenticated identity and a document"
      pattern: "FileID"
    - from: "wopi-host/cmd/wopi-host/main.go"
      to: "wopi-host/internal/config/config.go"
      via: "config.FromEnv() at startup; listen address defaults to 127.0.0.1:8091 (NEVER 8080)"
      pattern: "8091"
---

<objective>
Scaffold the reference WOPI host as an additive, isolated Go module at
`wopi-host/` (own go.mod — invisible to the C++/autotools build and to weekly
upstream merges), with the two foundation packages every later TRD builds on:
the session package (browser login sessions + WOPI access_token mint/lookup,
the Pitfall-2 "two unrelated credentials" boundary) and the filesystem document
storage package.

Purpose: AUTH-01's "new deployable outside wsd/browser" exists as a buildable
service skeleton; AUTH-02's short-lived WOPI access_token minting machinery is
implemented and tested before any HTTP surface consumes it.

Output: wopi-host/ module that builds, vets, and tests green; minimal main
serving /healthz on 127.0.0.1:8091.
</objective>

<file_tree>
wopi-host/                          ← CREATE (entire tree is new; nothing outside it is touched)
├── go.mod                          ← CREATE
├── go.sum                          ← CREATE
├── cmd/wopi-host/main.go           ← CREATE (minimal: config + /healthz; full wiring lands in TRD 03-04)
├── internal/
│   ├── config/config.go            ← CREATE
│   ├── session/session.go          ← CREATE
│   ├── session/session_test.go     ← CREATE
│   ├── storage/storage.go          ← CREATE
│   └── storage/storage_test.go     ← CREATE
└── testdata/hello.odt              ← CREATE (byte-copy of test/data/hello.odt)
</file_tree>

<execution_context>
@~/.claude/devflow/workflows/execute-trd.md
@~/.claude/devflow/templates/summary.md
</execution_context>

<embedded_context>

<codebase_examples>
Org Go conventions (verified from /Users/justin/dev/aofamily/browser/go and
/Users/justin/dev/eden-platform-go): plain `net/http` with Go 1.22+ pattern
routing, no frameworks; table-driven tests with `stretchr/testify` optional —
prefer stdlib assertions here to keep the dep tree minimal; `go test -race
./... -count=1` is the org CI test invocation.

eden-platform-go consumption (verified from /Users/justin/dev/aoid/go.mod and
/Users/justin/dev/aofamily/browser/go/go.mod): the module path is
`github.com/aocybersystems/eden-platform-go` (lowercase alias org) while the
repo lives at `github.com/AO-Cyber-Systems/eden-platform-go` (private). This
dev machine ALREADY has the required mapping configured (verified 2026-07-08):

```
git config --global url."https://<token>@github.com/AO-Cyber-Systems/".insteadOf "https://github.com/aocybersystems/"
go env GOPRIVATE  # github.com/aocybersystems/*
```

So `go get github.com/aocybersystems/eden-platform-go@latest` works locally
as-is. Pin the resolved pseudo-version in go.mod (aofamily pins
v0.0.0-20260701141824-09fedb51e64b; use `@latest` and record what resolves).
Do NOT add a `replace` directive — it would break CI (aoid uses replace only
because it never ships through shared CI).
</codebase_examples>

<anti_patterns>
- Do NOT touch ANY file outside wopi-host/ in this TRD. No .gitignore edits
  (tracked upstream file), no configure.ac, no Makefile.am, no wsd/, no
  browser/. The nested go.mod makes the module invisible to everything else.
- Do NOT make the WOPI access_token a JWT or embed AOID claims in it
  (research Pitfall 2): it is an opaque uuid coolwsd merely echoes back.
  Any code that base64/JWT-parses a WOPI token is wrong by construction.
- Do NOT use LLM-generated fixture bytes for tests (no_llm_test_data
  constraint): storage tests write hand-typed literal byte strings;
  testdata/hello.odt is a byte-copy of the upstream-authored
  test/data/hello.odt (`cp test/data/hello.odt wopi-host/testdata/hello.odt`).
- No property-based testing libraries (no_property_based_default), no
  .feature files (no_gherkin_layer) — descriptive Go subtest names carry the
  behavior descriptions.
- Module path MUST be `github.com/AO-Cyber-Systems/EdenDocs/wopi-host` (the
  actual repo path + subdirectory) so the nested module stays go-gettable.
  The research's literal `edendocs-wopi-host` suggestion predates the firm
  in-repo placement decision; a nested module's path must match its location.
</anti_patterns>

<error_recovery>
- If `go get github.com/aocybersystems/eden-platform-go@latest` fails with an
  auth error: verify `git config --global --get-regexp insteadOf` still shows
  the AO-Cyber-Systems mapping and GOPRIVATE includes
  `github.com/aocybersystems/*`. Do NOT vendor the module and do NOT switch to
  a replace directive — surface the auth problem instead.
- If `go mod tidy` pulls a surprising amount of eden-platform-go's dependency
  graph: expected — the module is large but Go module graph pruning means only
  packages actually imported (oidcrp and its deps) are built. go.sum will be
  long; that is fine.
- If port 8091 is occupied during a local run: that is a local environment
  issue; the listen address is configurable via EDENDOCS_WOPI_LISTEN_ADDR.
  NEVER fall back to 8080 under any circumstance.
</error_recovery>

</embedded_context>

<context>
@.planning/PROJECT.md
@.planning/objectives/03-aoid-authentication-integration-oidc/OBJECTIVE.md
@.planning/objectives/03-aoid-authentication-integration-oidc/3-RESEARCH.md
</context>

<research_context>
- Recommended structure (3-RESEARCH.md "Architecture Patterns"): in-repo
  `wopi-host/` module, `cmd/wopi-host/main.go` entrypoint, internal packages
  for session and storage. Port 8091 for dev/CI (firm recommendation).
- WOPI access_token contract (Pitfall 2): minted by the WOPI host
  (`uuid.NewString()`), stored → (AOID identity, file, expiry). coolwsd never
  inspects it. Short-lived (default 15 min).
- `access_token_ttl` (consumed later by the launch page, TRD 03-03) is the
  token EXPIRY as Unix milliseconds per MS-WOPI convention — expose
  `WopiSession.ExpiresAt time.Time` so 03-03 can compute `UnixMilli()`.
- Storage backend decision (research Open Question 4, resolved at planning):
  filesystem-backed store under a configurable data dir. Sufficient for a
  reference deployment; Redis/S3 documented later as production swap-ins.
</research_context>

<gotchas>
- HARD RULE: never bind, reference, or curl port 8080 anywhere — code,
  comments, tests, or docs. WOPI host default is 127.0.0.1:8091.
- eden-platform-go requires `go 1.26.1`; set `go 1.26` in wopi-host/go.mod
  (local toolchain is go1.26.4; CI will pin '1.26' via setup-go in TRD 03-04).
- EdenDocs is a PUBLIC repo and eden-platform-go is PRIVATE: external users
  cannot build wopi-host without org access. This is accepted (org mandate:
  never hand-roll OIDC) — note it in code comments near the go.mod require
  and it will be documented in TRD 03-05's runbook.
- MPL-2.0 discipline: wopi-host is NEW Eden-authored code. Give each .go file
  the standard MPL-2.0 header block used by Eden-authored files in
  scripts/eden/ (SPDX-License-Identifier: MPL-2.0, AO Cyber Systems
  copyright).
</gotchas>

## Test list

Behavior cases (outside-in within this TRD's layer; all are package-level unit
tests — the HTTP layer above them lands in 03-02/03-03):

**internal/session:**
1. MintWopiToken returns an opaque token that LookupWopiToken resolves to the
   same identity + fileID before expiry.
2. LookupWopiToken returns not-found for a token past its TTL (inject a fake
   `now func() time.Time`; no sleeps).
3. LookupWopiToken returns not-found for an unknown/garbage token.
4. Minted tokens are unique across calls and contain no identity material
   (not decodable as JWT — assert no "." separators / not base64 JSON).
5. CreateWebSession/LookupWebSession round-trips an Identity; expired web
   sessions are not returned.
6. Concurrent mint+lookup from multiple goroutines is race-free (exercised
   under `go test -race`).

**internal/storage:**
7. Write then Read round-trips exact bytes (hand-typed literal content).
8. Stat reports the correct size and a non-zero mod time for a seeded file.
9. List returns seeded files (seed testdata/hello.odt by copying in-test).
10. fileIDs containing `/`, `\`, or `..` are rejected with an error (path
    traversal) — Read, Write, and Stat all reject.
11. Write is atomic: a failed/partial write never leaves a truncated
    destination file (write to temp + rename; simulate by writing, then
    verifying no `*.tmp` residue).
12. Read of a nonexistent fileID returns a typed not-found error the WOPI
    layer can map to HTTP 404.

<tasks>

<task type="auto">
  <name>Task 1: Scaffold the wopi-host module, config, and minimal main</name>
  <files>wopi-host/go.mod, wopi-host/go.sum, wopi-host/cmd/wopi-host/main.go, wopi-host/internal/config/config.go, wopi-host/testdata/hello.odt</files>
  <action>
1. `mkdir -p wopi-host/cmd/wopi-host wopi-host/internal/config wopi-host/testdata`
2. `cd wopi-host && go mod init github.com/AO-Cyber-Systems/EdenDocs/wopi-host`
   then `go get github.com/aocybersystems/eden-platform-go@latest` and
   `go get github.com/google/uuid@latest`. Record the resolved
   eden-platform-go pseudo-version in the SUMMARY.
3. `cp test/data/hello.odt wopi-host/testdata/hello.odt` (upstream-authored
   fixture; do NOT generate a document).
4. internal/config/config.go — `type Config struct` + `FromEnv() (Config, error)`
   reading (env var → field, default):
   - EDENDOCS_WOPI_LISTEN_ADDR → ListenAddr, default "127.0.0.1:8091"
   - EDENDOCS_WOPI_PUBLIC_URL → PublicURL, default "http://127.0.0.1:8091"
     (origin the BROWSER uses; also the PostMessageOrigin value)
   - EDENDOCS_WOPI_BASE_URL → WopiBaseURL, default = PublicURL (origin
     COOLWSD uses to call back — differs in docker mode:
     http://host.docker.internal:8091)
   - EDENDOCS_WOPI_AOID_ISSUER → AOIDIssuer (no default; required by 03-02)
   - EDENDOCS_WOPI_OIDC_CLIENT_ID / _CLIENT_SECRET → ClientID/ClientSecret
   - EDENDOCS_WOPI_STATE_KEY → StateHMACKey []byte (base64); if unset,
     generate 32 random bytes at startup and log a WARN that sessions won't
     survive restarts (reference-host behavior)
   - EDENDOCS_WOPI_COOLWSD_URL → CoolwsdURL, default "http://127.0.0.1:9980"
   - EDENDOCS_WOPI_DATA_DIR → DataDir, default "./data"
   - EDENDOCS_WOPI_TOKEN_TTL → WopiTokenTTL time.Duration, default 15m
   - EDENDOCS_WOPI_REQUIRE_PROOF → RequireProof bool, default true
   Validate: ListenAddr must not contain ":8080" — return an explicit error
   ("port 8080 is banned project-wide; use 8091").
5. cmd/wopi-host/main.go — minimal for this wave: load config, log resolved
   (secret-redacted) config, `http.NewServeMux` with
   `GET /healthz` → 200 `{"status":"ok"}`, `http.ListenAndServe`.
   A `// NOTE: full route wiring lands in TRD 03-04` comment.
6. MPL-2.0 header on every .go file.
Commit: `feat(03-01): scaffold additive wopi-host Go module (config + healthz on 8091)`
(explicit file list; verify nothing outside wopi-host/ is staged).
  </action>
  <verify>cd wopi-host && go vet ./... && go build ./... && cd .. && git status --porcelain | grep -v '^?? wopi-host/' | grep -v '^A  wopi-host/' | grep -v '.planning/' | (! grep -q .) && ! grep -rn "8080" wopi-host/ --include='*.go' | grep -v 'banned' | grep -q . && cmp -s wopi-host/testdata/hello.odt test/data/hello.odt</verify>
  <done>wopi-host module builds and vets clean; healthz main exists; config defaults to 127.0.0.1:8091; hello.odt fixture is byte-identical to the upstream original; zero changes outside wopi-host/.</done>
  <recovery>If eden-platform-go fetch fails, see error_recovery (auth mapping). If go.mod ends up with a toolchain directive newer than 1.26, pin `go 1.26` and rerun go mod tidy.</recovery>
</task>

<task type="auto" tdd="true">
  <name>Task 2: internal/session — identity, web sessions, WOPI token mint/lookup</name>
  <files>wopi-host/internal/session/session.go, wopi-host/internal/session/session_test.go</files>
  <action>
RED first (test list cases 1-6), then GREEN. Design:

```go
type Identity struct {
    Subject string // AOID sub (account UUID) → CheckFileInfo UserId
    Name    string // display name after fallback chain → UserFriendlyName
    Email   string
    IsAdmin bool   // v1: always false (research Pitfall 1)
}

type WopiSession struct {
    Identity  Identity
    FileID    string
    ExpiresAt time.Time // 03-03 computes access_token_ttl = ExpiresAt.UnixMilli()
}

func NewStore(now func() time.Time) *Store  // inject clock for tests
func (s *Store) CreateWebSession(id Identity, ttl time.Duration) (sid string)
func (s *Store) LookupWebSession(sid string) (Identity, bool)
func (s *Store) DeleteWebSession(sid string)
func (s *Store) MintWopiToken(id Identity, fileID string, ttl time.Duration) (token string, sess WopiSession)
func (s *Store) LookupWopiToken(token string) (WopiSession, bool)
```

- Tokens and sids: `uuid.NewString()`. In-memory maps guarded by sync.RWMutex
  (reference host is single-instance; document Redis as the production
  swap-in in a package comment, do not build it).
- Lookup checks expiry against s.now() and deletes lazily when expired.
- One test per behavior case, descriptive subtest names
  (e.g. `t.Run("expired wopi token is not returned", ...)`).
Commits: `test(03-01): session store behavior (RED)` then
`feat(03-01): session store — web sessions + opaque WOPI token mint (GREEN)`.
  </action>
  <verify>cd wopi-host && go test -race ./internal/session/ -count=1 -v 2>&1 | tail -20 && go vet ./internal/session/</verify>
  <done>All 6 session behavior cases pass under -race; tokens are opaque uuids; expiry enforced via injected clock (no time.Sleep in tests).</done>
  <recovery>If -race flags a data race, guard all map access with the RWMutex — do not switch to sync.Map (keep the code reviewable).</recovery>
</task>

<task type="auto" tdd="true">
  <name>Task 3: internal/storage — filesystem document store</name>
  <files>wopi-host/internal/storage/storage.go, wopi-host/internal/storage/storage_test.go</files>
  <action>
RED first (test list cases 7-12), then GREEN. Design:

```go
type FileInfo struct { ID string; Size int64; ModTime time.Time }
var ErrNotFound = errors.New("storage: file not found")
var ErrBadFileID = errors.New("storage: invalid file id")

func New(dataDir string) (*Store, error)      // mkdir -p dataDir
func (s *Store) List() ([]FileInfo, error)
func (s *Store) Stat(fileID string) (FileInfo, error)
func (s *Store) Read(fileID string) ([]byte, FileInfo, error)
func (s *Store) Write(fileID string, r io.Reader) (FileInfo, error) // temp file + os.Rename (atomic)
```

- sanitize(fileID): reject empty, any `/` `\\`, or `..` — return ErrBadFileID
  (WOPI layer maps to 404/400). fileID == exact base filename in DataDir.
- Tests use t.TempDir(); fixture content is hand-typed literals plus one case
  copying wopi-host/testdata/hello.odt to prove binary round-trip fidelity
  (cmp bytes).
Commits: `test(03-01): filesystem storage behavior (RED)` then
`feat(03-01): filesystem document store with traversal guard + atomic writes (GREEN)`.
  </action>
  <verify>cd wopi-host && go test -race ./... -count=1 && go vet ./... && go build ./...</verify>
  <done>All 6 storage behavior cases pass; whole module green under -race; traversal attempts rejected by every method.</done>
  <recovery>If os.Rename fails across devices in some environment, create the temp file inside DataDir itself (same filesystem) — this is the required implementation anyway.</recovery>
</task>

</tasks>

<validation_gates>
<lint>cd wopi-host && go vet ./...</lint>
<test>cd wopi-host && go test -race ./... -count=1</test>
<build>cd wopi-host && go build ./...</build>
</validation_gates>

<verification>
- `git status --porcelain` shows changes ONLY under wopi-host/ (additive-module
  guarantee; AUTH-04's zero-wsd discipline starts here).
- `cd wopi-host && go build ./... && go vet ./... && go test -race ./... -count=1` all green.
- grep proves no 8080 anywhere under wopi-host/ (except the banned-port error
  string).
- wopi-host/testdata/hello.odt byte-identical to test/data/hello.odt.
</verification>

<success_criteria>
An isolated wopi-host Go module exists that builds/vets/tests green with the
session (WOPI-token minting) and storage foundations fully unit-tested, a
runnable /healthz main on 127.0.0.1:8091, and zero footprint outside
wopi-host/.
</success_criteria>

<output>
After completion, create `.planning/objectives/03-aoid-authentication-integration-oidc/03-01-SUMMARY.md`
recording: resolved eden-platform-go pseudo-version, exported API surfaces of
session/storage (03-02/03-03 build against them), and confirmation nothing
outside wopi-host/ changed.
</output>
