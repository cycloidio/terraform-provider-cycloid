package utils_test

import (
	"testing"

	"github.com/cycloidio/terraform-provider-cycloid/tests/utils"
	"github.com/stretchr/testify/assert"
)

func TestRepoRoot(t *testing.T) {
	_, err := utils.GetRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunCmdOutErrOutput(t *testing.T) {
	stdout, stderr, err := utils.RunCmdOutErr("sh", "-euc", `
	echo -n stdout
	echo -n stderr 1>&2
	`)
	if err != nil {
		t.Fatal("runcmd returned error", err)
	}

	assert.Equal(t, "stdout", stdout, "cmd should return stdout")
	assert.Equal(t, "stderr", stderr, "cmd should return stderr")
}
