package provider

import (
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cycloidio/cycloid-cli/pkg/testcfg"
	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
)

// testcfg.NewConfig lists catalog versions right after an async refresh, so on a
// fresh backend (CI wipes volumes) the version isn't there yet. NewConfig is
// idempotent, so retry it — the scan finishes between attempts. Upstream fix:
// cycloid-cli testcfg should await the version pull.
const (
	bootstrapMaxAttempts    = 8
	bootstrapRetryDelay     = 10 * time.Second
	bootstrapCatalogRaceErr = "failed to find latest catalog repo version after refresh"
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
	if err := os.Setenv("CY_TEST_WRITE_API_KEY_FILE", "0"); err != nil {
		log.Fatalf("failed to set CY_TEST_WRITE_API_KEY_FILE: %v", err)
	}

	var cfg *testcfg.Config
	var err error
	for attempt := 1; attempt <= bootstrapMaxAttempts; attempt++ {
		cfg, err = testcfg.NewConfig("tfprovider")
		if err == nil {
			break
		}
		// Only retry the known async catalog-scan race; fail fast on anything
		// else (bad licence, unreachable backend, etc.) instead of looping.
		if !strings.Contains(err.Error(), bootstrapCatalogRaceErr) || attempt == bootstrapMaxAttempts {
			log.Fatalf("testcfg bootstrap failed (attempt %d/%d): %v", attempt, bootstrapMaxAttempts, err)
		}
		log.Printf("testcfg bootstrap attempt %d/%d hit the async catalog-scan race; retrying in %s",
			attempt, bootstrapMaxAttempts, bootstrapRetryDelay)
		time.Sleep(bootstrapRetryDelay)
	}
	sharedCfg = cfg

	// Populate env vars so testAccPreCheck, provider.Configure, and
	// NewTestDependencyManager all continue to work without modification.
	for k, v := range map[string]string{
		"CY_API_KEY": cfg.APIKey,
		"CY_API_URL": cfg.APIUrl,
		"CY_ORG":     cfg.Org,
	} {
		if err := os.Setenv(k, v); err != nil {
			log.Fatalf("failed to set %s: %v", k, err)
		}
	}

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
		// Credential is intentionally empty — the catalog repo is a public GitHub URL
		// and requires no authentication. TestAccCatalogRepositoryResource now handles
		// the empty-credential case by omitting the field from the HCL config.
		tc.Repositories.Catalog = TestConfigRepo{
			URL:    ptr.Value(cfg.CatalogRepo.URL),
			Branch: cfg.CatalogRepo.Branch,
		}
	}

	// Component: derive stack info from the bootstrapped component.
	// testcfg uses "stack-e2e-stackforms" / "default" as defaults.
	if cfg.Component != nil && cfg.Component.ServiceCatalog != nil {
		ref := ptr.Value(cfg.Component.ServiceCatalog.Ref) // e.g. "cycloid:stack-e2e-stackforms"
		if colonIdx := strings.Index(ref, ":"); colonIdx >= 0 {
			tc.Component = &TestConfigComponent{
				StackCanonical: ref[colonIdx+1:],
				UseCase:        cfg.Component.UseCase,
			}
		}
	}

	return tc
}
