---
objective: 03-aoid-authentication-integration-oidc
trd: "05"
type: standard
wave: 4
depends_on: ["03-04"]
files_modified:
  - docs/eden/WOPI-HOST.md
  - wopi-host/README.md
autonomous: false
requirements: [AUTH-01, AUTH-03]
user_setup:
  - service: aoid
    why: "The WOPI host must exist as an oauth_clients row in a real AOID instance before the real-IdP flow can run — registration is exclusively a tenant-scoped, admin-authenticated Connect-RPC call (no static-config path exists)"
    env_vars:
      - name: EDENDOCS_WOPI_OIDC_CLIENT_ID
        source: "Output of AOID admin CreateClient RPC"
      - name: EDENDOCS_WOPI_OIDC_CLIENT_SECRET
        source: "CreateClient response — the plaintext secret is emitted EXACTLY ONCE at creation; capture immediately"
      - name: EDENDOCS_WOPI_AOID_ISSUER
        source: "The AOID deployment's issuer URL (deployed instance or local dev instance from ~/dev/aoid)"
    dashboard_config:
      - task: "Run AOID admin CreateClient: client_type=confidential, token_endpoint_auth_method=client_secret_basic, grant_types=[authorization_code], redirect_uris=[http://127.0.0.1:8091/callback], auto_consent=true (first-party org service)"
        location: "AOID admin Connect-RPC (requires admin auth + tenant access — see runbook section written by Task 1)"

must_haves:
  truths:
    - "docs/eden/WOPI-HOST.md lets a person who has never seen this objective run the WOPI host against fake AND real AOID: architecture (two-token model), env reference, AOID client-registration runbook, production alias_groups XML, and the v1 limitations register"
    - "A human has SEEN the real flow: launch page → AOID login → editor iframe loads hello.odt → the AOID account's display name appears in the editor's user list — and the result is recorded"
    - "The v1 limitations (IsAdminUser always false + the ent[]/aud=aoedge v2 path, no avatar claim, SupportsLocks=false, single-tenant issuer) are documented as decisions with their research rationale, not silent gaps"
  artifacts:
    - path: "docs/eden/WOPI-HOST.md"
      provides: "Operator runbook + architecture doc for the reference WOPI host"
      contains: "alias_groups"
    - path: "wopi-host/README.md"
      provides: "Quickstart (build/run/test/e2e) pointing at the full runbook"
      contains: "8091"
  key_links:
    - from: "docs/eden/WOPI-HOST.md"
      to: "AOID admin CreateClient RPC"
      via: "step-by-step client-registration runbook including one-time secret capture and per-environment redirect_uris"
      pattern: "CreateClient"
    - from: "docs/eden/WOPI-HOST.md"
      to: "coolwsd.xml storage.wopi.alias_groups"
      via: "production trust config: the mode=\"groups\" XML block + proof_key existence note (coolwsd signs unconditionally when the file exists)"
      pattern: "mode=\"groups\""
---

<objective>
Write the operator/integrator documentation for the reference WOPI host and
close the objective with the one verification only a human can give: a real
person, through a real browser, authenticating against a real AOID and seeing
their own AOID identity inside the EdenDocs editor.

Purpose: TRD 03-04 proved the machinery headlessly against a fake IdP;
success criteria 1 and 3 of the objective are stated from the USER's
perspective and need a human eye on a real IdP round trip (visual identity
rendering cannot be grepped).

Output: docs/eden/WOPI-HOST.md + wopi-host/README.md, and a recorded
human-verified real-AOID pass (or a precisely-scoped deferral if no AOID
instance is reachable).
</objective>

<file_tree>
docs/eden/WOPI-HOST.md   ← CREATE (docs/eden/ already exists — Objective 2's checklist lives there)
wopi-host/README.md      ← CREATE
</file_tree>

<execution_context>
@~/.claude/devflow/workflows/execute-trd.md
@~/.claude/devflow/templates/summary.md
</execution_context>

<embedded_context>

<codebase_examples>
AOID client registration model (verified from
/Users/justin/dev/aoid/internal/oauth/admin/service.go CreateClient and
migrations/0015_oauth_clients.up.sql): registration is ONLY via the
tenant-scoped admin Connect-RPC (requireAdmin + HasTenantAccess). The RPC
bcrypt-hashes the secret, enforces the auth-method/client-type CHECK
constraints, and audit-logs ActionClientCreated. A hand-written SQL INSERT is
forbidden (bypasses all three). The plaintext client_secret appears exactly
once, in the CreateClient response.

Production alias_groups block to document (shape verified from
wsd/HostUtil.cpp; goes in the DEPLOYED coolwsd.xml, one host+aliases per
group):
```xml
<storage>
  <wopi allow="true">
    <alias_groups mode="groups">
      <group>
        <host allow="true">https://wopi.example.aocyber.ai</host>
        <alias>https://wopi-alt.example.aocyber.ai</alias>
      </group>
    </alias_groups>
  </wopi>
</storage>
```
</codebase_examples>

<anti_patterns>
- Do NOT document (or perform) a direct SQL insert into aoid.oauth_clients.
- Do NOT write docs that duplicate research prose — the runbook documents
  what EXISTS (commands, env vars, config blocks) with one-line rationales
  and pointers to 3-RESEARCH.md for depth.
- Do NOT let the checkpoint replace automation: Claude starts every service,
  registers the client if an AOID admin path is scriptable in-session, and
  hands the user a URL — the human contributes only login credentials and
  eyes on the rendered identity (automation-first rule).
- Do NOT mark AUTH-03 complete on the fake-IdP evidence alone — the visual
  identity check against real AOID is the point of this TRD.
</anti_patterns>

<error_recovery>
- If no AOID instance is reachable (none deployed, local dev instance won't
  start): STOP at the checkpoint, record exactly what was attempted, and
  surface the deferral to the user — the objective's headless evidence
  (03-04) stands, but success criteria 1/3 remain OPEN and the objective must
  not be marked complete. Do not simulate the pass.
- If AOID rejects the redirect_uri: it must EXACTLY match the registered
  value including scheme/port (http://127.0.0.1:8091/callback for the local
  run) — re-register rather than fighting it.
- If login succeeds but the editor shows "UnknownUser": the account has no
  `name` and the userinfo fallback chain ended empty — check the WOPI host
  log for the CheckFileInfo UserFriendlyName value; fix the mapping only if
  the log shows an empty Name despite non-empty claims.
- If the iframe is blank with a console CSP error: add
  `--o:net.frame_ancestors=http://127.0.0.1:8091` to the coolwsd launch
  (03-04 research note) and record that the production runbook needs the
  frame_ancestors key alongside alias_groups.
</error_recovery>

</embedded_context>

<context>
@.planning/PROJECT.md
@.planning/objectives/03-aoid-authentication-integration-oidc/3-RESEARCH.md
@.planning/objectives/03-aoid-authentication-integration-oidc/03-04-SUMMARY.md
</context>

<research_context>
- v1 limitations register (all research-verified, to be documented as
  DECISIONS): IsAdminUser hard-false — AOID v1.0 has no admin claim reachable
  by a generic RP; the v2 path is ent[] entitlements + SkipClientIDCheck
  local decode (aud is hardcoded "aoedge" — AOID's own TODO) — Pitfall 1.
  No avatar — no picture claim exists; browser falls back to user.svg —
  Pitfall 5. SupportsLocks=false — lock-conflict semantics deliberately
  descoped — Open Question 1. Single-tenant issuer — per-tenant discovery
  documented as extension point — Open Question 3.
- Public-repo note for the docs: EdenDocs is public, eden-platform-go is
  private — external builders of wopi-host need org access (GOPRIVATE +
  insteadOf mapping); document the exact two commands.
- Port policy section is mandatory: 8091 wopi-host, 8092 fake-aoid,
  9980 coolwsd, 9981 e2e docker host port; 8080 is BANNED project-wide.
</research_context>

<gotchas>
- HARD RULE: no 8080 anywhere in the docs except a single explicit
  prohibition sentence in the port-policy section.
- docs/eden/ is shared with Objective 2's TRADEMARK-MPL-CHECKLIST.md — add
  the new file, touch nothing else there. verify-branding.sh's sweep scans
  shipped surfaces, not docs/, but keep Collabora references in the doc
  factual/attributional only (e.g. "coolwsd", "Collabora Online fork") per
  the trademark checklist's allowed-classes taxonomy.
- The checkpoint task needs the human to have an AOID account with a display
  name set — note it in the checkpoint instructions (otherwise the fallback
  chain shows the email, which is a PASS for the mapping but a weaker visual
  check).
</gotchas>

<tasks>

<task type="auto">
  <name>Task 1: Write docs/eden/WOPI-HOST.md + wopi-host/README.md</name>
  <files>docs/eden/WOPI-HOST.md, wopi-host/README.md</files>
  <action>
docs/eden/WOPI-HOST.md sections:
1. What this is — reference WOPI host architecture; diagram of the TWO
   unrelated credentials (AOID OIDC tokens stay server-side in the WOPI
   host; the WOPI access_token is a host-minted opaque uuid coolwsd echoes
   back) — research Pitfall 2, this is the doc's centerpiece.
2. Quickstart — build, run against fake-aoid (exact commands, ports 8091/
   8092), run the e2e (`./wopi-host/scripts/wopi-e2e.sh`, both modes).
3. Env reference — every EDENDOCS_WOPI_* var from internal/config with
   defaults and the docker-mode PublicURL/WopiBaseURL distinction.
4. AOID client registration runbook — the CreateClient RPC parameters from
   user_setup/codebase_examples, one-time secret capture warning,
   per-environment redirect_uris, auto_consent recommendation.
5. coolwsd trust (production) — the alias_groups mode="groups" XML block +
   the mode-attribute silent-rejection footgun (Pitfall 4) + proof_key note
   (coolwsd signs unconditionally when {configdir}/proof_key exists; the
   WOPI host REJECTS unproven requests — that is where "validation enabled"
   lives) + frame_ancestors knob.
6. v1 limitations & decisions — the four items from research_context, each
   with rationale and the concrete v2 path.
7. Port policy + public-repo/private-dep note (the two GOPRIVATE commands).
wopi-host/README.md: 30-line quickstart pointing at the full doc.
Both files MPL-2.0-headed (markdown comment header consistent with
docs/eden/ conventions).
Commit: `docs(03-05): WOPI host runbook + README (registration, trust, v1 limits)`
  </action>
  <verify>test -f docs/eden/WOPI-HOST.md && test -f wopi-host/README.md && grep -q 'mode="groups"' docs/eden/WOPI-HOST.md && grep -q 'CreateClient' docs/eden/WOPI-HOST.md && grep -q 'IsAdminUser' docs/eden/WOPI-HOST.md && grep -qi 'exactly once' docs/eden/WOPI-HOST.md && ! grep -n '8080' docs/eden/WOPI-HOST.md wopi-host/README.md | grep -vi 'banned\|never' | grep -q .</verify>
  <done>Runbook covers architecture, quickstart, env reference, AOID registration, production trust config, v1 limitations, port policy; README quickstart exists; no stray 8080 references.</done>
  <recovery>If doc claims drift from implemented behavior, the CODE from 03-01..04 is ground truth — fix the doc, never retrofit code to match prose in this TRD.</recovery>
</task>

<task type="checkpoint:human-verify">
  <name>Task 2: Real-AOID end-to-end — human sees their AOID identity in the editor</name>
  <files>docs/eden/WOPI-HOST.md</files>
  <action>
Claude automates everything up to the login screen:
1. Determine the reachable AOID instance with the user (deployed issuer, or
   local dev instance from ~/dev/aoid).
2. Register the client if scriptable in-session (admin RPC per the runbook);
   otherwise walk the user through it (user_setup). Capture client_id/secret
   into the local env (never into git).
3. Start the stack: coolwsd (docker reference container with aliasgroup1, or
   native build), wopi-host on 8091 with the REAL issuer + credentials,
   seeded hello.odt.
4. Print the URL: http://127.0.0.1:8091/open?file=hello.odt

USER VERIFIES (browser):
- [ ] Visiting the URL redirects to the real AOID login (Authorization Code
      flow — success criterion 1)
- [ ] After login, the editor iframe loads hello.odt
- [ ] The user list / avatar area shows YOUR AOID display name (generic
      avatar icon is EXPECTED — AOID has no avatar claim; criterion 3)
- [ ] Type a change and save; no error surfaces (PutFile against the real
      session)
Record the outcome (+ screenshots if convenient) in the SUMMARY. On PASS,
append a "Verified against real AOID on <date>, issuer <redacted-host>" line
to the runbook's Quickstart section. On FAIL or NO-AOID-REACHABLE, record
per error_recovery and leave the objective open.
  </action>
  <verify>Human confirmation of the four checklist items; SUMMARY records the outcome; runbook carries the verified-on line (PASS case).</verify>
  <done>A human has confirmed the real-AOID OIDC round trip and visible AOID identity in the editor — the last two success criteria of the objective that headless evidence cannot cover.</done>
  <recovery>See error_recovery: redirect_uri mismatch → re-register; UnknownUser → check CheckFileInfo log line before touching mapping; blank iframe → frame_ancestors knob; no AOID reachable → precise deferral, never simulate.</recovery>
</task>

</tasks>

<validation_gates>
<lint>! grep -rn '8080' docs/eden/WOPI-HOST.md wopi-host/README.md | grep -vi 'banned\|never' | grep -q .</lint>
<test>grep -q 'Verified against real AOID' docs/eden/WOPI-HOST.md || echo 'PENDING: human checkpoint not yet passed'</test>
<build>test -f docs/eden/WOPI-HOST.md</build>
</validation_gates>

<verification>
- Runbook complete per Task 1 verify; README quickstart present.
- Human checkpoint outcome recorded in 03-05-SUMMARY.md (PASS with the four
  checklist items, or a precise deferral).
- All four AUTH requirements now have their final evidence trail:
  AUTH-01/03 human-verified real flow (this TRD), AUTH-02/04 headless e2e
  (03-04).
</verification>

<success_criteria>
An operator can stand up the reference WOPI host against real AOID using only
the runbook, and a human has watched the full journey — AOID login to their
own name rendered inside the EdenDocs editor — confirming the objective's
user-facing success criteria.
</success_criteria>

<output>
After completion, create `.planning/objectives/03-aoid-authentication-integration-oidc/03-05-SUMMARY.md`
recording the checkpoint outcome, the AOID instance used (issuer host
redacted if deployed), client-registration notes (secret NOT recorded), and
any deltas discovered between fake-IdP and real-AOID behavior.
</output>
