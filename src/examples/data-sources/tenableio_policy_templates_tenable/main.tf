# Every Tenable-provided template a user-defined scan template can be based on,
# leaving out agent templates.
data "tenableio_policy_templates_tenable" "all" {}

output "network_policy_templates" {
  value = [for t in data.tenableio_policy_templates_tenable.all.templates : t.title if !t.is_agent]
}
