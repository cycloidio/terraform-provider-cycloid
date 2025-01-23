package e2e

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This test will list all folder with `test_` name under `./tests/e2e`
// It will then apply the `./ci/tf-test.sh run_test` script on each of them in parallel
func TestE2e(t *testing.T) {
	pwd, err := os.Getwd()
	require.NoError(t, err, "pwd")

	dir, err := os.ReadDir(pwd + "/tests/e2e")
	require.NoError(t, err, "ls /tests/e2e")

	for _, testDir := range dir {
		testCase := testDir.Name()
		if strings.HasPrefix(testCase, "test_") {
			t.Run(testCase, func(t *testing.T) {
				t.Parallel()
				t.Helper()

				cmd := exec.Command("sh", "-euc",
					`./ci/tf-test.sh run_test`,
				)

				cmd.Env = append(cmd.Env, os.Environ()...)
				cmd.Env = append(cmd.Env, []string{
					"TF_TEST_TARGET_DIR=" + "tests/e2e/" + testCase,
				}...)

				out, err := cmd.CombinedOutput()
				assert.NoError(t, err, string(out))
			})
		}
	}
}
