# Changelog

All notable changes to the Tenable.io Terraform provider. Releases before
0.7.0 are described only by their git tags and release notes.

## 0.7.0 (Unreleased)

FEATURES:

* **New Data Source:** `tenableio_scan_template_tenable` looks up a
  Tenable-provided scan template by `name` (`advanced`) or `title`
  (`Advanced Network Scan`), so configuration no longer needs to carry template
  uuids. `is_agent` chooses between a network and an agent template that share a
  title.
* **New Data Source:** `tenableio_policy_template_tenable` does the same for the
  policy template list, which is where the base of a user-defined scan template
  comes from.
* **New Data Source:** `tenableio_scan_templates_tenable` and
  `tenableio_policy_templates_tenable` list every template in each list.
* **New Data Source:** `tenableio_scan_template_custom` looks up a user-defined
  scan template (Scan Templates > User Defined in the UI) by `name` or `id`, for
  use as `tenableio_scan.policy_id`. A name shared by two templates fails the
  lookup and lists both with their ids.
* **New Data Source:** `tenableio_scan_templates_custom` lists every
  user-defined scan template.
* **New Resource:** `tenableio_scan_template_custom` manages a user-defined scan
  template, with 171 typed settings in `discovery`, `assessment`, `report`,
  `advanced` and `agent` blocks, and `plugin_family` blocks. Only configured
  settings are sent and tracked. Removing one replaces the template, since
  Tenable.io keeps settings an update leaves out. The plan rejects a uuid from
  the scan template list, a compliance template, and settings documented only
  for the other kind of template (network or agent). A `moved` block migrates a
  `tenableio_policy` without recreating it.

DEPRECATIONS:

* `tenableio_policy` is deprecated in favour of `tenableio_scan_template_custom`,
  and will be removed in a future release. Migrate with a `moved` block.
* `tenableio_policies` is deprecated in favour of
  `tenableio_scan_templates_custom`, and will be removed in a future release.
* Terraform shows a deprecation warning wherever either is used, and the provider
  logs one on every operation.

BUG FIXES:

* `tenableio_policy` read its name, description and metadata from the wrong
  shape. `GET /policies/{id}` nests the settings and carries no metadata, so
  every create failed with "Tenable.io stored name as """. It now reads the
  details for the settings and the policy list for the metadata.
* `tenableio_policy`: changing `template_uuid` now replaces the policy. The API
  ignores it on update, so the change never settled.

NOTES:

* A scan takes its `template_uuid` from the scan list and a policy from the
  policy list. The same template is not guaranteed to have the same uuid in
  both, so use the matching data source rather than reusing one uuid for both.
* Examples no longer hard-code template uuids.
