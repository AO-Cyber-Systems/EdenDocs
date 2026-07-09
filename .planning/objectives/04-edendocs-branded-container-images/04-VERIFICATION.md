---
objective: 04-edendocs-branded-container-images
verified: 2026-07-09T08:22:16Z
status: human_needed
score: 3/3 must-haves authored & wired (live CI green-run deferred by design)
re_verification:
  previous_status: none
  note: "Initial verification"
gaps: []
human_verification:
  - test: "First-run only — populate the org engine-pin mirror asset, then trigger the deferred publish pipeline"
    commands:
      - "gh release create engine-pin --repo AO-Cyber-Systems/EdenDocs --notes 'Pinned boot-validated engine snapshot for EdenDocs release images' || true"
      - "gh release upload engine-pin engine-main-assets.tar.gz --repo AO-Cyber-Systems/EdenDocs   # the ea91ef33… boot-green tarball; fetch-engine-pinned.sh hard-fails on any sha256 drift"
      - "git push origin eden-main   # USER-APPROVED rollout — triggers .github/workflows/docker-publish.yml on the eden-main push"
    expected: "docker-publish.yml runs green on ubuntu-latest: build-deps → fetch-engine-pinned (sha256 OK) → build-release (non-debug instdir) → buildx build (verify-caps 'post-COPY caps' line, no cap loss) → boot-health 'BOOT HEALTH PASSED: coolwsd accepts on 9980, no SIGSEGV' → 'MPL-IN-IMAGE PASSED' → 'PROXY SMOKE PASSED' (>60s WS survives, negative control dies) → trivy clean → push ghcr.io/ao-cyber-systems/edendocs:{main,sha-<short>,latest} → 'COSIGN KEYLESS SIGN + ATTEST + VERIFY PASSED'."
    why_human: "Publishing a signed image to the live GHCR registry is a rollout step requiring explicit user approval per the HARD RULE; it also cannot run on this macOS dev host (no Linux docker engine to build/boot the image). The engine-pin GitHub Release asset must be populated once before the first run or fetch-engine-pinned.sh returns a wget 404. Deferred by design, not a defect."
  - test: "Independent post-publish signature verification (after the run above is green)"
    commands:
      - "cosign verify ghcr.io/ao-cyber-systems/edendocs@<digest> --certificate-identity-regexp='https://github.com/AO-Cyber-Systems/EdenDocs/.*' --certificate-oidc-issuer='https://token.actions.githubusercontent.com'"
    expected: "cosign confirms the Sigstore/Fulcio keyless signature + CycloneDX SBOM attestation for the pushed digest."
    why_human: "Requires the live-published digest, which only exists after the user-approved push completes."
---

# Objective 4: EdenDocs-Branded Container Images — Verification Report

**Objective Goal:** Customers can deploy a signed, EdenDocs-branded container image, built on the from-source pattern, that survives real-world reverse-proxy conditions rather than only direct-connect access.

**Verified:** 2026-07-09T08:22:16Z
**Status:** human_needed (artifacts complete & correctly wired; one user-approved rollout action outstanding)
**Re-verification:** No — initial verification

## Goal Achievement

The heavy end-to-end proof for this objective (a green `docker-publish.yml` run that builds, boot-validates, proxy-smoke-tests, pushes, and cosign-signs the image) is **intentionally deferred** to a user-approved `git push origin eden-main`, per the rollout-boundary rule (publishing signed images to a live registry is an approval-gated rollout, and this macOS host cannot build/run the Linux image). Verification therefore confirms the **artifacts are authored so that IF the workflow runs it WOULD satisfy each success criterion**, and classifies the live green-run as `human_needed / deferred-by-design`. No authoring defect was found.

### Observable Truths

| # | Truth (Success Criterion) | Status | Evidence |
| - | ------------------------- | ------ | -------- |
| 1 | Image builds from the `from-source` pattern (COPY of prebuilt engine + EdenDocs wsd/browser + branding), NOT the from-packages repackage | ✓ VERIFIED (authoring) | `docker/eden/Dockerfile:28` `COPY instdir /`; apt list `:20-24` is runtime deps only (no collabora/coolwsd packages); `build-release.sh:18-26` non-debug configure + branding flags; `build-release.sh:46-57` stages instdir + engine + branding |
| 2 | `verify-caps` stage passes AND a doc session survives >60s through a real reverse proxy (not direct 127.0.0.1:9980) | ✓ VERIFIED (authoring); live run deferred | `Dockerfile:42-49` real `COPY --from` + getcap assert (exit 1 on cap loss), forced into runtime at `:56`; `proxy-smoke-test.sh:10,60-61` holds WS 80s; `:42-43` asserts loaded `36000s`; `:83` negative control expects death; `docker-publish.yml:181` runs it against the built `:scan` image |
| 3 | Published images are cosign-signed, in `ghcr.io/ao-cyber-systems` (lowercase), carrying EdenDocs LABEL metadata | ✓ VERIFIED (authoring); live run deferred | `docker-publish.yml:32` lowercase image; `:27` `id-token: write` (keyless); `:219-224` `cosign sign --yes` + `cosign verify` OIDC identity; `:89-103` metadata-action EdenDocs OCI LABELs; `:94-97` tags main/sha-/latest |

**Score:** 3/3 truths authored and fully wired. 0 gaps. Terminal live proof = 1 deferred human/rollout action.

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `scripts/eden/fetch-engine-pinned.sh` | sha256 fail-closed pinned fetch, refuses known-bad digest | ✓ VERIFIED | 43 lines; `:28-35` sha256 mismatch → exit 1; `:12,17-18,36` hard-refuse `076a4129…`; `:11` sources engine-pin.env |
| `scripts/eden/build-release.sh` | non-debug release build staging docker/eden/instdir | ✓ VERIFIED | 71 lines; `:18-26` no `--enable-debug`, `--with-lo-path=/opt/collaboraoffice`, branding flags, none of the 4 phone-home flags; `:46-57` stages coolwsd+engine+branding; `:60-69` loud context asserts |
| `docker/eden/engine-pin.env` | records pin sha256 + mirror URL | ✓ VERIFIED | 15 lines; `:9` `ENGINE_PIN_SHA256=ea91ef33…`; `:15` mirror URL; `:7` known-bad documented as NEVER-pin |
| `docker/eden/Dockerfile` | 3-stage from-source, verify-caps gate, EXPOSE 9980, USER 100, no LABELs | ✓ VERIFIED | 66 lines; 3 `FROM` stages; `:28` `COPY instdir /`; `:42-49` verify-caps; `:56` runtime forces the gate; `:57` EXPOSE 9980; `:58` USER 100; `:60` HEALTHCHECK `coolwsd --probe`; no `LABEL` directive |
| `docker/eden/.dockerignore` / `.gitignore` | keep instdir in build context, gitignore the staging tree | ✓ VERIFIED | .dockerignore excludes only `.git/.gitignore/.DS_Store/*.md` (instdir kept); .gitignore excludes `instdir/` (built fresh in CI) |
| `scripts/eden/proxy-smoke-test.sh` | >60s WS survival via verbatim coolwsd.conf, 8091, negative control | ✓ VERIFIED | 85 lines; `:14-15` byte-unmodified assert; `:34-35` co-located nginx + verbatim mount; `:42-43` 36000s structural assert; `:80,83` positive+negative cases |
| `scripts/eden/nginx-test-server{,-negative}.conf` | positive includes coolwsd.conf; negative uses 5s timeout | ✓ VERIFIED | positive `:9` includes coolwsd.conf, no own /cool/ dir; negative `:14` `proxy_read_timeout 5s`; both `listen 8091` |
| `.github/workflows/docker-publish.yml` | integration pipeline build→boot→proxy→trivy→push→keyless cosign | ✓ VERIFIED | 225 lines; wires all four TRD outputs; `:66,72,75,109,123,149,181,184,196,215` |
| `docs/eden/TRADEMARK-MPL-CHECKLIST.md` | container-specific rows + zero-upstream-divergence statement | ✓ VERIFIED | 235 lines; `:45` LABEL row, `:186-200` image name+OCI table, `:182` ZERO upstream divergences, `:210-235` MPL-in-image + keyless cosign |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | -- | --- | ------ | ------- |
| build-release.sh | docker/eden/instdir | `DESTDIR make install` + engine cp | ✓ WIRED | `build-release.sh:46-51`; Dockerfile `COPY instdir /` consumes it (`:28`) |
| fetch-engine-pinned.sh | engine-pin.env | `source` + fail-closed sha256 | ✓ WIRED | `:11` source; `:17-18,28-36` guards |
| Dockerfile verify-caps | runtime stage | `COPY --from=verify-caps /caps-verified` | ✓ WIRED | `:56` forces the gate into every build |
| proxy-smoke-test.sh | etc/nginx/coolwsd.conf | verbatim RO mount + git-diff assert | ✓ WIRED | `:14-15,35`; coolwsd.conf has `36000s` on `^~ /cool/` (byte-untouched) |
| docker-publish.yml | 04-01/02/03 outputs | build-deps→fetch→build-release→Dockerfile→proxy-smoke | ✓ WIRED | `:66,72,75,109-118,181` — the integration join |
| metadata-action | ghcr lowercase + OCI LABELs | build-push-action labels | ✓ WIRED | `:89-103,118,206` |
| cosign keyless | Sigstore/OIDC | `id-token: write` + sign/attest/verify | ✓ WIRED | `:27,219-224`; no stored key |
| boot-health gate | pinned-engine boot | `coolwsd --probe` on `:scan` | ✓ WIRED | `:123-140` aborts publish on SIGSEGV/no-accept |

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
| ----------- | -------------- | ----------- | ------ | -------- |
| DIST-01 | 04-01, 04-02, 04-04 | from-source branded image (engine + EdenDocs wsd/browser + branding), not from-packages | ✓ SATISFIED (authoring; live build deferred) | Dockerfile COPY instdir; build-release non-debug staging; CI build step |
| DIST-02 | 04-02, 04-03, 04-04 | verify-caps preserved + >60s session through real reverse proxy | ✓ SATISFIED (authoring; live run deferred) | verify-caps getcap gate; proxy-smoke >60s + negative control; CI runs both |
| DIST-03 | 04-04 | cosign-signed, published to AO-Cyber-Systems registry with EdenDocs LABELs | ✓ SATISFIED (authoring; live publish deferred) | lowercase ghcr push, keyless cosign sign/attest/verify, metadata-action LABELs |

REQUIREMENTS.md keeps DIST-01/02/03 unchecked (`[ ]`) pending the live run and records them "Code-complete … CI green-run proof pending user-approved push" (`REQUIREMENTS.md:89-91`). This is the correct state, not a gap. No orphaned requirements: every ID mapped to Objective 4 is claimed by at least one TRD's `requirements:` field.

### Anti-Patterns Found

None. No TODO/FIXME/placeholder/stub markers in any objective artifact. All scripts carry real logic with fail-closed assertions.

### Guardrail Checks

| Guardrail | Result |
| --------- | ------ |
| Port `:8080` in any artifact | ✓ CLEAN — every `8080` match is ban-prose in TRDs/RESEARCH/SUMMARYs; zero in Dockerfile, workflow, nginx confs, or proxy-smoke-test.sh. Host web = 8091, coolwsd = 9980 |
| Upstream files byte-untouched (build.yml, docker/from-source, docker/from-packages) since `8132747eb87` | ✓ CLEAN — `git diff` empty; no commit in range touched them |
| `etc/nginx/coolwsd.conf`, `COPYING`, `THIRDPARTYLICENSES`, `docker/cosign.pub` byte-untouched | ✓ CLEAN — verbatim reuse confirmed |
| Cosign key-based leak (`COSIGN_PRIVATE_KEY` / `generate-key-pair` / `--key`) | ✓ CLEAN — none; keyless OIDC only |
| `--enable-debug` in release build/Dockerfile/workflow | ✓ CLEAN — non-debug confirmed |
| Registry case | ✓ lowercase `ghcr.io/ao-cyber-systems/edendocs` |
| MPL/attribution in-image | ✓ COPYING + THIRDPARTYLICENSES staged (`docker-publish.yml:81-87`) and byte-verified in-image (`:149-175`) |
| Objective commits ancestors of HEAD (`195b0988290`) | ✓ all 8 (b0f64b7…159f4d1) are ancestors |

### Functional Verification

Not run. This is CI/packaging infrastructure (no app UI, no web/mobile surface to drive). The functional proof is the `docker-publish.yml` green run itself, which requires a Linux docker engine and is a user-approved rollout — see Human Verification. Playwright/Maestro N/A.

### Human Verification Required

The single outstanding item is the deferred-by-design live publish. Exact commands and expectations are in the frontmatter `human_verification` block. In order:

1. **(First run only) Populate the engine-pin mirror asset** so `fetch-engine-pinned.sh` can pull the sha256-pinned tarball:
   - `gh release create engine-pin --repo AO-Cyber-Systems/EdenDocs --notes '…'`
   - `gh release upload engine-pin engine-main-assets.tar.gz --repo AO-Cyber-Systems/EdenDocs`
2. **User-approved rollout** — `git push origin eden-main` (triggers the pipeline).
3. **Verify** — `cosign verify ghcr.io/ao-cyber-systems/edendocs@<digest> --certificate-identity-regexp='https://github.com/AO-Cyber-Systems/EdenDocs/.*' --certificate-oidc-issuer='https://token.actions.githubusercontent.com'`.

### Gaps Summary

No gaps. All artifacts across the four TRDs exist, are substantive, and are correctly wired to one another (build-release → instdir → Dockerfile → CI build → boot-health → proxy-smoke → push → cosign). Every success criterion is satisfied at the authoring level; the only thing standing between "authored" and "proven live" is the approval-gated `git push` (plus one-time mirror population), which is deferred by design and cannot run on this macOS host. Terminal status is therefore `human_needed`, not `gaps_found`.

---

*Verified: 2026-07-09T08:22:16Z*
*Verifier: Claude (verifier)*
