# e2e tests implementation

In this folder, every subfolder with a name prefixed with `test_` will be included in the e2e tests.

On each folder, the [`./ci/tf-test.sh`](./ci/tf-test.sh) script will be executed from the repo's root as followed:

```console
TF_TEST_TARGET_DIR=<dir_path> ./ci/tf-test.sh run_test
```

by default, all tests are executed in parallel.

## How to add a test

E2e tests are structured in a way that allow ops and dev to easily add a terraform use case in the tests.

All you have to do is to add valid terraform code in a subfolder with the prefix `test_` here.

The folders must be `snake_case` with only alphabetic characters and `_`.

They will be executed automatically when calling `go test`.

### Executing a test manually

You can execute a test by running [`./ci/tf-test.sh`](./ci/tf-test.sh) from the repo's root.

This is the same script used when starting `go test`

### Test implementation and override

You can look up the script implementation [here](./ci/tf-test.sh).

The script will execute the terraform code against the backend. It will do so with the following steps:

```
setup:
  create the org / api key for the test and inject them as TF_VARS

init:
  terraform init

pre_plan_apply:
  perform actions before planning the apply

plan_apply:
  terraform plan for the apply

post_plan_apply:
  actions after terraform plan and before the apply

apply:
  terraform apply of the plan

pre_plan_destroy:
  perform actions before terraform plan for destruction

plan_destroy:
  terraform plan destroy

post_plan_destroy:
  actions after terraform plan for destruction

destroy:
  terraform destroy

cleanup:
  any cleanup step

run_test:
  run the full test in all step order
```

You can override a step by putting a \${step_name}.sh script in the folder of your test, in that case, the script will be sourced in place of the default behavior.

You can also override the `run_test` order by overriding the command by making a `run_test.sh` script in your test folder.

The scripts must be executable to be used as override. It will have access to the same context as the _override_step function in this script.

If a vars.tfvars is set on the current repo, it will be added as a -var-file arg to the default steps.

By default, you test will have a cycloid child org create with from the folder name without the leading `test_` and `_` will be replaced to `-`.
  > Exemple: from a `test_resource_stack_visibiltiy` folder name, you will have an org named `resource-stack-visibility` org.

/!\\ The org is created at the `setup` stage, so if you override this step, be careful.

The API_KEY and org will be passed to terraform via the `TF_VAR_cycloid_api_key` and `TF_VAR_cycloid_org` variable.

For example, if you want to override the `run_test` step, you can do:

```sh
#! /usr/bin/env sh

# Let's only call plan apply destroy for a simple case:
setup

plan_apply

apply

destroy
```

All scripts on this repo have been written in [POSIX sh](https://www.grymoire.com/Unix/Sh.html) to accomodate MacOS users. Please, keep it that way to avoid issues with local testing.

### Making a test fail

Use `exit <code>` where code is different than `0`:

```sh
exit 1
```
