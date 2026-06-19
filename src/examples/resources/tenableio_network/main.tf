# Create a network for an isolated environment
resource "tenableio_network" "staging" {
  name           = "staging-environment"
  description    = "Network for staging infrastructure"
  assets_ttl_days = 90
}
