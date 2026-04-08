package provider

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
)

// testAccPreCheck validates that required environment variables are set for acceptance testing.
// When testing.Short() is true, skips the test so "go test -short" runs only unit tests.
func testAccPreCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping acceptance test in short mode")
	}
	if v := os.Getenv("CY_API_URL"); v == "" {
		t.Fatal("CY_API_URL is required for acceptance testing.")
	}

	if v := os.Getenv("CY_API_KEY"); v == "" {
		t.Fatal("CY_API_KEY is required for acceptance testing.")
	}

	if v := os.Getenv("CY_ORG"); v == "" {
		t.Fatal("CY_ORG is required for acceptance testing.")
	}
}

// testAccGetOrganizationCanonical returns the organization canonical from environment variable
// with fallback to "cycloid" for safety
func testAccGetOrganizationCanonical() string {
	org := os.Getenv("CY_ORG")
	if org == "" {
		return "cycloid" // fallback for safety
	}
	return org
}

// testAccGetTestConfig returns the centralised test config (dependencies). Skips the test if config cannot be loaded.
func testAccGetTestConfig(t *testing.T) *TestConfig {
	t.Helper()
	cfg, err := LoadTestConfig()
	if err != nil {
		t.Skipf("skipping: %v", err)
	}
	return cfg
}

// RandomCanonical returns baseName + 4 random digits for unique resource names in acceptance tests.
func RandomCanonical(baseName string) string {
	return fmt.Sprintf("%s%04d", baseName, rand.Intn(10000))
}
