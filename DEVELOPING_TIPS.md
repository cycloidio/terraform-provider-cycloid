# Tips

## How to work on it

Docs: https://developer.hashicorp.com/terraform/plugin/code-generation/openapi-generator#generator-config 

We have 2 required files:
* `openapi.yaml`: Which is the Swagger spec of the Cycloid API
* `generator_config.yaml`: Which is the definition of the Provider spec. It basically says which parts of the `openapi.yaml` are part of the Provider when generating

Any change to the `openapi.yaml` or `generator_config.yaml` will require to run 

```
tfplugingen-openapi generate --config generator_config.yml --output out_code_spec.json openapi.yaml
```

--
KNOWN BUGS:

**Credentials**

Renamed `raw` to `body` on the main NewCredentials. I've commented the `body` from the `Credential` because it's causing issues as it's generating multiple types and it does not compile. We should investigate this.

From the [docs](https://github.com/hashicorp/terraform-plugin-codegen-openapi/blob/main/DESIGN.md#resources) it should not be doing this

> Arrays and Objects will have their child attributes merged, so example_object.string_field and example_object.bool_field will be merged into the same SingleNestedAttribute schema.


**Organizaitons**

On Organization the `admins` are `MemberOrg` which itself has a `invited_by` which is also `MemberOrg` so it's a buckle and it needs to be broken

--

Which will regenerate the `out_code_spec.json` which is what everything else relies on for generating.

From that using the `generator_config.yaml` we can us it to generate 2 things

*Provider*

```
tfplugingen-framework generate provider --input ./out_code_spec.json --output ./provider
```

*Resources*

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
