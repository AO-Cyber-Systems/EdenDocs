/*
 * Copyright 2026 AO Cyber Systems
 *
 * SPDX-License-Identifier: MPL-2.0
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */

// Package proof implements verification of coolwsd's WOPI proof-key
// signatures (X-WOPI-Proof / X-WOPI-ProofOld / X-WOPI-TimeStamp), a
// byte-for-byte mirror of coolwsd's own wsd/ProofKey.cpp
// Proof::GetProof/SignProof.
//
// coolwsd unconditionally signs every outbound WOPI request IF a proof_key
// file exists at {COOLWSD_CONFIGDIR}/proof_key — there is no coolwsd.xml
// toggle for this (3-RESEARCH.md Pitfall 3). Verification is entirely the
// WOPI host's responsibility: this package is that verifier.
package proof

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"
)

// ErrStaleProof is returned when the X-WOPI-TimeStamp falls outside the
// accepted verification window.
var ErrStaleProof = errors.New("proof: timestamp outside verification window")

// ErrNoProofKey is returned when VerifyProof is called with a nil public
// key (e.g. discovery has not yet published a proof-key, per Pitfall 3).
var ErrNoProofKey = errors.New("proof: no public key available for verification")

// ErrNoMatch is returned by VerifyEither when no combination of
// (X-WOPI-Proof, X-WOPI-ProofOld) x (current key, old key) validates.
var ErrNoMatch = errors.New("proof: no valid proof header matched any known key")

// ticksToUnixEpoch is the number of .NET ticks (100ns units) between
// 0001-01-01T00:00:00Z and the Unix epoch (1970-01-01T00:00:00Z) — the
// well-known constant equal to .NET's DateTime.UnixEpoch.Ticks. Converting
// a raw X-WOPI-TimeStamp value through this offset (rather than adding
// ticks*100 nanoseconds directly to a year-1 time.Time) avoids overflowing
// Go's int64-nanosecond time.Duration, whose range only spans ~292 years —
// nowhere near enough to represent an offset from year 1.
const ticksToUnixEpoch = 621355968000000000

// ParseProofKey builds an *rsa.PublicKey from the base64-encoded big-endian
// modulus/exponent attributes coolwsd's /hosting/discovery XML publishes on
// its <proof-key> element (the "modulus"/"exponent" or "oldmodulus"/
// "oldexponent" attribute pair). The CAPI-blob "value"/"oldvalue"
// attributes are a different encoding and are deliberately ignored here.
func ParseProofKey(modulusB64, exponentB64 string) (*rsa.PublicKey, error) {
	modulus, err := base64.StdEncoding.DecodeString(modulusB64)
	if err != nil {
		return nil, fmt.Errorf("proof: decode modulus: %w", err)
	}
	exponent, err := base64.StdEncoding.DecodeString(exponentB64)
	if err != nil {
		return nil, fmt.Errorf("proof: decode exponent: %w", err)
	}
	if len(modulus) == 0 {
		return nil, fmt.Errorf("proof: empty modulus")
	}
	if len(exponent) == 0 {
		return nil, fmt.Errorf("proof: empty exponent")
	}

	n := new(big.Int).SetBytes(modulus)
	e := new(big.Int).SetBytes(exponent)
	return &rsa.PublicKey{N: n, E: int(e.Int64())}, nil
}

// VerifyProof verifies a single X-WOPI-Proof (or X-WOPI-ProofOld) header
// against pub, mirroring coolwsd's wsd/ProofKey.cpp byte construction:
//
//   - a length-prefixed (big-endian int32) accessToken
//   - a length-prefixed (big-endian int32) UPPERCASED full absolute request
//     URL (scheme://host:port/path?query — NOT just the path; proxies and
//     containers rewrite r.Host, so this must be reconstructed from a
//     trusted base URL, not read off the incoming request)
//   - a length-prefixed (big-endian int32) 8-byte big-endian encoding of
//     the raw .NET-tick timestamp
//
// hashed with SHA-256 and verified as an RSA PKCS1v15 signature.
//
// timestampHeader is the raw X-WOPI-TimeStamp value: .NET ticks (100ns
// units since 0001-01-01 00:00:00 UTC). window bounds how far the
// timestamp may drift from "now" in either direction before the proof is
// considered stale (MS-WOPI convention default is 20 minutes; this package
// takes window as a parameter rather than hardcoding it so callers can
// widen it for slower CI environments — see 3-RESEARCH.md error_recovery).
func VerifyProof(pub *rsa.PublicKey, accessToken, uri, timestampHeader, proofB64 string, window time.Duration) error {
	if pub == nil {
		return ErrNoProofKey
	}

	ticks, err := strconv.ParseInt(timestampHeader, 10, 64)
	if err != nil {
		return fmt.Errorf("proof: parse X-WOPI-TimeStamp %q: %w", timestampHeader, err)
	}

	ts := time.Unix(0, (ticks-ticksToUnixEpoch)*100)
	if d := time.Since(ts); d < -window || d > window {
		return fmt.Errorf("%w: %s from now", ErrStaleProof, d)
	}

	if proofB64 == "" {
		return fmt.Errorf("proof: empty proof header")
	}
	sig, err := base64.StdEncoding.DecodeString(proofB64)
	if err != nil {
		return fmt.Errorf("proof: decode signature: %w", err)
	}

	buf := new(bytes.Buffer)
	write := func(b []byte) error {
		if err := binary.Write(buf, binary.BigEndian, int32(len(b))); err != nil {
			return err
		}
		_, err := buf.Write(b)
		return err
	}
	if err := write([]byte(accessToken)); err != nil {
		return fmt.Errorf("proof: build signed buffer: %w", err)
	}
	if err := write([]byte(strings.ToUpper(uri))); err != nil {
		return fmt.Errorf("proof: build signed buffer: %w", err)
	}
	tb := make([]byte, 8)
	binary.BigEndian.PutUint64(tb, uint64(ticks))
	if err := write(tb); err != nil {
		return fmt.Errorf("proof: build signed buffer: %w", err)
	}

	h := sha256.Sum256(buf.Bytes())
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, h[:], sig); err != nil {
		return fmt.Errorf("proof: signature verification failed: %w", err)
	}
	return nil
}

// VerifyEither accepts a request as proven if ANY single combination of
// (proofCur against current, proofCur against old, proofOld against
// current, proofOld against old) verifies. This mirrors coolwsd's own
// rotation tolerance: ProofKey.cpp currently duplicates Proof and ProofOld
// unconditionally ("TODO: implement proper rotation"), so a genuinely
// rotating deployment may present either header signed by either the
// current or the previous key. current and/or old may be nil (e.g. before
// discovery has been fetched, or before any rotation has happened) — a nil
// key is simply skipped rather than erroring.
func VerifyEither(current, old *rsa.PublicKey, accessToken, uri, timestampHeader, proofB64, proofOldB64 string, window time.Duration) error {
	if proofB64 == "" && proofOldB64 == "" {
		return fmt.Errorf("%w: no proof headers present", ErrNoMatch)
	}

	var errs []error
	try := func(pub *rsa.PublicKey, proof string) bool {
		if pub == nil || proof == "" {
			return false
		}
		if err := VerifyProof(pub, accessToken, uri, timestampHeader, proof, window); err != nil {
			errs = append(errs, err)
			return false
		}
		return true
	}

	if try(current, proofB64) {
		return nil
	}
	if try(old, proofB64) {
		return nil
	}
	if try(current, proofOldB64) {
		return nil
	}
	if try(old, proofOldB64) {
		return nil
	}

	return fmt.Errorf("%w: %w", ErrNoMatch, errors.Join(errs...))
}
