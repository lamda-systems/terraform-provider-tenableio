# Deliberately invalid. `qa/run.sh` asserts that `terraform validate` REJECTS
# this configuration, so a regression that drops the whitespace guard shows up
# as a QA failure rather than as a mystery apply error months later.
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

resource "tenableio_tag_category" "trailing_space" {
  name = "Production "
}

resource "tenableio_tag_category" "padded_description" {
  name        = "Staging"
  description = "  padded  "
}

# A literal uuid rather than a reference: Terraform skips evaluating a resource
# whose dependency already produced an error, so a cross-reference here would
# quietly stop this case from being checked at all.
resource "tenableio_tag_value" "leading_space" {
  category_uuid = "00000000-0000-4000-8000-000000000001"
  value         = " London"
}

resource "tenableio_network" "tab" {
  name = "lab\t"
}
