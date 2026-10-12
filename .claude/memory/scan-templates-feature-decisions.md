---
name: scan-templates-feature-decisions
description: "Decisions for the scan templates feature (branch feature/scan_templates, Oct 2026) — naming, replacement, secrets via Key Vault"
metadata:
  node_type: memory
  type: project
  originSessionId: 0dc8e9d8-1047-411f-9df7-29aecb703fb4
  modified: 2026-10-12T10:43:29.902Z
---

Decisions made by the user on 2026-10-12 for the scan templates work:

- `tenableio_policy` has never been applied against a live tenant — treat it as broken, not production-verified (unlike tags/scans/dynamic filters).
- Naming: resource `scan_template_custom` (user-defined templates = API `/policies`), data-only `scan_template_tenable` (`/editor/scan/templates`) and `policy_template_tenable` (`/editor/policy/templates`). `policy_template_custom` = the compliance flavour (Policy Compliance Auditing / SCAP based templates, audits-heavy) — same /policies API, different schema; each resource must refuse the other's flavour.
- Custom `.audit` files are used heavily — custom audits (via file upload) are first-class, not an afterthought.
- Template catalogue: the user tried the dump tool (`src/tools/templatecatalog`), but one template's details were ~38k lines of JSON, so it was set aside (2026-10-12). Typed attributes are generated from the Create scan OpenAPI definition instead; a catalogue may still come later from a pipeline run and would only add rows to `scanSettings`.
- `tenableio_policy`/`tenableio_policies`: keep this release, deprecate (schema DeprecationMessage + log warning) and announce in CHANGELOG.md (started by this work if absent).
- Credentials: all credential types at once, as a separate `tenableio_managed_credential` referenced by UUID from templates (no inline credentials). Pipeline uses latest azurerm, so `ephemeral "azurerm_key_vault_secret"` is available.
- Typed attributes, not maps ([[typed-attributes-over-maps]]); cover all Tenable templates.
- Removing a configured setting/plugin family forces replacement (PUT preserves omitted settings).
- Credentials and compliance are in scope now. The pipeline Terraform executor reads secrets from Azure Key Vault; secrets must be marked sensitive and never land in state — plan is write-only attributes fed by `ephemeral "azurerm_key_vault_secret"`.
- Policy `visibility` behaviour is unverified by the user.
- Users must be able to look up Tenable templates by human name, not only UUID.

Status at 2026-10-12 (branch feature/scan_templates, uncommitted until the memory move): done — `*_template[s]_tenable` and `scan_template[s]_custom` data sources, `tenableio_scan_template_custom` resource (171 typed settings, plugin families, plan-time template checks, moved-block migration), `tenableio_policy`/`tenableio_policies` deprecated and read path fixed. Next — `tenableio_managed_credential` (write-only secrets) + credential references on templates; then audits/custom `.audit` upload and `tenableio_policy_template_custom`. Pending infra: Go 1.26.9 bump in `Dockerfile-local` + `src/go.mod` (stdlib govulncheck findings), `python3-venv` in the image.

**Why:** these were explicit answers to the planning questions; they constrain the schema design.
**How to apply:** follow them when implementing; re-confirm anything marked unverified with TF_ACC=1.
