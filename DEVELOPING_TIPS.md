# Tips

## Dev tooling

### Requirements

To make the dev environment work, you need those binaries installed on your machine.

No environment is provided, so you will have to install them yourself.

| executable | version | doc | required |
| ---        | ---     | --- | ---      |
| `go` | v1.22.9 | | true |
| `cy` | latest  | Will be used to fetch credentials from our prod console. You need to have a configured access to our `cycloid` org, see [the docs](https://docs.cycloid.io/reference/cli/) | true |
| `jq` | latest | to parse json, required in scripts | true |
| `curl` | any | fetch stuff | true |
| `docker` | latest | a recent version with `docker compose` plugin. | true |
| `openapi-generator-cli` | latest | required for the swagger to openAPI conversion [docs](openapi-generator-cli) | true |
| `terraform-plugin-doc` | latest | required for docgen [docs](https://github.com/hashicorp/terraform-plugin-docs) | true |
| `make` | any | run tasks | true |
| `shellcheck` | latest | lint shell files | false |

All scripts are wrote in POSIX sh (for MacOs/Linux/WSL compatibility).

### Local Backend for testing

The local backend is served via `compose.yml` file used through the [`./ci/dc.sh`](./ci/dc.sh) script (for secret injection).

use the following make commands to interact with it:

To initialize the backend, you can use the `./ci/init_backend.sh` script, see examples below.

<details>
<summary>

`make` backend commands

</summary>
All commands are idempotent (meaning you can repeat them to get the same result).

> [!Note]
> API key for an org will change at each execution.
> The script will, by default, write the API key to a file named`./.ci/${org_name}/api-key`.
> The default root org is `fake-root`

```
start-backend:        Start the backend
init-backend:         Lauch init script (root org, admin and token creation)
stop-backend:         Stop the backend and clean volumes
wait-for-backend:     wait for backend to be healthy
docker-pull:          pull the backend image
```

</details>

### Testing

E2E testing has been implemented using terraform files and a script called [./ci/tf-test.sh](./ci/tf-test.sh). Documentation on the implementation is [here](./tests/e2e/README.md)

The tests can be either started manually on a test case using the script or using the `go test` to start them all in parallel:

manually (from repo root): `TF_TEST_TARGET_DIR=<tf_test_dir> ./ci/tf-test.sh run_test`
via go: `go test -v ./... -count=1`.

You can use helper commands in `make`:

<details>
<summary>using `make` commands</summary>

```
e2e-tests:           run the whole e2e test cases
e2e-test-manual:     execute the test script manually on target_dir with cmd
```
</details>


By default, tests using `go test` will spin up the backend using the `./ci/dc.sh up_default` command, execute the tests, then exec `./ci/dc.sh clean`.

You can change this behavior with environment variables, see the section below.

#### Test Configuration

If you need to add configuration to a test, it must be via environment variable and documented here.

##### `go test` configuration

By default, `go test` will spin up the backend and before tests and clean it afterwards.

Since most of terraform actions are idempotent anyway, you can speed up your `dev -> build -> test` loop by persisting the backend between test runs.

Configuration is made using environment variable, see below:

<details>
<summary>go test config</summary>

[Implementation](./tests/e2e/main_test.go)

```
// if true, tests will spin up a backend using docker compose - implies TEST_DC_CLEAN=true
TEST_DC_UP=true
// if true, tests will clean the backend after tests are runned
TEST_DC_CLEAN=true
// if true, tests will init admin user and API_KEY before testing, the init is idempotent
TEST_BACKEND_INIT=true

// You can provide yourself the backend licence, but it will fetch the staging one by default using cy
API_LICENCE_KEY=
```
</details>

#### Adding tests

see [the tests Readme](./tests/e2e/README.md) for more information on how to add e2e tests.

## Code generation

Docs: https://developer.hashicorp.com/terraform/plugin/code-generation/openapi-generator#generator-config

We have 2 required files:
* `openapi.yaml`: Which is the Swagger spec of the Cycloid API
* `generator_config.yaml`: Which is the definition of the Provider spec. It basically says which parts of the `openapi.yaml` are part of the Provider when generating

The `tfplugingen-openapi` has several issues, namely it doesn't support (issues exists since a long time on this and it's not going to be fixed anytime soon):
- Recursive attributes (we have one issue with the admin model that is a memberOf)
- `$ref` or attributed merging

So a script has been made in go, contained in [the swagger_converter directory](./swagger_converter) that will:
  1. Fetch our production swagger
  1. Convert the swagger to an openAPI v3 using `openapi-generator-cli`
  1. Convert stuff (Injecting attributes, merging `$ref`, etc...)
  1. Output the generated openAPI on `./openapi.yaml` at repo's

from that we can generate the `out_code_spec.json` using `tfplugingen-openapi` (see `tf-generate` target).

Once the `out_code_spec.json` is generated, the `tfplugingen-framework` cli will generate the code.

> [!CAUTION]
> The code generation is not very friendly, and the `tfplugingen-openapi` has a lot of bug
> I advise that we find a way to work closer to the `out_code_spec.json` that is way more flexible
> to use when managing terraform resources.
>
> Maintaining the openAPI of this repo up to date could be really painful, be careful when trying to update.

---
<details>
<summary>KNOWN BUGS:</summary>

**Credentials**

Renamed `raw` to `body` on the main NewCredentials. I've commented the `body` from the `Credential` because it's causing issues as it's generating multiple types and it does not compile. We should investigate this.

From the [docs](https://github.com/hashicorp/terraform-plugin-codegen-openapi/blob/main/DESIGN.md#resources) it should not be doing this

> Arrays and Objects will have their child attributes merged, so example_object.string_field and example_object.bool_field will be merged into the same SingleNestedAttribute schema.


**Organizations**

On Organization the `admins` are `MemberOrg` which itself has a `invited_by` which is also `MemberOrg` so it's a buckle and it needs to be broken.

-> This is fixed by the [conversion script](./swagger_converter/converter.go) that will remove the recursive attribute.

--

</details>


From that using the `out_code_spec.json` we can us it to generate:

#### *Provider*

```
tfplugingen-framework generate provider --input ./out_code_spec.json --output ./provider
```

#### *Resources*

To do generate a *new* resource run:

```
tfplugingen-framework scaffold resource --name credential --output-dir ./provider
```

And add to the `provider/provider.go#Resources` the `New*` for the generated resource so it's part of the list of resources

To *update* an already created resource run:

```
tfplugingen-framework generate resources --input ./out_code_spec.json --output .
```

To regenerate the documentation just use:

```
tfplugindocs generate ./...
```

## How to run it locally

Docs: https://developer.hashicorp.com/terraform/plugin/code-generation/workflow-example#setup-terraform-for-testing

On the `$HOME` add the following on the file `.terraformrc`:

```hcl
provider_installation {

  dev_overrides {
    # Example GOBIN path, will need to be replaced with your own GOBIN path. Default is $GOPATH/bin
    "registry.terraform.io/cycloid/cycloid" = "/home/xescugc/go/bin"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
```

Then you can just run `terraform plan`, the current `main.tf` is just connecting to staging so that is where things are gonna be created
after you ran the `terraform apply -auto-approve`.
