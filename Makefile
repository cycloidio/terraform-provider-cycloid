SHELL = /usr/bin/env
.SHELLFLAGS = sh -c

.DEFAULT: help
.PHONY: help build install tf-generate new-resource install install-provider plan apply destroy docs

help: ## Show this help
	@grep -F -h "##" $(MAKEFILE_LIST) | grep -F -v fgrep | sed -e 's/:.*##/:##/' | column -t -s '##'

build: args=""
build: ## build the provider
	@go build -gcflags 'all=-l' -trimpath $(args)

tf-generate: ## Will regenerate the new provider spec and models
	@tfplugingen-openapi generate --config generator_config.yml --output out_code_spec.json openapi.yaml
	# Catalogs repoository generation is specific
	@./datasource_stacks/gen.sh
	@./resource_catalog_repository/gen.sh
	@tfplugingen-framework generate resources --input ./out_code_spec.json --output .
	@tfplugingen-framework generate data-sources --input ./out_code_spec.json --output .

new-resource: ## Generates boilplate code for new resource R
	@tfplugingen-framework scaffold resource --name $(R) --output-dir ./provider

.ONESHELL: install
install: ## Install the tools
		deps='
		github.com/hashicorp/terraform-plugin-codegen-openapi/cmd/tfplugingen-openapi@latest
		github.com/hashicorp/terraform-plugin-codegen-framework/cmd/tfplugingen-framework@latest
		github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs
		'
		for dep in $deps; do
			echo "installing $dep";
			go install "$dep";
		done

install-provider: ## Install the provider
	@go install .

plan: install-provider ## Will run a plan
	@terraform plan

apply: install-provider ## Will run an apply
	@terraform apply -auto-approve

destroy: install-provider ## Will run a destroy
	@terraform destroy -auto-approve

docs: ## Generates the provider docs
	@tfplugindocs generate --examples-dir examples/ --provider-dir . --provider-name cycloid ./..

