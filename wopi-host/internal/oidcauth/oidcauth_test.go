/*
 * Copyright 2026 AO Cyber Systems
 *
 * SPDX-License-Identifier: MPL-2.0
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */

package oidcauth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	oidcrp "github.com/aocybersystems/eden-platform-go/platform/oidcrp"

	"github.com/AO-Cyber-Systems/EdenDocs/wopi-host/internal/config"
	"github.com/AO-Cyber-Systems/EdenDocs/wopi-host/internal/oidcauth"
	"github.com/AO-Cyber-Systems/EdenDocs/wopi-host/internal/oidctest"
	"github.com/AO-Cyber-Systems/EdenDocs/wopi-host/internal/session"
)

// --- test harness -----------------------------------------------------

type harness struct {
	rpURL string
	cfg   config.Config
}

// newHarness wires a fake AOID IdP (with the given options) and a real
// Authenticator behind a trivial protected page, both on httptest servers,
// and returns the RP's base URL plus the resolved config used to build it
// (tests need cfg.StateHMACKey to hand-craft signed states directly).
func newHarness(t *testing.T, idpOpts ...oidctest.Option) *harness {
	t.Helper()

	mux := http.NewServeMux()
	rpSrv := httptest.NewServer(mux)
	t.Cleanup(rpSrv.Close)

	idp := oidctest.New(idpOpts...)
	idpSrv := httptest.NewServer(idp.Handler())
	t.Cleanup(idpSrv.Close)
	idp.SetIssuer(idpSrv.URL)

	cfg := config.Config{
		AOIDIssuer:   idpSrv.URL,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		StateHMACKey: []byte("test-state-hmac-key-0123456789ab"),
		PublicURL:    rpSrv.URL,
	}

	sessions := session.NewStore(time.Now)
	auth, err := oidcauth.NewAuthenticator(context.Background(), cfg, sessions)
	if err != nil {
		t.Fatalf("NewAuthenticator: %v", err)
	}

	mux.HandleFunc("/login", auth.LoginHandler)
	mux.HandleFunc("/callback", auth.CallbackHandler)
	mux.HandleFunc("/logout", auth.LogoutHandler)
	mux.Handle("/protected", auth.RequireLogin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, ok := oidcauth.FromContext(r.Context())
		if !ok {
			http.Error(w, "missing identity in context", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(identity)
	})))

	return &harness{rpURL: rpSrv.URL, cfg: cfg}
}

// newLoginClient returns an http.Client with a cookie jar that does NOT
// auto-follow redirects, so tests can inspect and mutate each hop.
func newLoginClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar.New: %v", err)
	}
	return &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func mustGet(t *testing.T, client *http.Client, rawURL string) *http.Response {
	t.Helper()
	resp, err := client.Get(rawURL)
	if err != nil {
		t.Fatalf("GET %s: %v", rawURL, err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

// driveToCallback performs the first three hops of a real login flow
// (protected page -> /login -> IdP authorize -> back to /callback) and
// returns the absolute callback URL (carrying a real code + real signed
// state) without fetching it, so callers can mutate it or fetch it
// directly.
func driveToCallback(t *testing.T, h *harness, client *http.Client) string {
	t.Helper()
	r1 := mustGet(t, client, h.rpURL+"/protected")
	if r1.StatusCode != http.StatusFound {
		t.Fatalf("GET /protected: status %d, want 302", r1.StatusCode)
	}
	r2 := mustGet(t, client, h.rpURL+r1.Header.Get("Location"))
	if r2.StatusCode != http.StatusFound {
		t.Fatalf("GET /login: status %d, want 302", r2.StatusCode)
	}
	r3 := mustGet(t, client, r2.Header.Get("Location"))
	if r3.StatusCode != http.StatusFound {
		t.Fatalf("GET idp authorize: status %d, want 302", r3.StatusCode)
	}
	callbackURL := r3.Header.Get("Location")
	if !strings.Contains(callbackURL, "/callback?") {
		t.Fatalf("expected IdP redirect to /callback, got %q", callbackURL)
	}
	return callbackURL
}

func tamperQueryParam(t *testing.T, rawURL, param string) string {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse %s: %v", rawURL, err)
	}
	q := u.Query()
	v := q.Get(param)
	if v == "" {
		t.Fatalf("%s has no %s param to tamper", rawURL, param)
	}
	last := v[len(v)-1]
	replacement := "a"
	if last == 'a' {
		replacement = "b"
	}
	q.Set(param, v[:len(v)-1]+replacement)
	u.RawQuery = q.Encode()
	return u.String()
}

func is4xx(status int) bool { return status >= 400 && status < 500 }

// --- test cases ---------------------------------------------------------

// Case 1: full happy path.
func TestFullHappyPathLoginFlow(t *testing.T) {
	h := newHarness(t)
	client := newLoginClient(t)

	r1 := mustGet(t, client, h.rpURL+"/protected")
	if r1.StatusCode != http.StatusFound {
		t.Fatalf("GET /protected: status %d, want 302", r1.StatusCode)
	}
	loginLoc := r1.Header.Get("Location")
	if !strings.HasPrefix(loginLoc, "/login?") {
		t.Fatalf("expected redirect to /login, got %q", loginLoc)
	}

	r2 := mustGet(t, client, h.rpURL+loginLoc)
	if r2.StatusCode != http.StatusFound {
		t.Fatalf("GET /login: status %d, want 302", r2.StatusCode)
	}
	authorizeLoc := r2.Header.Get("Location")
	authURL, err := url.Parse(authorizeLoc)
	if err != nil {
		t.Fatalf("parse authorize URL: %v", err)
	}
	q := authURL.Query()
	if q.Get("code_challenge") == "" {
		t.Error("authorize URL missing code_challenge")
	}
	if q.Get("code_challenge_method") != "S256" {
		t.Errorf("code_challenge_method = %q, want S256", q.Get("code_challenge_method"))
	}
	if q.Get("nonce") == "" {
		t.Error("authorize URL missing nonce")
	}
	if !strings.Contains(q.Get("state"), ".") {
		t.Errorf("state %q does not look like a signed HMAC token", q.Get("state"))
	}

	r3 := mustGet(t, client, authorizeLoc)
	if r3.StatusCode != http.StatusFound {
		t.Fatalf("GET idp authorize: status %d, want 302", r3.StatusCode)
	}
	callbackLoc := r3.Header.Get("Location")
	if !strings.Contains(callbackLoc, "/callback?") {
		t.Fatalf("expected IdP redirect to /callback, got %q", callbackLoc)
	}

	r4 := mustGet(t, client, callbackLoc)
	if r4.StatusCode != http.StatusFound {
		t.Fatalf("GET /callback: status %d, want 302", r4.StatusCode)
	}
	if got := r4.Header.Get("Location"); got != "/protected" {
		t.Errorf("post-login redirect = %q, want /protected", got)
	}
	if len(r4.Cookies()) == 0 {
		t.Fatal("/callback did not set a session cookie")
	}

	r5 := mustGet(t, client, h.rpURL+"/protected")
	if r5.StatusCode != http.StatusOK {
		t.Fatalf("GET /protected after login: status %d, want 200", r5.StatusCode)
	}
	var id session.Identity
	if err := json.NewDecoder(r5.Body).Decode(&id); err != nil {
		t.Fatalf("decode identity: %v", err)
	}
	if id.Subject != oidctest.DefaultIdentity().Subject {
		t.Errorf("Subject = %q, want %q", id.Subject, oidctest.DefaultIdentity().Subject)
	}
}

// Case 2: identity mapping fallback chain (name -> preferred_username ->
// email), IsAdmin always false.
func TestIdentityMappingFallbackChain(t *testing.T) {
	for _, tc := range []struct {
		name     string
		identity oidctest.Identity
		wantName string
	}{
		{
			name:     "name present wins",
			identity: oidctest.Identity{Subject: "sub-1", Name: "Full Name", PreferredUsername: "handle", Email: "a@example.com", EmailVerified: true},
			wantName: "Full Name",
		},
		{
			name:     "name absent falls back to preferred_username",
			identity: oidctest.Identity{Subject: "sub-2", PreferredUsername: "handle2", Email: "b@example.com", EmailVerified: true},
			wantName: "handle2",
		},
		{
			name:     "name and preferred_username absent falls back to email",
			identity: oidctest.Identity{Subject: "sub-3", Email: "c@example.com", EmailVerified: true},
			wantName: "c@example.com",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t, oidctest.WithIdentity(tc.identity))
			client := newLoginClient(t)

			callbackURL := driveToCallback(t, h, client)
			cb := mustGet(t, client, callbackURL)
			if cb.StatusCode != http.StatusFound {
				t.Fatalf("GET /callback: status %d, want 302", cb.StatusCode)
			}

			resp := mustGet(t, client, h.rpURL+"/protected")
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("GET /protected: status %d, want 200", resp.StatusCode)
			}
			var id session.Identity
			if err := json.NewDecoder(resp.Body).Decode(&id); err != nil {
				t.Fatalf("decode identity: %v", err)
			}
			if id.Subject != tc.identity.Subject {
				t.Errorf("Subject = %q, want %q", id.Subject, tc.identity.Subject)
			}
			if id.Name != tc.wantName {
				t.Errorf("Name = %q, want %q", id.Name, tc.wantName)
			}
			if id.IsAdmin {
				t.Error("IsAdmin must always be false in v1 (3-RESEARCH.md Pitfall 1)")
			}
		})
	}
}

// Case 3: tampered state is rejected, no session created.
func TestCallbackTamperedStateRejected(t *testing.T) {
	h := newHarness(t)
	client := newLoginClient(t)

	callbackURL := driveToCallback(t, h, client)
	tampered := tamperQueryParam(t, callbackURL, "state")

	resp := mustGet(t, client, tampered)
	if !is4xx(resp.StatusCode) {
		t.Fatalf("tampered state: status %d, want 4xx", resp.StatusCode)
	}
	if len(resp.Cookies()) != 0 {
		t.Fatal("tampered state must not set a session cookie")
	}

	guard := mustGet(t, client, h.rpURL+"/protected")
	if guard.StatusCode != http.StatusFound {
		t.Fatalf("protected page after rejected callback: status %d, want 302 (no session)", guard.StatusCode)
	}
}

// Case 4: id_token nonce mismatching the in-flight nonce is rejected.
func TestCallbackNonceMismatchRejected(t *testing.T) {
	h := newHarness(t, oidctest.WithBadNonce())
	client := newLoginClient(t)

	callbackURL := driveToCallback(t, h, client)
	resp := mustGet(t, client, callbackURL)
	if !is4xx(resp.StatusCode) {
		t.Fatalf("nonce mismatch: status %d, want 4xx", resp.StatusCode)
	}
	if len(resp.Cookies()) != 0 {
		t.Fatal("nonce mismatch must not set a session cookie")
	}
}

// Case 5: replaying the same callback URL twice is rejected the second
// time (single-use in-flight Pop).
func TestCallbackReplayRejected(t *testing.T) {
	h := newHarness(t)
	client := newLoginClient(t)

	callbackURL := driveToCallback(t, h, client)

	first := mustGet(t, client, callbackURL)
	if first.StatusCode != http.StatusFound {
		t.Fatalf("first callback: status %d, want 302", first.StatusCode)
	}

	second := mustGet(t, client, callbackURL)
	if !is4xx(second.StatusCode) {
		t.Fatalf("replayed callback: status %d, want 4xx", second.StatusCode)
	}
}

// Case 6: an expired signed state is rejected.
func TestCallbackExpiredStateRejected(t *testing.T) {
	h := newHarness(t)
	client := newLoginClient(t)

	st := oidcrp.State{
		Tenant:    "default",
		Idp:       "aoid",
		ReturnURL: "/protected",
		Nonce:     "expired-nonce",
		CreatedAt: time.Now().Add(-11 * time.Minute).Unix(),
	}
	signed := oidcrp.SignState(h.cfg.StateHMACKey, st)

	resp := mustGet(t, client, h.rpURL+"/callback?code=dummy-code&state="+url.QueryEscape(signed))
	if !is4xx(resp.StatusCode) {
		t.Fatalf("expired state: status %d, want 4xx", resp.StatusCode)
	}
	if len(resp.Cookies()) != 0 {
		t.Fatal("expired state must not set a session cookie")
	}
}

// Case 7: a token response missing id_token is rejected.
func TestCallbackMissingIDTokenRejected(t *testing.T) {
	h := newHarness(t, oidctest.WithoutIDToken())
	client := newLoginClient(t)

	callbackURL := driveToCallback(t, h, client)
	resp := mustGet(t, client, callbackURL)
	if !is4xx(resp.StatusCode) {
		t.Fatalf("missing id_token: status %d, want 4xx", resp.StatusCode)
	}
	if len(resp.Cookies()) != 0 {
		t.Fatal("missing id_token must not set a session cookie")
	}
}

// Case 8: an off-origin or protocol-relative ReturnURL never survives —
// the post-login redirect always falls back to "/".
func TestLoginReturnURLAllowlisted(t *testing.T) {
	for _, tc := range []struct {
		name string
		ret  string
	}{
		{"absolute-off-origin", "https://evil.example"},
		{"protocol-relative", "//evil.example"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			client := newLoginClient(t)

			r1 := mustGet(t, client, h.rpURL+"/login?return="+url.QueryEscape(tc.ret))
			if r1.StatusCode != http.StatusFound {
				t.Fatalf("GET /login: status %d, want 302", r1.StatusCode)
			}
			r2 := mustGet(t, client, r1.Header.Get("Location"))
			if r2.StatusCode != http.StatusFound {
				t.Fatalf("GET idp authorize: status %d, want 302", r2.StatusCode)
			}
			callbackURL := r2.Header.Get("Location")
			r3 := mustGet(t, client, callbackURL)
			if r3.StatusCode != http.StatusFound {
				t.Fatalf("GET /callback: status %d, want 302", r3.StatusCode)
			}
			if got := r3.Header.Get("Location"); got != "/" {
				t.Errorf("post-login redirect = %q, want fallback to /", got)
			}
		})
	}
}

// Case 9: RequireLogin without a session cookie redirects to /login with a
// return param carrying the original path.
func TestRequireLoginRedirectsWithReturnParam(t *testing.T) {
	h := newHarness(t)
	client := newLoginClient(t)

	resp := mustGet(t, client, h.rpURL+"/protected")
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status %d, want 302", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	if !strings.HasPrefix(loc, "/login?return=") {
		t.Fatalf("Location = %q, want prefix /login?return=", loc)
	}
	u, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("parse Location: %v", err)
	}
	if got := u.Query().Get("return"); got != "/protected" {
		t.Errorf("return param = %q, want /protected", got)
	}
}

// Case 10: logout clears the session; a subsequent protected GET
// redirects to /login again.
func TestLogoutClearsSession(t *testing.T) {
	h := newHarness(t)
	client := newLoginClient(t)

	callbackURL := driveToCallback(t, h, client)
	cb := mustGet(t, client, callbackURL)
	if cb.StatusCode != http.StatusFound {
		t.Fatalf("callback: status %d, want 302", cb.StatusCode)
	}

	ok := mustGet(t, client, h.rpURL+"/protected")
	if ok.StatusCode != http.StatusOK {
		t.Fatalf("protected after login: status %d, want 200", ok.StatusCode)
	}

	logout := mustGet(t, client, h.rpURL+"/logout")
	if logout.StatusCode != http.StatusFound || logout.Header.Get("Location") != "/" {
		t.Fatalf("logout: status %d location %q, want 302 to /", logout.StatusCode, logout.Header.Get("Location"))
	}

	after := mustGet(t, client, h.rpURL+"/protected")
	if after.StatusCode != http.StatusFound {
		t.Fatalf("protected after logout: status %d, want 302 (session cleared)", after.StatusCode)
	}
}
