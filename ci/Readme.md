# Tooling readme

This is where the project's tooling lies.

## Rules

- All scripts must be written in POSIX sh
- All scripts must be executed from repo's root
- Use the `.ci/` folder from repo's root as cache / temporary folder
- All functions/lib must be in the `lib.sh` file. Don't try to split it, it's too cumbersome in sh.
- Every script should print an help when called with `./script.sh -h`
- All CI procedure MUST be in specific `ci-` prefixed scripts

## Script summary

Look up their `./script --help`, here is a summary:

- `dc.sh` -> docker compose wrapper that will add the missing secrets using `cy`
- `init-backend.sh` -> create root or child orgs, with admin and api-key and authorize.
- `lib.sh` -> contains all the common stuff
- `tf-test.sh` -> script that implement the e2e testing
  -> The `tf_test_default_steps` contains the default testing steps
- `wait-for-backend` -> wait for the backend to be healthy
