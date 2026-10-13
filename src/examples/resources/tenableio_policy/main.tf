# Custom scan policy based on the Advanced Network Scan template
data "tenableio_policy_template_tenable" "advanced" {
  title = "Advanced Network Scan"
}

resource "tenableio_policy" "custom" {
  template_uuid = data.tenableio_policy_template_tenable.advanced.uuid
  name          = "custom-advanced-scan"
  description   = "Custom policy for internal network assessment"
  visibility    = "shared"
}
