# Discover the asset attributes and operators usable in dynamic tag rules
data "tenableio_tag_asset_filters" "all" {}

# The properties available for tenableio_tag_value filters
output "filter_properties" {
  value = [for f in data.tenableio_tag_asset_filters.all.filters : f.name]
}

# The operators a specific property supports
output "operating_system_operators" {
  value = one([
    for f in data.tenableio_tag_asset_filters.all.filters :
    f.operators if f.name == "operating_system"
  ])
}
