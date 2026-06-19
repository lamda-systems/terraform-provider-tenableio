# List all networks
data "tenableio_networks" "all" {}

output "network_names" {
  value = [for n in data.tenableio_networks.all.networks : n.name]
}
