# Terraform Provider Cycloid

How it works:

```hcl
terraform {
  required_providers {
    cycloid = {
      source = "cycloidio/cycloid"
    }
  }
}

resource "cycloid_organization" "child_test" {
  name = "terraform organization test"
}

provider "cycloid" {
  url                    = "URL"
  jwt                    = "JWT"
  organization_canonical = "ORG_CAN"
}
```

## Contibuting (cycloid only)

Please look at the [DEVELOPING_TIPS](./DEVELOPING_TIPS.md) file.

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
│  - cycloidio/cycloid in /home/stammfrei/projects/cycloid/terraform-provider-cycloid.git/branches/24-stack-visibility-support
│
│ The behavior may therefore not match any released version of the provider and applying changes may cause the state to become incompatible with published releases.
╵
```

You can look in `tests/e2e` for example of minimal terraform files for testing the provider.
