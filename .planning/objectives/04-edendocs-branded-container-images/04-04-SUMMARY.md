---
objective: 04-edendocs-branded-container-images
job: "04"
subsystem: infra
tags: [docker, ghcr, cosign, keyless, sigstore, oidc, sbom, cyclonedx, trivy, metadata-action, buildx, verify-caps, boot-health, reverse-proxy, mpl, trademark, dist-01, dist-02, dist-03]

# Dependency graph
requires:
  - objective: 04-01-release-build-engine-pin
    provides: "scripts/eden/build-deps.sh, scripts/eden/fetch-engine-pinned.sh (sha256 ea91ef33…, fail-closed on 076a4129…), scripts/eden/build-release.sh (non-debug instdir), docker/eden/engine-pin.env (pin + mirror release-asset URL)"
  - objective: 04-02-branded-dockerfile-verify-caps
    provides: "docker/eden/Dockerfile (3-stage assemble→verify-caps→runtime, ARG base_image=ubuntu:24.04, LABEL-free, EXPOSE 9980, coolwsd --probe HEALTHCHECK)"
  - objective: 04-03-reverse-proxy-smoke-harness
    provides: "scripts/eden/proxy-smoke-test.sh (+ nginx-test-server{,-negative}.conf) — >60s WS survival through the verbatim etc/nginx/coolwsd.conf on host port 8091, with a short-timeout negative control"
provides:
  - ".github/workflows/docker-publish.yml — the DIST-03 publish pipeline joining all three Wave-1 blocks: build-deps → fetch-engine-pinned → build-release → stage MPL COPYING → buildx build (verify-caps in-build) → boot-health gate → LABEL identity + MPL-in-image integrity → proxy-smoke >60s → trivy → push :main/:sha-<short>/:latest to ghcr.io/ao-cyber-systems/edendocs → keyless cosign sign + CycloneDX SBOM attest + verify"
  - "docs/eden/TRADEMARK-MPL-CHECKLIST.md §9 (Container image surfaces) + the filled-in §3 container-LABEL row"
affects: [05-upstream-tracking-workflow]

# Tech tracking
tech-stack:
  added: []   # all actions are GitHub-hosted (metadata-action, build-push-action, cosign-installer, sbom-action/syft, trivy-action); no repo dependency added
  patterns:
    - "Two-phase buildx: build once with load:true as :scan (verify-caps runs in-build), gate it locally (boot-health + LABEL/MPL + proxy-smoke + trivy), then push:true (cache-hit) with provenance:false so the signed digest is a plain single-platform manifest"
    - "Keyless cosign (Sigstore/GitHub OIDC, id-token: write) signs the immutable pushed DIGEST (steps.push.outputs.digest), never a mutable tag; CycloneDX SBOM via syft + cosign attest; self-verify with --certificate-identity-regexp + --certificate-oidc-issuer — uniform AO-Cyber-Systems org convention, zero key-based signing"
    - "OCI LABELs injected at CI via docker/metadata-action (Dockerfile stays LABEL-free / edit-free per release); anti-trademark gate matches Collabora BRAND forms (Online/Office/Productivity + domains), not the English word collaborat* (checklist §4 exception #12)"
    - "MPL-in-image completeness: make install ships THIRDPARTYLICENSES only, so the workflow additively stages the repo's byte-identical COPYING into the build context, then sha256-verifies BOTH in-image against the repo copies"
    - "Engine-pin defense-in-depth: fetch-engine-pinned.sh (sha256 gate) + a boot-health gate that boots the pinned engine in the FINAL image (coolwsd accepts on 9980, no SIGSEGV) and aborts the publish on any crash"

key-files:
  created:
    - ".github/workflows/docker-publish.yml"
  modified:
    - "docs/eden/TRADEMARK-MPL-CHECKLIST.md"

key-decisions:
  - "Keyless cosign only (Sigstore/OIDC, id-token: write) — no keypair, no COSIGN_PRIVATE_KEY; the pushed DIGEST is signed, a CycloneDX SBOM is attested, and cosign verify self-checks against the repo OIDC identity"
  - "provenance:false on the push build (deviation from the TRD outline's provenance: mode=max + sbom: true): keeps steps.push.outputs.digest a plain image manifest so the signed digest is unambiguous; the required CycloneDX SBOM is provided by the explicit cosign attest step"
  - "MPL COPYING is staged into the build context by the workflow (Rule 2 — missing critical functionality): make install (dist_doc_DATA) ships THIRDPARTYLICENSES but NOT COPYING, so the image would otherwise lack the MPL notice; staging the repo's byte-identical COPYING keeps the change entirely inside docker-publish.yml (zero Wave-1 file edits) and makes the must-have (COPYING+THIRDPARTYLICENSES byte-intact in-image) genuinely true"
  - "Anti-Collabora LABEL gate targets the trademark brand forms, and the description label is 'Privacy-first self-hosted document editor' (no 'collaborative') — a plain grep -qi collabora would false-FATAL on the English word; this mirrors checklist §4 exception #12"
  - "Local validation is lint/structural only per the rollout boundary: the macOS dev host cannot build/run the image, and pushing eden-main IS the live publish+sign rollout — held for the user's explicit approval. Task 3 (populate engine-pin mirror, push, drive CI green) is DEFERRED with exact commands below"

patterns-established:
  - "docker-publish.yml release-image pipeline: reusable author-then-drive-green shape (same as TRD 01-02) where the workflow's own in-line gates (verify-caps, boot-health, proxy-smoke, trivy, cosign verify) ARE the integration tests"

requirements-completed: []   # DIST-01/DIST-02/DIST-03 are CODE-COMPLETE but their integrated proof is the single GREEN docker-publish.yml run, which is DEFERRED to the user-approved push — left Pending in REQUIREMENTS.md (see Post-TRD Verification)

# Verification evidence
verification:
  gates_defined: 3          # lint / test / build
  gates_passed: 1           # lint PASS locally; test + build are the live CI run (DEFERRED, not run)
  auto_fix_cycles: 0        # no code-bug auto-fix cycles; two local-gate self-corrections (see Issues Encountered)
  tdd_evidence: false       # TDD-EXCEPTION (infra/release workflow — the workflow's own gates are the tests, driven green in CI)
  test_pairing: false       # N/A — CI workflow, not unit-tested source

# Metrics
duration: ~30min
completed: 2026-07-09
---

# Objective 4 TRD 04: Docker Publish + Keyless Cosign Summary

**Additive `.github/workflows/docker-publish.yml` that joins the three Wave-1 blocks into the DIST-03 pipeline — build-deps → sha256-pinned engine → non-debug release build → buildx (verify-caps in-build) → boot-health gate → LABEL identity + MPL byte-integrity → reverse-proxy >60s smoke → Trivy → push to ghcr.io/ao-cyber-systems/edendocs → keyless cosign sign + CycloneDX SBOM attest + verify — plus the trademark/MPL checklist container rows. Authored, atomically committed, and lint/structurally validated on eden-main; the live publish is held for the user-approved push.**

## Performance

- **Duration:** ~30 min
- **Completed:** 2026-07-09T08:10:57Z
- **Tasks:** 2 executed + committed (Task 1, Task 2); Task 3 DEFERRED (live rollout)
- **Files modified:** 2 (1 created, 1 modified)

## Accomplishments
- Authored the additive `docker-publish.yml` (build.yml byte-untouched) wiring all three Wave-1 building blocks into one push-triggered publish+sign pipeline — **actionlint CLEAN** (including shellcheck on every `run:` block).
- Extended `docs/eden/TRADEMARK-MPL-CHECKLIST.md`: filled the §3 container-LABEL placeholder row and added §9 (container image surfaces, MPL-in-image integrity, keyless-cosign signing, zero-upstream-divergence note).
- Closed a real gap the TRD outline missed: `make install` ships only `THIRDPARTYLICENSES`, so the workflow additively stages the repo's byte-identical `COPYING` into the build context — the image now genuinely ships both MPL files, sha256-verified in-image.
- Hardened the anti-Collabora LABEL gate to match the trademark brand forms (not the English word `collaborat*`), avoiding a self-inflicted false-FATAL on our own image description.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: Author docker-publish.yml | `python3 -c "yaml.safe_load(...)" && actionlint docker-publish.yml && grep -q <14 structural gates: eden-main trigger, lowercase ghcr, fetch-engine-pinned/build-release/proxy-smoke wired, docker/eden/Dockerfile, id-token: write, cosign sign+attest, metadata-action, trivy, no COSIGN_KEY, no 8080, no --enable-debug>` | 0 | PASS |
| 2: Checklist container rows | `grep -qi 'ghcr.io/ao-cyber-systems/edendocs' && grep -qi metadata-action && grep -qi cosign && grep -qi 'ZERO upstream-file divergences' && grep -qi COPYING && ! grep -q 8080` (docs/eden/TRADEMARK-MPL-CHECKLIST.md) | 0 | PASS |
| 3: Populate mirror, push, drive CI green, verify signed image | `gh run list --workflow=docker-publish.yml --branch eden-main --limit 1 --json conclusion --jq '.[0].conclusion' | grep -x success && ...` | — | **DEFERRED** (live publish rollout — requires the user-approved push; see "Deferred to Live CI" below) |

## Task Commits

Each executed task was committed atomically to eden-main:

1. **Task 1: docker-publish.yml** - `1067e76a353` (feat)
2. **Task 2: checklist container rows** - `159f4d1c2ab` (docs)

_Task 3 has no commit — it is the deferred live rollout (no push, no gh release, no workflow trigger performed)._
_(An accidental `df-tools commit --help` invocation created a stray "--help" commit of DevFlow cache files; it was immediately undone with `git reset --mixed HEAD~1`. Those `.planning/.*` caches remain untracked; HEAD is clean.)_

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| lint | `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/docker-publish.yml'))"; actionlint .github/workflows/docker-publish.yml` | 0 | PASS |
| test | `gh run list --workflow=docker-publish.yml --branch eden-main --limit 1 --json conclusion --jq '.[0].conclusion' | grep -x success` | — | **DEFERRED** (the live CI run — held for the user-approved push) |
| build | `true` (the CI run IS the build+publish; the green run is the proof) | — | **DEFERRED** |

Additional static gates (all run locally, all PASS): `build.yml` + `docker/from-source/` + `docker/from-packages/` + `docker/cosign.pub` byte-untouched vs the pre-04-04 tip; only the 2 intended files changed; `id-token: write` present; keyless cosign sign+attest+verify present; no `COSIGN_KEY`/`COSIGN_PRIVATE_KEY`/`generate-key-pair`; `ghcr.io/ao-cyber-systems/edendocs` lowercase; no `8080` in workflow or checklist; only `secrets.GITHUB_TOKEN` referenced (no new secret).

## TDD Evidence

N/A — **TDD-EXCEPTION** (declared in the TRD frontmatter): this is infra/release work (a CI workflow). The workflow's own gates (verify-caps build stage, boot-health, proxy-smoke >60s, trivy, cosign verify) ARE the tests, driven green as the integrated DIST-01/02/03 proof (same shape as TRD 01-02). `actionlint` + the green run are the gate; the green run is DEFERRED to the user-approved push.

## Post-TRD Verification

- **Auto-fix cycles used:** 0 (no code-bug auto-fixes; two local-gate self-corrections during authoring — see Issues Encountered)
- **Must-haves verified:** 2/5 locally — must-have #1 (workflow exists + correct shape/trigger/order) and must-have #5 (checklist container rows) are statically verified. Must-haves #2 (green run + log evidence), #3 (published LABELed image, MPL byte-intact), and #4 (cosign-signed digest + SBOM, cosign verify) are **DEFERRED** to the live CI run (they inherently require the build+publish that only the user-approved push triggers).
- **Gate failures:** None (lint PASS; test/build deferred, not failed).
- **Requirements DIST-01/02/03:** left **Pending** in REQUIREMENTS.md — code-complete, but the integrated green-run proof is deferred. Marked as such in ROADMAP/STATE, not checked off.

## Files Created/Modified
- `.github/workflows/docker-publish.yml` (created) — the additive DIST-03 release publish+sign pipeline (build.yml untouched).
- `docs/eden/TRADEMARK-MPL-CHECKLIST.md` (modified) — filled the §3 container-LABEL row; added §9 container image surfaces.

## Decisions Made
See `key-decisions` in the frontmatter. Headline: keyless-cosign-only (no keypair); `provenance:false` for an unambiguous signed digest; additively stage `COPYING` into the build context (Rule 2); trademark-form anti-Collabora gate; lint/structural-only local validation with the live publish deferred to the user-approved push.

## Deviations from Plan

### Auto-fixed / design deviations

**1. [Rule 2 - Missing Critical] Stage the MPL `COPYING` into the image build context**
- **Found during:** Task 1 (authoring the MPL-in-image gate).
- **Issue:** The TRD outline asserted `COPYING` is present in the image at `/usr/share/doc/coolwsd/COPYING`, but `make install` (Makefile.am `dist_doc_DATA`) installs only `THIRDPARTYLICENSES` — `COPYING` is not installed. The gate as-outlined would have FATAL'd in live CI (a false negative), and the published image would lack the MPL notice text (an MPL-completeness gap).
- **Fix:** Added a workflow step (after `build-release.sh`, before the `:scan` build) that copies the repo's byte-identical `COPYING` next to the staged `THIRDPARTYLICENSES`, and made the MPL gate locate both by path-agnostic `find`, grep `Mozilla Public` on `COPYING`, and sha256-compare BOTH against the repo copies. Kept entirely inside `docker-publish.yml` — zero Wave-1 file edits.
- **Files modified:** `.github/workflows/docker-publish.yml`
- **Verification:** actionlint clean; Task-1 structural gates pass. (In-image assertion runs in the deferred CI job.)
- **Committed in:** `1067e76a353`

**2. [Rule 1 - Bug] Trademark-form anti-Collabora gate + Collabora-free description label**
- **Found during:** Task 1.
- **Issue:** The TRD's plain `grep -qi 'collabora' labels.json` combined with the example description "…collaborative document editor" would MATCH the substring "collabora" inside "collaborative" and FATAL on our own label.
- **Fix:** Changed the description to "Privacy-first self-hosted document editor" (no `collabora` substring) AND changed the gate to match the Collabora BRAND forms (`Collabora Online/Office/Productivity` or the `collabora{online,office}` domains), per checklist §4 exception #12. This is the semantically correct trademark guard.
- **Files modified:** `.github/workflows/docker-publish.yml` (+ documented in checklist §9)
- **Verification:** actionlint clean; described in checklist §9.
- **Committed in:** `1067e76a353`

**3. [Rule 1 - Correctness] `provenance:false` on the push build (vs outline's `provenance: mode=max` + `sbom: true`)**
- **Found during:** Task 1.
- **Issue:** buildkit provenance/SBOM attestations turn the pushed artifact into an image index, making "sign the digest" ambiguous for `cosign sign`/`syft`/`cosign attest` on the deferred live run.
- **Fix:** Set `provenance:false` so `steps.push.outputs.digest` is a plain single-platform manifest; the required CycloneDX SBOM is provided by the explicit `cosign attest --type cyclonedx` step (must-have still satisfied).
- **Files modified:** `.github/workflows/docker-publish.yml`
- **Committed in:** `1067e76a353`

**4. [Cosmetic] Added `org.opencontainers.image.source` label + reworded a comment to drop the `--enable-debug` literal**
- Added the `source=https://github.com/AO-Cyber-Systems/EdenDocs` OCI label (named in must-have #3 and research example 6/7).
- Reworded the top comment describing build.yml to say "debug-enabled" instead of the literal `--enable-debug`, so the file passes the TRD's own `! grep -q -- '--enable-debug'` gate (same class as TRD 04-03's "NEVER 8080" comment reword).
- **Committed in:** `1067e76a353`

---

**Total deviations:** 4 (1 missing-critical, 2 correctness, 1 cosmetic).
**Impact on plan:** All necessary for correctness/MPL-completeness or to make the TRD's own gates pass. No scope creep — every change is inside the two intended files; zero Wave-1 or upstream file edits.

## Issues Encountered

- **`df-tools commit --help` made an accidental commit.** Invoking the tool with `--help` was treated as a commit message and committed 4 untracked DevFlow cache files (`.planning/.awareness-cache.json`, `.dup-detect-log.jsonl`, `.skill-active`, `state.json`) as a `"--help"` commit. Undone immediately with `git reset --mixed HEAD~1` (no commit hook involved); the caches are back to untracked. All real task commits used explicit `--files`.
- **Two local-gate self-corrections during authoring** (not deviations from plan intent): (a) the top comment's `--enable-debug` literal tripped the TRD's own no-`--enable-debug` gate → reworded to "debug-enabled"; (b) the checklist §9 phrase "ZERO upstream-file divergences" was initially split by markdown bold + a line-wrap so the Task-2 grep missed it → reworded to a contiguous line. Both re-validated green.

## Deferred to Live CI (the user-approved rollout)

**Nothing below was executed.** Pushing `eden-main` triggers the live build → publish → sign to `ghcr.io/ao-cyber-systems/edendocs`, which is a live rollout requiring the user's explicit approval (not given). Exact commands the user runs to complete the publish, in order:

1. **(First run only) Populate the engine-pin mirror** — the `engine-pin` Release asset does not exist yet, so `fetch-engine-pinned.sh` would 404 on the first CI run. Obtain a boot-validated known-good `engine-main-assets.tar.gz` (prefer the recorded sha256 `ea91ef33ba510eb1379568c58afe9ddc54aa05985f4ff963ab8e0a8cdfa6c59f`; NEVER the known-SIGSEGV `076a41297c2ed659638617ed27cb6d4cff01f1558f36fb0e0b1abd6f4a14868e`), then:
   ```
   gh release create engine-pin --repo AO-Cyber-Systems/EdenDocs \
     --notes "pinned EdenDocs release engine (boot-validated)"
   gh release upload engine-pin engine-main-assets.tar.gz --repo AO-Cyber-Systems/EdenDocs
   ```
   If `ea91ef33…` is no longer obtainable, re-pin per the TRD error_recovery: fetch the current rolling engine, boot-validate it, and if green update `docker/eden/engine-pin.env` to the new sha256 (committed atomically) before uploading.

2. **Push to trigger the pipeline** (this is the rollout — the single `git push` that produces the signed published image):
   ```
   git push origin eden-main
   ```

3. **Watch the run:**
   ```
   gh run watch "$(gh run list --workflow=docker-publish.yml --branch eden-main --limit 1 --json databaseId --jq '.[0].databaseId')" --exit-status
   ```

4. **Verify the signed published image:**
   ```
   cosign verify "ghcr.io/ao-cyber-systems/edendocs:main" \
     --certificate-identity-regexp="https://github.com/AO-Cyber-Systems/EdenDocs/.*" \
     --certificate-oidc-issuer="https://token.actions.githubusercontent.com"
   ```

5. **On green,** capture the run URL + published digest + `cosign verify` block into checklist §8/§9 and this SUMMARY, then mark DIST-01/02/03 complete.

## User Setup Required

None blocking. **Optional (per the TRD `user_setup`):** after the first publish, the new GHCR package is PRIVATE by default (still satisfies DIST-03 — org members can pull). To allow external pull, set visibility to Public: GitHub → AO-Cyber-Systems/EdenDocs → Packages → edendocs → Package settings → Change visibility. This is a human action; do not force it.

## Next Objective Readiness
- Objective 4 authoring is complete (4/4 TRDs written; 04-04's artifact committed). The only remaining step is the user-approved push that drives the integrated green run and closes DIST-01/02/03.
- Objective 5 (Upstream Tracking Workflow) can proceed: the §7 divergence allowlist is unchanged by Objective 4 (all container work is additive), which §9 of the checklist records.

## Explicit rollout-boundary confirmation
- **Did NOT `git push`** to origin (no live publish triggered).
- **Did NOT** create/upload the `engine-pin` GitHub Release asset.
- **Did NOT** run `gh workflow run` / `gh run rerun` or otherwise trigger any GitHub Actions run.
- All validation performed locally was lint/static/structural only (actionlint, yamllint, python YAML parse, and greps).

---
*Objective: 04-edendocs-branded-container-images*
*Completed: 2026-07-09*

## Self-Check: PASSED

- Files exist: `.github/workflows/docker-publish.yml`, `docs/eden/TRADEMARK-MPL-CHECKLIST.md`, `.planning/objectives/04-edendocs-branded-container-images/04-04-SUMMARY.md` — all FOUND.
- Commits exist: `1067e76a353` (Task 1), `159f4d1c2ab` (Task 2) — both FOUND on eden-main.
- `build.yml` + `docker/from-source/` + `docker/from-packages/` byte-untouched; only the 2 intended files changed by 04-04.
