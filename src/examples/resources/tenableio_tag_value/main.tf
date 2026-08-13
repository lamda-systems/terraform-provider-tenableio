# Create tag values under a category
resource "tenableio_tag_value" "production" {
  category_uuid = tenableio_tag_category.environment.uuid
  value         = "Production"
  description   = "Production environment assets"
}

resource "tenableio_tag_value" "staging" {
  category_uuid = tenableio_tag_category.environment.uuid
  value         = "Staging"
  description   = "Staging environment assets"
}

# A dynamic tag: Tenable.io automatically applies it to every asset the
# filter rules match. "and" rules must all match; "or" rules match any.
resource "tenableio_tag_value" "datacenter_servers" {
  category_uuid = tenableio_tag_category.environment.uuid
  value         = "Datacenter Servers"

  filters = {
    asset = {
      and = [
        {
          property = "asset_class"
          operator = "equals"
          values   = ["server"]
        },
        {
          property = "ipv4"
          operator = "equals"
          values   = ["10.0.0.0/16", "10.1.0.0/16"]
        },
      ]
    }
  }
}
