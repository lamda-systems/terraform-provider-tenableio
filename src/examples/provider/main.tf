terraform {
  required_providers {
    tenableio = {
      source = "registry.terraform.io/lamda-systems/tenableio"
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

  # Override the API base URL (default: https://cloud.tenable.com).
  # Also settable via TENABLEIO_BASE_URL.
  # base_url = "https://cloud.tenable.com"

  # Send an extra HTTP header on every API request, e.g. to authenticate
  # against a forward proxy. Also settable via TENABLEIO_PROXY_AUTH_HEADER
  # and TENABLEIO_PROXY_AUTH_VALUE.
  # proxy_auth_header = "Proxy-Authorization"
  # proxy_auth_value  = "Bearer your-proxy-token"
}
