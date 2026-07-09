module github.com/AO-Cyber-Systems/EdenDocs/wopi-host

go 1.26.1

// NOTE: github.com/aocybersystems/eden-platform-go is a PRIVATE org module
// (real repo: github.com/AO-Cyber-Systems/eden-platform-go). Building
// wopi-host requires org GitHub access (git config url.insteadOf + GOPRIVATE)
// even though EdenDocs itself is a public repo — see the runbook in
// TRD 03-05 for the required local/CI setup. This is an accepted org
// mandate: never hand-roll OIDC (3-RESEARCH.md).
require github.com/google/uuid v1.6.0

require (
	github.com/aocybersystems/eden-platform-go v0.0.0-20260708235425-c5fd1ee7cedb
	github.com/coreos/go-oidc/v3 v3.20.0
	github.com/golang-jwt/jwt/v5 v5.3.1
	golang.org/x/oauth2 v0.36.0
)

require (
	github.com/go-jose/go-jose/v4 v4.1.4 // indirect
	golang.org/x/sync v0.20.0 // indirect
)
