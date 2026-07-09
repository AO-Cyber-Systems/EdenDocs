# State Archive

Append-only log. Written by df-tools `add-decision` and `record-metric`.
STATE.md stays lean; this file grows over time.

## Decisions

- [Objective 03-aoid-authentication-integration-oidc]: eden-platform-go pinned at v0.0.0-20260708235425-c5fd1ee7cedb (resolved @latest); no replace directive, no vendoring — private-module access via global git insteadOf + GOPRIVATE
- [Objective 03-aoid-authentication-integration-oidc]: WOPI access_token is a bare opaque uuid.NewString() with injected-clock TTL (Pitfall 2 boundary); Config is a fmt.Stringer with always-redacted secrets; AUTH-01/AUTH-02 deliberately NOT marked complete after 03-01 — later TRDs (03-02/03/04) finish their requirement text
- [Objective 03-aoid-authentication-integration-oidc]: e2e-probe sends Origin: <coolwsd origin> on WS dials — coolwsd allowedOrigin() 403s Origin-less upgrades; faithful browser emulation since cool.html is served by coolwsd
- [Objective 03]: Real-AOID human-verify PASSED 2026-07-09 (issuer https://localhost:8443); AOID login is federation-only — dev runs need an upstream IdP + aoid fix 9d53d31 (federation-admin KMS Encrypt wiring, local/unpushed)

## Performance Metrics

| Objective | Duration | Tasks | Files |
|-----------|----------|-------|-------|
| Objective 03-aoid-authentication-integration-oidc P01 | 25min | 3 tasks | 9 files |
| Objective 03-aoid-authentication-integration-oidc P03 | ~30min | 3 tasks | 9 files |
| Objective 03-aoid-authentication-integration-oidc P03-04 | 2h | 3 tasks | 7 files |
| Objective 03 P05 | ~3h | 2 tasks | 6 files |
| Objective 04 P02 | 3min | 2 tasks | 3 files |

