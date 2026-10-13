# Deliberately invalid. `qa/run.sh` asserts that `terraform validate` REJECTS
# this configuration: the same plugin family twice
# would leave its status ambiguous.
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

resource "tenableio_scan_template_custom" "families" {
  template_uuid = "00000000-0000-4000-8000-000000000001"
  name          = "duplicate-family"

  plugin_family {
    name   = "Backdoors"
    status = "enabled"
  }

  plugin_family {
    name   = "Backdoors"
    status = "disabled"
  }
}
