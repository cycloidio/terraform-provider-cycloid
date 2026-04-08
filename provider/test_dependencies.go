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

	project, err := dm.provider.Middleware.CreateProject(
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
			return dm.provider.Middleware.DeleteProject(org, ptr.Value(project.Canonical))
		},
	})

	t.Logf("Created test project: %s", ptr.Value(project.Canonical))
	return project, nil
}

// EnsureTestProject creates a project if it doesn't exist, or returns the existing one.
// org must match the organization used by the resource under test.
func (dm *TestDependencyManager) EnsureTestProject(ctx context.Context, t *testing.T, org, name, description string) (*models.Project, error) {
	if dm.provider.Middleware == nil {
		t.Logf("Middleware not available, skipping project lookup: %s", name)
		return &models.Project{Canonical: &name}, nil
	}

	projects, err := dm.provider.Middleware.ListProjects(org)
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
	env, err := dm.provider.Middleware.CreateEnv(org, project, name, name, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create test environment: %w", err)
	}

	canonical := *env.Canonical
	dm.cleanupItems = append(dm.cleanupItems, cleanupItem{
		resourceType: "environment",
		canonical:    canonical,
		cleanupFunc: func() error {
			return dm.provider.Middleware.DeleteEnv(org, project, canonical)
		},
	})

	t.Logf("Created test environment: %s in project %s", canonical, project)
	return env, nil
}

// EnsureTestEnvironment creates the environment if it doesn't already exist, returning the full environment model.
func (dm *TestDependencyManager) EnsureTestEnvironment(ctx context.Context, t *testing.T, org, project, name string) (*models.Environment, error) {
	if dm.provider.Middleware == nil {
		t.Logf("Middleware not available, skipping environment lookup: %s", name)
		return &models.Environment{Canonical: &name}, nil
	}

	envs, err := dm.provider.Middleware.ListProjectsEnv(org, project)
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
// Delete failures are ignored (no error or warning logged).
func (dm *TestDependencyManager) Cleanup(ctx context.Context, t *testing.T) {
	if len(dm.cleanupItems) == 0 {
		return
	}

	for i := len(dm.cleanupItems) - 1; i >= 0; i-- {
		_ = dm.cleanupItems[i].cleanupFunc()
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
