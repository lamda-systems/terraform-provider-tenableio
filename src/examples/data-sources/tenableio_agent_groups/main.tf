# List all agent groups
data "tenableio_agent_groups" "all" {}

output "agent_group_names" {
  value = [for g in data.tenableio_agent_groups.all.groups : g.name]
}
