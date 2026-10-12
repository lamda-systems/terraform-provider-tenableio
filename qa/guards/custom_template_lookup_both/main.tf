# Deliberately invalid. `qa/run.sh` asserts that `terraform validate` REJECTS
# this configuration: a user-defined template lookup that sets both `id` and
# `name` is refused before any API call, rather than silently preferring one.
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

data "tenableio_scan_template_custom" "lookup" {
  id   = 45
  name = "Internal - Credentialed"
}
