# Thin wrapper around just. Run "make help" or "just help" for targets.
# Requires just: https://github.com/casey/just

.DEFAULT_GOAL := help

help:
	@just help

build:
	@just build $(args)

test:
	@just test

test-unit:
	@just test-unit

test-acc:
	@just test-acc

test-acc-one:
	@just test-acc-one $(TEST)

convert-swagger:
	@just convert-swagger

tf-generate:
	@just tf-generate

new-resource:
	@just new-resource $(R)

install:
	@just install

install-provider:
	@just install-provider

plan:
	@just plan

apply:
	@just apply

destroy:
	@just destroy

docs:
	@just docs

watch:
	@just watch

playground:
	@just playground
