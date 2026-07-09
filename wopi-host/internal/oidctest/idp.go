/*
 * Copyright 2026 AO Cyber Systems
 *
 * SPDX-License-Identifier: MPL-2.0
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */

// Package oidctest provides a hand-written, deterministic fake AOID OIDC
// issuer for use in tests (in-process, mounted on an httptest.Server) and
// in the standalone cmd/fake-aoid binary (TRD 03-04's e2e script). It
// mirrors AOID's real discovery/JWKS/authorize/token/userinfo endpoint
// shapes (verified 2026-07-08 against
// /Users/justin/dev/aoid/internal/oauth/http/discovery.go) closely enough
// that production oidcauth code paths are identical whether they talk to
// this fake or real AOID.
//
// There is no login UI: /oauth/authorize auto-approves every request and
// immediately redirects back with a code. The canned identity is hand
// written (never LLM-generated or randomized) and can be overridden per
// test via New's functional Options.
package oidctest

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// kid is the fixed key ID advertised in the JWKS and stamped into every
// id_token's JWS header — the two MUST match or signature verification
// fails ("failed to verify signature" — see the TRD's error_recovery
// notes).
const kid = "fake-aoid-1"

// Identity is the canned claim set the fake IdP embeds in id_tokens and
// serves from /oauth/userinfo. Zero-value fields are simply omitted from
// the claims map, which is how tests exercise AUTH-03's UserFriendlyName
// fallback chain (name -> preferred_username -> email).
type Identity struct {
	Subject           string
	Name              string
	PreferredUsername string
	Email             string
	EmailVerified     bool
}

// DefaultIdentity is the hand-written canned identity used when no
// WithIdentity option is supplied — the values the TRD's anti_patterns
// section mandates verbatim.
func DefaultIdentity() Identity {
	return Identity{
		Subject:       "aoid-e2e-user-1",
		Name:          "Eden Test User",
		Email:         "e2e@aocyber.ai",
		EmailVerified: true,
	}
}

// Option configures an IdP at construction time.
type Option func(*IdP)

// WithIdentity overrides the canned identity embedded in id_tokens and
// userinfo responses (e.g. to drop `name` and exercise the
// preferred_username/email fallback chain).
func WithIdentity(id Identity) Option {
	return func(p *IdP) { p.identity = id }
}

// WithoutIDToken makes /oauth/token omit the id_token field from its
// response entirely, exercising oidcrp.ErrMissingIDToken.
func WithoutIDToken() Option {
	return func(p *IdP) { p.omitIDToken = true }
}

// WithBadNonce makes every minted id_token carry a fixed nonce that never
// matches the nonce the caller sent to /oauth/authorize, exercising
// oidcrp.ErrNonceMismatch without disturbing PKCE validation.
func WithBadNonce() Option {
	return func(p *IdP) { p.badNonce = true }
}

// authRequest is what /oauth/authorize records for a single issued code.
type authRequest struct {
	Nonce               string
	CodeChallenge       string
	CodeChallengeMethod string
	ClientID            string
}

// IdP is an in-process fake AOID OIDC issuer.
type IdP struct {
	key         *rsa.PrivateKey
	identity    Identity
	omitIDToken bool
	badNonce    bool

	mu           sync.Mutex
	issuer       string
	codes        map[string]authRequest
	accessTokens map[string]Identity
}

// New generates a fresh RSA-2048 signing key and returns a ready-to-mount
// IdP. Call SetIssuer once the hosting httptest.Server (or net.Listener)
// URL is known — the issuer MUST equal the server's own base URL exactly,
// or go-oidc's `oidc: id token issued by a different provider` check will
// fail.
func New(opts ...Option) *IdP {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(fmt.Sprintf("oidctest: generating RSA key: %v", err))
	}
	p := &IdP{
		key:          key,
		identity:     DefaultIdentity(),
		codes:        make(map[string]authRequest),
		accessTokens: make(map[string]Identity),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// SetIssuer sets the issuer URL embedded in the discovery document and
// every id_token's iss claim. Must be called with the exact base URL the
// hosting server will be reached at.
func (p *IdP) SetIssuer(issuer string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.issuer = strings.TrimRight(issuer, "/")
}

// Issuer returns the currently configured issuer URL.
func (p *IdP) Issuer() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.issuer
}

// Handler mounts every AOID-shaped route on a fresh mux.
func (p *IdP) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/openid-configuration", p.handleDiscovery)
	mux.HandleFunc("GET /.well-known/jwks.json", p.handleJWKS)
	mux.HandleFunc("GET /oauth/authorize", p.handleAuthorize)
	mux.HandleFunc("POST /oauth/token", p.handleToken)
	mux.HandleFunc("GET /oauth/userinfo", p.handleUserinfo)
	return mux
}

func (p *IdP) handleDiscovery(w http.ResponseWriter, _ *http.Request) {
	iss := p.Issuer()
	doc := map[string]any{
		"issuer":                                iss,
		"authorization_endpoint":                iss + "/oauth/authorize",
		"token_endpoint":                        iss + "/oauth/token",
		"userinfo_endpoint":                     iss + "/oauth/userinfo",
		"jwks_uri":                              iss + "/.well-known/jwks.json",
		"response_types_supported":              []string{"code"},
		"code_challenge_methods_supported":      []string{"S256"},
		"scopes_supported":                      []string{"openid", "profile", "email", "offline_access"},
		"claims_supported":                      []string{"sub", "tnt", "email", "email_verified", "name", "preferred_username"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "private_key_jwt", "none"},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(doc)
}

func (p *IdP) handleJWKS(w http.ResponseWriter, _ *http.Request) {
	pub := p.key.PublicKey
	n := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes())
	jwks := map[string]any{
		"keys": []map[string]any{
			{
				"kty": "RSA",
				"use": "sig",
				"alg": "RS256",
				"kid": kid,
				"n":   n,
				"e":   e,
			},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(jwks)
}

// handleAuthorize auto-approves every request (no login UI — deterministic
// test infrastructure) and immediately 302s back to redirect_uri with a
// freshly minted one-time code.
func (p *IdP) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	redirectURI := q.Get("redirect_uri")
	state := q.Get("state")

	dest, err := url.Parse(redirectURI)
	if err != nil || redirectURI == "" {
		http.Error(w, "oidctest: bad redirect_uri", http.StatusBadRequest)
		return
	}

	code := uuid.NewString()
	p.mu.Lock()
	p.codes[code] = authRequest{
		Nonce:               q.Get("nonce"),
		CodeChallenge:       q.Get("code_challenge"),
		CodeChallengeMethod: q.Get("code_challenge_method"),
		ClientID:            q.Get("client_id"),
	}
	p.mu.Unlock()

	dq := dest.Query()
	dq.Set("code", code)
	dq.Set("state", state)
	dest.RawQuery = dq.Encode()
	http.Redirect(w, r, dest.String(), http.StatusFound)
}

func (p *IdP) handleToken(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, "invalid_request")
		return
	}
	if r.Form.Get("grant_type") != "authorization_code" {
		writeOAuthError(w, "unsupported_grant_type")
		return
	}

	code := r.Form.Get("code")
	verifier := r.Form.Get("code_verifier")

	p.mu.Lock()
	req, ok := p.codes[code]
	if ok {
		delete(p.codes, code) // one-time use
	}
	p.mu.Unlock()

	if !ok {
		writeOAuthError(w, "invalid_grant")
		return
	}

	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	if req.CodeChallengeMethod != "S256" || challenge != req.CodeChallenge {
		writeOAuthError(w, "invalid_grant")
		return
	}

	accessToken := "fake-aoid-opaque-" + uuid.NewString()
	p.mu.Lock()
	p.accessTokens[accessToken] = p.identity
	p.mu.Unlock()

	resp := map[string]any{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   300,
	}

	if !p.omitIDToken {
		idToken, err := p.mintIDToken(req.ClientID, req.Nonce)
		if err != nil {
			http.Error(w, "oidctest: mint id_token: "+err.Error(), http.StatusInternalServerError)
			return
		}
		resp["id_token"] = idToken
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (p *IdP) mintIDToken(clientID, nonce string) (string, error) {
	if p.badNonce {
		nonce = "wrong-nonce-does-not-match-request"
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss":            p.Issuer(),
		"aud":            clientID,
		"sub":            p.identity.Subject,
		"email":          p.identity.Email,
		"email_verified": p.identity.EmailVerified,
		"nonce":          nonce,
		"iat":            now.Unix(),
		"exp":            now.Add(5 * time.Minute).Unix(),
	}
	if p.identity.Name != "" {
		claims["name"] = p.identity.Name
	}
	if p.identity.PreferredUsername != "" {
		claims["preferred_username"] = p.identity.PreferredUsername
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = kid
	return tok.SignedString(p.key)
}

func (p *IdP) handleUserinfo(w http.ResponseWriter, r *http.Request) {
	const prefix = "Bearer "
	authz := r.Header.Get("Authorization")
	if !strings.HasPrefix(authz, prefix) {
		http.Error(w, "oidctest: missing bearer token", http.StatusUnauthorized)
		return
	}

	p.mu.Lock()
	id, ok := p.accessTokens[strings.TrimPrefix(authz, prefix)]
	p.mu.Unlock()
	if !ok {
		http.Error(w, "oidctest: invalid access token", http.StatusUnauthorized)
		return
	}

	resp := map[string]any{
		"sub":            id.Subject,
		"email":          id.Email,
		"email_verified": id.EmailVerified,
	}
	if id.Name != "" {
		resp["name"] = id.Name
	}
	if id.PreferredUsername != "" {
		resp["preferred_username"] = id.PreferredUsername
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func writeOAuthError(w http.ResponseWriter, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": code})
}
