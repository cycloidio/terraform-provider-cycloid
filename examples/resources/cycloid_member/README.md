# Cycloid Member Example

This example will guide you on how to use Terraform to create an organization member in Cycloid.

The process consists of the following steps:
1. Create a new member invitation in Cycloid

Once the member is created with Terraform, you can navigate to the Cycloid console to view the new member.

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
