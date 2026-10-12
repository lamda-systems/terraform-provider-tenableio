# Deliberately invalid. `qa/run.sh` asserts that `terraform validate` REJECTS
# this configuration: report_paranoia has no effect
# unless override_accuracy is on; Tenable.io would store "Normal".
#
# Nothing here is ever applied.

terraform {
  required_providers {
    tenableio = {
      source = "registry.terraform.io/lamda-systems/tenableio"
    }
  }
}

provider "tenableio" {}

resource "tenableio_scan_template_custom" "paranoid" {
  template_uuid = "00000000-0000-4000-8000-000000000001"
  name          = "paranoia-without-override"

  assessment {
    report_paranoia = "Paranoid (more false alarms)"
  }
}
