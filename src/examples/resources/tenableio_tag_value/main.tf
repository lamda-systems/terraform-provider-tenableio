# Create tag values under a category
resource "tenableio_tag_value" "production" {
  category_uuid = tenableio_tag_category.environment.uuid
  value         = "Production"
  description   = "Production environment assets"
}

resource "tenableio_tag_value" "staging" {
  category_uuid = tenableio_tag_category.environment.uuid
  value         = "Staging"
  description   = "Staging environment assets"
}
