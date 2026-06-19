# Exclude a set of hosts from scanning
resource "tenableio_exclusion" "maintenance_window" {
  name    = "maintenance-hosts"
  members = "10.0.1.100,10.0.1.101,10.0.1.102"
}

# Scheduled exclusion during a weekly maintenance window
resource "tenableio_exclusion" "weekly_maintenance" {
  name        = "weekly-maintenance-window"
  description = "Exclude production load balancers during maintenance"
  members     = "10.0.0.0/24"

  schedule {
    enabled   = true
    starttime = "2026-01-01 02:00:00"
    endtime   = "2026-01-01 06:00:00"
    timezone  = "US/Eastern"
    rrules    = "FREQ=WEEKLY;INTERVAL=1;BYDAY=SU"
  }
}
