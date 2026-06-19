# On-demand scan targeting specific IPs
resource "tenableio_scan" "assessment" {
  template_uuid = "893d91d1-5440-4f8c-9a6b-b50cfba86652d24bd260ef5f9e66"
  name          = "central-region-assets"
  text_targets  = "192.0.2.1-192.0.2.255"
  emails        = "[email protected]"
}

# Recurring weekly scan using a custom policy
resource "tenableio_scan" "weekly" {
  template_uuid   = "329692d8-ea42-4e96-acd6-7da6c3571c27d24bd260ef5f9e66"
  name            = "western-region-weekly"
  text_targets    = "10.0.0.0/24"
  policy_id       = tenableio_policy.custom.id
  enabled         = true
  launch          = "WEEKLY"
  starttime       = "20240101T130000"
  rrules          = "FREQ=WEEKLY;INTERVAL=1;BYDAY=MO"
  timezone        = "US/Mountain"
  scan_time_window = 180
}
