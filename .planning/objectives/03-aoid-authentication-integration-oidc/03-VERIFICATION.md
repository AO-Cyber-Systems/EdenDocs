---
objective: 03-aoid-authentication-integration-oidc
verified: 2026-07-09T04:05:00Z
status: passed
score: 19/19 must-haves verified (human checkpoint confirmed 2026-07-09)
human_verification:
  - test: "Visit http://127.0.0.1:8091/open?file=hello.odt with wopi-host configured against a REAL AOID issuer (registered OAuth client, redirect_uri http://127.0.0.1:8091/callback)"
    expected: "Browser redirects to the real AOID login page (Authorization Code + PKCE flow)"
    why_human: "TRD 03-05 checkpoint:human-verify — requires a reachable AOID instance + admin CreateClient registration; headless e2e proved the identical code path against the fake AOID-shaped IdP only"
  - test: "Complete AOID login and observe the launch page"
    expected: "Editor iframe loads hello.odt via the hidden auto-submitting POST form"
    why_human: "Real-IdP round trip + visual iframe render cannot be grepped; fake-IdP equivalent passed headlessly"
  - test: "Inspect the editor's user list / avatar area"
    expected: "The AOID account's display name appears (generic avatar icon is EXPECTED — AOID has no avatar claim; documented v1 limitation)"
    why_human: "AUTH-03's final proof is visual identity rendering inside the editor (success criterion 3); CheckFileInfo emits UserFriendlyName correctly per unit tests + e2e, but a human eye must confirm the render"
  - test: "Type a change and save"
    expected: "No error surfaces; PutFile round-trips against the real session"
    why_human: "Confirms the full real-AOID session end to end (headless PutFile proof exists for the fake-IdP path)"
notes:
  - kind: external_setup
    note: "GITOPS_PAT Actions secret absent on AO-Cyber-Systems/EdenDocs — wopi-host.yml run 28991414576 red at 'Download modules' (git ls-remote exit 128 on private eden-platform-go). Exact documented signature; same module graph vets/builds/tests green locally. Fix: gh secret set GITOPS_PAT --repo AO-Cyber-Systems/EdenDocs, then rerun. Not a code gap."
  - kind: exogenous_blocker
    note: "build.yml red at the pre-existing engine smoke step (Objective 2 / 02-05 BLOCKED-EXOGENOUS engine-tarball SIGSEGV). The additive wopi-host e2e step rides after it; not this objective's gap."
  - kind: requirements_bookkeeping
    note: "REQUIREMENTS.md marks AUTH-01/02/04 complete and AUTH-03 pending — consistent with the open 03-05 checkpoint. On checkpoint PASS, mark AUTH-03 complete and append the 'Verified against real AOID on <date>' line to docs/eden/WOPI-HOST.md Quickstart."
---

# Objective 3: AOID Authentication Integration (OIDC) Verification Report

**Objective Goal:** Users authenticate against AOID before opening a document, via a new reference WOPI host service (outside `wsd/`/`browser/`), and their AOID identity is visible inside the editor — with no OIDC code added to coolwsd itself.
**Verified:** 2026-07-09T04:05:00Z
**Status:** passed
**Re-verification:** No — initial verification

**Human checkpoint resolution (2026-07-09):** all 4 `human_verification`
items were confirmed by the operator against a real local-dev AOID (issuer
`https://localhost:8443`, wopi-host on 8093 for this run — 8091 was locally
occupied): real-AOID login redirect, editor iframe loaded hello.odt, AOID
display name "Eden Test User" rendered in the editor, save round-tripped
without error. Score: **19/19**. AUTH-03 marked complete per the
`requirements_bookkeeping` note; details + deltas in `03-05-SUMMARY.md`.

## Goal Achievement

### Success Criteria (ROADMAP.md — the contract)

| # | Criterion | Status | Evidence |
|---|-----------|--------|----------|
| 1 | Launch page → AOID OIDC Authorization Code flow → embedded editor iframe | ✓ VERIFIED (headless, fake-IdP) / real-AOID walkthrough → human item | oidcauth via `eden-platform-go/platform/oidcrp` (PKCE S256); independent e2e re-run step [4/10] "OIDC round trip -> /open lands with a session" PASSED; launch page renders hidden auto-submitting POST form + iframe (launch.go:141-146) |
| 2 | CheckFileInfo/GetFile/PutFile + short-lived WOPI access_tokens coolwsd accepts | ✓ VERIFIED | Independent docker e2e re-run 2026-07-09 (this verifier): coolwsd (collabora/code:latest) opened a real document session; CheckFileInfo/GetFile/PutFile status=200 log-greps + "proof: verified ok" all gated inside the script; token matched `^[0-9a-f-]{36}$` (opaque uuid, 15m TTL default) |
| 3 | Editor UI displays name/avatar/admin flag from CheckFileInfo | ? HUMAN NEEDED (by design) | Code-side complete: `UserFriendlyName` (Name→Email→"EdenDocs User" fallback, never empty), top-level `IsAdminUser` (v1 hard-false, documented), `UserExtraInfo` omitted (no AOID avatar claim → browser generic fallback, documented v1 limitation). Visual render = TRD 03-05 open checkpoint |
| 4 | Trust ONLY via `storage.wopi.alias_groups mode="groups"` + proof-key validation; no OIDC in `wsd/` | ✓ VERIFIED | E2e positive proof (trusted host loads) AND negative proof (`untrusted.invalid` → `error: cmd=internal kind=unauthorized`) in my re-run; `git log 04703653ab2..HEAD` touches zero `wsd/`/`browser/src` files; `wsd/` oidc/oauth grep hits are only the pre-existing upstream `noAuthHeader` identifier (substring false positive) |

### Observable Truths (aggregated TRD must_haves: 19)

| TRD | Truths | Status |
|-----|--------|--------|
| 03-01 | Additive module / opaque short-lived tokens / storage traversal guard / vet+race green | 4/4 ✓ (suite re-run green: 7 packages ok under `-race -count=1`) |
| 03-02 | 302→PKCE flow via oidcrp / claim mapping (sub, name→preferred_username→email, IsAdmin=false) / tamper rejection / fake-aoid on 8092 | 4/4 ✓ (oidcauth tests green; `oidcrp.ExchangeAndVerify` present; fake-aoid bans 8080, binds 8092) |
| 03-03 | Exact CheckFileInfo contract / GetFile+PutFile w/ 401 gate / X-WOPI-Proof(/Old) RSA-SHA256 verify / POST-form launch page | 4/4 ✓ (wopi.go:49-65,207-224; proof.go VerifyPKCS1v15; launch.go POST form + access_token_ttl=UnixMilli) |
| 03-04 | One-command headless e2e / alias_groups-only trust positive+negative / zero wsd//browser/src commits / path-filtered CI | 4/4 ✓ (e2e independently re-run PASSED; git purity gate empty; wopi-host.yml exists — red only on missing GITOPS_PAT secret, external) |
| 03-05 | Runbook complete / **human has SEEN real flow** / v1 limitations documented | 2/3 — human-verify checkpoint OPEN (recorded pending by design; commit 7a1d139c075) |

**Score:** 18/19 truths verified; 1 pending human checkpoint.

### Required Artifacts (all three levels)

| Artifact | Exists | Substantive | Wired | Status |
|----------|--------|-------------|-------|--------|
| `wopi-host/go.mod` (nested module, eden-platform-go pin) | ✓ | ✓ | ✓ | VERIFIED |
| `wopi-host/internal/session/session.go` (MintWopiToken) | ✓ | 178 L | imported by wopi/launch/main | VERIFIED |
| `wopi-host/internal/storage/storage.go` | ✓ | 195 L | imported by wopi/launch/main | VERIFIED |
| `wopi-host/testdata/hello.odt` | ✓ | byte-identical to `test/data/hello.odt` (cmp) | e2e seed | VERIFIED |
| `wopi-host/internal/oidcauth/oidcauth.go` (oidcrp.ExchangeAndVerify) | ✓ | 319 L | main.go:105-107,115 | VERIFIED |
| `wopi-host/internal/oidctest/idp.go` + `cmd/fake-aoid/main.go` | ✓ | 355/57 L | tests + e2e script | VERIFIED |
| `wopi-host/internal/proof/proof.go` (VerifyPKCS1v15) | ✓ | 193 L | wopi middleware | VERIFIED |
| `wopi-host/internal/coolwsd/discovery.go` | ✓ | 223 L | wopi + launch | VERIFIED |
| `wopi-host/internal/wopi/wopi.go` (IsAdminUser) | ✓ | 272 L | main.go:111 | VERIFIED |
| `wopi-host/internal/launch/launch.go` (access_token_ttl) | ✓ | 207 L | main.go:115 behind RequireLogin | VERIFIED |
| `wopi-host/cmd/wopi-host/main.go` (full route wiring) | ✓ | 172 L, all routes | — | VERIFIED |
| `wopi-host/cmd/e2e-probe/main.go` (coolclient) | ✓ | 223 L | e2e script | VERIFIED |
| `wopi-host/scripts/wopi-e2e.sh` (alias_groups) | ✓ | 14 KB, executable | CI + local | VERIFIED |
| `.github/workflows/wopi-host.yml` (GOPRIVATE) | ✓ | ✓ | red on GITOPS_PAT only | VERIFIED (code) |
| `docs/eden/WOPI-HOST.md` (alias_groups, CreateClient, mode="groups") | ✓ | 348 L | — | VERIFIED |
| `wopi-host/README.md` (8091) | ✓ | 89 L | — | VERIFIED |

### Key Link Verification

All 15 declared key links pattern-verified (34/34 grep checks PASS), including the load-bearing ones: `wopi.go→LookupWopiToken` (token gate), `wopi.go→X-WOPI-Proof` (proof middleware), `launch.go→MintWopiToken`/`urlsrc`, `main.go→identityFrom` (wave-2 seam closure at main.go:95-101), `wopi-e2e.sh→alias_groups[@mode]=groups`, `e2e-probe→load url=`.

### Requirements Coverage

| Requirement | Source TRDs | Status | Evidence |
|-------------|------------|--------|----------|
| AUTH-01 | 03-01, 03-02, 03-04, 03-05 | ✓ SATISFIED (code + headless e2e; real-IdP confirmation folded into human checkpoint) | oidcrp Authorization Code + PKCE flow; e2e step [4/10] |
| AUTH-02 | 03-01, 03-03, 03-04 | ✓ SATISFIED | CheckFileInfo/GetFile/PutFile 200s + opaque uuid token in independent e2e re-run |
| AUTH-03 | 03-02, 03-03, 03-05 | ? NEEDS HUMAN (pending, by design) | Claim mapping + CheckFileInfo pass-through tested; visual render = open checkpoint. REQUIREMENTS.md correctly shows Pending |
| AUTH-04 | 03-03, 03-04 | ✓ SATISFIED | alias_groups-only trust (positive+negative e2e), RequireProof default true + "proof: verified ok", zero wsd//browser/src commits since 04703653ab2 |

No orphaned requirements: REQUIREMENTS.md maps exactly AUTH-01..04 to Objective 3; TRD frontmatter collectively claims all four.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| proof.go | 157 | "TODO" (quotes upstream coolwsd comment) | ℹ️ Info | None — explanatory |
| config.go | 138 | "placeholder" (redaction doc comment) | ℹ️ Info | None |
| wopi.go | 58 | "placeholder" (avatar-omission rationale) | ℹ️ Info | None |

No blockers, no stub implementations, no empty returns, no console-log-only handlers.

### Functional Verification (independent e2e re-run by verifier)

`COOLWSD_MODE=docker WOPI_HOST_PORT=18091 ./wopi-host/scripts/wopi-e2e.sh` → **`WOPI E2E PASSED: OIDC -> launch -> CheckFileInfo/GetFile/PutFile -> proof + alias_groups trust (mode=docker)`** (2026-07-09, this verification session — not just the SUMMARY's claim). All 10 steps passed, including the negative trust proof (`error: cmd=internal kind=unauthorized` for `untrusted.invalid`). Port 18091 override used because 8091 is squatted locally by an unrelated aofamily-ai dev server; CI default remains 8091. Port 8080 appears nowhere in the new surface except ban-enforcement code/comments (verified by grep).

### Hard Constraints

- **No port 8080:** ✓ — every occurrence in wopi-host//docs is the ban constant, validation error, or prohibition text; config refuses `:8080` at startup.
- **Zero wsd//browser/src commits since 04703653ab2:** ✓ — `git log --name-only 04703653ab2..HEAD | grep -E '^(wsd|browser/src)/'` empty.
- **Objective 2 surfaces untouched:** ✓ — no `eden-branding/` or `scripts/eden/` files in any objective-3 commit.
- **All 14 SUMMARY-claimed commit hashes exist** in git history.

### Human Verification Required

See frontmatter `human_verification` — the four TRD 03-05 checkpoint items (real-AOID login redirect, iframe loads hello.odt, AOID display name visible, save round-trip). Prerequisites the human/operator owns first: (a) reachable AOID instance, (b) OAuth client registration via AOID admin CreateClient RPC (redirect_uri `http://127.0.0.1:8091/callback`, one-time secret capture), (c) optionally `gh secret set GITOPS_PAT --repo AO-Cyber-Systems/EdenDocs` to green the wopi-host CI workflow.

### Gaps Summary

No code gaps. Everything automatable is verified — the full test suite is green under `-race`, all 16 artifacts pass exists/substantive/wired, all 34 must-have pattern checks pass, and the dual-mode e2e was independently re-run and passed in this session. The single open item is the designed TRD 03-05 `checkpoint:human-verify` (real-AOID browser walkthrough → AUTH-03 / success criteria 1+3 final proof), plus two external-environment items (GITOPS_PAT secret; Objective 2's exogenous engine blocker) that are not this objective's code gaps.

---

_Verified: 2026-07-09T04:05:00Z_
_Verifier: Claude (verifier)_
