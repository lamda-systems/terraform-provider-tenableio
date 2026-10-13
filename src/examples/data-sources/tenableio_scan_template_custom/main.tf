# A template built in the Tenable.io UI (Scan Templates > User Defined),
# looked up by the name it was given there.
data "tenableio_scan_template_custom" "credentialed" {
  name = "Internal - Credentialed"
}

# Names need not be unique in Tenable.io. When two templates share one, the
# lookup fails and lists them with their ids; look the right one up by id.
data "tenableio_scan_template_custom" "legacy" {
  id = 45
}

data "tenableio_scan_template_tenable" "advanced" {
  title = "Advanced Network Scan"
}

resource "tenableio_scan" "weekly" {
  template_uuid = data.tenableio_scan_template_tenable.advanced.uuid
  policy_id     = data.tenableio_scan_template_custom.credentialed.id
  name          = "internal-weekly"
  text_targets  = "10.0.0.0/24"
}
