# Every Tenable-provided template a scan can be created from.
data "tenableio_scan_templates_tenable" "all" {}

output "scan_template_names" {
  value = { for t in data.tenableio_scan_templates_tenable.all.templates : t.name => t.title }
}
