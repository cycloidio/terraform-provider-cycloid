#!/usr/bin/env make

SHELL = /usr/bin/env
.SHELLFLAGS = sh -c

.DEFAULT: help
.PHONY: help lint build install tf-generate new-resource install provider-install
.PHONY: plan apply destroy docs test e2e-tests e2e-test-manual compose
.PHONY:	start-backend stop-backend init-backend wait-for-backend docker-pull clean

help: ## Show this help
	@grep -F -h "##" $(MAKEFILE_LIST) | grep -F -v fgrep | sed -e 's/:.*##/:##/' | column -t -s '##'

build: ARGS=""
build: ## build the provider
	@go build -gcflags "all=-N -l" -trimpath

test: e2e-tests
	@true

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

provider-install: build ## install the provider
	@go install .

plan: install ## Will run a plan
	@terraform plan

apply: install ## Will run an apply
	@terraform apply -auto-approve

destroy: install ## Will run a destroy
	@terraform destroy -auto-approve

docs: ## Generates the provider docs
	@tfplugindocs generate ./..

e2e-tests: TEST_DC_UP="true" TEST_DC_CLEAN="true" TEST_BACKEND_INIT="true" ARGS=""
e2e-tests: ## Run the whole e2e test cases
	go test -v ./tests/e2e -count=1 $(ARGS)

.ONESHELL:
e2e-test-manual: TARGET_DIR CMD="run_test"
e2e-test-manual: ## Execute the test script manually on TARGET_DIR with CMD
	@test -d $(TARGET_DIR) || { echo "folder '$(TARGET_DIR)' does not exists."; exit 1; }
	TF_TEST_TARGET_DIR=$(TARGET_DIR) ./ci/tf-test.sh "$(CMD)"

compose: ARGS="help"
compose: ## docker compose alias that uses the project ./ci/dc.sh, pass arguments with ARGS="..."
		./ci/dc.sh $(ARGS)

list:
	echo $(MAKEFILE_LIST)

start-backend: ## Start the backend
	./ci/dc.sh cmd up -d

init-backend: ## Lauch init script (root org, admin and token creation)
	./ci/init-backend.sh init_root_org

stop-backend: ## Stop the backend and clean volumes
	./ci/dc.sh cmd down -v

wait-for-backend: ## wait for backend to be healthy
	@./ci/wait-for-backend.sh

clean: stop-backend ## Clean the project
	@true

docker-pull: ## pull the backend image
	test -x "$(which aws)" || { echo "aws cli with credentials is required to pull the backend image"; exit 1; }
	aws ecr get-login-password --region eu-west-1 | docker login --username aws --password-stdin 661913936052.dkr.ecr.eu-west-1.amazonaws.com

lint-sh: ## lint sh files
	@shellcheck ci/*.sh -x ci/lib.sh

lint-go: ## lint go files
	@go vet -c 2

lint: lint-sh lint-go ## lint files
	@true
