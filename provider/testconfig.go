package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/viper"
)

const (
	// Env prefix for overriding test config via environment variables (e.g. CY_TEST_CONFIG_REPOSITORY).
	testConfigEnvPrefix = "CY_TEST_"
	testConfigName      = "test_config"
	testConfigType      = "yaml"
)

// TestConfig holds test environment dependency data (credentials, repos, etc.).
// Loaded from test_config.yaml with env var override (CY_TEST_*).
type TestConfig struct {
	ConfigRepository string               `mapstructure:"config_repository"`
	Repositories     TestConfigRepos      `mapstructure:"repositories"`
	Component       *TestConfigComponent `mapstructure:"component"`
}

// TestConfigComponent holds stack_canonical, use_case, stack_version, and input_variables for component acceptance tests.
// Set in test_config.yaml; component tests skip when stack_canonical is empty.
// stack_ref is built at test time as <org>:<stack_canonical>.
// InputVariables is serialized as JSON in the resource and passed through jsondecode() in Terraform.
type TestConfigComponent struct {
	StackCanonical   string                                  `mapstructure:"stack_canonical"`
	UseCase          string                                  `mapstructure:"use_case"`
	StackVersion     string                                  `mapstructure:"stack_version"`
	InputVariables   map[string]map[string]map[string]any    `mapstructure:"input_variables"`
}

// TestConfigRepos holds default repo URLs/branches and credential for repository tests.
type TestConfigRepos struct {
	Config  TestConfigRepo `mapstructure:"config"`
	Catalog TestConfigRepo `mapstructure:"catalog"`
}

// TestConfigRepo holds url, branch, and credential for a repo.
type TestConfigRepo struct {
	URL        string `mapstructure:"url"`
	Branch     string `mapstructure:"branch"`
	Credential string `mapstructure:"credential"`
}

var (
	testConfig     *TestConfig
	testConfigOnce sync.Once
)

// LoadTestConfig loads test config from YAML with env override (CY_TEST_*).
// Config file is searched at repo root first, then provider dir, then current dir.
func LoadTestConfig() (*TestConfig, error) {
	var err error
	testConfigOnce.Do(func() {
		testConfig, err = loadTestConfigOnce()
	})
	return testConfig, err
}

func loadTestConfigOnce() (*TestConfig, error) {
	v := viper.New()
	v.SetConfigName(testConfigName)
	v.SetConfigType(testConfigType)
	v.SetEnvPrefix(testConfigEnvPrefix)
	v.AutomaticEnv()

	// Search paths: repo root first, then provider dir, then current dir
	if root := findRepoRoot(); root != "" {
		v.AddConfigPath(root)
	}
	if providerDir := findProviderDir(); providerDir != "" {
		v.AddConfigPath(providerDir)
	}
	v.AddConfigPath(".")
	v.AddConfigPath("./provider")

	// Optional config file; defaults are still applied from env or struct zero values
	if cfgPath := os.Getenv("CY_TEST_CONFIG_FILE"); cfgPath != "" {
		v.SetConfigFile(cfgPath)
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil, fmt.Errorf("test config file %s not found: place test_config.yaml at repo root (or set CY_TEST_CONFIG_FILE). See README_TEST_CONFIG.md", testConfigName+"."+testConfigType)
		}
		return nil, fmt.Errorf("reading test config: %w", err)
	}

	var cfg TestConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling test config: %w", err)
	}
	return &cfg, nil
}

// findRepoRoot walks up from the current directory until it finds go.mod (repo root).
func findRepoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// findProviderDir returns a path where test_config.yaml may live (provider or cwd).
func findProviderDir() string {
	if _, err := os.Stat("provider/test_config.yaml"); err == nil {
		return "provider"
	}
	if _, err := os.Stat("test_config.yaml"); err == nil {
		return "."
	}
	return filepath.Join(".", "provider")
}
