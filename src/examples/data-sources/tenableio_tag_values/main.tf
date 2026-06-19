# List all tag values
data "tenableio_tag_values" "all" {}

output "tag_values" {
  value = [for v in data.tenableio_tag_values.all.values : "${v.category_name}:${v.value}"]
}
