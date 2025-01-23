# Tooling readme

This is where the project's tooling lies.

## Rules

- All scripts must be written in POSIX sh
- All scripts must be executed from repo's root
- Use the `.ci/` folder from repo's root as cache / temporary folder
- All functions/lib must be in the `lib.sh` file. Don't try to split it, it's too cumbersome in sh.
- Every script should print an help when called with `./script.sh -h`
- All CI procedure MUST be in specific `ci-` prefixed scripts
