# List all scanners
data "tenableio_scanners" "all" {}

output "scanner_names" {
  value = [for s in data.tenableio_scanners.all.scanners : s.name]
}
