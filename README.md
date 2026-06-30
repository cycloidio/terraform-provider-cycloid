# Terraform Provider Cycloid

## User documentation

Follow the provider documentation on the [terraform registry](https://registry.terraform.io/providers/cycloidio/cycloid/latest/docs).

## Contibuting (cycloid only)

Please look at the [DEVELOPING_TIPS](./DEVELOPING_TIPS.md) file.

## Development environment (devenv.sh)

The dev toolchain (matching Go, `golangci-lint`, `tfplugindocs`, `just`,
`opentofu`/`terraform`, the `docker compose` client and helpers) is pinned in
[`devenv.nix`](./devenv.nix). The same shell is used locally and by CI, so a
green local run matches a green CI run.

With [direnv](https://direnv.net/):

```
direnv allow      # one-time; auto-enters the shell on cd, loads .env if present
```

Without direnv:

```
devenv shell      # enter the toolchain shell
# or run a one-off command in it:
devenv shell -- just test-unit
```

CI runs on Cycloid's self-hosted runner (`runs-on: [self-hosted, cycloid]`) and
enters this same devenv via `devenv shell -- <cmd>`. See
[`.github/workflows/ci.yml`](./.github/workflows/ci.yml).

## Testing

All commands run inside the devenv shell (prefix one-offs with
`devenv shell --`, or enter the shell first with `devenv shell`).

### Standard (unit) tests

No backend or credentials needed — `TF_ACC` is unset so acceptance tests skip:

```
devenv shell -- just test-unit      # go test ./... -short
```

### Acceptance tests

Acceptance tests (`TF_ACC=1`) run against a **local Cycloid backend** brought up
with docker compose (`youdeploy-api`, `plugin-manager`/`-registry`,
`docker-registry`, concourse, vault, db, redis, …). You need:

- `CY_SAAS_API_KEY` — an API key for the `cycloid` org on Cycloid SaaS, used by
  `just env` to mint `.env` (registry/git creds) via `cy uri interpolate`.
- `API_LICENCE_KEY` — the first-boot licence for the local backend.

One-shot (fresh stack → mint `.env` → full suite):

```
export CY_SAAS_API_KEY=…  API_LICENCE_KEY=…
devenv shell -- just test-acc-fresh
```

Or step by step:

```
devenv shell                # enter the shell
just be-start               # bring up the local backend stack
just env                    # mint .env (needs CY_SAAS_API_KEY)
just test-acc               # TF_ACC=1 go test ./... -v
just test-acc-one TestAccProjectResource   # a single test
just be-stop                # tear the stack down (docker compose down -v)
```

CI runs the same suite (parallel-safe) via `just be-start` + `just test-acc`;
it recovers the registry/licence credentials at runtime through the `cy` CLI
instead of `.env`.

## Use a development version of the provider

To use or test a developpement version of the provider you'll need to clone this repo an follow these commands:

#### Build the provider

```
make build
```

#### Install it

```
make install
```

#### Add this file somewhere convenient

Write this for example here `.ci/dev.tfrc`, put the absolute path to this repo.

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/cycloidio/cycloid" = "/absolute/path/to/the/terraform/repo"
  }

  direct {}
}
```

#### Export the `TF_CLI_CONFIG_FILE` env var

The value should point to the file we created

```
TF_CLI_CONFIG_FILE=/absolute/path/to/the/terraform/repo/.ci/dev.tfrc
```

#### Now you can execute terraform with the local provider

```
TF_CLI_CONFIG_FILE=/path/to/.ci/dev.tfrc terraform plan
```

Terraform should output a warning indicating a local dev provider override that looks like this:

```
╷
│ Warning: Provider development overrides are in effect
│
│ The following provider development overrides are set in the CLI configuration:
│  - cycloidio/cycloid in /absolute/path/to/your/terraform-provider-cycloid
│
│ The behavior may therefore not match any released version of the provider and applying changes may cause the state to become incompatible with published releases.
╵
```

You can look in `tests/e2e` for example of minimal terraform files for testing the provider.
