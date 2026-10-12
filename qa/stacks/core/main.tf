# Folders, networks, agent groups, exclusions, policies and scans, wired
# together so the stack also exercises dependency ordering: the scan references
# a folder, a policy and a scanner discovered through a data source. Template
# uuids are looked up by name or title, never written out.

terraform {
  required_providers {
    tenableio = {
      source = "registry.terraform.io/lamda-systems/tenableio"
    }
  }
}

provider "tenableio" {}

# --- Tenable-provided templates ---------------------------------------------

# A policy takes its template from the policy list, a scan from the scan list.
# The mock gives the two lists different uuids, so mixing them up fails here.
data "tenableio_policy_template_tenable" "advanced" {
  name = "advanced"
}

data "tenableio_scan_template_tenable" "basic" {
  title = "Basic Network Scan"
}

data "tenableio_scan_template_tenable" "advanced" {
  name = "advanced"
}

# "Malware Scan" is the title of a network and an agent template alike;
# is_agent is what makes this lookup unambiguous.
data "tenableio_scan_template_tenable" "agent_malware" {
  title    = "malware scan"
  is_agent = true
}

data "tenableio_scan_templates_tenable" "all" {}

data "tenableio_policy_templates_tenable" "all" {}

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
  template_uuid = data.tenableio_policy_template_tenable.advanced.uuid
  name          = "qa-advanced-scan"
  description   = "Custom policy for the QA network"
  visibility    = "shared"
}

# No description and no visibility: pins the "" and "private" defaults.
resource "tenableio_policy" "minimal" {
  template_uuid = data.tenableio_policy_template_tenable.advanced.uuid
  name          = "qa-minimal-policy"
}

# The same template looked up the two ways a user would. A lookup reads during
# plan unless something it waits on is still changing, so the first waits on an
# id that is unknown until create, the second on depends_on.
data "tenableio_scan_template_custom" "by_id" {
  id = tenableio_policy.advanced.id
}

data "tenableio_scan_template_custom" "by_name" {
  name       = "qa-advanced-scan"
  depends_on = [tenableio_policy.advanced]
}

data "tenableio_scan_templates_custom" "all" {
  depends_on = [tenableio_policy.advanced, tenableio_policy.minimal]
}

# --- scans -----------------------------------------------------------------

data "tenableio_scanners" "all" {}

locals {
  # Scanners are provisioned by Tenable, never created through the API, so the
  # stack discovers one rather than declaring it.
  scanner_id = data.tenableio_scanners.all.scanners[0].id
}

# A described scan is the case that broke in production: GET /scans/{id} does
# not report a description, so a provider that reads its state back from there
# sees "" and proposes restoring the text on every plan -- and the update that
# follows cannot verify its own echo either. The second plan being empty is what
# proves that is fixed.
resource "tenableio_scan" "ondemand" {
  template_uuid = data.tenableio_scan_template_tenable.basic.uuid
  name          = "qa-ondemand"
  description   = "daily scan on Tenable test devices"
  text_targets  = "192.0.2.1-192.0.2.255"
  folder_id     = tenableio_folder.qa.id
  scanner_id    = local.scanner_id
  emails        = "qa@example.com"
}

# No description, but a scan_time_window and a full schedule: those are the
# other settings GET /scans/{id} does not report back, and reading them as zero
# is what kept re-proposing them on every plan.
resource "tenableio_scan" "weekly" {
  template_uuid    = data.tenableio_scan_template_tenable.advanced.uuid
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

output "custom_template_count" {
  value = length(data.tenableio_scan_templates_custom.all.templates)
}

# Both lookups land on the policy created above, and report what it was given.
output "custom_lookups_agree" {
  value = alltrue([
    data.tenableio_scan_template_custom.by_name.id == tenableio_policy.advanced.id,
    data.tenableio_scan_template_custom.by_id.name == "qa-advanced-scan",
    data.tenableio_scan_template_custom.by_id.description == "Custom policy for the QA network",
    data.tenableio_scan_template_custom.by_id.template_uuid == data.tenableio_policy_template_tenable.advanced.uuid,
    data.tenableio_scan_template_custom.by_id.shared,
  ])
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

output "scan_template_count" {
  value = length(data.tenableio_scan_templates_tenable.all.templates)
}

output "policy_template_count" {
  value = length(data.tenableio_policy_templates_tenable.all.templates)
}

# The lookup matched case-insensitively and kept the title as written.
output "agent_malware_template_name" {
  value = data.tenableio_scan_template_tenable.agent_malware.name
}

output "agent_malware_template_title" {
  value = data.tenableio_scan_template_tenable.agent_malware.title
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

# The description the scan details endpoint never reports back.
output "ondemand_scan_description" {
  value = tenableio_scan.ondemand.description
}

output "weekly_scan_description" {
  value = tenableio_scan.weekly.description
}

# Settings the details endpoint never echoes back. The empty re-plan is the real
# assertion; these pin the values as well, so a silent zeroing cannot pass.
output "weekly_scan_time_window" {
  value = tenableio_scan.weekly.scan_time_window
}

output "weekly_scan_launch" {
  value = tenableio_scan.weekly.launch
}

output "weekly_scan_timezone" {
  value = tenableio_scan.weekly.timezone
}
