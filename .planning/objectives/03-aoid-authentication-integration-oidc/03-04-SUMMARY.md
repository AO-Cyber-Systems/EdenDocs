---
objective: 03-aoid-authentication-integration-oidc
trd: 03-04
subsystem: wopi-host
tags: [oidc, wopi, coolwsd, e2e, alias-groups, proof-key, ci]
requirements: [AUTH-01, AUTH-02, AUTH-04]
dependency-graph:
  requires:
    - 03-02 (oidcauth relying party + fake-aoid IdP)
    - 03-03 (wopi/launch/coolwsd/proof packages + log-line contract)
  provides:
    - Fully wired cmd/wopi-host binary (all routes)
    - Headless coolwsd WS client (cmd/e2e-probe)
    - Dual-mode behavioral e2e (wopi-host/scripts/wopi-e2e.sh)
    - Engine-independent wopi-host CI + additive build.yml e2e step
  affects:
    - 03-05 (real-AOID verification reuses the same script/stack)
tech-stack:
  added:
    - github.com/gorilla/websocket v1.5.4-0.20250319132907 (cmd/e2e-probe ONLY)
  patterns:
    - identityFrom seam closure (oidcauth.FromContext -> launch.New)
    - Origin-header browser emulation for coolwsd WS upgrades
    - IdP-first deterministic startup ordering (start -> wait -> start -> wait)
key-files:
  created:
    - wopi-host/cmd/e2e-probe/main.go
    - wopi-host/scripts/wopi-e2e.sh
    - .github/workflows/wopi-host.yml
  modified:
    - wopi-host/cmd/wopi-host/main.go (full rewrite: route wiring)
    - wopi-host/go.mod, wopi-host/go.sum (gorilla/websocket)
    - .github/workflows/build.yml (additive steps only)
decisions:
  - "e2e-probe sends Origin: <coolwsd origin> on the WS dial — coolwsd's allowedOrigin() 403s upgrades with no Origin header; in real usage cool.html is served BY coolwsd so the browser Origin IS coolwsd's own origin (faithful emulation, not a bypass)"
  - "WOPI_HOST_PORT env override (default 8091) in wopi-e2e.sh — CI/default behavior unchanged; exists solely to dodge unrelated local listeners on shared dev machines"
  - "PutFile confirmation is the script's log-grep job, not the probe's — probe phase 2 treats post-save timeout/disconnect as success"
metrics:
  duration: ~2h (incl. one continuation across turn-budget boundary)
  tasks: 3/3
  completed: 2026-07-08
---

# Objective 03 TRD 04: Wiring + Trust E2E Summary

One deterministic command (`wopi-host/scripts/wopi-e2e.sh`) now behaviorally proves the whole AUTH chain headlessly — AOID-shaped OIDC login via fake-aoid, WOPI-host-minted opaque uuid token, real coolwsd document session, proof-verified CheckFileInfo/GetFile/PutFile callbacks, and alias_groups-only trust (positive AND negative) — passing locally in docker mode with CI wired for both the path-filtered wopi-host workflow and the main build pipeline.

## What Was Built

**Task 1 — Service wiring (`a2520e12e83`, `1e6f792cd01`):**
- `cmd/wopi-host/main.go`: full route map on one `http.ServeMux` — `GET /healthz`, `/login`, `/callback`, `/logout` (oidcauth), `/wopi/` (token-gated internally, NOT RequireLogin-wrapped), `/` + `/open` (cookie-gated via `authr.RequireLogin(launchHandler)`). The wave-2 decoupling seam closes here: `identityFrom := func(r) { return oidcauth.FromContext(r.Context()) }` injected into `launch.New`. Graceful shutdown, `EDENDOCS_WOPI_LOG_FILE` MultiWriter logging for the e2e log-grep contract. oidcauth init failure is FATAL (`log.Fatalf`) — no half-alive service can pass /healthz.
- `cmd/e2e-probe/main.go`: headless coolwsd WebSocket client mirroring `browser/js/global.js` makeDocAndWopiSrcUrl + `socket.ts` `_onSocketOpen` (`coolclient 0.1` hello → `load url=`). Two-phase read loop: phase 1 hard pass/fail on `status:`/`error:` frames (`-expect-unauthorized` inverts), phase 2 (`-save`) sends `save dontTerminateEdit=1 dontSaveIfUnmodified=0` and drains diagnostically — PutFile confirmation is the script's log grep. gorilla/websocket imported ONLY here.

**Task 2 — Dual-mode e2e (`0cfcd849a44`, `1188370fe27`):**
- `wopi-host/scripts/wopi-e2e.sh`: native (CI, coolwsd 9980, `--o:storage.wopi.alias_groups[@mode]=groups` + `group[0].host` overrides, ephemeral ssh-keygen proof_key) / docker (local, ephemeral `edendocs-wopi-e2e` container on 9981, `aliasgroup1` env, `coolconfig generate-proof-key`) modes. Deterministic startup ordering: coolwsd → fake-aoid (wait ready) → wopi-host (wait ready) → 10 ordered assertions. `fail()` tails both wopi-host and coolwsd logs. Fetch-to-file-then-grep throughout (no `curl | grep`).

**Task 3 — CI wiring (`840d7be73db`, `44926c18572`):**
- `.github/workflows/wopi-host.yml`: path-filtered (wopi-host/**), setup-go 1.26 with `cache-dependency-path: wopi-host/go.sum`, VERBATIM aofamily private-module step (warn branch when GITOPS_PAT absent), `go mod download` / `vet` / `build` / `test -race ./... -count=1` in `wopi-host/`.
- `.github/workflows/build.yml`: purely additive steps appended AFTER verify-branding — setup-go, private-module step, `COOLWSD_MODE=native ./wopi-host/scripts/wopi-e2e.sh`. Verified zero removed/modified existing lines (`git diff HEAD~1 -- .github/workflows/build.yml | grep '^-'` → empty).

## Local E2E Evidence (verification of record)

`WOPI_HOST_PORT=18091 COOLWSD_MODE=docker ./wopi-host/scripts/wopi-e2e.sh` (full transcript captured at /tmp/wopi-e2e-local.txt during the run; port override local-only, see Deviations):

```
=== WOPI E2E: mode=docker ===
--- [1/10] fake-aoid healthy on 8092 ---
--- [2/10] wopi-host healthy on 18091 ---
--- [3/10] coolwsd healthy (docker) ---
--- [4/10] OIDC round trip -> /open lands with a session ---
--- [5/10] minted token is an opaque uuid, not a JWT (Pitfall 2 guard) ---
--- [6-8/10] POSITIVE trust: coolclient load -> CheckFileInfo/GetFile/proof -> forced PutFile ---
e2e-probe: PASS — got status: frame
e2e-probe: sent forced save command; draining frames until timeout for PutFile to land
--- [9/10] NEGATIVE trust: untrusted WOPISrc host rejected ---
e2e-probe: recv: error: cmd=internal kind=unauthorized
e2e-probe: PASS — got expected unauthorized error frame: error: cmd=internal kind=unauthorized
--- [10/10] zero-upstream-diff gate (wsd/, browser/) ---
WOPI E2E PASSED: OIDC -> launch -> CheckFileInfo/GetFile/PutFile -> proof + alias_groups trust (mode=docker)
```

- **Positive trust proof:** coolwsd (fresh `collabora/code:latest` container, trusted host registered ONLY via `aliasgroup1`) accepted the WOPISrc, called back CheckFileInfo + GetFile with valid X-WOPI-Proof signatures, and the forced save round-tripped through PutFile — all four log-grep gates (`wopi: CheckFileInfo…status=200`, `wopi: GetFile…status=200`, `proof: verified ok`, `wopi: PutFile…status=200`) passed inside the script (a miss on any one is a hard `fail()`).
- **Negative trust proof:** the SAME coolwsd instance rejected `http://untrusted.invalid/wopi/files/x` with `error: cmd=internal kind=unauthorized` BEFORE any network callback (wopi-host log verified free of `untrusted.invalid`).
- **Token shape guard:** minted access_token matched `^[0-9a-f-]{36}$` (opaque uuid, not a JWT).

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: wire wopi-host + e2e-probe | `go vet ./... && go build ./... && go test -race ./... -count=1` + gorilla-confinement greps | 0 | PASS |
| 2: dual-mode e2e passes locally | `WOPI_HOST_PORT=18091 COOLWSD_MODE=docker ./wopi-host/scripts/wopi-e2e.sh` → `WOPI E2E PASSED` | 0 | PASS |
| 3: CI wiring | YAML parse OK; `git diff … build.yml \| grep '^-'` empty (additive); run-green check DEFERRED-TO-ORCHESTRATOR | 0 | PASS (local checks) |

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| lint | `cd wopi-host && go vet ./... && bash -n scripts/wopi-e2e.sh` | 0 | PASS |
| test | `cd wopi-host && go test -race ./... -count=1` | 0 | PASS (all 7 packages) |
| build | `gh run list --workflow=wopi-host.yml …` | n/a | **DEFERRED-TO-ORCHESTRATOR** (no pushes from this executor; orchestrator merges/pushes/watches) |

Zero-upstream-diff gate (AUTH-04): `git log --name-only --pretty=format: 04703653ab2..HEAD | grep -E '^(wsd|browser/src)/'` → **empty** (no objective-3 commit touches wsd/ or browser/src).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] e2e-probe WS dial rejected 403 by coolwsd's allowedOrigin()**
- **Found during:** Task 2, first probe dial (handshake 403 Forbidden)
- **Issue:** `wsd/ClientRequestDispatcher.cpp` `allowedOrigin()` rejects any WS upgrade with no `Origin` header; gorilla/websocket sends none by default. (The local fork has a WOPI-token Origin bypass, but the docker container runs upstream 26.04 without it.)
- **Fix:** probe now derives `Origin: <scheme>://<host>` from `-coolwsd-url` and sends it on the dial — faithful emulation of the real browser flow, where cool.html is served BY coolwsd so the page Origin IS coolwsd's origin.
- **Files:** wopi-host/cmd/e2e-probe/main.go — **Commit:** `0cfcd849a44`

**2. [Rule 1 - Bug] Startup race: wopi-host fatal before fake-aoid listened**
- **Found during:** Task 2, first e2e run (step [4/10] 404; wopi-host log: `oidcauth init: … dial tcp 127.0.0.1:8092: connection refused`)
- **Issue:** script started fake-aoid and wopi-host back-to-back; wopi-host's single, non-retried OIDC discovery at startup lost the race and (correctly) log.Fatal'd.
- **Fix:** deterministic ordering — start fake-aoid → wait for `.well-known/openid-configuration` → start wopi-host → wait for `/healthz` — before any test step. main.go needed no change (init failure was already fatal by design).
- **Files:** wopi-host/scripts/wopi-e2e.sh — **Commit:** `1188370fe27`

**3. [Rule 3 - Blocking] Port 8091 occupied locally by an unrelated long-running dev server**
- **Found during:** Task 2 diagnosis — an unrelated `aofamily-ai` `go run ./cmd/server` (PID 82513, up since 2026-07-07) was bound to 8091, silently answering the script's healthz probe and masking the real crash. It is a live, intentional process for a parallel project — NOT killed.
- **Fix:** `WOPI_HOST_PORT` env override in the script (default **8091**, byte-identical CI behavior; the port plan 8091/8092/9980/9981 is unchanged). Local run used `WOPI_HOST_PORT=18091`.
- **Files:** wopi-host/scripts/wopi-e2e.sh — **Commit:** `1188370fe27`

## User Setup Required — GITOPS_PAT (LOUD)

**The `GITOPS_PAT` Actions secret is ABSENT on AO-Cyber-Systems/EdenDocs** (`gh secret list` showed no repo-level secrets). Until it is set, `.github/workflows/wopi-host.yml` and the new build.yml steps will fail at `go mod download` on the private `eden-platform-go` module — the workflow's warn branch surfaces this explicitly ("WARN: GITOPS_PAT secret not set").

**Action for the user:**
```
gh secret set GITOPS_PAT --repo AO-Cyber-Systems/EdenDocs
```
with a PAT that has read access to `AO-Cyber-Systems/eden-platform-go` (same secret aofamily CI uses). No token was extracted from local config, vendored, or committed by this executor.

## CI Delegation Notes (for the orchestrator)

- **No pushes were made from this executor** (per hard constraint). The Task 3 `<verify>`'s run-green check and the `<build>` validation gate are **DEFERRED-TO-ORCHESTRATOR**: after merge/push, watch the `wopi-host` workflow — it must go green once GITOPS_PAT is set (it never builds coolwsd; the engine blocker cannot affect it).
- **build.yml is EXPECTED red** at its pre-existing engine smoke step (upstream engine-tarball SIGSEGV — BLOCKED-EXOGENOUS, owned by Objective 2's 02-05 rerun action, see STATE.md). Our appended e2e step rides after it; its first green happens when the engine republish lands. Do not thrash CI over that redness.
- Local docker-mode e2e is the **verification of record** while the engine blocker holds (per TRD `<verification>`).

## Orchestrator CI Evidence (post-merge push, 2026-07-09)

- Pushed eden-main `7b0ec91787c..f132f891102` to origin.
- **wopi-host.yml run 28991066118: FAILURE at "Download modules"** — exact signature of the documented GITOPS_PAT gap (`git ls-remote https://github.com/aocybersystems/eden-platform-go … exit status 128`; the warn branch printed "WARN: GITOPS_PAT secret not set"). No code defect: the same module graph vets/builds/tests green locally. The `<build>` gate remains **OPEN pending user setup** — rerun via `gh run rerun 28991066118 --repo AO-Cyber-Systems/EdenDocs` (or push) after `gh secret set GITOPS_PAT --repo AO-Cyber-Systems/EdenDocs`.
- build.yml run 28991066192: triggered by the same push; expected red at the pre-existing engine smoke step (02-05's BLOCKED-EXOGENOUS SIGSEGV). Recorded separately when complete.

## Self-Check: PASSED

- Files exist: wopi-host/cmd/wopi-host/main.go, wopi-host/cmd/e2e-probe/main.go, wopi-host/scripts/wopi-e2e.sh (executable), .github/workflows/wopi-host.yml, build.yml additive block — all FOUND.
- Commits exist: a2520e12e83, 1e6f792cd01, 0cfcd849a44, 1188370fe27, 840d7be73db, 44926c18572 — all FOUND in git log.
- No proof_key / e2e-probe binary / upstream-tree residue in `git status`.
