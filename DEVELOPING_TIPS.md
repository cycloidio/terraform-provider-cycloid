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
| `openapi-generator-cli` | latest | required for the swagger to openAPI conversion [docs](openapi-generator-cli) | true |
| `terraform-plugin-doc` | latest | required for docgen [docs](https://github.com/hashicorp/terraform-plugin-docs) | true |
| `make` | any | run tasks | true |

## Code generation

Docs: https://developer.hashicorp.com/terraform/plugin/code-generation/openapi-generator#generator-config

We have 2 required files:
* `openapi.yaml`: Which is the Swagger spec of the Cycloid API
* `generator_config.yaml`: Which is the definition of the Provider spec. It basically says which parts of the `openapi.yaml` are part of the Provider when generating

The `tfplugingen-openapi` has several issues, namely it doesn't support (issues exists since a long time on this and it's not going to be fixed anytime soon):
- Recursive attributes (we have one issue with the admin model that is a memberOf)
- `$ref` or attributed merging

So a script has been made in go, contained in [the swagger_converter directory](./swagger_converter).
Its main purpose is to fix the openapi spec to remove or merge problematic attributes.

The script will:
1. Fetch our production swagger
1. Convert the swagger to an openAPI v3 using `openapi-generator-cli`
1. Fix compatibility issues in the OpenAPI (mainly `$ref` and recursive attributes.)
1. Output the generated openAPI on `./openapi.yaml` at repo's

from that we can generate the `out_code_spec.json` using `tfplugingen-openapi` (see `tf-generate` target).

All the current exception handling is made in the Convert function in the `converter.go` file:

<details>

<summary>code</summary>

https://github.com/cycloidio/terraform-provider-cycloid/blob/284bea0538cd047e940b9b49dbd922cef86afc56/swagger_converter/converter.go#L59-L169

</details>

You can append you changes here.

Once the `out_code_spec.json` is generated, the `tfplugingen-framework` cli will generate the code.

> [!CAUTION]
> The code generation is not very friendly, and the `tfplugingen-openapi` has a lot of bug
> I advise that we find a way to work closer to the `out_code_spec.json` that is way more flexible
> to use when managing terraform resources.
>
> If you need to add extensive changes to a resource schema, I would advise to do it this way:
> - If the resource is small, with little or no nested arrays/objects -> write it by hand like the `resource_stack`
> - If you only need to ignore some changes. Generate it using the `generator_config.yml` and use the ignore keyword
> - If you need extensive changes (rename attributes, add attributes outside of the swagger model, and so on)
>   - Create another generator_config.yml only for your resource and put it in the resource folder
>   - Ignore as much field as possible in the gen
>   - Generate the spec.json
>   - Edit the spec.json (try to do it programatically with a `gen.sh` script for example)
>   - Commit the generator config, the spec.json and the script.
>   - add the script to the `tf-generate` target in make
>
> Look up the [catalog repository generation](./resource_catalog_repository/).
>
> Maintaining the openAPI of this repo up to date could be really painful, be careful when trying to update.

---
<details>
<summary>Known bugs</summary>

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

> [!CAUTION]
> Some attributes have been documented manually
> those addition should be pushed in the schema
> Check the diff if you re-generate the docs

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
