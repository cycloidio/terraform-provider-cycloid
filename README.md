# Terraform Provider Cycloid

How to generate it

Read https://developer.hashicorp.com/terraform/plugin/code-generation/openapi-generator#generator-config

```
tfplugingen-openapi generate --config generator_config.yml --output out_code_spec.json openapi.yaml
```

then read https://developer.hashicorp.com/terraform/plugin/code-generation/framework-generator

```
tfplugingen-framework generate all --input out_code_spec.json --output provider
```
