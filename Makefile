.PHONY: help
help: ## Show this help
	@grep -F -h "##" $(MAKEFILE_LIST) | grep -F -v fgrep | sed -e 's/:.*##/:##/' | column -t -s '##'

.PHONY: tf-generate
tf-generate: ## Will regenerate the new provider spec and models
	@tfplugingen-openapi generate --config generator_config.yml --output out_code_spec.json openapi.yaml
	@tfplugingen-framework generate resources --input ./out_code_spec.json --output .

.PHONY: install
install: ## Install the provider
	@go install .

.PHONY: plan
plan: install ## Will run a plan
	@terraform plan

.PHONY: apply
apply: install ## Will run an apply
	@terraform apply -auto-approve

.PHONY: destroy
destroy: install ## Will run a destroy
	@terraform destroy -auto-approve

.PHONY: docs
docs: ## Generates the provider docs
	@tfplugindocs generate ./..

