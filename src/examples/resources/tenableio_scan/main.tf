data "tenableio_scan_template_tenable" "basic" {
  title = "Basic Network Scan"
}

data "tenableio_scan_template_tenable" "advanced" {
  title = "Advanced Network Scan"
}

# On-demand scan targeting specific IPs
resource "tenableio_scan" "assessment" {
  template_uuid = data.tenableio_scan_template_tenable.basic.uuid
  name          = "central-region-assets"
  text_targets  = "192.0.2.1-192.0.2.255"
  emails        = "[email protected]"
}

# Recurring weekly scan using a custom policy
resource "tenableio_scan" "weekly" {
  template_uuid    = data.tenableio_scan_template_tenable.advanced.uuid
  name             = "western-region-weekly"
  text_targets     = "10.0.0.0/24"
  policy_id        = tenableio_scan_template_custom.internal.id
  enabled          = true
  launch           = "WEEKLY"
  starttime        = "20240101T130000"
  rrules           = "FREQ=WEEKLY;INTERVAL=1;BYDAY=MO"
  timezone         = "US/Mountain"
  scan_time_window = 180
}
