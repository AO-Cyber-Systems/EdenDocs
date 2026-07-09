module github.com/AO-Cyber-Systems/EdenDocs/wopi-host

go 1.26

// NOTE: github.com/aocybersystems/eden-platform-go is a PRIVATE org module
// (real repo: github.com/AO-Cyber-Systems/eden-platform-go). Building
// wopi-host requires org GitHub access (git config url.insteadOf + GOPRIVATE)
// even though EdenDocs itself is a public repo — see the runbook in
// TRD 03-05 for the required local/CI setup. This is an accepted org
// mandate: never hand-roll OIDC (3-RESEARCH.md).
require github.com/google/uuid v1.6.0

require github.com/golang-jwt/jwt/v5 v5.3.1
