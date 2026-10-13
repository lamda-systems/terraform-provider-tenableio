# Look a template up by the title the Tenable.io UI shows...
data "tenableio_scan_template_tenable" "basic" {
  title = "Basic Network Scan"
}

# ...or by its short system name.
data "tenableio_scan_template_tenable" "advanced" {
  name = "advanced"
}

# Some titles belong to both a network and an agent template. is_agent
# chooses between them; without it the lookup fails and lists both.
data "tenableio_scan_template_tenable" "agent_malware" {
  title    = "Malware Scan"
  is_agent = true
}

resource "tenableio_scan" "assessment" {
  template_uuid = data.tenableio_scan_template_tenable.basic.uuid
  name          = "central-region-assets"
  text_targets  = "192.0.2.1-192.0.2.255"
}
