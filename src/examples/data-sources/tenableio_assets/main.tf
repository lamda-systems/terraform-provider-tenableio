# List all assets seen in the last 30 days (default)
data "tenableio_assets" "recent" {}

# List assets seen in the last 7 days
data "tenableio_assets" "last_week" {
  date_range = 7
}

output "asset_count" {
  value = length(data.tenableio_assets.recent.assets)
}
