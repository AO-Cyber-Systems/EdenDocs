/*
 * Copyright 2026 AO Cyber Systems
 *
 * SPDX-License-Identifier: MPL-2.0
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */

package proof

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"math/big"
	"strconv"
	"strings"
	"testing"
	"time"
)

// testKey generates a fresh 2048-bit RSA key for a single test. 2048 is
// plenty for PKCS1v15/SHA-256 test vectors and far faster than the 4096-bit
// key coolwsd generates in production (Proof::Proof(Type) in ProofKey.cpp).
func testKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate test RSA key: %v", err)
	}
	return priv
}

// nowTicks returns .NET ticks (100ns units since 0001-01-01 UTC) for the
// current instant, mirroring the X-WOPI-TimeStamp convention. Computed via
// the Unix-epoch offset (see ticksToUnixEpoch in proof.go) rather than
// time.Since(year1) directly — the latter saturates time.Duration's
// int64-nanosecond range long before reaching a 2020s "now".
func nowTicks() int64 {
	return ticksToUnixEpoch + time.Now().UnixNano()/100
}

// signProof is a genuine mirror of coolwsd's wsd/ProofKey.cpp SignProof:
// the exact same len-prefixed byte construction as VerifyProof, but signing
// instead of verifying. This is the "signer half" of the mirror pair the
// TRD calls for — no recorded coolwsd traffic needed at this layer.
func signProof(t *testing.T, priv *rsa.PrivateKey, accessToken, uri string, ticks int64) string {
	t.Helper()
	buf := new(bytes.Buffer)
	write := func(b []byte) {
		if err := binary.Write(buf, binary.BigEndian, int32(len(b))); err != nil {
			t.Fatalf("write length prefix: %v", err)
		}
		buf.Write(b)
	}
	write([]byte(accessToken))
	write([]byte(strings.ToUpper(uri)))
	tb := make([]byte, 8)
	binary.BigEndian.PutUint64(tb, uint64(ticks))
	write(tb)

	h := sha256.Sum256(buf.Bytes())
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, h[:])
	if err != nil {
		t.Fatalf("sign proof: %v", err)
	}
	return base64.StdEncoding.EncodeToString(sig)
}

func TestVerifyProof_RoundTrip(t *testing.T) {
	priv := testKey(t)
	accessToken := "tok-123"
	uri := "http://127.0.0.1:9091/wopi/files/hello.odt?access_token=tok-123"
	ticks := nowTicks()
	proofB64 := signProof(t, priv, accessToken, uri, ticks)
	tsHeader := strconv.FormatInt(ticks, 10)

	if err := VerifyProof(&priv.PublicKey, accessToken, uri, tsHeader, proofB64, 20*time.Minute); err != nil {
		t.Fatalf("VerifyProof: unexpected error: %v", err)
	}
}

func TestVerifyProof_Tampered(t *testing.T) {
	priv := testKey(t)
	accessToken := "tok-123"
	uri := "http://127.0.0.1:9091/wopi/files/hello.odt?access_token=tok-123"
	ticks := nowTicks()
	proofB64 := signProof(t, priv, accessToken, uri, ticks)
	tsHeader := strconv.FormatInt(ticks, 10)

	cases := []struct {
		name                     string
		accessToken, uri, ts, pf string
	}{
		{"tampered access_token", "tok-456", uri, tsHeader, proofB64},
		{"tampered url", accessToken, uri + "&extra=1", tsHeader, proofB64},
		{"tampered ticks", accessToken, uri, strconv.FormatInt(ticks+1, 10), proofB64},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := VerifyProof(&priv.PublicKey, c.accessToken, c.uri, c.ts, c.pf, 20*time.Minute); err == nil {
				t.Fatalf("expected verification error for %s, got nil", c.name)
			}
		})
	}
}

func TestVerifyProof_StaleTimestamp(t *testing.T) {
	priv := testKey(t)
	accessToken := "tok-1"
	uri := "http://127.0.0.1:9091/wopi/files/x"
	// 30 minutes in the past, in .NET-tick units, outside a 20-minute window.
	staleTicks := nowTicks() - int64(30*time.Minute/(100*time.Nanosecond))
	proofB64 := signProof(t, priv, accessToken, uri, staleTicks)
	tsHeader := strconv.FormatInt(staleTicks, 10)

	err := VerifyProof(&priv.PublicKey, accessToken, uri, tsHeader, proofB64, 20*time.Minute)
	if err == nil {
		t.Fatalf("expected stale-timestamp error, got nil")
	}
}

func TestParseProofKey_MatchesSigner(t *testing.T) {
	priv := testKey(t)
	modB64 := base64.StdEncoding.EncodeToString(priv.PublicKey.N.Bytes())
	expB64 := base64.StdEncoding.EncodeToString(big.NewInt(int64(priv.PublicKey.E)).Bytes())

	pub, err := ParseProofKey(modB64, expB64)
	if err != nil {
		t.Fatalf("ParseProofKey: %v", err)
	}
	if pub.N.Cmp(priv.PublicKey.N) != 0 || pub.E != priv.PublicKey.E {
		t.Fatalf("ParseProofKey produced a different key than the signer's own public key")
	}

	accessToken := "tok-abc"
	uri := "http://127.0.0.1:9091/wopi/files/x?access_token=tok-abc"
	ticks := nowTicks()
	proofB64 := signProof(t, priv, accessToken, uri, ticks)
	tsHeader := strconv.FormatInt(ticks, 10)
	if err := VerifyProof(pub, accessToken, uri, tsHeader, proofB64, 20*time.Minute); err != nil {
		t.Fatalf("VerifyProof with ParseProofKey-derived key: %v", err)
	}
}

func TestVerifyEither_RotationTolerance(t *testing.T) {
	current := testKey(t)
	old := testKey(t)
	accessToken := "tok-either"
	uri := "http://127.0.0.1:9091/wopi/files/x"
	ticks := nowTicks()
	tsHeader := strconv.FormatInt(ticks, 10)

	// coolwsd currently duplicates Proof and ProofOld ("TODO: implement
	// proper rotation" in ProofKey.cpp) — simulate the case coolwsd's own
	// rotation tolerance is meant to cover: a valid signature under the OLD
	// key arriving in X-WOPI-ProofOld, alongside garbage in X-WOPI-Proof.
	validProofOld := signProof(t, old, accessToken, uri, ticks)
	garbage := base64.StdEncoding.EncodeToString([]byte("not-a-real-signature-at-all"))

	if err := VerifyEither(&current.PublicKey, &old.PublicKey, accessToken, uri, tsHeader, garbage, validProofOld, 20*time.Minute); err != nil {
		t.Fatalf("VerifyEither should accept a valid ProofOld against the old key: %v", err)
	}

	// Neither current nor old key validates either header -> reject.
	if err := VerifyEither(&current.PublicKey, &old.PublicKey, accessToken, uri, tsHeader, garbage, garbage, 20*time.Minute); err == nil {
		t.Fatalf("expected VerifyEither to reject when no combination matches")
	}
}

func TestVerifyEither_MissingHeaders(t *testing.T) {
	current := testKey(t)
	if err := VerifyEither(&current.PublicKey, nil, "tok", "http://x/y", "", "", "", 20*time.Minute); err == nil {
		t.Fatalf("expected error when both proof headers are empty")
	}
}
