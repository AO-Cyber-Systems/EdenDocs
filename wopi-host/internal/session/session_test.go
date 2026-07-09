/*
 * Copyright 2026 AO Cyber Systems
 *
 * SPDX-License-Identifier: MPL-2.0
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */

package session

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeClock lets tests advance time deterministically without sleeping.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock(start time.Time) *fakeClock {
	return &fakeClock{now: start}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func testIdentity() Identity {
	return Identity{
		Subject: "aoid-subject-uuid-1234",
		Name:    "Ada Lovelace",
		Email:   "ada@example.com",
		IsAdmin: false,
	}
}

func TestMintWopiTokenAndLookup(t *testing.T) {
	t.Run("mint then lookup resolves the same identity and fileID before expiry", func(t *testing.T) {
		clock := newFakeClock(time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC))
		store := NewStore(clock.Now)
		id := testIdentity()

		token, minted := store.MintWopiToken(id, "hello.odt", 15*time.Minute)
		if token == "" {
			t.Fatal("MintWopiToken returned an empty token")
		}

		got, ok := store.LookupWopiToken(token)
		if !ok {
			t.Fatal("LookupWopiToken did not find the just-minted token")
		}
		if got.Identity != id {
			t.Errorf("Identity mismatch: got %+v, want %+v", got.Identity, id)
		}
		if got.FileID != "hello.odt" {
			t.Errorf("FileID mismatch: got %q, want %q", got.FileID, "hello.odt")
		}
		if !got.ExpiresAt.Equal(minted.ExpiresAt) {
			t.Errorf("ExpiresAt mismatch: got %v, want %v", got.ExpiresAt, minted.ExpiresAt)
		}
	})
}

func TestLookupWopiTokenExpiry(t *testing.T) {
	t.Run("expired wopi token is not returned", func(t *testing.T) {
		clock := newFakeClock(time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC))
		store := NewStore(clock.Now)
		id := testIdentity()

		token, _ := store.MintWopiToken(id, "hello.odt", 1*time.Minute)

		// Not yet expired.
		if _, ok := store.LookupWopiToken(token); !ok {
			t.Fatal("token should still be valid immediately after minting")
		}

		clock.Advance(2 * time.Minute)

		if _, ok := store.LookupWopiToken(token); ok {
			t.Error("expired token was returned by LookupWopiToken")
		}
	})
}

func TestLookupWopiTokenUnknown(t *testing.T) {
	t.Run("unknown or garbage token is not found", func(t *testing.T) {
		clock := newFakeClock(time.Now())
		store := NewStore(clock.Now)

		if _, ok := store.LookupWopiToken("not-a-real-token"); ok {
			t.Error("garbage token unexpectedly resolved")
		}
		if _, ok := store.LookupWopiToken(""); ok {
			t.Error("empty token unexpectedly resolved")
		}
	})
}

func TestMintedTokensAreOpaque(t *testing.T) {
	t.Run("minted tokens are unique and contain no identity material", func(t *testing.T) {
		clock := newFakeClock(time.Now())
		store := NewStore(clock.Now)
		id := testIdentity()

		seen := make(map[string]bool)
		for i := 0; i < 50; i++ {
			token, _ := store.MintWopiToken(id, "hello.odt", 15*time.Minute)
			if seen[token] {
				t.Fatalf("duplicate token minted: %s", token)
			}
			seen[token] = true

			// A JWT would contain two "." separators (header.payload.sig).
			// An opaque uuid must not.
			if strings.Contains(token, ".") {
				t.Fatalf("token looks JWT-shaped (contains '.'): %s", token)
			}
			// The identity's subject/email/name must never appear literally
			// in the token — it is a random opaque handle, not an encoding
			// of the identity.
			if strings.Contains(token, id.Subject) || strings.Contains(token, id.Email) {
				t.Fatalf("token appears to embed identity material: %s", token)
			}
		}
	})
}

func TestWebSessionRoundTrip(t *testing.T) {
	t.Run("create then lookup round-trips an Identity", func(t *testing.T) {
		clock := newFakeClock(time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC))
		store := NewStore(clock.Now)
		id := testIdentity()

		sid := store.CreateWebSession(id, 30*time.Minute)
		if sid == "" {
			t.Fatal("CreateWebSession returned an empty session id")
		}

		got, ok := store.LookupWebSession(sid)
		if !ok {
			t.Fatal("LookupWebSession did not find the just-created session")
		}
		if got != id {
			t.Errorf("Identity mismatch: got %+v, want %+v", got, id)
		}
	})

	t.Run("expired web session is not returned", func(t *testing.T) {
		clock := newFakeClock(time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC))
		store := NewStore(clock.Now)
		id := testIdentity()

		sid := store.CreateWebSession(id, 1*time.Minute)
		clock.Advance(2 * time.Minute)

		if _, ok := store.LookupWebSession(sid); ok {
			t.Error("expired web session was returned by LookupWebSession")
		}
	})

	t.Run("deleted web session is not returned", func(t *testing.T) {
		clock := newFakeClock(time.Now())
		store := NewStore(clock.Now)
		id := testIdentity()

		sid := store.CreateWebSession(id, 30*time.Minute)
		store.DeleteWebSession(sid)

		if _, ok := store.LookupWebSession(sid); ok {
			t.Error("deleted web session was still returned by LookupWebSession")
		}
	})
}

func TestConcurrentMintAndLookup(t *testing.T) {
	t.Run("concurrent mint and lookup from multiple goroutines is race-free", func(t *testing.T) {
		clock := newFakeClock(time.Now())
		store := NewStore(clock.Now)
		id := testIdentity()

		var wg sync.WaitGroup
		tokens := make(chan string, 100)

		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				token, _ := store.MintWopiToken(id, "hello.odt", 15*time.Minute)
				tokens <- token
			}(i)
		}

		go func() {
			wg.Wait()
			close(tokens)
		}()

		for token := range tokens {
			// Concurrent lookups of freshly-minted tokens; some may race
			// ahead of their own mint goroutine's channel send, but the
			// lookup call itself must never panic or corrupt state — that's
			// what -race is checking here.
			store.LookupWopiToken(token)
		}
	})
}
