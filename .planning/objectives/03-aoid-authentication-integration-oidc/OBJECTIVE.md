---
work: feature
---

# AOID Authentication Integration (OIDC)

## Goal

Users authenticate against AOID before opening a document, via a new reference WOPI host service (outside this repo's `wsd/`/`browser/` tree), and their AOID identity is visible inside the editor — with no OIDC code added to coolwsd itself.

**Requirements:** AUTH-01, AUTH-02, AUTH-03, AUTH-04
**Depends on:** Objective 1 (running coolwsd for end-to-end round trip); independent of Objective 2

---
*Created: 2026-07-09 (auto-scaffold via bootstrapObjectiveMd)*
