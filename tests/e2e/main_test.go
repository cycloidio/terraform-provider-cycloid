package e2e

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/cycloidio/terraform-provider-cycloid/tests/utils"
	"github.com/pkg/errors"
)

var (
	// if true, tests will spin up a backend using docker compose - implies a clean
	TEST_DC_UP = strings.ToLower(utils.EnvDefault("TEST_DC_UP", "true"))
	// if true, tests will clean the backend after tests are runned
	TEST_DC_CLEAN = strings.ToLower(utils.EnvDefault("TEST_DC_CLEAN", "true"))
	// if true, tests will init admin user and API_KEY before testing
	TEST_BACKEND_INIT = strings.ToLower(utils.EnvDefault("TEST_BACKEND_INIT", "true"))

	API_LICENCE_KEY = os.Getenv("API_LICENCE_KEY")
)

// Cleanup after tests
func Cleanup(code int) {
	if TEST_DC_UP == "true" && TEST_DC_CLEAN == "true" {
		fmt.Println("SETUP: cleaning backend...")
		out, err := utils.RunSh(`
			./ci/dc.sh clean
		`)
		if err != nil {
			fmt.Println("failed to docker compose down", out, err)
		}
	}

	os.Exit(code)
}

// Setup the tests and execute them
func TestMain(m *testing.M) {
	repoRoot, err := utils.GetRepoRoot()
	if err != nil {
		fmt.Println(err)
		Cleanup(1)
	}

	// IMPORTANT -- we make so that in the e2e tests, all commands are executed from repo root
	// much easier to handle the scripts this way
	if err := os.Chdir(repoRoot); err != nil {
		fmt.Println("failed to chdir:", err)
		Cleanup(1)
	}

	err = utils.LoadProjectDotEnv()
	if err != nil {
		fmt.Println(err)
		Cleanup(1)
	}

	if API_LICENCE_KEY == "" {
		key, err := fetchLicenceKey()
		if err != nil {
			fmt.Println("failed to get backend licence:", err)
			Cleanup(1)
		}

		os.Setenv("API_LICENCE_KEY", key)
	}

	if TEST_DC_UP == "true" {
		fmt.Println("SETUP: starting backend...")
		if out, err := utils.RunSh(`
		./ci/dc.sh up_default
	`); err != nil {
			fmt.Println("failed to start backend:\n", out, "\n", err)
			Cleanup(1)
		}

		if err := utils.WaitForBackend(30); err != nil {
			fmt.Println("Backend is no healthy\n", err)
			Cleanup(1)
		}
	}

	if TEST_BACKEND_INIT == "true" {
		fmt.Println("SETUP: init backend...")
		if err := utils.WaitForBackend(30); err != nil {
			fmt.Println("Backend is no healthy\n", err)
			Cleanup(1)
		}

		if out, err := utils.RunSh(`
		./ci/init-backend.sh init_root_org
	`); err != nil {
			fmt.Println("failed to init backend:\n", out, "\n", err)
			Cleanup(1)
		}
	}

	Cleanup(m.Run())
}

// Return the licence key
func fetchLicenceKey() (string, error) {
	fmt.Println("Fetching API licence key for tests using the CLI.")
	// Get the licence and propagate it
	credentialKey, ok := os.LookupEnv("API_LICENCE_CREDENTIAL_CANONICAL")
	if !ok {
		return "", errors.New("API_LICENCE_CREDENTIAL_CANONICAL env var is required for tests to fetch the licence.")
	}

	licenceCred, err := utils.GetCyCredential("cycloid", credentialKey)
	if err != nil {
		return "", errors.Wrap(err, "failed to get API_LICENCE_KEY for backend using cycloid\n")
	}

	key, ok := licenceCred.Raw.Raw.(map[string]any)["licence_key"].(string)
	if !ok {
		return "", errors.Errorf("fail to parse licence credential: %v", licenceCred)
	}

	return key, nil
}
