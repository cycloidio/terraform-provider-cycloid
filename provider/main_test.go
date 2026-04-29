package provider

import (
	"log"
	"os"
	"testing"

	"github.com/cycloidio/cycloid-cli/pkg/testcfg"
	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
)

// sharedCfg holds the bootstrapped test configuration for the whole test binary.
// Set by TestMain when TF_ACC=1 and CY_TEST_PROVISION_API is not "0".
// Tests that need shared fixtures (config repo, project, environment, etc.) read from this.
var sharedCfg *testcfg.Config

// TestMain bootstraps the local Cycloid stack once per test binary run and tears it
// down after all tests complete. Running without TF_ACC set skips the bootstrap so
// that unit tests work without a live backend.
//
// Do not call t.Parallel() inside TestAcc* functions — the bootstrap provisions
// shared resources (catalog repo, git-server repos) that are not concurrent-safe.
func TestMain(m *testing.M) {
	if os.Getenv("TF_ACC") == "" {
		os.Exit(m.Run())
	}

	// Disable .api_key write so we don't scribble into this repo's working tree.
	os.Setenv("CY_TEST_WRITE_API_KEY_FILE", "0")

	cfg, err := testcfg.NewConfig("tfprovider")
	if err != nil {
		log.Fatalf("testcfg bootstrap failed: %v", err)
	}
	sharedCfg = cfg

	// Populate env vars so testAccPreCheck, provider.Configure, and
	// NewTestDependencyManager all continue to work without modification.
	os.Setenv("CY_API_KEY", cfg.APIKey)
	os.Setenv("CY_API_URL", cfg.APIUrl)
	os.Setenv("CY_ORG", cfg.Org)

	// Pre-populate LoadTestConfig so tests don't need test_config.yaml.
	primeTestConfig(testConfigFromBootstrap(cfg))

	code := m.Run()

	cfg.Cleanup()
	os.Exit(code)
}

// testConfigFromBootstrap maps the bootstrapped testcfg.Config onto the TestConfig
// shape expected by existing tests (config_repository, repositories, component).
func testConfigFromBootstrap(cfg *testcfg.Config) *TestConfig {
	tc := &TestConfig{}

	if cfg.ConfigRepo != nil && cfg.ConfigRepo.Canonical != nil {
		tc.ConfigRepository = *cfg.ConfigRepo.Canonical
		if cfg.ConfigRepo.URL != nil {
			tc.Repositories.Config = TestConfigRepo{
				URL:        ptr.Value(cfg.ConfigRepo.URL),
				Branch:     cfg.ConfigRepo.Branch,
				Credential: cfg.ConfigRepo.CredentialCanonical,
			}
		}
	}

	if cfg.CatalogRepo != nil {
		tc.Repositories.Catalog = TestConfigRepo{
			URL:        ptr.Value(cfg.CatalogRepo.URL),
			Branch:     cfg.CatalogRepo.Branch,
			Credential: cfg.CatalogRepo.CredentialCanonical,
		}
	}

	if cfg.CatalogRepoVersionStacks != nil {
		tc.Component = &TestConfigComponent{
			StackCanonical: "stack-e2e-stackforms",
			UseCase:        "default",
			StackVersion:   "stacks",
		}
	}

	return tc
}
