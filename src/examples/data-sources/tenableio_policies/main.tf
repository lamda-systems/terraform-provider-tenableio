# List all policies
data "tenableio_policies" "all" {}

output "policy_names" {
  value = [for p in data.tenableio_policies.all.policies : p.name]
}
