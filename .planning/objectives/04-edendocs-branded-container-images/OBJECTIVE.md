---
work: infra
tdd: skip
---

# EdenDocs-Branded Container Images

## Goal

Customers can deploy a signed, EdenDocs-branded container image, built on the
from-source pattern, that survives real-world reverse-proxy conditions rather
than only direct-connect access. The runnable verification gates in each TRD
(shellcheck/hadolint/actionlint lint, structural greps, byte-integrity checks,
and the docker-publish.yml green run: build → boot-health → >60s proxy smoke →
trivy → push → keyless cosign) ARE the tests — this is CI/packaging infra, not
app UI work, so no Maestro/widget TDD applies.

Requirements: DIST-01, DIST-02, DIST-03.

---
*Created: 2026-07-09 (auto-scaffold via bootstrapObjectiveMd)*
