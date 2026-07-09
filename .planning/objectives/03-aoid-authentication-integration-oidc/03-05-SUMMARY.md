---
objective: 03-aoid-authentication-integration-oidc
trd: 03-05
subsystem: wopi-host
tags: [oidc, aoid, wopi, runbook, human-verify, federation, tls, private-ca]
requirements: [AUTH-01, AUTH-03]
requirements-completed: [AUTH-01, AUTH-03]
dependency-graph:
  requires:
    - 03-01..03-04 (complete wopi-host stack + dual-mode e2e)
  provides:
    - docs/eden/WOPI-HOST.md operational runbook (architecture, env reference, AOID CreateClient registration, alias_groups XML, v1 limitations register)
    - wopi-host/README.md quickstart
    - EDENDOCS_WOPI_OIDC_CA_FILE (private-CA trust for outbound OIDC TLS; no skip-verify path)
    - Human-verified real-AOID end-to-end record (AUTH-03 closed)
  affects:
    - any future production AOID deployment of the WOPI host (runbook is the contract)
tech-stack:
  added: []
  patterns:
    - Private-CA trust via oidc.ClientContext-injected http.Client (discovery + JWKS + token exchange ride one CA-pooled client; verification never disabled)
    - Checkpoint stack-state convention — .checkpoint-05/ holds env files (600, git-ignored), PIDs, logs, launcher + resume scripts
key-files:
  created:
    - wopi-host/internal/oidcauth/oidcauth_cafile_test.go
  modified:
    - wopi-host/internal/config/config.go (OIDCCAFile field)
    - wopi-host/internal/oidcauth/oidcauth.go (newHTTPClientForCA + ClientContext wiring)
    - docs/eden/WOPI-HOST.md (env var row, AOID-side blocker + resolution notes, verified-against-real-AOID record)
decisions:
  - "TLS trust for AOID's self-signed dev CA = explicit CA-bundle env var (EDENDOCS_WOPI_OIDC_CA_FILE), NOT InsecureSkipVerify — adds a root, never disables verification; macOS SSL_CERT_FILE is unreliable for Go and keychain edits are invasive"
  - "wopi-host ran on 8093 for this checkpoint ONLY because 8091 was occupied by an unrelated aofamily dev server; code default remains 127.0.0.1:8091 and the registered redirect_uri was scoped to this run"
  - "AOID login rode federation through fake-aoid registered as a real external IdP (user-authorized scaffolding, disclosed at checkpoint): AOID's login surface is federation-only and no first-credential seeding path exists — the AOID<->wopi-host leg (session, code, token exchange, JWKS) was fully real"
  - "User-authorized one-method fix in the aoid repo (9d53d31): federationAdminKMSAdapter.Encrypt delegated to crypto.WrapCipher via optional-interface assertion — unblocked CreateExternalIdP on wrap-key deployments"
metrics:
  duration: ~3h across three executor turns (incl. AOID stack bring-up + aoid-repo fix)
  tasks: 2/2
  completed: 2026-07-09
---

# Objective 03 TRD 05: Real-AOID Verification Summary

**Human-verified the full real-AOID OIDC flow in a browser (login redirect → editor iframe → "Eden Test User" identity rendered → clean save), against a local dev AOID at issuer `https://localhost:8443`, with the runbook, a private-CA trust env var, and the registration bootstrap all landed.**

## Performance

- **Duration:** ~3h wall across three executor turns (runbook was Task 1, prior turn)
- **Completed:** 2026-07-09
- **Tasks:** 2/2 (Task 1 runbook; Task 2 checkpoint:human-verify)
- **Files modified:** 4 in EdenDocs (+2 in aoid, committed there)

## Accomplishments

- **Checkpoint outcome: PASS — human-verified 2026-07-09.** The operator confirmed all four items: (1) `http://127.0.0.1:8093/open?file=hello.odt` redirected to the real AOID login, (2) the editor iframe loaded hello.odt, (3) the AOID display name **"Eden Test User"** appeared in the editor (generic avatar icon as expected — no AOID avatar claim), (4) a typed change saved with no error (PutFile round trip).
- **AOID instance:** local dev, native binary from `~/dev/aoid` (branch `fix/oidc-idtoken-email-claims`), FIPS build (`GOFIPS140=v1.0.0`), software signing tier (`softkey://`, `AOID_KMS_WRAP_KEY`), issuer `https://localhost:8443`, Postgres in docker on 5434.
- **Client registration (per the §4 runbook, real admin RPCs over mTLS `aoid-superadmin` fixture cert — no SQL inserts):** tenant `edendocs` → `OAuthAdminService.CreateClient` → **client_id `UeeboUCigLiyYItuyMeXVw`** (confidential, `client_secret_basic`, `authorization_code`, redirect_uri `http://127.0.0.1:8093/callback`, auto_consent). **Secret NOT recorded** — captured once into a git-ignored local env file (`.checkpoint-05/wopi-host.env`, mode 600), never printed or committed.
- **New config surface:** `EDENDOCS_WOPI_OIDC_CA_FILE` — PEM CA bundle for outbound OIDC TLS (discovery/JWKS/token exchange). Tested with a full login flow against an httptest TLS IdP, an unknown-authority control, and CA-file error cases.
- **Headless pre-verification:** the complete 11-hop chain (wopi-host → AOID authorize → federation → JIT provisioning → auto-consent code → token exchange → session → launch page with opaque WOPI token) was driven headlessly and recorded in `.checkpoint-05/bootstrap/redirect-chain.txt` before the human run.

## Fake-IdP ↔ Real-AOID deltas discovered

1. **AOID's browser login surface is federation-only** — `/login` is an IdP picker; password/WebAuthn RPCs exist but there is no admin/dev path to seed a first credential (ProvisionAccount carries no credential; ChangePassword/WebAuthn enrollment need an existing session). A local dev AOID therefore needs an upstream IdP registered; this run used the repo's fake-aoid (canned "Eden Test User", auto-approves) JIT-provisioned into the real AOID — disclosed to and accepted by the operator.
2. **AOID v1.0 boot had federation-admin KMS Encrypt deliberately unwired**, hard-blocking `CreateExternalIdP` with an `oidc_client_secret`. Fixed with user authorization in the **aoid repo: commit `9d53d31` on `fix/oidc-idtoken-email-claims` (local, unpushed)** — one-method delegation from `federationAdminKMSAdapter.Encrypt` to the existing `crypto.WrapCipher.Encrypt`, with round-trip + degradation tests.
3. **`tenants.federation_return_url_allowlist` has no admin-RPC surface** (empty allowlist rejects every federation login with `invalid_return_url`); set via the same direct `UPDATE` AOID's own `cmd/aoid/dev_seed_e2e.go` issues.
4. **AOID enforces a FIPS build gate** (`GOFIPS140` baked into the binary) — plain `go build` binaries refuse to boot.
5. **Port note:** this run used **8093, not 8091**, because 8091 was occupied by an unrelated aofamily dev server. The code default (`127.0.0.1:8091`) and all docs are unchanged; only this run's registered redirect_uri used 8093.

## Task Evidence

| Task | Verify | Result | Status |
|---|---|---|---|
| 1: Runbook + README | docs review vs must_haves (prior turn) | runbook covers fake+real AOID, registration, trust, v1 limitations | PASS |
| 2 (code): CA-file trust | `go build ./... && go vet ./... && go test ./...` in wopi-host | exit 0, all packages ok | PASS |
| 2 (stack): headless chain | 11-hop curl chain + `SELECT display_name FROM aoid.accounts` | code→token→session→launch page; JIT row "Eden Test User" | PASS |
| 2 (checkpoint): human verify | operator browser walkthrough 2026-07-09 | all 4 items confirmed ("Looks good") | **PASS (human)** |

## Task Commits

1. **Task 1: runbook + README** — `bbc28d7ca63` (docs), merge `ffe5b6ddf2a`
2. **Task 2: CA-file trust** — `ef4aaca1d5b` (feat)
3. **Task 2: blocker documented** — `4c603f5f293` (docs)
4. **Task 2: blocker resolved + dev-setup notes** — `e2ed6d1933b` (docs)
5. **aoid repo (authorized):** `9d53d31` — `fix(boot): wire federation-admin KMS Encrypt to the chassis wrap cipher` (branch `fix/oidc-idtoken-email-claims`, local/unpushed)

## Post-TRD Verification

- Auto-fix cycles used: 0 (deviations were pre-authorized user decisions, not gate failures)
- Must-haves verified: 3/3 (runbook completeness; human has SEEN the real flow — recorded; v1 limitations documented)
- Gate failures: None

## Deviations from Plan

- **[User-authorized] aoid repo fix `9d53d31`** — Rule-4 stop was raised for the unwired KMS Encrypt shim; user replied "fix aoid". Scoped to one adapter method + tests; in-flight work in that repo untouched.
- **[Rule 3 - blocking] wopi-host port 8093 for this run** — 8091 occupied by an unrelated local dev server.
- **[Rule 3 - blocking] fresh Postgres volume** — inconsistent `schema_migrations` state (version=3 dirty) after the AOID binary swap; dev DB wiped and the entire bootstrap re-run from scratch on the fixed binary.
- **[Rule 3 - blocking] coolwsd proof-key inotify restart** — generating the proof key post-boot stops coolwsd by design; `docker start` with the persisted key resolved it.
- **[Disclosed scaffolding] fake-aoid as upstream federation IdP + `federation_return_url_allowlist` UPDATE** — both disclosed in the checkpoint report and accepted by the operator with approval.

## Stack / teardown

The verification stack (AOID 8443, fake-aoid 8092, wopi-host 8093, coolwsd 9981, Postgres 5434) is left running; PIDs, env files, evidence, and teardown commands live in `.planning/objectives/03-aoid-authentication-integration-oidc/.checkpoint-05/README.md` (secrets git-ignored, never committed).

## Self-Check: PASSED

- docs/eden/WOPI-HOST.md verified-line present: FOUND
- wopi-host/internal/oidcauth/oidcauth_cafile_test.go: FOUND
- Commits `bbc28d7ca63`, `ef4aaca1d5b`, `4c603f5f293`, `e2ed6d1933b` in EdenDocs history: FOUND
- Commit `9d53d31` in aoid history: FOUND
