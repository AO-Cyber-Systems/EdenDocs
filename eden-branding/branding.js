/* -*- js -*- */
/*
 * Copyright the EdenDocs contributors / AO Cyber Systems.
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */

/* MANDATORY for BRAND-03: window.brandProductName is a bare global with NO
 * other producer for the admin console — coolwsd.xml's brandProductName key
 * only reaches the editor via the hidden-input path (2-RESEARCH.md §4).
 * This script executes (via the %BRANDING_JS% tag) before adminBody.html's
 * inline script by template order. Keep to plain ES5 global assignments —
 * it runs unbundled, in an old-style script context, in BOTH the editor and
 * the admin console. */
window.brandProductName = 'EdenDocs';
window.brandProductURL = 'https://aocyber.ai';
