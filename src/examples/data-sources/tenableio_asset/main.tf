# Look up a specific asset by UUID
data "tenableio_asset" "web_server" {
  id = "f0c1d2e3-a4b5-c6d7-e8f9-0a1b2c3d4e5f"
}

output "asset_ips" {
  value = data.tenableio_asset.web_server.ipv4
}
