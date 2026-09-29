---
generated: "2026-09-29"
profile: "general"
profile_source: file
components: ["wopi-host/"]
unsupported_areas: ["browser/", "cypress_test/", "cypress_test/eslint_plugin/", "engine/rust_uno/", "qt/test/"]
counts: { gap: 13, weak: 1, info: 20 }
---

# Stack Report: edendocs

Proposals only — nothing here has been applied. Review, then change CI/runners yourself.

## Gaps

| ID | Component | Finding | Evidence | Proposal |
|---|---|---|---|---|
| JS-AUDIT | browser/ | CI has no dependency audit | browser/package.json | Audit dependencies in CI. `npm audit --audit-level=high` |
| JS-LINT | browser/ | CI has no JS/TS linter | browser/package.json | Lint in CI. `npx eslint .` |
| JS-TYPE | browser/ | CI has no TypeScript type check | browser/package.json | Type-check in CI. `npx tsc --noEmit` |
| JS-AUDIT | cypress_test/ | CI has no dependency audit | cypress_test/package.json | Audit dependencies in CI. `npm audit --audit-level=high` |
| JS-LINT | cypress_test/ | CI has no JS/TS linter | cypress_test/package.json | Lint in CI. `npx eslint .` |
| JS-AUDIT | cypress_test/eslint_plugin/ | CI has no dependency audit | cypress_test/eslint_plugin/package.json | Audit dependencies in CI. `npm audit --audit-level=high` |
| JS-LINT | cypress_test/eslint_plugin/ | CI has no JS/TS linter | cypress_test/eslint_plugin/package.json | Lint in CI. `npx eslint .` |
| JS-AUDIT | qt/test/ | CI has no dependency audit | qt/test/package.json | Audit dependencies in CI. `npm audit --audit-level=high` |
| JS-LINT | qt/test/ | CI has no JS/TS linter | qt/test/package.json | Lint in CI. `npx eslint .` |
| JS-TYPE | qt/test/ | CI has no TypeScript type check | qt/test/package.json | Type-check in CI. `npx tsc --noEmit` |
| GO-FMT | wopi-host/ | CI has no Go format check | wopi-host/go.mod | Fail CI on unformatted files; a bare `gofmt -l` lists them and exits 0. `test -z "$(gofmt -l .)"` |
| GO-VULN | wopi-host/ | CI has no govulncheck | wopi-host/go.mod | Scan dependencies for known vulnerabilities in CI, on a schedule as well as on push. `govulncheck ./...` |
| LOCAL-MIRROR | wopi-host/ | gates run only in CI, with no local runner target: lint (go vet ./...), test (go test -race ./... -count=1), sast (uses:github/codeql-action/init) | .github/workflows/wopi-host.yml:wopi-host: go vet ./...; .github/workflows/wopi-host.yml:wopi-host: go test -race ./... -count=1; .github/workflows/codeql-analysis.yml:analyze: uses:github/codeql-action/init | Add a task/make/just target per STACK.md key so each CI gate also runs locally. Keys: lint, test, sast. `make lint` |

## Weak

| ID | Component | Finding | Evidence | Proposal |
|---|---|---|---|---|
| GO-TIDY | wopi-host/ | CI has no go.mod tidy check | wopi-host/go.mod | Fail CI when go.mod/go.sum are not tidy (Go >= 1.23). `go mod tidy -diff` |

## Info

| ID | Component | Finding | Evidence | Proposal |
|---|---|---|---|---|
| CI-HYGIENE | (root) | workflow hygiene: build.yml (no permissions:, 4 actions not pinned to a commit SHA); codeql-analysis.yml (no permissions:, 3 actions not pinned to a commit SHA, no timeout-minutes, no concurrency); docker-publish.yml (11 actions not pinned to a commit SHA); notify-website.yml (no permissions:, no timeout-minutes, no concurrency); wopi-host.yml (no permissions:, 2 actions not pinned to a commit SHA) | .github/workflows/build.yml; .github/workflows/codeql-analysis.yml; .github/workflows/docker-publish.yml | Set a least-privilege `permissions:` block, pin actions by SHA, and add `timeout-minutes` and a `concurrency` group. |
| DOCKER-LINT | (root) | CI has no Dockerfile linter (hadolint) | docker/eden/Dockerfile; docker/from-packages/Dockerfile; snap/Dockerfile | Lint Dockerfiles (hadolint must be installed; it is not assumed). `hadolint docker/eden/Dockerfile` |
| DOCKER-PIN | (root) | base images pinned by tag, not digest: debian:stable-slim, ubuntu:24.04 | docker/eden/Dockerfile; docker/from-packages/Dockerfile; snap/Dockerfile | Pin each FROM to an @sha256 digest (keep the tag in a comment for readability). |
| JS-E2E | cypress_test/ | an e2e framework is configured but CI never runs it | cypress_test/package.json | Run the e2e suite in CI. `npx playwright test` |
| GO-COVER | wopi-host/ | CI runs Go tests without coverage | wopi-host/go.mod | Record coverage (no threshold is implied). `go test -coverprofile=coverage.out ./...` |
| GO-FIX | wopi-host/ | CI has no `go fix -diff` | wopi-host/go.mod | Report pending modernizer fixes; `go fix -diff` exits non-zero on a diff. `go fix -diff ./...` |
| GO-LINT | wopi-host/ | CI has no Go linter (golangci-lint / staticcheck) | wopi-host/go.mod | Add a linter beyond vet; staticcheck needs no config. `staticcheck ./...` |

## Draft notes

| Key | Candidate | Status | Source |
|---|---|---|---|
| build | `./scripts/eden/build-deps.sh` | off_stack | ci |
| build | `./scripts/eden/build-release.sh` | off_stack | ci |
| build | `./scripts/eden/build.sh` | off_stack | ci |
| build | `make build` | off_stack | runner |
| test | `./scripts/eden/proxy-smoke-test.sh "${{ env.IMAGE }}:scan"` | off_stack | ci |
| test | `./scripts/eden/smoke-test.sh` | off_stack | ci |
| — | — | info | — |
| test | `npm test` | sub_area | manifest |
| — | — | info | — |
| — | — | info | — |
| — | — | info | — |
| — | — | info | — |
| test | `npm test` | sub_area | manifest |
