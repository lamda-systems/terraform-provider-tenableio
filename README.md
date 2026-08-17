# Terraform Provider for Tenable.io

[![CI](https://github.com/lamda-systems/terraform-provider-tenableio/actions/workflows/ci.yml/badge.svg)](https://github.com/lamda-systems/terraform-provider-tenableio/actions/workflows/ci.yml)

Custom Terraform provider for managing [Tenable.io](https://www.tenable.com/) Vulnerability Management resources.

## What it manages

**Resources**

| Resource | Manages |
| --- | --- |
| `tenableio_scan` | Scan configurations |
| `tenableio_policy` | Scan policies (templates) |
| `tenableio_folder` | Scan folders |
| `tenableio_exclusion` | Scan exclusions |
| `tenableio_network` | Networks |
| `tenableio_tag_category` | Tag categories |
| `tenableio_tag_value` | Tag values, static or [dynamic](#dynamic-tags) |
| `tenableio_agent_group` | Agent groups |

**Data sources**

| Data source | Returns |
| --- | --- |
| `tenableio_scans` / `tenableio_policies` | Scan configurations, scan policies |
| `tenableio_asset` / `tenableio_assets` | A single asset, or the workbench asset list |
| `tenableio_folders` / `tenableio_exclusions` | Scan folders, scan exclusions |
| `tenableio_networks` / `tenableio_scanners` | Networks, scanners |
| `tenableio_agent_groups` | Agent groups |
| `tenableio_tag_categories` / `tenableio_tag_values` | Tag categories, tag values |
| `tenableio_tag_asset_filters` | Asset attributes and operators usable in dynamic tag rules |

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

resource "tenableio_folder" "reports" {
  name = "quarterly-reports"
}
```

### Multiple provider instances

Set `prefix` on an aliased provider to give it its own set of environment
variables. Each variable falls back independently to its unprefixed
`TENABLEIO_` equivalent, so an alias can override only what differs:

```terraform
provider "tenableio" {
  alias  = "eu"
  prefix = "TENABLEIO_EU" # TENABLEIO_EU_ACCESS_KEY, TENABLEIO_EU_SECRET_KEY, ...

  base_url = "https://eu.cloud.tenable.com"
}
```

Every setting resolves from the provider block first, then the prefixed
environment variable, then the unprefixed one. See the
[provider documentation](docs/index.md) for the full list.

### Forward proxy authentication

`proxy_auth_header` / `proxy_auth_value` (or `TENABLEIO_PROXY_AUTH_HEADER` /
`TENABLEIO_PROXY_AUTH_VALUE`) add an extra HTTP header to every API request,
for deployments that reach Tenable.io through an authenticating proxy. Both
must be set together.

Aliased providers inherit this pair like any other setting: an instance that
declares no proxy variables of its own falls back to the shared
`TENABLEIO_PROXY_AUTH_HEADER` / `TENABLEIO_PROXY_AUTH_VALUE`, and it can
override just one of them. To opt a single instance out of an inherited proxy,
set both attributes to `""` in its provider block.

> **Set shared proxy config in the environment, not in a provider block.**
> Terraform configures each `provider` block independently, so attributes on
> the default provider are invisible to aliased ones. Putting the header name
> in the default block while the token comes from
> `TENABLEIO_PROXY_AUTH_VALUE` makes every alias resolve a value with no
> header, failing with `Missing Proxy Auth Header`. Either export both
> variables, or repeat `proxy_auth_header` in each aliased block.

### Dynamic tags

Give a `tenableio_tag_value` a set of `filters` and Tenable.io applies the tag
automatically to every asset the rules match:

```terraform
resource "tenableio_tag_category" "environment" {
  name = "Environment"
}

resource "tenableio_tag_value" "datacenter_servers" {
  # Reference the category by UUID, not by name — see the note below.
  category_uuid = tenableio_tag_category.environment.uuid
  value         = "Datacenter Servers"

  filters = {
    asset = {
      and = [
        { property = "asset_class", operator = "equals", values = ["server"] },
        { property = "ipv4", operator = "equals", values = ["10.0.0.0/16"] },
      ]
    }
  }
}
```

Use the `tenableio_tag_asset_filters` data source to discover which properties
and operators your tenant supports. Without `filters`, the tag is static.

> **Reference categories by `category_uuid`, not `category_name`.** The API
> creates a category when the given name is not found, and a bare name string
> creates no dependency edge in Terraform — so a tag value can be created
> before its `tenableio_tag_category`, silently producing a second, unmanaged
> category of the same name. Values sharing a not-yet-existing name can also
> race each other under Terraform's default parallelism. On top of that,
> `category_name` is computed from the API response and forces replacement when
> it changes, so renaming a category in the Tenable UI destroys and recreates
> every tag value pinned to it by name. A UUID reference avoids all three.

## Development

This project uses a VS Code devcontainer. Open the repo in VS Code and select **Reopen in Container** to get all tooling (Go, Terraform, linters, security scanners) pre-installed at pinned versions.

```bash
# Build
cd src && make build

# Test
cd src && make test

# Acceptance tests (hit a real Tenable.io tenant; needs credentials)
cd src && make testacc

# Lint
cd src && make lint

# Security checks (gosec + govulncheck)
cd src && make security

# Regenerate docs/ from schemas, examples and templates
cd src && make docs

# Build and install into ~/.terraform.d/plugins for local testing
cd src && make install

# Run full pre-commit checks
bash .githooks/pre-commit
```

Activate the pre-commit hook:

```bash
cd src && make setup
```

Adding a resource or data source also means adding its example, doc template
and a `make docs` run — see [CLAUDE.md](CLAUDE.md) for the full checklist and
project conventions.

## Security

Security scanning runs on every push and pull request via the CI pipeline:

- **[CodeQL](https://codeql.github.com/)** — semantic code analysis, reported to the GitHub [Security tab](https://github.com/lamda-systems/terraform-provider-tenableio/security/code-scanning).
- **[gosec](https://github.com/securego/gosec)** — static analysis for Go security vulnerabilities. Results are uploaded as SARIF to the same Security tab.
- **[govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck)** — checks Go dependencies against the Go vulnerability database. It is a **hard gate**: a known vulnerability fails the build. The full JSON report is also uploaded as a workflow artifact.
- **[Dependabot](https://docs.github.com/en/code-security/dependabot)** — automated dependency update PRs.

Lint and security run in parallel, and the build/test job runs only if both pass. The suite also runs on a weekly schedule so new advisories surface without a push.

## Releases

Merging to `main` releases automatically once CI passes: the workflow tags the commit, then GoReleaser builds the binaries and GPG-signs the checksums.

The version bump is inferred from the branch name in the merge commit: `major/…` and `minor/…` bump those components, and **everything else is a patch bump**. Name the branch accordingly before merging. To land a change without releasing, include `[skip release]` (with the space) in the commit message.

## Documentation

Full resource and data source documentation is available on the [Terraform Registry](https://registry.terraform.io/providers/lamda-systems/tenableio/latest/docs) or in the [`docs/`](docs/) directory. `docs/` is generated — edit the schema descriptions, `src/examples/` and `src/templates/`, then run `make docs`.

## License

See [LICENSE](LICENSE).
