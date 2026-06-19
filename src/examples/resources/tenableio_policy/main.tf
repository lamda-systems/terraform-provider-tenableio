# Custom scan policy based on Advanced Network Scan template
resource "tenableio_policy" "custom" {
  template_uuid = "329692d8-ea42-4e96-acd6-7da6c3571c27d24bd260ef5f9e66"
  name          = "custom-advanced-scan"
  description   = "Custom policy for internal network assessment"
  visibility    = "shared"
}
