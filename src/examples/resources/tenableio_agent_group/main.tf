# Create an agent group for production servers
resource "tenableio_agent_group" "production" {
  name = "production-servers"
}
