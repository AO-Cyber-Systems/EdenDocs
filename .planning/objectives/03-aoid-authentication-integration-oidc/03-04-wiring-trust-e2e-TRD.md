---
objective: 03-aoid-authentication-integration-oidc
trd: "04"
type: standard
wave: 3
depends_on: ["03-02", "03-03"]
files_modified:
  - wopi-host/cmd/wopi-host/main.go
  - wopi-host/cmd/e2e-probe/main.go
  - wopi-host/go.mod
  - wopi-host/go.sum
  - wopi-host/scripts/wopi-e2e.sh
  - .github/workflows/wopi-host.yml
  - .github/workflows/build.yml
autonomous: true
requirements: [AUTH-01, AUTH-02, AUTH-04]
user_setup:
  - service: github
    why: "CI must fetch the private eden-platform-go module from a public repo's workflows"
    env_vars:
      - name: GITOPS_PAT
        source: "GitHub repo/org Actions secret on AO-Cyber-Systems/EdenDocs — a PAT with read access to AO-Cyber-Systems/eden-platform-go (same secret aofamily CI uses). gh secret list showed NO repo-level secrets on EdenDocs; add via: gh secret set GITOPS_PAT --repo AO-Cyber-Systems/EdenDocs (user supplies the token value)"

must_haves:
  truths:
    - "One command (wopi-host/scripts/wopi-e2e.sh) proves the full chain headlessly: OIDC login (fake AOID) → launch page mints a WOPI token → coolwsd opens a real document session → coolwsd calls back CheckFileInfo + GetFile with valid proof signatures → a forced save round-trips through PutFile"
    - "coolwsd trusts the WOPI host ONLY because storage.wopi.alias_groups is set with mode=\"groups\" — proven positively (trusted WOPISrc loads) AND negatively (a WOPISrc on an unregistered host is rejected as unauthorized by the same coolwsd instance)"
    - "Zero commits in this objective touch wsd/ or browser/src — verified mechanically from git history (AUTH-04's no-OIDC-in-coolwsd clause)"
    - "A path-filtered wopi-host CI workflow (vet/build/test -race) is green on eden-main independently of the engine-tarball blocker; the build.yml e2e step is wired additively even though the pipeline is currently red upstream of it (BLOCKED-EXOGENOUS engine SIGSEGV — 02-05's blocker, not ours)"
  artifacts:
    - path: "wopi-host/cmd/wopi-host/main.go"
      provides: "Full route wiring: /healthz, /login, /callback, /logout, /, /open, /wopi/files/* with oidcauth identity injected into launch"
      contains: "wopi/files"
    - path: "wopi-host/cmd/e2e-probe/main.go"
      provides: "Headless coolwsd WebSocket client: coolclient handshake + load + forced save; positive and expect-unauthorized modes"
      contains: "coolclient"
    - path: "wopi-host/scripts/wopi-e2e.sh"
      provides: "Dual-mode (native CI / local docker) end-to-end orchestration with fail() log-surfacing and log-grep assertions"
      contains: "alias_groups"
    - path: ".github/workflows/wopi-host.yml"
      provides: "Engine-independent Go CI for wopi-host (private-module fetch via GITOPS_PAT, vet/build/test -race)"
      contains: "GOPRIVATE"
  key_links:
    - from: "wopi-host/scripts/wopi-e2e.sh"
      to: "coolwsd runtime config"
      via: "native mode: --o:storage.wopi.alias_groups[@mode]=groups --o:storage.wopi.alias_groups.group[0].host=http://127.0.0.1:8091 --o:storage.wopi.alias_groups.group[0].host[@allow]=true (keys verified against wsd/HostUtil.cpp parse loop); docker mode: -e aliasgroup1"
      pattern: "alias_groups\\[@mode\\]=groups"
    - from: "wopi-host/cmd/e2e-probe/main.go"
      to: "coolwsd /cool/<enc>/ws WebSocket"
      via: "URL + handshake mirroring browser/js/global.js makeDocAndWopiSrcUrl and browser/src/app/socket.ts (coolclient → load url=)"
      pattern: "load url="
    - from: "wopi-host/scripts/wopi-e2e.sh"
      to: "wopi-host structured logs"
      via: "greps the 03-03 log-line contract: 'wopi: CheckFileInfo … status=200', 'wopi: GetFile … status=200', 'wopi: PutFile … status=200', 'proof: verified ok'"
      pattern: "wopi: PutFile"
    - from: "wopi-host/cmd/wopi-host/main.go"
      to: "wopi-host/internal/oidcauth + internal/launch"
      via: "launch.New(..., identityFrom = oidcauth.FromContext-based extractor wrapped by RequireLogin) — the wave-2 decoupling seam closes here"
      pattern: "identityFrom"
---

<objective>
Close the loop: wire every package into the wopi-host binary, register the
WOPI host with coolwsd via `storage.wopi.alias_groups mode="groups"` (config
only — the single coolwsd-side change this objective is allowed), and prove
AUTH-01/02/04 end to end with a headless, deterministic e2e script that runs
in two modes (CI native coolwsd on 9980; local docker reference container) —
plus CI wiring that keeps wopi-host green independently of the current
engine-tarball blocker.

Purpose: this TRD is where "coolwsd accepts the minted token for a real
document session" stops being a claim and becomes a grep-able log line.

Output: runnable wopi-host binary, e2e-probe, wopi-host/scripts/wopi-e2e.sh, a
green wopi-host.yml workflow, an additive build.yml e2e step, and a PASSING
local e2e run recorded in the SUMMARY.
</objective>

<file_tree>
wopi-host/
├── cmd/wopi-host/main.go        ← MODIFY (full wiring)
├── cmd/e2e-probe/main.go        ← CREATE
├── go.mod / go.sum              ← MODIFY (add gorilla/websocket for the probe)
wopi-host/scripts/wopi-e2e.sh         ← CREATE (Eden-owned script dir, additive)
.github/workflows/wopi-host.yml  ← CREATE
.github/workflows/build.yml      ← MODIFY (additive steps only: setup-go + private-module config + e2e step after verify-branding)
</file_tree>

<execution_context>
@~/.claude/devflow/workflows/execute-trd.md
@~/.claude/devflow/templates/summary.md
</execution_context>

<embedded_context>

<codebase_examples>
alias_groups config keys — verified against wsd/HostUtil.cpp parseAliases
(reads `storage.wopi.alias_groups.group[i].host`, `.host[@allow]` bool,
`.alias[j]`; mode at `storage.wopi.alias_groups[@mode]`, which MUST be
exactly "groups" once any group[0] exists or the whole block is LOG_ERR'd and
ignored — research Pitfall 4). coolwsd's --o: command-line overrides set these
exact keys on the layered Poco config, so NATIVE mode needs no XML editing:

```bash
./coolwsd --disable-ssl \
  --o:sys_template_path="$PWD/systemplate" \
  --o:child_root_path="$PWD/jails" \
  --o:cache_files.path="$PWD/cache" \
  --o:storage.wopi.alias_groups[@mode]=groups \
  --o:storage.wopi.alias_groups.group[0].host=http://127.0.0.1:8091 \
  --o:storage.wopi.alias_groups.group[0].host[@allow]=true \
  --o:admin_console.username=admin --o:admin_console.password=admin &
```
(Launch pattern mirrors scripts/eden/smoke-test.sh:23-28, which also shows the
setcap + systemplate bootstrap this script must repeat, the `./coolwsd --probe`
readiness loop, and the fail() helper with /tmp/coolwsd.log tail.)

Proof key for native mode — build.sh configures --enable-debug, and
wsd/ProofKey.cpp resolves the key path to DEBUG_ABSSRCDIR/proof_key (the repo
root) in debug builds. Generate ephemeral + clean up:
```bash
[ -f ./proof_key ] || ssh-keygen -t rsa -b 2048 -N "" -m PEM -f ./proof_key
# trap: rm -f ./proof_key ./proof_key.pub  (NEVER commit these)
```

Docker mode — the long-running edendocs-code reference container already
publishes 127.0.0.1:9980, so the e2e spins its OWN container on host port
9981 (never 8080):
```bash
docker run --rm -d --name edendocs-wopi-e2e -p 127.0.0.1:9981:9980 \
  -e aliasgroup1='http://host\.docker\.internal:8091' \
  -e extra_params='--o:ssl.enable=false' \
  collabora/code:latest
```
In docker mode: COOLWSD_URL=http://127.0.0.1:9981 (browser/probe side),
EDENDOCS_WOPI_BASE_URL=http://host.docker.internal:8091 (coolwsd-callback
side), wopi-host listens on 0.0.0.0:8091 so the container can reach it.

WebSocket URL + handshake — verified from browser/js/global.js
makeDocAndWopiSrcUrl (line ~2063) and browser/src/app/socket.ts _onSocketOpen
(line ~407):
```
docurl = WOPISRC + "?access_token=" + TOKEN + "&access_token_ttl=" + TTL + "&permission=edit"
wsURL  = ws://127.0.0.1:{PORT}/cool/ + urlQueryEscape(docurl) + "/ws"
         + "?WOPISrc=" + urlQueryEscape(WOPISRC) + "&compat=" + "/ws"
msg1   = "coolclient 0.1 <epoch-ms> <epoch-ms>"
msg2   = "load url=" + urlQueryEscape(docurl)
```
Success signal: any server text frame starting with "status:" (doc loaded ⇒
coolwsd completed CheckFileInfo + GetFile). Failure: frame starting with
"error:" (e.g. `error: cmd=internal kind=unauthorized` for untrusted WOPI
hosts). Forced save: send `save dontTerminateEdit=1 dontSaveIfUnmodified=0`
(wsd protocol; forces PutFile even on an unmodified doc).

Private-module CI step — copy VERBATIM from aofamily
.github/workflows/browser.yml:39-48 (proven pattern):
```yaml
- name: Configure private-module fetch
  env:
    GO_PRIVATE_TOKEN: ${{ secrets.GITOPS_PAT }}
  run: |
    if [ -n "$GO_PRIVATE_TOKEN" ]; then
      git config --global url."https://${GO_PRIVATE_TOKEN}@github.com/AO-Cyber-Systems/".insteadOf "https://github.com/aocybersystems/"
      go env -w GOPRIVATE=github.com/aocybersystems/*
    else
      echo "WARN: GITOPS_PAT secret not set — private-module fetch will fail."
    fi
```
</codebase_examples>

<anti_patterns>
- Do NOT edit wsd/, browser/, coolwsd.xml.in, configure.ac, or any tracked
  upstream file. coolwsd trust is runtime config (--o: overrides / container
  env) + documentation only.
- Do NOT edit scripts/eden/smoke-test.sh, verify-branding.sh, build.sh, or
  eden-branding/ (Objective 1/2 surfaces; 02-05 is mid-flight). wopi-e2e.sh
  is a NEW file; build.yml gets ADDITIVE steps only (append after
  verify-branding — never reorder or modify existing steps).
- Do NOT attempt to fix the current build.yml redness at the smoke step — it
  is the upstream engine-tarball SIGSEGV (BLOCKED-EXOGENOUS, owned by 02-05,
  see STATE.md). Our e2e step rides after it and will go green when the
  engine republish lands. Do not thrash CI.
- Do NOT use 127.0.0.1 with a different PORT as the "untrusted host" negative
  test — HostUtil matches HOSTNAMES only (port/scheme ignored, per
  wsd/ClientRequestDispatcher.cpp comment "port and scheme from wopi host
  config are currently ignored"). The negative WOPISrc must use a different
  hostname (use http://untrusted.invalid/wopi/files/x — never resolves, and
  rejection happens BEFORE any network call).
- Do NOT put `mode="groups"` anywhere without group[0] defined, and never
  define group[0] without mode="groups" (silent-rejection footgun both ways —
  Pitfall 4).
- NEVER reference port 8080 — ports in play: 8091 (wopi-host), 8092
  (fake-aoid), 9980 (coolwsd native / in-container), 9981 (docker host port
  for the e2e's own container).
- No `curl | grep -q` in the script (EPIPE/pipefail false-negative, proven in
  CI run 28959752077) — fetch to file, then grep, mirroring smoke-test.sh.
</anti_patterns>

<error_recovery>
- Probe gets `error: cmd=internal kind=unauthorized` in POSITIVE mode:
  alias_groups didn't take. Grep coolwsd's log (/tmp/coolwsd.log native;
  `docker logs` docker mode) for "alias" / "Admins did not set the
  alias_groups mode" — check the [@mode] override spelling first (Pitfall 4).
- Probe times out with no status: and wopi-host log shows `proof: REJECTED`:
  compare signed-URL reconstruction — coolwsd signs the URL it REQUESTED
  (WopiBaseURL form). In docker mode that is the host.docker.internal URL.
- Probe times out and wopi-host log shows NO CheckFileInfo at all: coolwsd
  couldn't reach the WOPI host — in docker mode confirm wopi-host bound
  0.0.0.0:8091 and host.docker.internal resolves in-container
  (`docker exec … getent hosts host.docker.internal`).
- WS handshake fails (non-101): re-check the /cool/ URL encoding against
  global.js makeDocAndWopiSrcUrl — the docurl segment is query-escaped as a
  single path component; consult browser devtools against the reference
  container as ground truth if needed.
- PutFile grep times out: some builds debounce saves — retry the save command
  once after 5s; if still absent, check wopi-host log for `wopi: PutFile`
  with a non-200 status (fix the status handling, not the assertion).
- wopi-host.yml red on `go mod download` with 404/auth: GITOPS_PAT missing —
  surface to user (user_setup), do not vendor and do not commit tokens.
- CI fix cycles: diagnose from `gh run view --log-failed` BEFORE any fix
  commit; ≤4 fix cycles, then stop and record (Objective 1/2 discipline).
</error_recovery>

</embedded_context>

<context>
@.planning/PROJECT.md
@.planning/objectives/03-aoid-authentication-integration-oidc/3-RESEARCH.md
@.planning/objectives/03-aoid-authentication-integration-oidc/03-02-SUMMARY.md
@.planning/objectives/03-aoid-authentication-integration-oidc/03-03-SUMMARY.md
@.planning/STATE.md
</context>

<research_context>
- AUTH-04's "proof-key validation enabled" means: coolwsd signs
  unconditionally when a proof_key file exists (no wsd toggle — Pitfall 3);
  the WOPI host verifies and rejects (built in 03-03). This TRD's job is
  ensuring the key EXISTS in native mode (debug builds read
  DEBUG_ABSSRCDIR/proof_key) and asserting `proof: verified ok` appears in
  the e2e logs.
- Trust verification must be behavioral, not config-file inspection
  (Pitfall 4): curl/WS through coolwsd and assert accept + reject, never just
  "the XML looks right".
- The engine-tarball SIGSEGV (STATE.md blocker, 2026-07-08): build.yml is red
  at the smoke step for reasons unrelated to this objective. wopi-host.yml
  (path-filtered, no coolwsd build) is this TRD's green CI evidence; the
  build.yml e2e step's first green ride happens after the upstream republish.
- If the editor iframe were ever CSP-blocked (frame-ancestors) in later
  browser testing: coolwsd derives allowed frame ancestors from the WOPI
  host; if needed, `--o:net.frame_ancestors=http://127.0.0.1:8091` is the
  knob. Not required for the headless probe (no iframe involved) — noted for
  03-05.
</research_context>

<gotchas>
- go.mod gains `github.com/gorilla/websocket` (probe only). Keep it out of
  server packages — import it solely in cmd/e2e-probe.
- The e2e script must be runnable from the repo root
  (`cd "$(dirname "$0")/../.."` like smoke-test.sh), `set -euo pipefail`,
  fail() helper that tails BOTH /tmp/coolwsd.log (or docker logs) AND the
  wopi-host log file.
- Data dir seeding: `cp wopi-host/testdata/hello.odt "$DATA_DIR/hello.odt"`
  (mktemp -d). The fake IdP accepts any client_id/secret — use
  "edendocs-e2e" / "e2e-secret" literals.
- Cookie-jar curl: `curl -c jar -b jar -L` follows login → authorize →
  callback → return; then fetch /open?file=hello.odt to a FILE and sed/grep
  out access_token, access_token_ttl, form action (WOPISrc + urlsrc).
- Sanity assertion (Pitfall 2 regression): the extracted access_token must
  match `^[0-9a-f-]{36}$` (uuid) — i.e. NOT a JWT with dots.
- git-history gate for AUTH-04: from the first objective-3 commit onward, no
  commit touches wsd/ or browser/src/. Implement as a verify command over
  `git log --name-only`, scoped by the 03-01 SUMMARY's first commit hash.
- build.yml additions: one `actions/setup-go@v5` step (go-version '1.26',
  cache-dependency-path wopi-host/go.sum), the private-module step, and the
  e2e step — ALL appended after the verify-branding step; do not alter
  existing steps (02-05's pending rerun references this workflow file at its
  old commit, which is unaffected, but keep the diff purely additive anyway).
- MPL-2.0 headers on new .go files and the new shell script.
</gotchas>

## Test list

This TRD is the system layer (outside-in: outermost). Its "tests" are the e2e
script's ordered assertions, executed in both modes:

1. fake-aoid healthy on 8092 (discovery fetch-to-file + grep).
2. wopi-host healthy on 8091 (/healthz).
3. coolwsd healthy (native: --probe loop on 9980; docker: capabilities fetch
   on 9981).
4. OIDC flow completes headlessly: cookie-jar curl lands on /open with a
   session (page contains a form with access_token).
5. Minted token is an opaque uuid, not a JWT (Pitfall 2 guard).
6. POSITIVE trust: e2e-probe connects, sends coolclient+load, receives
   `status:` ⇒ coolwsd accepted the WOPISrc host AND the token.
7. wopi-host log contains `wopi: CheckFileInfo … status=200`,
   `wopi: GetFile … status=200`, and `proof: verified ok` (proof-key
   validation actually exercised).
8. Forced save: probe sends `save dontTerminateEdit=1 dontSaveIfUnmodified=0`;
   wopi-host log gains `wopi: PutFile … status=200` within 60s.
9. NEGATIVE trust: probe in -expect-unauthorized mode against
   WOPISrc=http://untrusted.invalid/wopi/files/x receives an `error:` frame
   (kind unauthorized) and NO CheckFileInfo for that host appears in the
   wopi-host log ⇒ trust exists ONLY via alias_groups registration.
10. Zero-upstream-diff gate: `git status --porcelain wsd/ browser/` is empty
    and the objective's commit history touches neither tree.

<tasks>

<task type="auto">
  <name>Task 1: Full main.go wiring + e2e-probe WebSocket client</name>
  <files>wopi-host/cmd/wopi-host/main.go, wopi-host/cmd/e2e-probe/main.go, wopi-host/go.mod, wopi-host/go.sum</files>
  <action>
main.go: construct config → session.NewStore(time.Now) → storage.New →
coolwsd.New(cfg.CoolwsdURL, 1h) → oidcauth.NewAuthenticator → wopi.New →
launch.New with identityFrom bridging oidcauth's context helper:
mount GET /healthz, GET /login, GET /callback, GET /logout, /wopi/files/
(wopi handler), and launch at / and /open (wrapped so identityFrom sees
oidcauth's session). Log startup config (secrets redacted) + all 03-03
contract lines flow to stdout AND a file when EDENDOCS_WOPI_LOG_FILE is set
(the e2e greps a file, mirroring /tmp/coolwsd.log conventions). Graceful
shutdown on SIGINT/SIGTERM.

e2e-probe: `go get github.com/gorilla/websocket@latest` (import ONLY here).
Flags: -coolwsd-url (http://127.0.0.1:9980), -wopisrc, -token, -ttl,
-timeout (60s), -save (bool), -expect-unauthorized (bool).
Build docurl/wsURL and send the two handshake messages EXACTLY per the
codebase_examples shape; then read frames:
- default mode: exit 0 on first frame starting "status:"; if -save, then
  send `save dontTerminateEdit=1 dontSaveIfUnmodified=0`, keep reading until
  timeout (PutFile assertion happens in the SCRIPT via log grep), exit 0.
  Exit 1 on "error:" frame (print it) or timeout.
- -expect-unauthorized: exit 0 on an "error:" frame containing
  "unauthorized" (print it); exit 1 on "status:" or timeout.
Print every received frame's first 120 chars to stderr for diagnosability.
Commits: `feat(03-04): wire wopi-host service (all routes on 8091)` /
`feat(03-04): headless coolwsd WS e2e-probe`.
  </action>
  <verify>cd wopi-host && go vet ./... && go build ./... && go test -race ./... -count=1 && grep -q 'gorilla/websocket' go.mod && ! grep -rn 'gorilla/websocket' internal/ | grep -q .</verify>
  <done>wopi-host binary wires every package (module still fully green); e2e-probe builds with the documented flag set; gorilla/websocket confined to cmd/e2e-probe.</done>
  <recovery>If the identityFrom bridge fights the middleware shape from 03-02/03-03 SUMMARYs, adapt ONLY main.go glue — do not refactor the internal packages in this TRD.</recovery>
</task>

<task type="auto">
  <name>Task 2: wopi-host/scripts/wopi-e2e.sh — dual-mode end-to-end + local PASS</name>
  <files>wopi-host/scripts/wopi-e2e.sh</files>
  <action>
Write the orchestration script implementing test-list 1-10. Structure:
- Mode select: COOLWSD_MODE=${COOLWSD_MODE:-auto} → "native" if ./coolwsd
  exists, else "docker".
- Common: build `go build -o "$TMPD/bin/..." ./cmd/...` (all three
  binaries); seed DATA_DIR; start fake-aoid (8092) and wopi-host (8091,
  EDENDOCS_WOPI_AOID_ISSUER=http://127.0.0.1:8092, LOG_FILE set,
  REQUIRE_PROOF=true; in docker mode LISTEN_ADDR=0.0.0.0:8091 and
  EDENDOCS_WOPI_BASE_URL=http://host.docker.internal:8091); trap kills all
  PIDs, removes TMPD, removes generated proof_key{,.pub}, and
  `docker rm -f edendocs-wopi-e2e` in docker mode.
- native: repeat smoke-test.sh's setcap + systemplate bootstrap; generate
  ./proof_key if absent (ssh-keygen per codebase_examples — NEVER committed);
  launch coolwsd with the three alias_groups --o: overrides + the standard
  flags; readiness via ./coolwsd --probe loop.
- docker: spin edendocs-wopi-e2e on 127.0.0.1:9981 with aliasgroup1 env per
  codebase_examples; readiness via capabilities fetch-to-file loop (≤120s).
- Assertions in test-list order, every check fetch-to-file + grep, every
  failure through fail() (tails coolwsd log, wopi-host log, docker logs as
  applicable). Extract form fields with grep -o/sed from the /open HTML.
- Zero-upstream-diff gate: `[ -z "$(git status --porcelain wsd/ browser/)" ]`.
- Final line: `echo "WOPI E2E PASSED: OIDC → launch → CheckFileInfo/GetFile/PutFile → proof + alias_groups trust (mode=$MODE)"`.
Then RUN it locally in docker mode until green:
`COOLWSD_MODE=docker ./wopi-host/scripts/wopi-e2e.sh` — this local PASS is the
verification of record while build.yml is engine-blocked. Capture the full
output for the SUMMARY.
Commit: `feat(03-04): dual-mode WOPI e2e — OIDC round trip + coolwsd trust proofs`.
  </action>
  <verify>bash -n wopi-host/scripts/wopi-e2e.sh && COOLWSD_MODE=docker ./wopi-host/scripts/wopi-e2e.sh 2>&1 | tee /tmp/wopi-e2e-local.txt | tail -5 && grep -q 'WOPI E2E PASSED' /tmp/wopi-e2e-local.txt && ! grep -n '8080' wopi-host/scripts/wopi-e2e.sh | grep -vi 'never\|banned' | grep -q . && ! git status --porcelain | grep -E '^.. (wsd/|browser/|proof_key)' | grep -q .</verify>
  <done>Local docker-mode run prints WOPI E2E PASSED with all 10 assertions (positive AND negative trust, proof verified, PutFile round trip); no proof_key or upstream-tree residue in git status.</done>
  <recovery>Work the error_recovery ladder in order (unauthorized → alias config; proof REJECTED → URL reconstruction; no CheckFileInfo → reachability; WS non-101 → URL encoding). If docker networking on this mac blocks host.docker.internal, document and fall back to running the container with --add-host=host.docker.internal:host-gateway.</recovery>
</task>

<task type="auto">
  <name>Task 3: CI wiring — wopi-host.yml green + additive build.yml e2e step</name>
  <files>.github/workflows/wopi-host.yml, .github/workflows/build.yml</files>
  <action>
1. .github/workflows/wopi-host.yml: on push to eden-main +
   workflow_dispatch, paths: [wopi-host/**, wopi-host/scripts/wopi-e2e.sh,
   .github/workflows/wopi-host.yml]; job: ubuntu-latest,
   actions/checkout@v4, actions/setup-go@v5 (go-version '1.26',
   cache-dependency-path: wopi-host/go.sum), the VERBATIM private-module
   step from codebase_examples, then working-directory wopi-host:
   `go mod download`, `go vet ./...`, `go build ./...`,
   `go test -race ./... -count=1`.
2. build.yml — APPEND after the verify-branding step (purely additive):
   setup-go (same pins), the private-module step, and
   `- name: WOPI host end-to-end (AUTH-01..04, native coolwsd on 9980 + wopi-host on 8091)`
   running `./wopi-host/scripts/wopi-e2e.sh` with COOLWSD_MODE=native.
3. Ensure GITOPS_PAT exists on the repo (user_setup). If the user provides
   the token value in-session: `gh secret set GITOPS_PAT --repo
   AO-Cyber-Systems/EdenDocs`. If not available, note it and proceed — the
   wopi-host.yml warn-branch surfaces it loudly.
4. Push and watch `gh run watch` for the wopi-host workflow — it MUST go
   green (it never builds coolwsd, so the engine blocker cannot affect it).
   build.yml is EXPECTED red at its pre-existing smoke step (engine SIGSEGV,
   BLOCKED-EXOGENOUS, 02-05's resume action) — confirm from the failed-run
   log that the failure is the smoke step (before our new step), then STOP:
   that redness is not ours to fix. Record run IDs + the note that the e2e
   step's first green ride lands with the engine republish.
Commits: `ci(03-04): wopi-host workflow (vet/build/test, private-module fetch)` /
`ci(03-04): append WOPI e2e step to build pipeline`.
  </action>
  <verify>gh run list --workflow=wopi-host.yml --branch eden-main --limit 1 --json conclusion --jq '.[0].conclusion' | grep -q success && git diff HEAD~2 -- .github/workflows/build.yml | grep '^-' | grep -v '^---' | (! grep -q .) && ! git log --name-only --pretty=format: 04703653ab2..HEAD | grep -E '^(wsd|browser/src)/' | grep -q .</verify>
  <done>wopi-host.yml is green on eden-main; build.yml diff is purely additive (no existing lines removed/modified); build.yml redness (if any) is confirmed to be the pre-existing engine smoke failure, documented in the SUMMARY with run IDs.</done>
  <recovery>If wopi-host.yml fails on the private module: GITOPS_PAT gate (user_setup) — surface and stop rather than vendoring. ≤4 fix cycles on anything else; diagnose from logs before each fix commit.</recovery>
</task>

</tasks>

<validation_gates>
<lint>cd wopi-host && go vet ./... && bash -n scripts/wopi-e2e.sh</lint>
<test>cd wopi-host && go test -race ./... -count=1</test>
<build>gh run list --workflow=wopi-host.yml --branch eden-main --limit 1 --json conclusion --jq '.[0].conclusion' | grep -q success</build>
</validation_gates>

<verification>
- `COOLWSD_MODE=docker ./wopi-host/scripts/wopi-e2e.sh` prints WOPI E2E PASSED
  locally (verification of record while the engine blocker holds).
- Positive AND negative trust proofs both asserted in the same run
  (alias_groups is the only reason coolwsd talks to us — AUTH-04).
- wopi-host log evidence: CheckFileInfo/GetFile/PutFile all status=200 with
  `proof: verified ok`.
- wopi-host.yml green on eden-main; build.yml changes purely additive.
- `git log --name-only <first-obj3-commit>..HEAD | grep -E '^(wsd|browser/src)/'`
  returns nothing (zero OIDC code in coolwsd — AUTH-04).
</verification>

<success_criteria>
One deterministic command proves the whole AUTH chain headlessly — AOID-shaped
OIDC login, WOPI-host-minted short-lived token, coolwsd document session with
proof-verified CheckFileInfo/GetFile/PutFile callbacks, and trust that exists
if-and-only-if the host is registered in alias_groups mode="groups" — with
engine-independent CI green and the e2e wired into the main pipeline for when
the exogenous engine blocker clears.
</success_criteria>

<output>
After completion, create `.planning/objectives/03-aoid-authentication-integration-oidc/03-04-SUMMARY.md`
including: the local WOPI E2E PASSED transcript (both trust proofs + log-grep
evidence), wopi-host.yml run ID/URL, build.yml run status with the
engine-blocker note (+ resume linkage to 02-05's rerun action), and the
zero-upstream-diff gate output.
</output>
