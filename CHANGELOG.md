# Changelog

## Unreleased

### Added
- Local docker-compose stack (`compose.yml`) mirroring cycloid-cli's 12-service QA stack
  (youdeploy-api, plugin-manager, plugin-registry, docker-registry, concourse, vault,
  redis, mysql, git-server, mailpit). Managed via `just be-start / be-stop / be-reset`.
- Acceptance test bootstrap via `TestMain` — calls `testcfg.NewConfig` (from
  `cycloid-cli/pkg/testcfg`) to provision admin user, API key, config repo, catalog repo,
  test project, environment, and component automatically on `just be-start`.
- `.env.sample` and `just env` recipe for secret resolution via `cy uri interpolate`
  (mirrors `cycloid-cli` `make .env`).
- `just test-acc-fresh` recipe: one-shot path from cold docker state to green test suite.
- Acceptance tests for 5 plugin resources: `cycloid_plugin_registry`,
  `cycloid_plugin_manager`, `cycloid_plugin_registry_plugin`, `cycloid_plugin_version`,
  `cycloid_plugin` (full install → running → uninstall lifecycle).
- `ensurePluginHelloWorld` helper: idempotent pull/tag/push of
  `docker.io/cycloid/plugin-hello-world:1.0.0` to the local docker-registry.

### Changed
- `TESTING.md` rewritten: local docker-compose stack is now the primary test path.
  Remote env override path documented as secondary.
- `go.mod` now consumes `cycloid-cli/pkg/testcfg` (promoted from `internal/testcfg`).

### Deprecated
- `TEST_DEPENDENCIES_ACC_RUN.md` (remote-env workflow notes) — kept for reference,
  will be removed in a future cycle.
- `test_config.yaml` / `README_TEST_CONFIG.md` — no longer required for local stack runs.

### Fixed
- `provider/stack_datasource_test.go`: removed non-existent `QuotaEnabled` field from
  `models.ServiceCatalog` literal (pre-existing `go vet` failure).
