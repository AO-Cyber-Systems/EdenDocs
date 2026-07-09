/*
 * Copyright 2026 AO Cyber Systems
 *
 * SPDX-License-Identifier: MPL-2.0
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */

package launch

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/AO-Cyber-Systems/EdenDocs/wopi-host/internal/coolwsd"
	"github.com/AO-Cyber-Systems/EdenDocs/wopi-host/internal/session"
	"github.com/AO-Cyber-Systems/EdenDocs/wopi-host/internal/storage"
)

// minimal discovery fixture for launch tests — only the odt action matters
// here; proof keys are irrelevant to the launch page (browser-facing, no
// proof headers).
const launchDiscoveryXML = `<wopi-discovery>
    <net-zone name="external-http">
        <app name="writer">
            <action default="true" ext="odt" name="edit" urlsrc="http://127.0.0.1:9980/browser/abc123/cool.html?"/>
        </app>
    </net-zone>
</wopi-discovery>`

type env struct {
	handler  http.Handler
	sessions *session.Store
	cfg      Config
}

// newEnv wires a launch handler with an injected identity stub — launch
// must never import oidcauth (TRD decoupling rule); TRD 03-04 injects the
// real oidcauth context helper in production.
func newEnv(t *testing.T, identityFrom func(*http.Request) (session.Identity, bool)) *env {
	t.Helper()

	discoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		_, _ = w.Write([]byte(launchDiscoveryXML))
	}))
	t.Cleanup(discoSrv.Close)
	disco := coolwsd.New(discoSrv.URL, time.Hour)

	sessions := session.NewStore(time.Now)

	dataDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dataDir, "hello.odt"), []byte("hello odt bytes"), 0o644); err != nil {
		t.Fatalf("seed data dir: %v", err)
	}
	store, err := storage.New(dataDir)
	if err != nil {
		t.Fatalf("storage.New: %v", err)
	}

	cfg := Config{
		WopiBaseURL: "http://127.0.0.1:8091",
		TokenTTL:    15 * time.Minute,
	}
	return &env{
		handler:  New(sessions, store, disco, cfg, identityFrom),
		sessions: sessions,
		cfg:      cfg,
	}
}

func authedIdentity(*http.Request) (session.Identity, bool) {
	return session.Identity{
		Subject: "sub-uuid-1234",
		Name:    "Ada Lovelace",
		Email:   "ada@example.test",
	}, true
}

func anonIdentity(*http.Request) (session.Identity, bool) {
	return session.Identity{}, false
}

// --- test-list 16: unauthenticated root -> 302 /login?return=... ---

func TestRoot_Unauthenticated_RedirectsToLogin(t *testing.T) {
	e := newEnv(t, anonIdentity)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("GET / unauthenticated status = %d, want 302", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.HasPrefix(loc, "/login?return=") {
		t.Fatalf("Location = %q, want /login?return=...", loc)
	}
}

// --- test-list 17: /open renders the POST-form iframe embed ---

func TestOpen_RendersPostFormEmbed(t *testing.T) {
	e := newEnv(t, authedIdentity)

	req := httptest.NewRequest(http.MethodGet, "/open?file=hello.odt", nil)
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /open status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()

	// Form method must be POST (Pitfall 6 — never GET query-string).
	if !regexp.MustCompile(`(?i)<form[^>]+method="post"`).MatchString(body) {
		t.Errorf("rendered page missing a method=\"post\" form")
	}

	// action = urlsrc + "WOPISrc=" + urlencoded wopiSrc
	wopiSrc := e.cfg.WopiBaseURL + "/wopi/files/hello.odt"
	wantAction := "http://127.0.0.1:9980/browser/abc123/cool.html?WOPISrc=" + url.QueryEscape(wopiSrc)
	if !strings.Contains(body, `action="`+wantAction+`"`) {
		t.Errorf("form action missing/incorrect:\nwant substring: action=%q\nbody: %s", wantAction, body)
	}

	// Hidden access_token input carrying a uuid-shaped value.
	tokRe := regexp.MustCompile(`name="access_token"\s+value="([0-9a-f-]{36})"`)
	m := tokRe.FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("rendered page missing hidden access_token input with uuid value; body: %s", body)
	}
	token := m[1]

	// The minted token must resolve in the session store, and the rendered
	// access_token_ttl must equal that session's ExpiresAt.UnixMilli().
	sess, ok := e.sessions.LookupWopiToken(token)
	if !ok {
		t.Fatalf("access_token in page does not resolve in the session store")
	}
	wantTTL := strconv.FormatInt(sess.ExpiresAt.UnixMilli(), 10)
	ttlRe := regexp.MustCompile(`name="access_token_ttl"\s+value="(\d+)"`)
	mt := ttlRe.FindStringSubmatch(body)
	if mt == nil {
		t.Fatalf("rendered page missing access_token_ttl input")
	}
	if mt[1] != wantTTL {
		t.Errorf("access_token_ttl = %s, want %s (ExpiresAt.UnixMilli, MS-WOPI ms-epoch convention)", mt[1], wantTTL)
	}

	// iframe present and targeted by the form.
	if !strings.Contains(body, "<iframe") {
		t.Errorf("rendered page missing iframe")
	}

	// Auto-submit script.
	if !regexp.MustCompile(`document\.forms\[0\]\.submit\(\)`).MatchString(body) {
		t.Errorf("rendered page missing auto-submit script")
	}
}

// --- test-list 18: token never in any GET URL (Pitfall 6 regression) ---

func TestOpen_TokenNeverInGetURL(t *testing.T) {
	e := newEnv(t, authedIdentity)

	req := httptest.NewRequest(http.MethodGet, "/open?file=hello.odt", nil)
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /open status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()

	tokRe := regexp.MustCompile(`name="access_token"\s+value="([0-9a-f-]{36})"`)
	m := tokRe.FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("no access_token input found")
	}
	token := m[1]

	// The token may appear ONLY as the hidden form input value — never in
	// any href/src/action URL attribute.
	urlAttrRe := regexp.MustCompile(`(?:href|src|action)="([^"]*)"`)
	for _, match := range urlAttrRe.FindAllStringSubmatch(body, -1) {
		if strings.Contains(match[1], token) {
			t.Fatalf("access_token leaked into a URL attribute: %s", match[0])
		}
	}
}

// --- test-list 19: /open error paths ---

func TestOpen_UnknownFile_404(t *testing.T) {
	e := newEnv(t, authedIdentity)

	req := httptest.NewRequest(http.MethodGet, "/open?file=does-not-exist.odt", nil)
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /open unknown file status = %d, want 404", rec.Code)
	}
}

func TestOpen_MissingFileParam_400(t *testing.T) {
	e := newEnv(t, authedIdentity)

	req := httptest.NewRequest(http.MethodGet, "/open", nil)
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("GET /open without file param status = %d, want 400", rec.Code)
	}
}

// --- document list renders open links when authenticated ---

func TestRoot_Authenticated_ListsDocuments(t *testing.T) {
	e := newEnv(t, authedIdentity)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET / authenticated status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "hello.odt") {
		t.Errorf("document list missing hello.odt")
	}
	if !strings.Contains(body, "/open?file=hello.odt") {
		t.Errorf("document list missing open link for hello.odt")
	}
}
