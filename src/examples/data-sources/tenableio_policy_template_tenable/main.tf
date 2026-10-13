# The base of a user-defined scan template comes from the policy list, not the
# scan list: the same template can have a different uuid in each.
data "tenableio_policy_template_tenable" "advanced" {
  title = "Advanced Network Scan"
}

resource "tenableio_scan_template_custom" "internal" {
  template_uuid = data.tenableio_policy_template_tenable.advanced.uuid
  name          = "Internal - Baseline"
}
