# Create a tag category for organizing assets by environment
resource "tenableio_tag_category" "environment" {
  name        = "Environment"
  description = "Deployment environment classification"
}
