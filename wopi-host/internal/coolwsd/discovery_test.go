/*
 * Copyright 2026 AO Cyber Systems
 *
 * SPDX-License-Identifier: MPL-2.0
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */

package coolwsd

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// fixtureServer serves testdata/discovery.xml (hand-captured from the live
// edendocs-code reference container, coolwsd 26.04.2.1 — see
// 03-03-SUMMARY.md for provenance) and counts every request it receives, so
// tests can assert on caching behavior.
func fixtureServer(t *testing.T) (url string, hits *int32) {
	t.Helper()
	fixture, err := os.ReadFile("testdata/discovery.xml")
	if err != nil {
		t.Fatalf("read testdata/discovery.xml: %v", err)
	}
	hits = new(int32)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(hits, 1)
		w.Header().Set("Content-Type", "text/xml")
		_, _ = w.Write(fixture)
	}))
	t.Cleanup(srv.Close)
	return srv.URL, hits
}

func TestClient_URLSrc_ResolvesFromFixture(t *testing.T) {
	baseURL, _ := fixtureServer(t)
	c := New(baseURL, time.Hour)

	urlsrc, err := c.URLSrc(context.Background(), "odt")
	if err != nil {
		t.Fatalf("URLSrc(odt): unexpected error: %v", err)
	}
	if !strings.Contains(urlsrc, "cool.html?") {
		t.Fatalf("URLSrc(odt) = %q, want it to contain cool.html?", urlsrc)
	}

	// Second known extension from the same fixture (calc app).
	urlsrcOds, err := c.URLSrc(context.Background(), "ods")
	if err != nil {
		t.Fatalf("URLSrc(ods): unexpected error: %v", err)
	}
	if !strings.Contains(urlsrcOds, "cool.html?") {
		t.Fatalf("URLSrc(ods) = %q, want it to contain cool.html?", urlsrcOds)
	}
}

func TestClient_URLSrc_UnknownExtension(t *testing.T) {
	baseURL, _ := fixtureServer(t)
	c := New(baseURL, time.Hour)

	_, err := c.URLSrc(context.Background(), "not-a-real-extension")
	if !errors.Is(err, ErrUnknownExtension) {
		t.Fatalf("URLSrc(unknown) error = %v, want ErrUnknownExtension", err)
	}
}

func TestClient_ProofKeysParsedFromFixture(t *testing.T) {
	baseURL, _ := fixtureServer(t)
	c := New(baseURL, time.Hour)

	d, err := c.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: unexpected error: %v", err)
	}
	if d.ProofKey == nil {
		t.Fatalf("Discovery.ProofKey not parsed from fixture")
	}
	if d.OldProofKey == nil {
		t.Fatalf("Discovery.OldProofKey not parsed from fixture")
	}
	if d.ProofKey.N.Cmp(d.OldProofKey.N) != 0 {
		// The captured fixture duplicates current/old (coolwsd's own "TODO:
		// implement proper rotation") — they should be equal moduli here.
		t.Fatalf("expected ProofKey and OldProofKey moduli to match in the captured fixture (coolwsd duplicates them)")
	}
}

func TestClient_Get_IsCached(t *testing.T) {
	baseURL, hits := fixtureServer(t)
	c := New(baseURL, time.Hour)

	if _, err := c.Get(context.Background()); err != nil {
		t.Fatalf("first Get: %v", err)
	}
	if _, err := c.Get(context.Background()); err != nil {
		t.Fatalf("second Get: %v", err)
	}
	if got := atomic.LoadInt32(hits); got != 1 {
		t.Fatalf("expected exactly 1 HTTP fetch across two Get calls, got %d", got)
	}

	c.Invalidate()
	if _, err := c.Get(context.Background()); err != nil {
		t.Fatalf("Get after Invalidate: %v", err)
	}
	if got := atomic.LoadInt32(hits); got != 2 {
		t.Fatalf("expected a second HTTP fetch after Invalidate, got %d", got)
	}
}
