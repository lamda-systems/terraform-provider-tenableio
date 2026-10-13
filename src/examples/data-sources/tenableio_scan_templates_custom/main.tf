# Every user-defined scan template the API keys can see.
data "tenableio_scan_templates_custom" "all" {}

output "custom_template_ids" {
  value = { for t in data.tenableio_scan_templates_custom.all.templates : t.name => t.id... }
}
