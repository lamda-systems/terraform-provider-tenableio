# List all folders
data "tenableio_folders" "all" {}

output "folder_names" {
  value = [for f in data.tenableio_folders.all.folders : f.name]
}
