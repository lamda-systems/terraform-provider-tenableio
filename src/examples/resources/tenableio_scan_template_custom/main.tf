data "tenableio_policy_template_tenable" "advanced" {
  title = "Advanced Network Scan"
}

# A user-defined scan template. Only the settings written here are sent and
# tracked; everything else keeps the Advanced Network Scan defaults.
resource "tenableio_scan_template_custom" "internal" {
  template_uuid = data.tenableio_policy_template_tenable.advanced.uuid
  name          = "Internal - Baseline"
  description   = "Weekly baseline for the internal network"

  discovery {
    ping_the_remote_host = true
    portscan_range       = "1-65535"
    ssl_prob_ports       = "All ports"
  }

  assessment {
    override_accuracy = true
    report_paranoia   = "Avoid false alarms"
    thorough_tests    = true
  }

  report {
    reverse_lookup = true
  }

  advanced {
    safe_checks        = true
    max_hosts_per_scan = 30
  }

  plugin_family {
    name   = "Denial of Service"
    status = "disabled"
  }

  # Removing a setting or a plugin family replaces the template, which gives it
  # a new id. Create the replacement first so scans never point at nothing.
  lifecycle {
    create_before_destroy = true
  }
}

# An agent template takes the agent block; discovery does not apply to it.
data "tenableio_policy_template_tenable" "agent" {
  name = "agent_advanced"
}

resource "tenableio_scan_template_custom" "agents" {
  template_uuid = data.tenableio_policy_template_tenable.agent.uuid
  name          = "Agents - Baseline"

  agent {
    refresh_reporting_type           = "days"
    refresh_reporting_frequency_days = 14
    staggered_start_mins             = 30
  }
}

data "tenableio_scan_template_tenable" "advanced" {
  title = "Advanced Network Scan"
}

resource "tenableio_scan" "internal" {
  template_uuid = data.tenableio_scan_template_tenable.advanced.uuid
  policy_id     = tenableio_scan_template_custom.internal.id
  name          = "internal-weekly"
  text_targets  = "10.0.0.0/24"
}
