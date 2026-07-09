/*
 * Copyright 2026 AO Cyber Systems
 *
 * SPDX-License-Identifier: MPL-2.0
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */

// Package oidcauth is the AOID OIDC relying-party surface of the WOPI
// host: /login, /callback, /logout, and a RequireLogin middleware. It is
// a thin consumer of github.com/aocybersystems/eden-platform-go's
// platform/oidcrp package — the org mandate is to NEVER hand-roll OIDC
// state/nonce/PKCE handling (3-RESEARCH.md).
//
// AOID tokens (id_token / access_token / refresh_token) never leave this
// package: CallbackHandler maps verified id_token claims into a
// session.Identity and mints a server-side web session cookie. coolwsd and
// the browser never see an AOID token (3-RESEARCH.md Pitfall 2 — the WOPI
// access_token minted later by internal/session, TRD 03-03, is a
// completely separate, unrelated credential space).
package oidcauth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	oidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"golang.org/x/oauth2"

	oidcrp "github.com/aocybersystems/eden-platform-go/platform/oidcrp"

	"github.com/AO-Cyber-Systems/EdenDocs/wopi-host/internal/config"
	"github.com/AO-Cyber-Systems/EdenDocs/wopi-host/internal/session"
)

const (
	// cookieName is the WOPI host's browser login-session cookie.
	// SameSite=Lax is REQUIRED (not merely preferred): the OIDC redirect
	// back from the IdP is a top-level GET, and SameSite=Strict would
	// drop the cookie on that navigation.
	cookieName = "edendocs_wopi_sid"

	// tenantID / idpID are the single-tenant reference host's fixed
	// oidcrp.State.Tenant / .Idp values. oidcrp.VerifyState rejects empty
	// values here as a zero-value-attack defense, so these must always be
	// set even though this deployment has exactly one tenant and one IdP.
	tenantID = "default"
	idpID    = "aoid"

	stateMaxAge   = 10 * time.Minute
	inFlightTTL   = 10 * time.Minute
	webSessionTTL = 12 * time.Hour
)

type contextKey int

const identityContextKey contextKey = 0

// FromContext extracts the session.Identity RequireLogin injected into the
// request context. 03-03's WOPI protocol handlers and the launch page
// consume this.
func FromContext(ctx context.Context) (session.Identity, bool) {
	id, ok := ctx.Value(identityContextKey).(session.Identity)
	return id, ok
}

// Authenticator implements the AOID OIDC relying party. Construct once at
// process startup via NewAuthenticator — the ProviderCache/VerifierCache
// discovery+key-fetch happens exactly once here, never per-request (org
// sso-integration.md "Pitfall 4": raw, uncached go-oidc discovery or
// verifier construction on every request ignores AOID's JWKS
// max-age=300 caching contract).
type Authenticator struct {
	oauthCfg     *oauth2.Config
	verifier     *oidc.IDTokenVerifier
	stateKey     []byte
	inflight     oidcrp.InFlightStore
	sessions     *session.Store
	secureCookie bool
}

// NewAuthenticator discovers cfg.AOIDIssuer, builds a VerifierCache-backed
// ID token verifier, and returns a ready-to-mount Authenticator. ctx bounds
// the discovery round trip only.
func NewAuthenticator(ctx context.Context, cfg config.Config, sessions *session.Store) (*Authenticator, error) {
	if cfg.AOIDIssuer == "" {
		return nil, errors.New("oidcauth: config.AOIDIssuer is required")
	}
	if cfg.ClientID == "" {
		return nil, errors.New("oidcauth: config.ClientID is required")
	}
	if sessions == nil {
		return nil, errors.New("oidcauth: sessions store is required")
	}

	providers := oidcrp.NewProviderCache()
	provider, err := providers.Get(ctx, idpID, cfg.AOIDIssuer)
	if err != nil {
		return nil, fmt.Errorf("oidcauth: discovering AOID issuer %q: %w", cfg.AOIDIssuer, err)
	}

	verifiers := oidcrp.NewVerifierCache()
	verifier := verifiers.Get(idpID, provider, cfg.ClientID, nil)

	oauthCfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  strings.TrimRight(cfg.PublicURL, "/") + "/callback",
		// AOID scope set per 3-RESEARCH.md Pattern 1 — email/name/
		// preferred_username all ride in the id_token under these scopes,
		// so CallbackHandler never needs a separate userinfo round trip.
		Scopes: []string{oidc.ScopeOpenID, "profile", "email"},
	}

	return &Authenticator{
		oauthCfg:     oauthCfg,
		verifier:     verifier,
		stateKey:     cfg.StateHMACKey,
		inflight:     oidcrp.NewInMemoryInFlightStore(),
		sessions:     sessions,
		secureCookie: strings.HasPrefix(cfg.PublicURL, "https://"),
	}, nil
}

// LoginHandler starts the Authorization Code + PKCE (S256) flow: it mints
// a nonce and PKCE verifier, stores them in the in-flight store keyed by
// nonce, signs an oidcrp.State carrying the allowlisted return path, and
// redirects to the IdP's authorization endpoint.
func (a *Authenticator) LoginHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	nonce := uuid.NewString()
	pkceVerifier := oauth2.GenerateVerifier()
	returnURL := sanitizeReturn(r.URL.Query().Get("return"))

	st := oidcrp.State{
		Tenant:    tenantID,
		Idp:       idpID,
		ReturnURL: returnURL,
		Nonce:     nonce,
		CreatedAt: time.Now().Unix(),
	}
	signedState := oidcrp.SignState(a.stateKey, st)

	payload := oidcrp.InFlightPayload{
		PKCEVerifier: pkceVerifier,
		StoredNonce:  nonce,
		TenantID:     tenantID,
		IdpID:        idpID,
		ReturnURL:    returnURL,
		CreatedAt:    time.Now(),
	}
	if err := a.inflight.Put(ctx, nonce, payload, inFlightTTL); err != nil {
		http.Error(w, "oidcauth: failed to start login", http.StatusInternalServerError)
		return
	}

	authURL, err := oidcrp.BuildAuthURL(a.oauthCfg, signedState, nonce, pkceVerifier, nil)
	if err != nil {
		http.Error(w, "oidcauth: failed to build authorization URL", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, authURL, http.StatusFound)
}

// CallbackHandler completes the flow: verifies the signed state, pops the
// single-use in-flight record by nonce, exchanges the code + verifies the
// id_token (nonce-bound) via oidcrp.ExchangeAndVerify, maps claims to a
// session.Identity, mints a web session cookie, and redirects to the
// allowlisted return path.
func (a *Authenticator) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()

	st, err := oidcrp.VerifyState(a.stateKey, q.Get("state"), stateMaxAge)
	if err != nil {
		http.Error(w, "oidcauth: invalid or expired state", http.StatusBadRequest)
		return
	}

	payload, err := a.inflight.Pop(ctx, st.Nonce)
	if err != nil {
		http.Error(w, "oidcauth: expired or already-used login attempt", http.StatusBadRequest)
		return
	}

	code := q.Get("code")
	if code == "" {
		http.Error(w, "oidcauth: missing code", http.StatusBadRequest)
		return
	}

	_, _, claims, err := oidcrp.ExchangeAndVerify(ctx, a.oauthCfg, a.verifier, code, payload.PKCEVerifier, payload.StoredNonce)
	if err != nil {
		http.Error(w, "oidcauth: token exchange or verification failed", http.StatusUnauthorized)
		return
	}

	identity := mapClaims(claims)

	sid := a.sessions.CreateWebSession(identity, webSessionTTL)
	a.setSessionCookie(w, sid, webSessionTTL)

	http.Redirect(w, r, sanitizeReturn(payload.ReturnURL), http.StatusFound)
}

// LogoutHandler deletes the server-side session (if any) and expires the
// cookie, then redirects home.
func (a *Authenticator) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieName); err == nil {
		a.sessions.DeleteWebSession(c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   a.secureCookie,
		MaxAge:   -1,
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

// RequireLogin is middleware that resolves the session cookie to a
// session.Identity (injected into the request context, see FromContext)
// or redirects to /login?return=<original request URI>.
func (a *Authenticator) RequireLogin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(cookieName)
		if err != nil {
			a.redirectToLogin(w, r)
			return
		}
		identity, ok := a.sessions.LookupWebSession(c.Value)
		if !ok {
			a.redirectToLogin(w, r)
			return
		}
		ctx := context.WithValue(r.Context(), identityContextKey, identity)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *Authenticator) redirectToLogin(w http.ResponseWriter, r *http.Request) {
	ret := url.QueryEscape(r.URL.RequestURI())
	http.Redirect(w, r, "/login?return="+ret, http.StatusFound)
}

func (a *Authenticator) setSessionCookie(w http.ResponseWriter, sid string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    sid,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   a.secureCookie,
		Expires:  time.Now().Add(ttl),
	})
}

// mapClaims implements the AUTH-03 CheckFileInfo<->AOID claim mapping:
// UserId <- sub; UserFriendlyName <- name, falling back to
// preferred_username, falling back to email (AOID's own userinfo endpoint
// implements this same chain server-side — the RP replicates it directly
// over id_token claims, avoiding an extra userinfo round trip since the
// scopes we request already put email/name/preferred_username in the
// id_token). IsAdminUser is hardcoded false in v1: AOID's id_token carries
// no admin/role claim usable by a generic OIDC RP (3-RESEARCH.md
// Pitfall 1) — a v2 would source this from an `ent[]` entitlement claim
// scoped to aud=aoedge instead.
func mapClaims(claims map[string]any) session.Identity {
	return session.Identity{
		Subject: stringClaim(claims, "sub"),
		Name:    firstNonEmpty(stringClaim(claims, "name"), stringClaim(claims, "preferred_username"), stringClaim(claims, "email")),
		Email:   stringClaim(claims, "email"),
		IsAdmin: false,
	}
}

func stringClaim(claims map[string]any, key string) string {
	v, _ := claims[key].(string)
	return v
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// sanitizeReturn allowlists a caller-supplied post-login redirect target to
// same-origin paths only. oidcrp.State.ReturnURL is HMAC-signed but
// deliberately NOT validated as a URL by oidcrp itself — callers MUST
// allowlist before issuing the redirect, or a signed state carrying an
// attacker-supplied absolute/protocol-relative URL becomes an open
// redirect.
func sanitizeReturn(raw string) string {
	if raw == "" {
		return "/"
	}
	if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") {
		return "/"
	}
	return raw
}
