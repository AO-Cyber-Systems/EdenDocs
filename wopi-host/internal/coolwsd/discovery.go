/*
 * Copyright 2026 AO Cyber Systems
 *
 * SPDX-License-Identifier: MPL-2.0
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */

// Package coolwsd is a cached client for coolwsd's own /hosting/discovery
// WOPI discovery document: it resolves the urlsrc action template for a
// given file extension (used by internal/launch to build the editor iframe
// embed URL) and extracts coolwsd's proof-key public keys (consumed by
// internal/proof to verify X-WOPI-Proof/X-WOPI-ProofOld on every inbound
// WOPI request).
package coolwsd

import (
	"context"
	"crypto/rsa"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/AO-Cyber-Systems/EdenDocs/wopi-host/internal/proof"
)

// ErrUnknownExtension is returned by URLSrc when no <action> in the
// discovery document matches the requested file extension.
var ErrUnknownExtension = errors.New("coolwsd: no discovery action for extension")

// Discovery is the parsed, immutable result of a single /hosting/discovery
// fetch: the urlsrc action template per file extension, plus coolwsd's
// current and previous (rotation-tolerant) proof-key public keys.
type Discovery struct {
	// ProofKey is parsed from the <proof-key modulus=.. exponent=..>
	// attributes. Nil if the document had no <proof-key> element at all
	// (e.g. coolwsd has no {COOLWSD_CONFIGDIR}/proof_key file — 3-RESEARCH.md
	// Pitfall 3).
	ProofKey *rsa.PublicKey
	// OldProofKey is parsed from <proof-key oldmodulus=.. oldexponent=..>.
	OldProofKey *rsa.PublicKey

	// actions maps a bare file extension (no leading dot, e.g. "odt") to
	// its urlsrc action template, collected from every <action> in the
	// document regardless of net-zone/app nesting — first hit for a given
	// ext wins (3-RESEARCH.md/TRD guidance: don't assume zone/app order is
	// stable across coolwsd versions).
	actions map[string]string
}

// Client is a cached /hosting/discovery client. The zero value is not
// usable — construct with New.
type Client struct {
	baseURL    string
	ttl        time.Duration
	httpClient *http.Client

	mu        sync.Mutex
	cached    *Discovery
	fetchedAt time.Time
}

// New constructs a Client that fetches discovery from
// coolwsdURL+"/hosting/discovery" and caches the result for cacheTTL
// (production default: 1 hour — discovery only changes on a coolwsd
// upgrade/restart).
func New(coolwsdURL string, cacheTTL time.Duration) *Client {
	return &Client{
		baseURL:    strings.TrimSuffix(coolwsdURL, "/"),
		ttl:        cacheTTL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Get returns the cached Discovery, fetching (and caching) it first if
// there is no cached value or the cache has expired per the configured
// cacheTTL.
func (c *Client) Get(ctx context.Context) (*Discovery, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cached != nil && time.Since(c.fetchedAt) < c.ttl {
		return c.cached, nil
	}

	d, err := c.fetch(ctx)
	if err != nil {
		return nil, err
	}
	c.cached = d
	c.fetchedAt = time.Now()
	return c.cached, nil
}

// Invalidate drops the cached Discovery, forcing the next Get to refetch.
func (c *Client) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cached = nil
}

// URLSrc returns the urlsrc action template for ext (a bare extension, no
// leading dot — e.g. "odt", not ".odt"; a leading dot is stripped
// defensively if present), fetching/caching discovery as needed.
func (c *Client) URLSrc(ctx context.Context, ext string) (string, error) {
	d, err := c.Get(ctx)
	if err != nil {
		return "", err
	}
	ext = strings.TrimPrefix(strings.ToLower(ext), ".")
	urlsrc, ok := d.actions[ext]
	if !ok {
		return "", fmt.Errorf("%w: %q", ErrUnknownExtension, ext)
	}
	return urlsrc, nil
}

func (c *Client) fetch(ctx context.Context) (*Discovery, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/hosting/discovery", nil)
	if err != nil {
		return nil, fmt.Errorf("coolwsd: build discovery request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("coolwsd: fetch discovery: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("coolwsd: discovery returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("coolwsd: read discovery body: %w", err)
	}

	return parseDiscovery(body)
}

// XML shape of coolwsd's /hosting/discovery document. Only the attributes
// this package needs are mapped; unknown elements/attributes are ignored
// by encoding/xml automatically.
type discoveryXML struct {
	XMLName  xml.Name     `xml:"wopi-discovery"`
	NetZones []netZoneXML `xml:"net-zone"`
	ProofKey *proofKeyXML `xml:"proof-key"`
}

type netZoneXML struct {
	Apps []appXML `xml:"app"`
}

type appXML struct {
	Actions []actionXML `xml:"action"`
}

type actionXML struct {
	Ext    string `xml:"ext,attr"`
	URLSrc string `xml:"urlsrc,attr"`
}

// proofKeyXML maps the base64 modulus/exponent attribute pairs. The
// CAPI-blob "value"/"oldvalue" attributes are deliberately not mapped —
// coolwsd's own consumers (and this package) use modulus/exponent, per
// 3-RESEARCH.md Pattern 6.
type proofKeyXML struct {
	Modulus     string `xml:"modulus,attr"`
	Exponent    string `xml:"exponent,attr"`
	OldModulus  string `xml:"oldmodulus,attr"`
	OldExponent string `xml:"oldexponent,attr"`
}

func parseDiscovery(body []byte) (*Discovery, error) {
	var doc discoveryXML
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("coolwsd: parse discovery XML: %w", err)
	}

	d := &Discovery{actions: make(map[string]string)}

	for _, zone := range doc.NetZones {
		for _, app := range zone.Apps {
			for _, action := range app.Actions {
				if action.Ext == "" || action.URLSrc == "" {
					continue
				}
				ext := strings.ToLower(action.Ext)
				if _, exists := d.actions[ext]; exists {
					continue // first hit wins — don't assume zone/app order
				}
				d.actions[ext] = action.URLSrc
			}
		}
	}

	if doc.ProofKey != nil {
		if doc.ProofKey.Modulus != "" && doc.ProofKey.Exponent != "" {
			pub, err := proof.ParseProofKey(doc.ProofKey.Modulus, doc.ProofKey.Exponent)
			if err != nil {
				return nil, fmt.Errorf("coolwsd: parse proof-key: %w", err)
			}
			d.ProofKey = pub
		}
		if doc.ProofKey.OldModulus != "" && doc.ProofKey.OldExponent != "" {
			pub, err := proof.ParseProofKey(doc.ProofKey.OldModulus, doc.ProofKey.OldExponent)
			if err != nil {
				return nil, fmt.Errorf("coolwsd: parse old proof-key: %w", err)
			}
			d.OldProofKey = pub
		}
	}

	return d, nil
}
