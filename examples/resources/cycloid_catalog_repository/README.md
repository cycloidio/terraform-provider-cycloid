# Catalog repository Example

This example will show you how to use Terraform to create the Catalog repository in Cycloid

It consists of the following steps:
1. Create a new credential in Cycloid
2. Create a new catalog repository in Cycloid

Once the catalog repository is created with Terraform you can head over to the Cycloid console and see the new catalog repository.

## Run the example

> [!WARNING]  
> Remember to load yours TF variables before running the example.

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
