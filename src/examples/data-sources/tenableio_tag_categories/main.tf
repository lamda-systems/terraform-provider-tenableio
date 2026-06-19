# List all tag categories
data "tenableio_tag_categories" "all" {}

output "category_names" {
  value = [for c in data.tenableio_tag_categories.all.categories : c.name]
}
