# List all scans
data "tenableio_scans" "all" {}

# List scans in a specific folder
data "tenableio_scans" "folder" {
  folder_id = 3
}

output "scan_names" {
  value = [for s in data.tenableio_scans.all.scans : s.name]
}
