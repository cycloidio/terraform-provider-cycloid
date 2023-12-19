# Terraform Provider Cycloid

How to generate it

Read https://developer.hashicorp.com/terraform/plugin/code-generation/openapi-generator#generator-config

```
tfplugingen-openapi generate --config generator_config.yml --output out_code_spec.json openapi.yaml
```

KNOWN BUG:

Renamed `raw` to `body` on the main Credentials. I've commented the `body` from the `Credential` because it's causing issues as it's generating multiple types and it does not compile. We should investigate this.

From the [docs](https://github.com/hashicorp/terraform-plugin-codegen-openapi/blob/main/DESIGN.md#resources) it should not be doing this

> Arrays and Objects will have their child attributes merged, so example_object.string_field and example_object.bool_field will be merged into the same SingleNestedAttribute schema.

but it looks like it does, I have to test this later and open  an issue to check why as it's a big blocking.

To generate the provider schema run:

```
tfplugingen-framework generate provider --input ./out_code_spec.json --output ./provider
```

If it's a new resource run

```
tfplugingen-framework scaffold resource --name credential --output-dir ./provider
```

And add to the `provider/provider.go#Resources` the `New*` for the generated resource

then run

```
tfplugingen-framework generate resources --input ./out_code_spec.json --output ./provider
```

Which generates all the resources from the spec generated previously
