# User-defined scan templates. The second plan being empty is the assertion
# that matters: it proves every configured setting was persisted as planned and
# read back as planned, including the settings Tenable.io does not report back.

terraform {
  required_providers {
    tenableio = {
      source = "registry.terraform.io/lamda-systems/tenableio"
    }
  }
}

provider "tenableio" {}

data "tenableio_policy_template_tenable" "advanced" {
  name = "advanced"
}

data "tenableio_policy_template_tenable" "basic" {
  name = "basic"
}

data "tenableio_policy_template_tenable" "agent" {
  name = "agent_advanced"
}

data "tenableio_scan_template_tenable" "advanced" {
  name = "advanced"
}

# Every kind of setting, plus the two traps in the details response:
# max_compliance_check_timeout comes back as an integer, and the web crawler
# depth is not reported at all while scan_webapps is off.
resource "tenableio_scan_template_custom" "network" {
  template_uuid = data.tenableio_policy_template_tenable.advanced.uuid
  name          = "qa-network-template"
  description   = "Internal network baseline"

  discovery {
    ping_the_remote_host = false
    icmp_ping_retries    = 3
    portscan_range       = "1-1024"
    ssl_prob_ports       = "All ports"
  }

  assessment {
    override_accuracy  = true
    report_paranoia    = "Paranoid (more false alarms)"
    webcrawl_max_depth = 4
  }

  report {
    reverse_lookup          = true
    allow_post_scan_editing = false
  }

  advanced {
    safe_checks                  = true
    scan_allow_multi_target      = true
    max_compliance_check_timeout = 600
    debug_level                  = "2"
  }

  plugin_family {
    name   = "Backdoors"
    status = "disabled"
  }

  plugin_family {
    name   = "SCADA"
    status = "disabled"
  }
}

# A Basic template: individual discovery settings need mode = "Custom".
resource "tenableio_scan_template_custom" "basic" {
  template_uuid = data.tenableio_policy_template_tenable.basic.uuid
  name          = "qa-basic-template"

  discovery {
    mode     = "Custom"
    udp_ping = true
  }
}

# An agent template, with the agent-only settings, including the ones that
# travel as JSON integers.
resource "tenableio_scan_template_custom" "agent" {
  template_uuid = data.tenableio_policy_template_tenable.agent.uuid
  name          = "qa-agent-template"

  agent {
    refresh_reporting_type           = "days"
    refresh_reporting_frequency_days = 14
    staggered_start_mins             = 30
  }

  advanced {
    report_max_ports = 2048
  }
}

# Nothing but a name: no blocks at all must plan clean too.
resource "tenableio_scan_template_custom" "minimal" {
  template_uuid = data.tenableio_policy_template_tenable.advanced.uuid
  name          = "qa-minimal-template"
}

# A scan based on the template, and the template found again by id.
resource "tenableio_scan" "internal" {
  template_uuid = data.tenableio_scan_template_tenable.advanced.uuid
  name          = "qa-templated-scan"
  text_targets  = "192.0.2.0/24"
  policy_id     = tenableio_scan_template_custom.network.id
}

data "tenableio_scan_template_custom" "network" {
  id = tenableio_scan_template_custom.network.id
}

output "network_description" {
  value = data.tenableio_scan_template_custom.network.description
}

output "network_icmp_ping_retries" {
  value = tenableio_scan_template_custom.network.discovery.icmp_ping_retries
}

output "network_compliance_timeout" {
  value = tenableio_scan_template_custom.network.advanced.max_compliance_check_timeout
}

output "network_webcrawl_max_depth" {
  value = tenableio_scan_template_custom.network.assessment.webcrawl_max_depth
}

output "agent_refresh_days" {
  value = tenableio_scan_template_custom.agent.agent.refresh_reporting_frequency_days
}

output "scan_policy_matches" {
  value = tenableio_scan.internal.policy_id == tenableio_scan_template_custom.network.id
}

output "minimal_has_owner" {
  value = tenableio_scan_template_custom.minimal.owner != ""
}
