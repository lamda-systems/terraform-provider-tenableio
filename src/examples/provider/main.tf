terraform {
  required_providers {
    tenableio = {
      source = "registry.terraform.io/tenable/tenableio"
    }
  }
}

provider "tenableio" {
  # Credentials via environment variables:
  #   TENABLEIO_ACCESS_KEY
  #   TENABLEIO_SECRET_KEY
  #
  # Or set them directly (not recommended for production):
  # access_key = "your-access-key"
  # secret_key = "your-secret-key"
}
