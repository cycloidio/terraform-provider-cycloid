package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/cycloid-cli/cmd/cycloid/common"
	"github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/internal/ptr"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// TestDependencyManager handles creating and managing test dependencies using Cycloid middleware
type TestDependencyManager struct {
	provider     *CycloidProvider
	organization string
	cleanupItems []cleanupItem
}

type cleanupItem struct {
	resourceType string
	canonical    string
	cleanupFunc  func() error
}

// NewTestDependencyManager creates a new dependency manager for testing
func NewTestDependencyManager(t *testing.T) *TestDependencyManager {
	provider := &CycloidProvider{}

	provider.APIKey = os.Getenv("CY_API_KEY")
	provider.APIUrl = os.Getenv("CY_API_URL")
	provider.DefaultOrganization = os.Getenv("CY_ORG")

	if provider.APIKey != "" && provider.APIUrl != "" && provider.DefaultOrganization != "" {
		provider.Insecure = false
		provider.APIClient = common.NewAPI(
			common.WithURL(provider.APIUrl),
			common.WithToken(provider.APIKey),
			common.WithInsecure(provider.Insecure),
		)
		provider.Middleware = middleware.NewMiddleware(provider.APIClient)
	}

	org := testAccGetOrganizationCanonical()

	return &TestDependencyManager{
		provider:     provider,
		organization: org,
		cleanupItems: []cleanupItem{},
	}
}

// CreateTestProject creates a test project using middleware and returns the full project model.
// org must match the organization used by the resource under test so the project is in the same org.
func (dm *TestDependencyManager) CreateTestProject(ctx context.Context, t *testing.T, org, name, description string) (*models.Project, error) {
	if description == "" {
		description = fmt.Sprintf("Test project %s for acceptance testing", name)
	}

	cfg, err := LoadTestConfig()
	if err != nil {
		return nil, fmt.Errorf("loading test config: %w", err)
	}

	project, _, err := dm.provider.Middleware.CreateProject(
		org,
		name,
		name, // canonical same as name
		description,
		cfg.ConfigRepository,
		"", "", "", "", // owner, team, color, icon - use defaults
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create test project: %w", err)
	}

	dm.cleanupItems = append(dm.cleanupItems, cleanupItem{
		resourceType: "project",
		canonical:    ptr.Value(project.Canonical),
		cleanupFunc: func() error {
			_, err := dm.provider.Middleware.DeleteProject(org, ptr.Value(project.Canonical))
			return err
		},
	})

	t.Logf("Created test project: %s", ptr.Value(project.Canonical))
	return project, nil
}

// EnsureTestProject creates a project if it doesn't exist, or returns the existing one.
// org must match the organization used by the resource under test.
// Skips the test when credentials are not configured (middleware is nil).
func (dm *TestDependencyManager) EnsureTestProject(ctx context.Context, t *testing.T, org, name, description string) (*models.Project, error) {
	t.Helper()
	if dm.provider.Middleware == nil {
		t.Skip("skipping acceptance test: CY_API_URL, CY_API_KEY and CY_ORG must be set")
	}

	projects, _, err := dm.provider.Middleware.ListProjects(org)
	if err != nil {
		t.Logf("Warning: failed to list projects, will attempt to create: %v", err)
	}

	for _, p := range projects {
		if ptr.Value(p.Canonical) == name {
			t.Logf("Test project already exists: %s", name)
			return p, nil
		}
	}

	return dm.CreateTestProject(ctx, t, org, name, description)
}

// CreateTestEnvironment creates a test environment inside a project using middleware and returns the full environment model.
func (dm *TestDependencyManager) CreateTestEnvironment(ctx context.Context, t *testing.T, org, project, name string) (*models.Environment, error) {
	env, _, err := dm.provider.Middleware.CreateEnv(org, project, name, name, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create test environment: %w", err)
	}

	canonical := *env.Canonical
	dm.cleanupItems = append(dm.cleanupItems, cleanupItem{
		resourceType: "environment",
		canonical:    canonical,
		cleanupFunc: func() error {
			_, err := dm.provider.Middleware.DeleteEnv(org, project, canonical)
			return err
		},
	})

	t.Logf("Created test environment: %s in project %s", canonical, project)
	return env, nil
}

// EnsureTestEnvironment creates the environment if it doesn't already exist, returning the full environment model.
// Skips the test when credentials are not configured (middleware is nil).
func (dm *TestDependencyManager) EnsureTestEnvironment(ctx context.Context, t *testing.T, org, project, name string) (*models.Environment, error) {
	t.Helper()
	if dm.provider.Middleware == nil {
		t.Skip("skipping acceptance test: CY_API_URL, CY_API_KEY and CY_ORG must be set")
	}

	envs, _, err := dm.provider.Middleware.ListProjectsEnv(org, project)
	if err != nil {
		t.Logf("Warning: failed to list environments, will attempt to create: %v", err)
	}

	for _, e := range envs {
		if ptr.Value(e.Canonical) == name {
			t.Logf("Test environment already exists: %s in project %s", name, project)
			return e, nil
		}
	}

	return dm.CreateTestEnvironment(ctx, t, org, project, name)
}

// Cleanup runs in reverse order of creation so dependents (e.g. environment) are removed before dependencies (e.g. project).
func (dm *TestDependencyManager) Cleanup(ctx context.Context, t *testing.T) {
	t.Helper()
	for i := len(dm.cleanupItems) - 1; i >= 0; i-- {
		item := dm.cleanupItems[i]
		if err := item.cleanupFunc(); err != nil {
			t.Logf("Warning: cleanup of %s %q failed: %v", item.resourceType, item.canonical, err)
		}
	}
	dm.cleanupItems = []cleanupItem{}
}

// GetProvider returns the configured provider for use in tests
func (dm *TestDependencyManager) GetProvider() *CycloidProvider {
	return dm.provider
}

// GetProviderFactories returns the provider factories for use in resource.TestCase
func (dm *TestDependencyManager) GetProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"cycloid": providerserver.NewProtocol6WithError(dm.provider),
	}
}
