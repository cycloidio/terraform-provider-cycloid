# Credential Resource Example

This example will guide you on how to use Terraform to create a credential in Cycloid.

The process consists of the following steps:
1. Create a new credential in Cycloid

Once the credential is created with Terraform, you can navigate to the Cycloid console to view the new credential.

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
