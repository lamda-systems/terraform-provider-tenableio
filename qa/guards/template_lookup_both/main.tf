# Deliberately invalid. `qa/run.sh` asserts that `terraform validate` REJECTS
# this configuration: a template lookup that sets both `name` and `title` is
# refused before any API call, rather than silently picking one of them.
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

data "tenableio_policy_template_tenable" "lookup" {
  name  = "advanced"
  title = "Advanced Network Scan"
}
