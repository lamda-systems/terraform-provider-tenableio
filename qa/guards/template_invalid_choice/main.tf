# Deliberately invalid. `qa/run.sh` asserts that `terraform validate` REJECTS
# this configuration: a value outside a setting's
# documented choices is refused at validate time, with the choices listed.
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

resource "tenableio_scan_template_custom" "choice" {
  template_uuid = "00000000-0000-4000-8000-000000000001"
  name          = "invalid-choice"

  advanced {
    audit_trail = "verbose"
  }
}
