# External backend Resource Example

This example will guide you on how to use Terraform to create a external backend in Cycloid.

The process consists of the following steps:
1. Create a new credential in Cycloid
2. Create a new external backend in Cycloid

Once the external backend is created with Terraform, you can navigate to the Cycloid console to view the new external backend.

## Run the example

> [!WARNING]
> Ensure to load your TF variables before running the example.

From inside of this directory:

```bash
terraform init
terraform plan -out theplan
terraform apply theplan
```

## Remove the example

To remove the resources created by this example, run the following command:

```bash
terraform destroy
```
