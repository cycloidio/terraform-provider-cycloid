# Test configuration (acceptance tests)

Acceptance tests use a **centralised config** for dependency data (credentials, repos, etc.) so the test environment can be configured without editing code.

## Config file and env override

- **Config file:** `test_config.yaml` at the **repo root** (directory containing `go.mod`) is the single source of default values; the loader does not use in-code defaults. The loader also checks `provider/` and current dir. If no file is found, loading fails with a clear error.
- **Override with env vars:** Any key can be overridden with the `CY_TEST_` prefix. Examples:
  - `CY_TEST_CONFIG_REPOSITORY` – config repository canonical (used when creating projects)
  - `CY_TEST_REPOSITORIES_CONFIG_URL`, `CY_TEST_REPOSITORIES_CONFIG_CREDENTIAL` – config repo
  - `CY_TEST_REPOSITORIES_CATALOG_URL`, `CY_TEST_REPOSITORIES_CATALOG_BRANCH`, `CY_TEST_REPOSITORIES_CATALOG_CREDENTIAL` – catalog repo
- **Custom config file path:** Set `CY_TEST_CONFIG_FILE` to the path to a YAML file (overrides default path).

## Loading in tests

- **TestDependencyManager** uses the config for `CreateTestProject` (config repository canonical).
- **Helpers:** Call `testAccGetTestConfig(t)` to get `*TestConfig` and use `cfg.ConfigRepository`, `cfg.Repositories.Config` (URL, Branch, Credential), `cfg.Repositories.Catalog`, etc.

See `test_config.yaml` in this directory for the structure and `provider/testconfig.go` for the loader (Viper + YAML + env).
