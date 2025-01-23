package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
)

// Wrapper to run a command a /bin/sh -c <cmd> using RunCmd
func RunSh(commandBlock string) (string, error) {
	binSh, err := exec.LookPath("sh")
	if err != nil {
		return "", errors.Wrap(err, "failed to find 'sh' in $PATH")
	}

	cmd := exec.Command(binSh, "-exc", commandBlock)
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// Run a command and return stdout stderr and err
// is env is nil, defaults top os.Environ()
func RunCmdOutErr(cmd string, args ...string) (string, string, error) {
	proc := exec.Command(cmd, args...)
	proc.Env = os.Environ()

	var pOut, pErr strings.Builder
	proc.Stdout = &pOut
	proc.Stderr = &pErr
	err := proc.Run()
	return pOut.String(), pErr.String(), err
}

// Lookup $ENV_VAR, is not set, return defaultValue
func EnvDefault(envVar, defaultValue string) string {
	value, ok := os.LookupEnv(envVar)
	if !ok {
		return defaultValue
	}

	return value
}

// Converts os.Environ() env var list to map[string]string
func EnvListToMap(env []string) map[string]string {
	var result = make(map[string]string)
	for _, envVar := range env {
		key, val, ok := strings.Cut(envVar, "=")
		if ok {
			result[key] = val
		}
	}

	return result
}

// Convert env variables stored as map[string]string to []string with 'key=value' format.
func EnvMapToList(env map[string]string) []string {
	result := []string{}
	for k, v := range env {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}

	return result
}

// Check if a git repo exists at <path>
func RepoExists(path string) bool {
	repoDir, err := os.Stat(path)
	if err != nil {
		return false
	}

	if !repoDir.IsDir() {
		return false
	}

	if _, err = os.Stat(path + "/.git"); err != nil {
		return false
	}

	return true
}

// Fetch credential and return its value
// Uses cy cli to use the user's credentials
// It will use our production cycloid instance
func GetCyCredential(org, canonical string) (*models.Credential, error) {
	cmdOut, cmdErr, err := RunCmdOutErr(
		"cy", "--api-url", "https://http-api.cycloid.io", "--org", org, "credential", "get", "--output", "json",
		"--canonical", canonical,
	)

	if err != nil {
		return nil, errors.Wrap(err, cmdErr)
	}

	var cred *models.Credential
	err = json.Unmarshal([]byte(cmdOut), &cred)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse json from cycloid response.")
	}

	return cred, nil
}

// load the .env at project root - will also try to load a .dev.env for local overrides
// errors only if it can't load the project .env
func LoadProjectDotEnv() error {
	err := godotenv.Load(".env")
	if err != nil {
		return err
	}

	file, err := os.Stat(".dev.env")
	if err != nil {
		return nil
	}

	err = godotenv.Load(file.Name())
	if err != nil {
		return nil
	}

	fmt.Println("loaded .dev.env in env")
	return nil
}

// Curl status from backend and wait that database, pipelines and vault status are ok
func WaitForBackend(timeoutInSec int) error {
	// Wait for backend up
	for increment := range timeoutInSec {
		stdout, stderr, err := RunCmdOutErr("sh", "-euc",
			`./ci/dc.sh cmd ps --format json | jq -r '. | select(.Service == "youdeploy-api") | .Health'`,
		)
		if err != nil || increment > timeoutInSec {
			fmt.Println() // print missing newline
			return errors.Wrapf(err, "timeout while waiting for backend up:\n%s\n", stderr)
		}

		status := strings.TrimRight(stdout, "\n")
		if status == "healthy" { // output from cmd has a newline
			fmt.Println() // print missing newline
			return nil
		}

		fmt.Printf("\rbackend status is '%s' waiting to be healthy since %d sec...", status, increment)
		time.Sleep(1 * time.Second)
	}

	return errors.New("timeout while waiting for backend up\n")
}

func GetRepoRoot() (string, error) {
	dirName, err := os.Getwd()
	if err != nil {
		return "", err
	}

	pwd, err := filepath.Abs(dirName)
	if err != nil {
		return "", err
	}

	if !strings.HasPrefix(pwd, "/") {
		return "", errors.Errorf("not rooted path: %s", pwd)
	}

	for {
		if pwd == "/" {
			return "", errors.New("Cannot find repo's root, reached '/'")
		}

		_, gitErr := os.Stat(pwd + "/.git")
		_, ciErr := os.Stat(pwd + "/ci/")
		if gitErr == nil && ciErr == nil {
			return pwd, nil
		}

		pwd = filepath.Dir(pwd)
	}
}
