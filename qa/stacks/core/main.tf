# Folders, networks, agent groups, exclusions, policies and scans, wired
# together so the stack also exercises dependency ordering: the scan references
# a folder, a policy and a scanner discovered through a data source.

terraform {
  required_providers {
    tenableio = {
      source = "registry.terraform.io/lamda-systems/tenableio"
    }
  }
}

provider "tenableio" {}

# --- folders ---------------------------------------------------------------

resource "tenableio_folder" "qa" {
  name = "qa-scans"
}

# --- networks --------------------------------------------------------------

resource "tenableio_network" "qa" {
  name            = "qa-network"
  description     = "Isolated network for QA"
  assets_ttl_days = 90
}

# No description and no TTL: pins both defaults ("" and 180) end to end.
resource "tenableio_network" "minimal" {
  name = "qa-network-minimal"
}

# --- agent groups ----------------------------------------------------------

resource "tenableio_agent_group" "linux" {
  name = "qa-linux-agents"
}

# --- exclusions ------------------------------------------------------------

resource "tenableio_exclusion" "adhoc" {
  name    = "qa-maintenance-hosts"
  members = "10.0.1.100,10.0.1.101"
}

# schedule is a SingleNestedAttribute, so it takes `=` and braces, not a block.
resource "tenableio_exclusion" "weekly" {
  name        = "qa-weekly-window"
  description = "Exclude the QA subnet during the maintenance window"
  members     = "10.0.0.0/24"
  network_id  = tenableio_network.qa.uuid

  schedule = {
    enabled   = true
    starttime = "2026-01-01 02:00:00"
    endtime   = "2026-01-01 06:00:00"
    timezone  = "US/Eastern"
    rrules    = "FREQ=WEEKLY;INTERVAL=1;BYDAY=SU"
  }
}

# --- policies --------------------------------------------------------------

resource "tenableio_policy" "advanced" {
  template_uuid = "329692d8-ea42-4e96-acd6-7da6c3571c27d24bd260ef5f9e66"
  name          = "qa-advanced-scan"
  description   = "Custom policy for the QA network"
  visibility    = "shared"
}

# No description and no visibility: pins the "" and "private" defaults.
resource "tenableio_policy" "minimal" {
  template_uuid = "329692d8-ea42-4e96-acd6-7da6c3571c27d24bd260ef5f9e66"
  name          = "qa-minimal-policy"
}

# --- scans -----------------------------------------------------------------

data "tenableio_scanners" "all" {}

locals {
  # Scanners are provisioned by Tenable, never created through the API, so the
  # stack discovers one rather than declaring it.
  scanner_id = data.tenableio_scanners.all.scanners[0].id
}

resource "tenableio_scan" "ondemand" {
  template_uuid = "893d91d1-5440-4f8c-9a6b-b50cfba86652d24bd260ef5f9e66"
  name          = "qa-ondemand"
  text_targets  = "192.0.2.1-192.0.2.255"
  folder_id     = tenableio_folder.qa.id
  scanner_id    = local.scanner_id
  emails        = "qa@example.com"
}

resource "tenableio_scan" "weekly" {
  template_uuid    = "329692d8-ea42-4e96-acd6-7da6c3571c27d24bd260ef5f9e66"
  name             = "qa-weekly"
  text_targets     = "10.0.0.0/24"
  folder_id        = tenableio_folder.qa.id
  scanner_id       = local.scanner_id
  policy_id        = tenableio_policy.advanced.id
  enabled          = true
  launch           = "WEEKLY"
  starttime        = "20260101T130000"
  rrules           = "FREQ=WEEKLY;INTERVAL=1;BYDAY=MO"
  timezone         = "US/Mountain"
  scan_time_window = 180
}

# --- data sources ----------------------------------------------------------

data "tenableio_folders" "all" {
  depends_on = [tenableio_folder.qa]
}

data "tenableio_networks" "all" {
  depends_on = [tenableio_network.qa, tenableio_network.minimal]
}

data "tenableio_agent_groups" "all" {
  depends_on = [tenableio_agent_group.linux]
}

data "tenableio_exclusions" "all" {
  depends_on = [tenableio_exclusion.adhoc, tenableio_exclusion.weekly]
}

data "tenableio_policies" "all" {
  depends_on = [tenableio_policy.advanced, tenableio_policy.minimal]
}

data "tenableio_scans" "all" {
  depends_on = [tenableio_scan.ondemand, tenableio_scan.weekly]
}

data "tenableio_assets" "all" {}

# --- outputs ---------------------------------------------------------------

output "folder_count" {
  # Two seeded system folders plus the one created here.
  value = length(data.tenableio_folders.all.folders)
}

output "network_count" {
  value = length(data.tenableio_networks.all.networks)
}

output "agent_group_count" {
  value = length(data.tenableio_agent_groups.all.groups)
}

output "exclusion_count" {
  value = length(data.tenableio_exclusions.all.exclusions)
}

output "policy_count" {
  value = length(data.tenableio_policies.all.policies)
}

output "scan_count" {
  value = length(data.tenableio_scans.all.scans)
}

output "asset_count" {
  value = length(data.tenableio_assets.all.assets)
}

output "scanner_count" {
  value = length(data.tenableio_scanners.all.scanners)
}

# Defaults that must survive a round trip untouched.
output "minimal_network_description" {
  value = tenableio_network.minimal.description
}

output "minimal_network_ttl" {
  value = tenableio_network.minimal.assets_ttl_days
}

output "minimal_policy_visibility" {
  value = tenableio_policy.minimal.visibility
}

output "minimal_policy_description" {
  value = tenableio_policy.minimal.description
}
