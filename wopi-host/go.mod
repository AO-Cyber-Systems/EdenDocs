module go.aocyber.ai/edendocs/wopi-host

go 1.27.1

require (
	github.com/coreos/go-oidc/v3 v3.20.0
	github.com/golang-jwt/jwt/v5 v5.3.1
	// wopi-host requires org GitHub access (git config url.insteadOf + GOPRIVATE)
	// even though EdenDocs itself is a public repo — see the runbook in
	// TRD 03-05 for the required local/CI setup. This is an accepted org
	// mandate: never hand-roll OIDC (3-RESEARCH.md).
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.4-0.20250319132907-e064f32e3674
	go.aocyber.ai/eden-platform-go v0.1.0
	golang.org/x/oauth2 v0.36.0
)

require (
	github.com/go-jose/go-jose/v4 v4.1.4 // indirect
	golang.org/x/net v0.53.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
)
