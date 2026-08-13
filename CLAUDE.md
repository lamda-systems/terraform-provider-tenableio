# Tenable.io Terraform Provider

## Project Overview

Custom Terraform provider for managing Tenable.io (Vulnerability Management) resources via the Tenable API.

## Development Environment

This project uses a **VS Code devcontainer** (Dockerfile-local). All tooling — Go, Terraform, linters, security scanners — is installed inside the container. You are always running inside the devcontainer; there is no need for Docker-in-Docker or sidecar exec.

Every tool in `Dockerfile-local` is pinned to an explicit version via `ARG` (Terraform, golangci-lint, goreleaser, delve, tfplugindocs, gosec, govulncheck) — do not reintroduce `@latest`. `GOFLAGS` is deliberately left unset so the container uses Go's default `-mod=readonly`, matching CI: with `-mod=mod`, anything that shells out to `go build` (notably `tfplugindocs` during `make docs`) can silently rewrite `go.mod`/`go.sum`. `go get` and `go mod tidy` still work when you actually intend a dependency change.

## Tech Stack

- **Language**: Go 1.26+
- **Framework**: Terraform Plugin Framework (hashicorp/terraform-plugin-framework)
- **Terraform**: 1.14+
- **Linter**: golangci-lint v2
- **Security**: gosec, govulncheck
- **Release**: goreleaser v2
- **Docs**: tfplugindocs

## Tenable API

- **Base URL**: `https://cloud.tenable.com`
- **Auth**: `X-ApiKeys: accessKey=ACCESS_KEY;secretKey=SECRET_KEY;` header
- **User-Agent**: `Integration/1.0 (Tenable; TerraformProvider; Build/VERSION)`
- **Docs**: https://developer.tenable.com
- **LLM index**: https://developer.tenable.com/llms.txt — the root file only lists section indexes; the useful one is https://developer.tenable.com/reference/llms.txt, and any reference page can be fetched as markdown by appending `.md` to its URL.

Do not assume the API round-trips a field in the shape it accepts it. Several endpoints echo back a different structure than they take (see Tags below), so check the response schema of a GET before mapping it to state.

## Project Structure

```
src/
├── main.go                      # Provider entry point
├── go.mod / go.sum
├── GNUmakefile
├── .goreleaser.yml
├── internal/
│   ├── provider/                # Provider registration (provider.go) and
│   │                            # settings resolution (config.go)
│   ├── client/                  # Tenable API HTTP client
│   ├── resources/               # Terraform resources (CRUD)
│   └── datasources/             # Terraform data sources (read-only)
├── examples/                    # Example .tf files (used by tfplugindocs)
└── templates/                   # Doc templates for tfplugindocs
```

## Commands

```bash
# Build
cd src && make build

# Test
cd src && make test

# Acceptance tests (needs real Tenable.io creds)
cd src && make testacc

# Lint
cd src && make lint

# Security (gosec + govulncheck)
cd src && make security

# Generate docs
cd src && make docs

# Install locally for dev
cd src && make install

# Run full pre-commit checks (lint + security + tests)
bash .githooks/pre-commit
```

## CI Pipeline

GitHub Actions workflow (`.github/workflows/ci.yml`):
- **lint** and **security** run in parallel
- **build** (+ tests + coverage) runs only after both pass (`needs: [lint, security]`)
- Weekly scheduled run (Monday 04:25 UTC) for CodeQL and security checks
- Security job: CodeQL analysis, gosec (SARIF), govulncheck (JSON artifact)
- Build job: tests with coverage, Cobertura XML report, PR coverage comment
- Dependabot is configured for Go module and GitHub Actions updates (`.github/dependabot.yml`)

## Pre-commit Hook

The `.githooks/pre-commit` hook runs lint, gosec, govulncheck, and tests before each commit. Activate with `make setup` (from `src/`), which sets `core.hooksPath` to `.githooks/`. The same checks are available as a VS Code task ("Pre-commit Hook") for manual runs.

## Provider Configuration

Every provider setting resolves from three sources, in order (see `resolveSettings` in `src/internal/provider/config.go`):

1. The attribute in the `provider` block
2. The **prefixed** environment variable, when the `prefix` attribute is set
3. The unprefixed `TENABLEIO_*` environment variable

Fallback is **per variable, not per group**: an aliased provider can set only `TENABLEIO_EU_PROXY_AUTH_VALUE` and still inherit the shared `TENABLEIO_ACCESS_KEY`/`TENABLEIO_SECRET_KEY`. This is what makes multiple provider instances (one per region, per proxy, per tenant) workable:

```terraform
provider "tenableio" {
  alias  = "eu"
  prefix = "TENABLEIO_EU"   # reads TENABLEIO_EU_ACCESS_KEY, TENABLEIO_EU_SECRET_KEY, ...
}
```

Rules enforced in `config.go`:

- A prefix is trimmed of trailing `_` and must match `^[A-Za-z_][A-Za-z0-9_]*$`, so it can only produce valid env var names.
- An environment variable that is set but **empty** counts as unset and falls through to the next source.
- `proxy_auth_header` and `proxy_auth_value` are validated **together on the resolved values** (not on the config), since each half can arrive from a different source. Exactly one of the two set is an error; neither set is fine (no header is sent).
- Any provider attribute that is unknown at plan time is a hard error — provider config must be resolvable during plan.
- Diagnostics name the variable *that instance actually reads* (`TENABLEIO_EU_ACCESS_KEY`, not `TENABLEIO_ACCESS_KEY`).

## Environment Variables

Each of these also has a prefixed form (`<PREFIX>_ACCESS_KEY` etc.) when `prefix` is set on the provider block:

- `TENABLEIO_ACCESS_KEY` — Tenable.io API access key
- `TENABLEIO_SECRET_KEY` — Tenable.io API secret key
- `TENABLEIO_BASE_URL` — Override base URL (default: `https://cloud.tenable.com`)
- `TENABLEIO_PROXY_AUTH_HEADER` — Name of an extra HTTP header sent on every API request (e.g. `Proxy-Authorization`)
- `TENABLEIO_PROXY_AUTH_VALUE` — Value for that header

Not prefixable:

- `TF_ACC` — Set to `1` to run acceptance tests
- `TF_LOG` — Terraform log level (default: INFO in devcontainer)

## Tags and Dynamic Tag Filters

A tag value is **static** (applied to assets manually) or **dynamic** (Tenable.io applies it automatically to every asset matching a set of rules — what the UI presents as filters on asset class, IPv4, operating system, and so on). What makes a tag dynamic is the presence of a `filters` object on `POST /tags/values` or `PUT /tags/values/{uuid}`; the API then reports `type: "dynamic"`.

`tenableio_tag_value` exposes this as a nested `filters.asset.and` / `filters.asset.or` list of `{property, operator, values}` rules. `tenableio_tag_asset_filters` (`GET /tags/assets/filters`) lists every property usable as a rule, its supported operators, and its dropdown options — that endpoint is the source of truth for valid `property`/`operator` values, so point users at it rather than hard-coding a list.

**The request and response shapes differ, and this drives most of the implementation:**

| | Request (POST/PUT) | Response (GET details) |
|---|---|---|
| `filters.asset` | JSON object | JSON-formatted **string** |
| attribute key | `property` | `field` |
| operator | readable (`equals`) | short code (`eq`, `match`, `wc`) |
| value | string *or* array of strings | either |

Consequences baked into the code, do not "simplify" them away:

- `client.TagRule` has custom `MarshalJSON`/`UnmarshalJSON` ([tags.go](src/internal/client/tags.go)) — it emits `property` and collapses a single value to a bare string, and it parses both `property` and `field` plus both value forms. `TagValueResponseFilters.ParseAssetRules()` decodes the stringified response.
- Because the echo cannot be compared verbatim with configuration, `reconcileFilters` in [tag_value.go](src/internal/resources/tag_value.go) keeps the **configured** rules authoritative while the tag stays dynamic. Only coarse drift is detected: a tag turned static out-of-band clears `filters` from state, and a tag with no filters in state (import, or made dynamic out-of-band) adopts the parsed response rules.
- Removing `filters` from a dynamic tag forces replacement (`filtersRemovalRequiresReplace` plan modifier), because the API does not document whether omitting `filters` on update preserves or clears the rules.

Limits from the API: 40 rules per tag, 1,024 values per rule, 1 MB request body.

**Unverified against a live tenant** (implemented from docs only) — confirm with `TF_ACC=1` before relying on them: static→dynamic conversion via update, the exact response echo format, and clear-on-omit semantics.

## Conventions

- Provider name: `tenableio`
- Resource naming: `tenableio_<resource>` (e.g., `tenableio_scan`, `tenableio_folder`)
- Data source naming: `tenableio_<resource>` or `tenableio_<resources>` (plural for lists)
- Use Terraform Plugin Framework (not SDKv2)
- All API calls go through the centralized client in `internal/client/`
- One file per resource/data source
- Provider settings resolution lives in `internal/provider/config.go`, not in `Configure` — keep `Configure` a thin caller of `resolveSettings` so the resolution logic stays unit-testable without a Terraform run
- Adding a provider attribute means touching four places: the model struct and schema in `provider.go`, `resolveSettings` in `config.go`, the attribute list in `provider_test.go`, and `examples/provider/main.tf` (then `make docs`)
- Adding a resource or data source means registering it in `provider.go` **and** adding it to the expected list in `provider_test.go`, which asserts the full inventory
- JSON quirks (custom `MarshalJSON`/`UnmarshalJSON`, response shapes that differ from request shapes) belong in `internal/client/` behind plain Go types, so resources never deal with raw JSON. Cover each quirk with a table-driven unit test in `internal/client/<name>_test.go` — these run without credentials, unlike acceptance tests

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
