# Terraform Provider for Tenable.io

[![CI](https://github.com/lamda-systems/terraform-provider-tenableio/actions/workflows/ci.yml/badge.svg)](https://github.com/lamda-systems/terraform-provider-tenableio/actions/workflows/ci.yml)

Custom Terraform provider for managing [Tenable.io](https://www.tenable.com/) Vulnerability Management resources.

## Requirements

- [Terraform](https://www.terraform.io/downloads) >= 1.14
- [Go](https://go.dev/dl/) >= 1.26 (for building)
- Tenable.io API credentials ([access & secret keys](https://developer.tenable.com))

## Usage

```terraform
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
}
```

## Development

This project uses a VS Code devcontainer. Open the repo in VS Code and select **Reopen in Container** to get all tooling (Go, Terraform, linters, security scanners) pre-installed.

```bash
# Build
cd src && make build

# Test
cd src && make test

# Lint
cd src && make lint

# Security checks (gosec + govulncheck)
cd src && make security

# Run full pre-commit checks
bash .githooks/pre-commit
```

Activate the pre-commit hook:

```bash
cd src && make setup
```

## Security

Security scanning runs on every push and pull request via the CI pipeline:

- **[gosec](https://github.com/securego/gosec)** — static analysis for Go security vulnerabilities. Results are uploaded as SARIF to the GitHub [Security tab](https://github.com/lamda-systems/terraform-provider-tenableio/security/code-scanning).
- **[govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck)** — checks Go dependencies against the Go vulnerability database. Reports are uploaded as workflow artifacts.
- **[Dependabot](https://docs.github.com/en/code-security/dependabot)** — automated dependency update PRs.

## Documentation

Full resource and data source documentation is available on the [Terraform Registry](https://registry.terraform.io/providers/lamda-systems/tenableio/latest/docs) or in the [`docs/`](docs/) directory.

## License

See [LICENSE](LICENSE).
