# Tenable.io mock API

An in-memory fake of every Tenable.io endpoint the Terraform provider calls,
built with FastAPI. It exists so the provider can be exercised end to end —
`terraform plan`, `apply`, `destroy`, import, drift — without credentials and
without touching a live tenant.

The design rule is **fidelity over convenience**. The mock echoes back exactly
what it was given, rejects what the real API rejects, and reproduces the
response shapes that differ from the request shapes. A mock more forgiving than
production is worse than no mock at all, because it certifies provider bugs as
passing.

## Quick start

```bash
cd mockapi
make venv          # one-off: create .venv and install pinned deps
make test          # 144 tests
make run           # serves on 0.0.0.0:8080
```

Point the provider at it:

```bash
export TENABLEIO_BASE_URL=http://127.0.0.1:8080
export TENABLEIO_ACCESS_KEY=anything
export TENABLEIO_SECRET_KEY=anything
```

Any syntactically valid `X-ApiKeys` header is accepted unless you pin
`MOCK_ACCESS_KEY` / `MOCK_SECRET_KEY`. Browse the schema at
`http://127.0.0.1:8080/docs`.

## What it covers

| Family | Endpoints |
|---|---|
| Tag categories | `POST`/`GET` `/tags/categories`, `GET`/`PUT`/`DELETE` `/tags/categories/{uuid}` |
| Tag values | `POST`/`GET` `/tags/values`, `GET`/`PUT`/`DELETE` `/tags/values/{uuid}` |
| Asset filters | `GET /tags/assets/filters` |
| Folders | `POST`/`GET` `/folders`, `PUT`/`DELETE` `/folders/{id}` |
| Networks | `POST`/`GET` `/networks`, `GET`/`PUT`/`DELETE` `/networks/{uuid}` |
| Exclusions | `POST`/`GET` `/exclusions`, `GET`/`PUT`/`DELETE` `/exclusions/{id}` |
| Agent groups | `POST`/`GET` `/scanners/null/agent-groups`, `GET`/`DELETE` `/…/{id}` |
| Scanners | `GET /scanners`, `GET /scanners/{id}` (read-only) |
| Policies | `POST`/`GET` `/policies`, `GET`/`PUT`/`DELETE` `/policies/{id}` |
| Scans | `POST`/`GET` `/scans`, `GET`/`PUT`/`DELETE` `/scans/{id}` |
| Workbenches | `GET /workbenches/assets`, `GET /workbenches/assets/{id}/info` (read-only) |

## The shapes it deliberately reproduces

These are the places where the API is not self-consistent. Every one of them is
a chance for a provider to be subtly wrong, so the mock insists on them.

**Dynamic tag filters change shape between write and read.** `POST`/`PUT`
accept a `filters` *object*; every read path returns a JSON-formatted *string*,
with `property` renamed to `field`, readable operators shortened to codes, and a
single-element value list collapsed to a bare string:

```jsonc
// sent
{"asset": {"and": [{"property": "operating_system", "operator": "equals", "value": ["FreeBSD"]}]}}

// returned
{"asset": "{\"and\":[{\"field\":\"operating_system\",\"operator\":\"eq\",\"value\":\"FreeBSD\"}]}"}
```

**Creating a category rejects a duplicate name; creating a value does not.**
`POST /tags/categories` with an existing name returns 400 *"A category with the
name you specified already exists."* — it does **not** return the existing
category. But `POST /tags/values` with a `category_name` will happily create or
reuse a category. The asymmetry is real.

**Scans rename half their keys between `POST` and `GET`.** `POST /scans`
returns `{"scan": {...}}` with `id`, `text_targets`, `emails`. `GET /scans/{id}`
returns `{"info": {...}}` with `object_id`, `targets`,
`notification_email_address`, and the template under `scanner_name`.

**Several writes return nothing.** `POST /folders` returns only `{"id": N}`.
`PUT /folders/{id}`, `PUT /policies/{id}`, `PUT /scans/{id}` and every `DELETE`
return an empty body.

**Errors are Tenable-shaped, not FastAPI-shaped.** Validation failures come back
as 400 `{"statusCode", "error", "message"}`, never 422 `{"detail": [...]}`.

## Quirks: the behaviours the docs leave ambiguous

Where the public documentation genuinely does not resolve a behaviour, the mock
refuses to guess. It implements the conservative reading by default and exposes
the alternative as an environment variable. **A correct provider passes with
every combination** — that is what the switches are for.

| Variable | Default | Alternative | What it changes |
|---|---|---|---|
| `MOCK_OMITTED_DESCRIPTION` | `clears` | `preserves` | Whether a `PUT` with no `description` key wipes the stored text or keeps it |
| `MOCK_OMITTED_FILTERS` | `clears` | `preserves` | Whether a `PUT` with no `filters` key reverts a dynamic tag to static or keeps its rules |
| `MOCK_LOWERCASE_CATEGORY_NAMES` | off | on | Folds category names to lower case and echoes the folded form |
| `MOCK_REJECT_UNKNOWN_FIELDS` | off | on | Lint mode: 400 on a body field the endpoint does not define |

Why these matter:

- **`MOCK_OMITTED_DESCRIPTION=preserves`** reproduces the class of failure that
  motivated this mock. A provider that declares `description` with a `""`
  default but serialises it with Go's `omitempty` never puts the key on the
  wire when a user clears the field. The server echoes the stale text, and
  Terraform aborts with *"Provider produced inconsistent result after apply:
  .description: was cty.StringVal(""), now cty.StringVal("…")"*. Under
  `clears`, the identical provider looks fine — which is exactly why testing
  against one setting proves nothing.
- **`MOCK_LOWERCASE_CATEGORY_NAMES=on`** does the same for `name`. The
  documented example for `POST /tags/values` sends `"category_name": "Location"`
  and receives `"category_name": "location"`, which suggests production
  normalises, though nothing states it in prose. Any provider attribute that
  writes an echoed name straight into state fails its apply when this is on.
- **`MOCK_OMITTED_FILTERS`** is why the provider forces a replacement rather
  than an update when filters are removed from a dynamic tag: neither answer is
  documented, so it cannot rely on either.

Other settings: `MOCK_ACCESS_KEY`, `MOCK_SECRET_KEY`, `MOCK_USER`, `MOCK_SEED`
(default on), `MOCK_FROZEN_CLOCK` (default on, for byte-reproducible responses).

`make run-quirky` starts the server with the two quirks that reproduce the
reported failure already enabled.

## Inspecting what the provider actually sent

The mock records every request. This is how you assert on what went **on the
wire**, rather than on what the provider meant to send.

```bash
curl -s 'http://127.0.0.1:8080/__mock/requests?method=PUT&path=/tags/categories/UUID' | jq
```

```jsonc
{
  "method": "PUT",
  "path": "/tags/categories/00000000-0000-4000-8000-000000000004",
  "status": 200,
  "body": {"name": "Location"},
  "body_keys": ["name"]        // no "description" — the key never left the provider
}
```

`body_keys` is the important field. A body carrying `description: ""` is an
instruction to clear; a body with no `description` key is silence. They are
different requests, and only the key list tells them apart — inspecting values
cannot. That distinction is precisely what `omitempty` erases.

Admin endpoints (unauthenticated, never recorded, namespaced so they can never
collide with a real path):

| Endpoint | Purpose |
|---|---|
| `GET /__mock/health` | Readiness probe |
| `GET /__mock/requests` | Request log; filter with `?method=` and `?path=` |
| `POST /__mock/reset` | Clear all state and re-seed; `?requests_only=true` keeps objects |
| `GET /__mock/settings` | The active quirks, so a failing pipeline can report what it ran against |

## Driving it from the Go tests

`src/internal/client/mockapi_integration_test.go` runs the provider's real HTTP
client against this mock, which is what proves the two agree on the wire — that
the shapes returned here actually deserialize into the client's structs. The
tests skip unless `TENABLEIO_MOCK_URL` is set, so `make test` on the Go side
stays credential- and network-free:

```bash
cd mockapi && make run                    # one shell
cd src && TENABLEIO_MOCK_URL=http://127.0.0.1:8080 go test ./internal/client/ -run Mock -v
```

Each test resets the mock through `POST /__mock/reset` first, so they are
order-independent.

## Determinism

Identifiers are generated in sequence — UUIDs as
`00000000-0000-4000-8000-{n:012d}`, integer IDs as a shared counter — and the
clock is frozen at `2026-01-01T00:00:00Z`. A fixed sequence of requests always
produces byte-identical responses, so acceptance tests can assert on concrete
values. Set `MOCK_FROZEN_CLOCK=0` for a clock that advances.

Seeding covers only what the API cannot create: two scanners, two workbench
assets, the `My Scans` and `Trash` system folders, and the default network. Tag
categories and values are deliberately **not** seeded, so the request that
creates one is always visible in the log.

## Running in Docker

```bash
make docker-build
docker run --rm -p 8080:8080 -e MOCK_OMITTED_DESCRIPTION=preserves tenableio-mock:local
```

The devcontainer's compose file also defines a `mockapi` service, reachable from
the Go container at `http://mockapi:8080`.

## Layout

```
mockapi/
├── tenableio_mock/
│   ├── app.py          # factory, auth + recording middleware, admin endpoints
│   ├── config.py       # Settings and Quirks, all environment-driven
│   ├── errors.py       # Tenable-shaped error envelope, FastAPI handler overrides
│   ├── filters.py      # the request→response tag rule transform
│   ├── models.py       # pydantic request bodies
│   ├── seed.py         # baseline objects the API cannot create
│   ├── store.py        # in-memory state, deterministic identifiers
│   └── routers/        # one module per resource family
└── tests/              # 144 tests, no credentials needed
```

## Related

`qa/` drives real Terraform against this mock — see [qa/README.md](../qa/README.md).
Its `run.sh` starts and stops the server for you and runs every stack under each
quirk profile.
