package provider

import (
	"os"
	"testing"
)

// testAccPreCheck validates that required environment variables are set for acceptance testing
func testAccPreCheck(t *testing.T) {
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
