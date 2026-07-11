package provider

import (
	"context"
	"strings"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/stretchr/testify/assert"
)

// TestCredentialSchemaCanonicalForcesReplaceWhenConfigured guards the fix for
// the "Provider produced inconsistent result after apply" bug on `.canonical`.
// The Cycloid API cannot cleanly rename a credential canonical (a PUT that
// changes it renames server-side but responds 404), so a changed,
// explicitly-configured canonical must force replacement rather than an
// in-place update. This is enforced by a RequiresReplaceIfConfigured plan
// modifier added in credentialResource.Schema(); the credential schema is
// code-generated, so this also protects the post-processing from being lost on
// regeneration.
func TestCredentialSchemaCanonicalForcesReplaceWhenConfigured(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	resp := &fwresource.SchemaResponse{}
	NewCredentialResource().Schema(ctx, fwresource.SchemaRequest{}, resp)

	attr, ok := resp.Schema.Attributes["canonical"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("canonical attribute is not a schema.StringAttribute: %T", resp.Schema.Attributes["canonical"])
	}

	found := false
	for _, pm := range attr.PlanModifiers {
		if strings.Contains(pm.Description(ctx), "destroy and recreate") {
			found = true
			break
		}
	}
	assert.True(t, found, "canonical must force replacement when a configured value changes; the API cannot rename a credential canonical in place")
}
