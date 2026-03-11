package provider

//

// The fwresource import alias is so there is no collision
// with the more typical acceptance testing import:
// "github.com/hashicorp/terraform-plugin-testing/helper/resource"

// fwresource "github.com/hashicorp/terraform-plugin-framework/helper/resource"

// func TestThingResourceSchema(t *testing.T) {
// 	t.Parallel()
//
// 	ctx := context.Background()
// 	schemaRequest := fwresource.SchemaRequest{}
// 	schemaResponse := &fwresource.SchemaResponse{}
//
// 	// Instantiate the resource.Resource and call its Schema method
// 	NewTeamResource().Schema(ctx, schemaRequest, schemaResponse)
//
// 	if schemaResponse.Diagnostics.HasError() {
// 		t.Fatalf("Schema method diagnostics: %+v", schemaResponse.Diagnostics)
// 	}
//
// 	// Validate the schema
// 	diagnostics := schemaResponse.Schema.ValidateImplementation(ctx)
//
// 	if diagnostics.HasError() {
// 		t.Fatalf("Schema validation diagnostics: %+v", diagnostics)
// 	}
// }

// 	func TestAccTeamResource(t *testing.T) {
// 	t.Parallel()
// 	// var teamBefore, teamAfter teamResourceModel
// 	compareValueSame := statecheck.CompareValue(compare.ValuesSame())
//
// 	resource.Test(t, resource.TestCase{
// 		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
// 			"cycloid": providerserver.NewProtocol6WithError(&CycloidProvider{}),
// 		},
// 		PreCheck: func() { testAccPreCheck(t) },
// 		// Providers:    testAccProviders,
// 		// CheckDestroy: testAccCheckExampleResourceDestroy,
// 		Steps: []resource.TestStep{
// 			{
// 				Config: `
// resource "cycloid_team" "test_team" {
//   name         = "Test Team"
//   roles = [
//     "organization-admin",
//   ]
// }
// 				`,
// 				ConfigStateChecks: []statecheck.StateCheck{
// 					compareValueSame.AddStateValue(
// 						"cycloid_team.test_team", tfjsonpath.New("canonical"),
// 					),
// 				},
// 			},
// 		},
// 	})
// }
//
// func testAccPreCheck(t *testing.T) {
// 	if v := os.Getenv("CY_API_URL"); v == "" {
// 		t.Fatal("CY_API_URL is required for testing.")
// 	}
//
// 	if v := os.Getenv("CY_API_KEY"); v == "" {
// 		t.Fatal("CY_API_KEY is required for testing.")
// 	}
//
// 	if v := os.Getenv("CY_ORG"); v == "" {
// 		t.Fatal("CY_ORG is required for testing.")
// 	}
// }
