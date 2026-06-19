# Tenable.io Terraform Provider

## Project Overview

Custom Terraform provider for managing Tenable.io (Vulnerability Management) resources via the Tenable API.

## Tech Stack

- **Language**: Go 1.26+
- **Framework**: Terraform Plugin Framework (hashicorp/terraform-plugin-framework)
- **Terraform**: 1.14+
- **Linter**: golangci-lint v2
- **Release**: goreleaser v2
- **Docs**: tfplugindocs

## Tenable API

- **Base URL**: `https://cloud.tenable.com`
- **Auth**: `X-ApiKeys: accessKey=ACCESS_KEY;secretKey=SECRET_KEY;` header
- **User-Agent**: `Integration/1.0 (Tenable; TerraformProvider; Build/VERSION)`
- **Docs**: https://developer.tenable.com
- **LLM index**: https://developer.tenable.com/llms.txt

## Project Structure

```
src/
├── main.go                      # Provider entry point
├── go.mod / go.sum
├── GNUmakefile
├── .goreleaser.yml
├── internal/
│   ├── provider/                # Provider config and registration
│   ├── client/                  # Tenable API HTTP client
│   ├── resources/               # Terraform resources (CRUD)
│   └── datasources/             # Terraform data sources (read-only)
├── examples/                    # Example .tf files (used by tfplugindocs)
└── templates/                   # Doc templates for tfplugindocs
```

## Commands

```bash
# Build
cd src && go build -o terraform-provider-tenableio

# Test
cd src && go test ./...

# Acceptance tests (needs real Tenable.io creds)
cd src && TF_ACC=1 TENABLEIO_ACCESS_KEY=xxx TENABLEIO_SECRET_KEY=xxx go test ./... -v

# Lint
cd src && golangci-lint run ./...

# Generate docs
cd src && make docs

# Install locally for dev
cd src && go build -o terraform-provider-tenableio && \
  mkdir -p ~/.terraform.d/plugins/registry.terraform.io/lamda-systems/tenableio/0.1.0/linux_amd64 && \
  mv terraform-provider-tenableio ~/.terraform.d/plugins/registry.terraform.io/lamda-systems/tenableio/0.1.0/linux_amd64/
```

## Environment Variables

- `TENABLEIO_ACCESS_KEY` — Tenable.io API access key
- `TENABLEIO_SECRET_KEY` — Tenable.io API secret key
- `TENABLEIO_BASE_URL` — Override base URL (default: `https://cloud.tenable.com`)
- `TF_ACC` — Set to `1` to run acceptance tests
- `TF_LOG` — Terraform log level (default: INFO in devcontainer)

## Conventions

- Provider name: `tenableio`
- Resource naming: `tenableio_<resource>` (e.g., `tenableio_scan`, `tenableio_folder`)
- Data source naming: `tenableio_<resource>` or `tenableio_<resources>` (plural for lists)
- Use Terraform Plugin Framework (not SDKv2)
- All API calls go through the centralized client in `internal/client/`
- One file per resource/data source

## Adding New Resources or Data Sources

When adding a new resource or data source, documentation must be created alongside the code:

1. **Example file** — Create `src/examples/resources/tenableio_<name>/main.tf` (or `data-sources/` for data sources) with realistic usage showing required and key optional attributes.

2. **Template file** — Create `src/templates/resources/<name>.md.tmpl` (or `data-sources/`) using this structure:
   ```
   ---
   page_title: "{{.Name}} {{.Type}} - {{.ProviderName}}"
   subcategory: ""
   description: |-
   {{ .Description | plainmarkdown | trimspace | prefixlines "  " }}
   ---

   # {{.Name}} ({{.Type}})

   {{ .Description | trimspace }}

   ## Example Usage

   {{ tffile (printf "examples/resources/%s/main.tf" .Name) }}

   {{ .SchemaMarkdown | trimspace }}
   ```
   For data sources, replace `resources` with `data-sources` in the `tffile` path.

3. **Regenerate docs** — Run `cd src && make docs` which outputs to the repo-root `docs/` directory (where the Terraform Registry reads from).

4. **Naming** — Template files use the resource name without the provider prefix (e.g., `folder.md.tmpl` not `tenableio_folder.md.tmpl`). Example directories use the full name (e.g., `tenableio_folder/`).
