package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cycloidio/cycloid-cli/cmd/cycloid/common"
	"github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/cycloid-cli/internal/ptr"
	"github.com/stretchr/testify/require"
)

type organizationRoleAcceptanceRule struct {
	Action    string   `json:"action"`
	Effect    string   `json:"effect"`
	Resources []string `json:"resources"`
}

func TestAccOrganizationRoleRecreateFromDefaultAdminRole(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC is not set")
	}

	apiURL := os.Getenv("CY_API_URL")
	apiKey := os.Getenv("CY_API_KEY")
	org := os.Getenv("CY_ORG")
	if apiURL == "" || apiKey == "" || org == "" {
		t.Skip("CY_API_URL, CY_API_KEY or CY_ORG is not set")
	}

	ctx := context.Background()
	apiClient := common.NewAPI(common.WithURL(apiURL), common.WithToken(apiKey))
	mid := middleware.NewMiddleware(apiClient)

	sourceRole, _, err := mid.GetRole(org, "organization-admin")
	require.NoError(t, err)
	require.NotNil(t, sourceRole)
	require.Greater(t, len(sourceRole.Rules), 0)

	rules := make([]organizationRoleAcceptanceRule, 0, len(sourceRole.Rules))
	for _, rule := range sourceRole.Rules {
		if rule == nil {
			continue
		}
		rules = append(rules, organizationRoleAcceptanceRule{
			Action:    ptr.Value(rule.Action),
			Effect:    ptr.Value(rule.Effect),
			Resources: rule.Resources,
		})
	}
	require.Greater(t, len(rules), 0)

	rulesJSON, err := json.MarshalIndent(rules, "", "  ")
	require.NoError(t, err)

	roleCanonical := fmt.Sprintf("tf-acc-org-admin-%d", time.Now().UnixNano())
	roleName := fmt.Sprintf("%s recreated", strings.TrimSpace(ptr.Value(sourceRole.Name)))
	if len(roleName) < 3 {
		roleName = "Organization Admin Recreated"
	}
	roleDescription := fmt.Sprintf("Recreated from organization-admin at %s", time.Now().UTC().Format(time.RFC3339))

	terraformManifest := fmt.Sprintf(`
terraform {
  required_providers {
    cycloid = {
      source = "cycloidio/cycloid"
    }
  }
}

provider "cycloid" {}

locals {
  recreated_rules = jsondecode(<<JSON
%s
JSON
  )
}

resource "cycloid_organization_role" "recreated" {
  canonical   = %q
  name        = %q
  description = %q
  rules       = local.recreated_rules
}
`, string(rulesJSON), roleCanonical, roleName, roleDescription)

	workDir := t.TempDir()
	repoRoot := acceptanceRepoRoot(t)
	tfrcPath := filepath.Join(workDir, "terraformrc")
	mainTFPath := filepath.Join(workDir, "main.tf")

	tfrc := fmt.Sprintf(`provider_installation {
  dev_overrides {
    "registry.terraform.io/cycloidio/cycloid" = %q
  }
  direct {}
}
`, repoRoot)

	require.NoError(t, os.WriteFile(tfrcPath, []byte(tfrc), 0o600))
	require.NoError(t, os.WriteFile(mainTFPath, []byte(terraformManifest), 0o600))

	t.Cleanup(func() {
		_, _ = mid.DeleteRole(org, roleCanonical)
	})

	acceptanceRunCommand(t, ctx, repoRoot, os.Environ(), "go", "build", "-gcflags", "all=-l", "-trimpath", ".")

	terraformEnv := append(os.Environ(), "TF_CLI_CONFIG_FILE="+tfrcPath)
	acceptanceRunCommand(t, ctx, workDir, terraformEnv, "terraform", "init", "-input=false")
	acceptanceRunCommand(t, ctx, workDir, terraformEnv, "terraform", "apply", "-auto-approve", "-input=false")

	createdRole, _, err := mid.GetRole(org, roleCanonical)
	require.NoError(t, err)
	require.NotNil(t, createdRole)
	require.Equal(t, roleCanonical, ptr.Value(createdRole.Canonical))
	require.Equal(t, roleName, ptr.Value(createdRole.Name))
	require.Equal(t, len(rules), len(createdRole.Rules))

	acceptanceRunCommand(t, ctx, workDir, terraformEnv, "terraform", "destroy", "-auto-approve", "-input=false")

	_, _, err = mid.GetRole(org, roleCanonical)
	require.Error(t, err)
}

func acceptanceRepoRoot(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	require.NoError(t, err)

	repoRoot := filepath.Clean(filepath.Join(cwd, ".."))
	_, err = os.Stat(filepath.Join(repoRoot, "go.mod"))
	if err == nil {
		return repoRoot
	}

	_, err = os.Stat(filepath.Join(cwd, "go.mod"))
	require.NoError(t, err)
	return cwd
}

func acceptanceRunCommand(t *testing.T, ctx context.Context, workDir string, env []string, binary string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = workDir
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s failed: %v\n%s", binary, strings.Join(args, " "), err, string(output))
	}
}
