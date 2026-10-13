# Deliberately invalid. `qa/run.sh` asserts that `terraform validate` REJECTS
# this configuration: a preconfigured mode overrides
# the individual settings beside it, so pairing them is refused rather than
# silently ignored by Tenable.io.
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

resource "tenableio_scan_template_custom" "basic" {
  template_uuid = "00000000-0000-4000-8000-000000000001"
  name          = "mode-override"

  discovery {
    mode     = "Port scan (all ports)"
    udp_ping = true
  }
}
