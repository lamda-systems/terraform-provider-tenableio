# QA

Real Terraform, run against the [mock API](../mockapi/README.md), with no
credentials and no live tenant.

```bash
./run.sh          # everything: guards, then both stacks under every profile
./run.sh test tags
```

The suite builds the provider from `src/`, points Terraform at the binary
through a `dev_overrides` block (so there is no registry download and no
`terraform init`), starts a mock, and drives the stacks through a full
lifecycle.

## What a passing stack means

For each stack, in order:

1. **apply** succeeds.
2. **re-plan is empty.** This is the assertion that matters. An empty diff is
   the proof that the provider wrote back exactly what it planned. Nearly every
   bug this harness has caught showed up here first.
3. **second apply is a no-op**, which catches an Update path that breaks even
   when nothing changed.
4. **outputs match** their expected values — the checks in `check_outputs`
   pin the things that must survive a round trip, like a description defaulting
   to `""` and a name being echoed back unfolded.
5. **destroy** succeeds.

## Profiles

Each profile restarts the mock with different quirks. A correct provider has to
survive all of them, because the Tenable.io documentation does not say which
behaviour production actually implements.

| Profile | Mock configuration | Asks |
|---|---|---|
| `strict` | defaults | Does the provider work against the conservative reading? |
| `preserve` | `MOCK_OMITTED_DESCRIPTION=preserves`, `MOCK_OMITTED_FILTERS=preserves` | Does it survive an API that keeps fields left out of an update? |
| `normalise` | `MOCK_LOWERCASE_CATEGORY_NAMES=1` | Does it cope with an API that rewrites what it was sent? |

`preserve` is the one that used to break. A provider that declares `description`
with a `""` default but serialises it with Go's `omitempty` never puts the key
on the wire when a user clears it, so the server echoes the stale text back.

## Expected failures

Two kinds of failure are the *correct* outcome, and the harness asserts them
rather than tolerating them.

**`guards/`** holds deliberately invalid configuration. `terraform validate`
must reject every directory in there. Today that covers leading and trailing
whitespace, which Tenable.io strips server-side — so a stray space would plan
one string and apply another. Rejecting it at validate time stops that before
any API call. A guard that starts *passing* validation is a regression.

**`EXPECT_APPLY_ERROR`** in `run.sh` lists stack/profile pairs where the apply
is supposed to stop with a provider error:

```
EXPECT_APPLY_ERROR=("tags:normalise:Tenable.io Stored a Different Value")
```

Under `normalise` the server rewrites category names, and `name` is a Required
attribute. Terraform has no way to reconcile the configured value with the
stored one — whichever side state records, the other disagrees forever. The
provider detects it and fails loudly instead of leaving a plan that proposes the
same change on every run. Here a *clean* apply would be the bug, so the harness
fails if the guard does not fire.

## Stacks

| Stack | Covers |
|---|---|
| `stacks/tags` | `tag_category`, `tag_value` (static and dynamic, single- and multi-valued filters, `and` + `or`), and the three tag data sources |
| `stacks/core` | `folder`, `network`, `agent_group`, `exclusion`, `policy`, `scan`, and their data sources, wired together so dependency ordering is exercised too |

Both stacks deliberately include resources that omit optional attributes, so the
schema defaults (`""`, `180`, `private`) are pinned end to end rather than only
in the happy path.

## Working interactively

```bash
./run.sh up              # start a mock and leave it running
./run.sh up preserve     # ... with a profile's quirks
./run.sh apply tags
./run.sh destroy tags
./run.sh down
./run.sh clean           # stop the mock, drop tfstate, remove the built binary
```

Inspect what the provider actually sent while a stack is up:

```bash
curl -s 'http://127.0.0.1:8080/__mock/requests?method=PUT' | jq '.requests[] | {path, body_keys}'
```

`body_keys` lists the keys physically present in each request body. Asserting a
key is *absent* is how a serialiser that drops a meaningful empty value gets
caught; inspecting values cannot do it.

## About persistence

The mock keeps everything in memory, so its contents live and die with the
process. That is deliberate and, for this harness, sufficient: a full run starts
the mock, applies, and destroys within one process lifetime, and a fresh
container per profile is exactly what you want for repeatable results.

It only matters for interactive work. If you `./run.sh up`, apply a stack, then
restart the mock, the Terraform state file still refers to objects that no
longer exist and the next plan will propose recreating everything. Either avoid
restarting mid-session, or run `./run.sh clean` to reset both sides together.

If keeping a stack alive across restarts ever becomes worth it, the mock's store
is a handful of plain dicts — dumping it to JSON on shutdown and reloading on
boot behind a `MOCK_STATE_FILE` variable would be a small change. It has not
been needed yet, so it has not been built.

## Layout

```
qa/
├── run.sh                    # build, mock lifecycle, profiles, assertions
├── guards/whitespace/        # must FAIL terraform validate
└── stacks/
    ├── tags/
    └── core/
```

Generated at runtime and gitignored: `.bin/`, `.dev.tfrc`, `.mock.pid`,
`.mock.log`, `.last-run.log`, and each stack's `terraform.tfstate`.
