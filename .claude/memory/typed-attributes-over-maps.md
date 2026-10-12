---
name: typed-attributes-over-maps
description: "Schema design preference — typed attributes, avoid map attributes; if unavoidable, small scoped blocks"
metadata:
  node_type: memory
  type: feedback
  originSessionId: 0dc8e9d8-1047-411f-9df7-29aecb703fb4
  modified: 2026-10-12T10:35:47.761Z
---

Prefer typed Terraform attributes over free-form `map(string)` attributes in resource schemas. Use maps only when genuinely unavoidable, and then keep them small and scoped inside a block (e.g. `variable { name, value }` repeated blocks) rather than a top-level map.

**Why:** the provider is consumed by other people; typed attributes are easier to reason about, validate and document.

**How to apply:** when an API takes free-form settings (e.g. Tenable scan template settings), model them as typed attributes grouped in nested blocks; reach for a key/value block only for truly dynamic keys. See [[scan-templates-feature-decisions]].
