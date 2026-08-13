terraform {
  required_providers {
    tenableio = {
      source = "registry.terraform.io/lamda-systems/tenableio"
    }
  }
}

# Default provider. Reads the unprefixed environment variables:
#   TENABLEIO_ACCESS_KEY
#   TENABLEIO_SECRET_KEY
#   TENABLEIO_BASE_URL
#   TENABLEIO_PROXY_AUTH_HEADER
#   TENABLEIO_PROXY_AUTH_VALUE
provider "tenableio" {
  # Values can also be set directly (not recommended for credentials):
  # access_key = "your-access-key"
  # secret_key = "your-secret-key"

  # Override the API base URL (default: https://cloud.tenable.com).
  # base_url = "https://cloud.tenable.com"

  # Send an extra HTTP header on every API request, e.g. to authenticate
  # against a forward proxy. The header name and its value must both be
  # set, from either source.
  # proxy_auth_header = "Proxy-Authorization"
  # proxy_auth_value  = "Bearer your-proxy-token"
}

# Aliased providers use `prefix` so each one reads its own environment
# variables. This alias reads TENABLEIO_EU_ACCESS_KEY, TENABLEIO_EU_SECRET_KEY,
# TENABLEIO_EU_PROXY_AUTH_HEADER, TENABLEIO_EU_PROXY_AUTH_VALUE, and so on.
provider "tenableio" {
  alias  = "eu"
  prefix = "TENABLEIO_EU"

  base_url = "https://eu.cloud.tenable.com"
}

# Each variable falls back independently to its unprefixed TENABLEIO_
# equivalent. Exporting only TENABLEIO_US_PROXY_AUTH_VALUE gives this alias
# its own proxy token while it keeps using the shared TENABLEIO_ACCESS_KEY
# and TENABLEIO_SECRET_KEY.
provider "tenableio" {
  alias  = "us"
  prefix = "TENABLEIO_US"

  proxy_auth_header = "Proxy-Authorization"
}

# Resources and data sources select an instance with the `provider` argument.
resource "tenableio_folder" "eu_reports" {
  provider = tenableio.eu

  name = "eu-reports"
}
