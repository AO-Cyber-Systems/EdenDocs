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
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/AO-Cyber-Systems/EdenDocs/wopi-host/internal/config"
	"github.com/AO-Cyber-Systems/EdenDocs/wopi-host/internal/oidcauth"
	"github.com/AO-Cyber-Systems/EdenDocs/wopi-host/internal/oidctest"
	"github.com/AO-Cyber-Systems/EdenDocs/wopi-host/internal/session"
)

// newTLSHarness is newHarness with the fake IdP served over TLS on a
// private (httptest-generated, self-signed) certificate — the local-dev
// AOID topology EDENDOCS_WOPI_OIDC_CA_FILE exists for. It also returns a
// browser-simulating client that (like a real user's browser trusting the
// dev CA) can complete the TLS handshake with the IdP.
func newTLSHarness(t *testing.T, caFile string, idpOpts ...oidctest.Option) (*harness, *http.Client, error) {
	t.Helper()

	mux := http.NewServeMux()
	rpSrv := httptest.NewServer(mux)
	t.Cleanup(rpSrv.Close)

	idp := oidctest.New(idpOpts...)
	idpSrv := httptest.NewTLSServer(idp.Handler())
	t.Cleanup(idpSrv.Close)
	idp.SetIssuer(idpSrv.URL)

	if caFile == writeIdPCert {
		caFile = filepath.Join(t.TempDir(), "idp-ca.pem")
		pemBytes := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: idpSrv.Certificate().Raw,
		})
		if err := os.WriteFile(caFile, pemBytes, 0o600); err != nil {
			t.Fatalf("writing IdP cert PEM: %v", err)
		}
	}

	cfg := config.Config{
		AOIDIssuer:   idpSrv.URL,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		StateHMACKey: []byte("test-state-hmac-key-0123456789ab"),
		PublicURL:    rpSrv.URL,
		OIDCCAFile:   caFile,
	}

	sessions := session.NewStore(time.Now)
	auth, err := oidcauth.NewAuthenticator(context.Background(), cfg, sessions)
	if err != nil {
		return nil, nil, err
	}

	mux.HandleFunc("/login", auth.LoginHandler)
	mux.HandleFunc("/callback", auth.CallbackHandler)
	mux.Handle("/protected", auth.RequireLogin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	browser := newLoginClient(t)
	browser.Transport = idpSrv.Client().Transport

	return &harness{rpURL: rpSrv.URL, cfg: cfg}, browser, nil
}

// writeIdPCert is a sentinel caFile value telling newTLSHarness to write
// the IdP's own certificate to a temp PEM file and use that path.
const writeIdPCert = "\x00write-idp-cert"

func TestNewAuthenticatorTrustsPrivateCAViaOIDCCAFile(t *testing.T) {
	h, client, err := newTLSHarness(t, writeIdPCert)
	if err != nil {
		t.Fatalf("NewAuthenticator with OIDCCAFile: %v", err)
	}

	// Full login flow: the /callback code exchange is an outbound TLS call
	// to the private-CA IdP and must ride the same CA-trusting client as
	// discovery did.
	callbackURL := driveToCallback(t, h, client)
	resp := mustGet(t, client, callbackURL)
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("GET /callback: status %d, want 302 (login should complete over private-CA TLS)", resp.StatusCode)
	}
	r := mustGet(t, client, h.rpURL+"/protected")
	if r.StatusCode != http.StatusOK {
		t.Fatalf("GET /protected after login: status %d, want 200", r.StatusCode)
	}
}

func TestNewAuthenticatorRejectsPrivateCAWithoutOIDCCAFile(t *testing.T) {
	// Control: same TLS IdP, no CA file — discovery must fail on the
	// unknown authority instead of silently trusting it.
	_, _, err := newTLSHarness(t, "")
	if err == nil {
		t.Fatal("NewAuthenticator succeeded against a private-CA issuer without OIDCCAFile; want certificate verification error")
	}
	if !strings.Contains(err.Error(), "certificate") {
		t.Fatalf("error = %v, want a certificate verification failure", err)
	}
}

func TestNewAuthenticatorOIDCCAFileErrors(t *testing.T) {
	sessions := session.NewStore(time.Now)
	base := config.Config{
		AOIDIssuer:   "https://127.0.0.1:1", // never dialed: CA load fails first
		ClientID:     "test-client",
		StateHMACKey: []byte("test-state-hmac-key-0123456789ab"),
		PublicURL:    "http://127.0.0.1:8091",
	}

	t.Run("missing file", func(t *testing.T) {
		cfg := base
		cfg.OIDCCAFile = filepath.Join(t.TempDir(), "does-not-exist.pem")
		if _, err := oidcauth.NewAuthenticator(context.Background(), cfg, sessions); err == nil {
			t.Fatal("want error for missing CA file, got nil")
		}
	})

	t.Run("no valid PEM certificates", func(t *testing.T) {
		cfg := base
		cfg.OIDCCAFile = filepath.Join(t.TempDir(), "garbage.pem")
		if err := os.WriteFile(cfg.OIDCCAFile, []byte("not a certificate"), 0o600); err != nil {
			t.Fatalf("writing garbage file: %v", err)
		}
		if _, err := oidcauth.NewAuthenticator(context.Background(), cfg, sessions); err == nil {
			t.Fatal("want error for PEM-free CA file, got nil")
		}
	})
}
