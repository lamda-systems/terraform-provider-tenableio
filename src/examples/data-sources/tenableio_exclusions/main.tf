# List all exclusions
data "tenableio_exclusions" "all" {}

output "exclusion_names" {
  value = [for e in data.tenableio_exclusions.all.exclusions : e.name]
}
