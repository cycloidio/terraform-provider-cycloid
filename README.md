# Terraform Provider Cycloid

How to generate it

Read https://developer.hashicorp.com/terraform/plugin/code-generation/openapi-generator#generator-config

```
tfplugingen-openapi generate --config generator_config.yml --output out_code_spec.json openapi.yaml
```

If it's a new resource run

```
tfplugingen-framework scaffold resource --name credential --output-dir ./provider
```

And add to the `provider/provider.go#Resources` the `New*` for the generated resource

then run

```
tfplugingen-framework generate resources   --input ./out_code_spec.json   --output ./provider
```

Which generates all the resources from the spec generated previously
