# Tag categories, tag values (static and dynamic), and the tag data sources.
#
# Split from the core stack because tags are where the API's request and
# response shapes diverge most: filters go out as an object and come back as a
# JSON-formatted string with renamed keys. If anything is going to produce a
# perpetual diff, it is this stack.

terraform {
  required_providers {
    tenableio = {
      source = "registry.terraform.io/lamda-systems/tenableio"
    }
  }
}

provider "tenableio" {
  # base_url and credentials come from TENABLEIO_* in the environment; qa/run.sh
  # points them at the mock.
}

# --- categories ------------------------------------------------------------

resource "tenableio_tag_category" "environment" {
  name        = "Environment"
  description = "Deployment environment classification"
}

# No description: exercises the Optional+Computed attribute defaulting to "".
# Under MOCK_OMITTED_DESCRIPTION=preserves this is the shape that used to fail
# the apply.
resource "tenableio_tag_category" "location" {
  name = "Location"
}

# --- static values ---------------------------------------------------------

resource "tenableio_tag_value" "production" {
  category_uuid = tenableio_tag_category.environment.uuid
  value         = "Production"
  description   = "Production environment assets"
}

# Deliberately no description, to pin the empty-string default end to end.
resource "tenableio_tag_value" "staging" {
  category_uuid = tenableio_tag_category.environment.uuid
  value         = "Staging"
}

# The same value under a different category: uniqueness is on the
# (category, value) pair, so this must not collide with the one above.
resource "tenableio_tag_value" "london" {
  category_uuid = tenableio_tag_category.location.uuid
  value         = "Production"
  description   = "Reuses a value name from another category on purpose"
}

# --- dynamic values --------------------------------------------------------

# A single rule with a single value. The API collapses it to a bare string on
# the way back, so this is the case that proves the client can parse both forms.
resource "tenableio_tag_value" "freebsd" {
  category_uuid = tenableio_tag_category.environment.uuid
  value         = "FreeBSD hosts"
  description   = "Dynamic: one rule, one value"

  filters = {
    asset = {
      and = [
        {
          property = "operating_system"
          operator = "equals"
          values   = ["FreeBSD"]
        },
      ]
    }
  }
}

# Multiple rules, multiple values, and both branches. Exercises the array form
# of the value and the short operator codes coming back from the API.
resource "tenableio_tag_value" "datacenter" {
  category_uuid = tenableio_tag_category.location.uuid
  value         = "Datacenter"
  description   = "Dynamic: and + or, multi-valued"

  filters = {
    asset = {
      and = [
        {
          property = "ipv4"
          operator = "eq"
          values   = ["10.0.0.1", "10.0.0.2", "10.0.0.3"]
        },
        {
          property = "operating_system"
          operator = "contains"
          values   = ["Linux"]
        },
      ]
      or = [
        {
          property = "fqdn"
          operator = "wildcard"
          values   = ["*.dc1.example.com"]
        },
      ]
    }
  }
}

# --- data sources ----------------------------------------------------------

data "tenableio_tag_categories" "all" {
  depends_on = [
    tenableio_tag_category.environment,
    tenableio_tag_category.location,
  ]
}

data "tenableio_tag_values" "all" {
  depends_on = [
    tenableio_tag_value.production,
    tenableio_tag_value.staging,
    tenableio_tag_value.london,
    tenableio_tag_value.freebsd,
    tenableio_tag_value.datacenter,
  ]
}

# The catalogue of properties and operators usable in a dynamic tag rule.
data "tenableio_tag_asset_filters" "catalogue" {}

# --- outputs ---------------------------------------------------------------
#
# These double as assertions: run.sh checks them after apply.

output "category_count" {
  value = length(data.tenableio_tag_categories.all.categories)
}

output "value_count" {
  value = length(data.tenableio_tag_values.all.values)
}

output "dynamic_value_count" {
  value = length([
    for v in data.tenableio_tag_values.all.values : v if v.type == "dynamic"
  ])
}

# Must stay "" -- the whole point of the empty-string default.
output "location_description" {
  value = tenableio_tag_category.location.description
}

output "staging_description" {
  value = tenableio_tag_value.staging.description
}

# Must echo back exactly as written, with no case folding.
output "location_name" {
  value = tenableio_tag_category.location.name
}

output "filter_properties" {
  value = sort([for f in data.tenableio_tag_asset_filters.catalogue.filters : f.name])
}
