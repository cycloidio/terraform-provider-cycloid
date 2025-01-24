package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/cycloidio/terraform-provider-cycloid/swagger_converter/utils"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

var (
	ciDir = ".ci"
)

// Fetch the swagger from the config's Url, default: 'https://docs.cycloid.io/api/swagger.yaml'
// Write to .ci/swagger.yml
func (c Converter) Fetch() error {
	// if the file exists, don't fetch it
	if _, err := os.Stat(c.SwaggerFile); !os.IsNotExist(err) {
		return nil
	}

	if err := os.MkdirAll(ciDir, 0660); err != nil {
		return errors.Wrap(err, "failed to create .ci directory")
	}

	return utils.CurlToFile(c.Url, c.SwaggerFile)
}

// Convert the swagger.yaml to an openapi v3 spec to ./openapi.yaml using `openapi-generator-cli`
// Then applie all changes required for the provider codegen
func (c Converter) Convert() error {
	if err := utils.RunCmd("openapi-generator-cli", "generate",
		"--generator-name", "openapi-yaml",
		"--input-spec", c.SwaggerFile,
		"--output", ciDir,
	); err != nil {
		return errors.Wrap(err, "failed to convert input swagger to openapi v3")
	}

	fileContent, err := os.ReadFile(c.OpenApiSourceFile)
	if err != nil {
		return errors.Wrapf(err, "failed to open file openapi file at %s", c.OpenApiSourceFile)
	}

	var openApiData map[string]interface{}
	if err := yaml.Unmarshal(fileContent, &openApiData); err != nil {
		return errors.Wrap(err, "failed to parse yaml from swagger file")
	}

	fmt.Println("begin openApi modifications for tfplugingen")

	// We need to delete some keys in schema
	// some of our models are recursive, and it's not supported by tfplugingen-openapi
	// this issue is not likely to be fixed anytime soon
	// see: https://github.com/hashicorp/terraform-plugin-codegen-openapi/issues/132
	var keysToDelete = [][]string{
		{"components", "schemas", "MemberOrg", "properties", "invited_by"},
		{"components", "schemas", "MemberTeam", "properties", "invited_by"},
	}

	for _, key := range keysToDelete {
		fmt.Printf("deleting attribute '%s' in OpenApi spec\n", strings.Join(key, "."))
		_, err = utils.DeleteNestedKey(openApiData, key)
		if err != nil {
			return err
		}
	}

	// We must rename some key to avoid typing issues in generated code
	var keysToUpdate = []struct {
		Path   []string
		NewKey string
	}{
		// To avoid type collision with generated code
		{
			Path:   []string{"components", "schemas", "Credential", "properties", "raw"},
			NewKey: "body",
		},
		{
			Path:   []string{"components", "schemas", "UpdateCredential", "properties", "raw"},
			NewKey: "body",
		},
		{
			Path:   []string{"components", "schemas", "NewCredential", "properties", "raw"},
			NewKey: "body",
		},
		// For catalog repository on_create_{visibility,team} feature
		{
			Path:   []string{"components", "schemas", "NewServiceCatalogSource", "properties", "visibility"},
			NewKey: "on_create_visibility",
		},
		{
			Path:   []string{"components", "schemas", "NewServiceCatalogSource", "properties", "team_canonical"},
			NewKey: "on_create_team",
		},
	}

	for _, key := range keysToUpdate {
		fmt.Printf("renaming attribute '%s' to '%s' in OpenApi spec\n", strings.Join(key.Path, "."), key.NewKey)
		err := utils.RenameNestedKey(openApiData, key.Path, key.NewKey)
		if err != nil {
			return err
		}
	}

	// Append a TerraformCycloidProvider component
	// This component is required by the tf-gen tfplugingen-openapi
	// docs: https://developer.hashicorp.com/terraform/plugin/code-generation/openapi-generator#generator-config
	fmt.Println("adding TerraformProviderCycloid component in OpenApi spec")
	terraformProviderSchema := struct {
		Title      string                       `yaml:"title"`
		Required   []string                     `yaml:"required"`
		Properties map[string]map[string]string `yaml:"properties"`
	}{
		Title:    "TerraformProviderCycloid",
		Required: []string{"url", "jwt", "organization_canonical"},
		Properties: map[string]map[string]string{
			"url": {
				"type": "string",
			},
			"jwt": {
				"type": "string",
			},
			"organization_canonical": {
				"type": "string",
			},
		},
	}

	openApiData["components"].(map[string]interface{})["schemas"].(map[string]interface{})["TerraformProviderCycloid"] = terraformProviderSchema

	// Inject AWSStorage, GCPStorage and SwiftStorage for some components
	externalBackendMissingSchema := map[string]map[string]any{
		"aws_storage":   {"$ref": "#/components/schemas/AWSStorage"},
		"gcp_storage":   {"$ref": "#/components/schemas/GCPStorage"},
		"swift_storage": {"$ref": "#/components/schemas/SwiftStorage"},
		"engine":        {"type": "string", "enum": []string{"aws_storage", "gcp_storage", "swift_storage"}},
	}

	for _, schema := range []string{"NewExternalBackend", "UpdateExternalBackend", "NewInfraImportExternalBackend"} {
		for key, value := range externalBackendMissingSchema {
			fmt.Println("Adding key", key, "to schema", schema)
			err = utils.UpdateMapValue(openApiData, []string{"components", "schemas", schema, "properties", key}, value)
			if err != nil {
				return errors.Wrapf(err, "failed to update schema 'components.schema.%s.properties' with '%s'", schema, key)
			}
		}
	}

	// terraform-plugin-codegen-openapi doesn't support schema composition
	// see: https://github.com/hashicorp/terraform-plugin-codegen-openapi/issues/56
	// Be careful if you see a log message in the style of:
	//   level=WARN msg="skipping resource schema mapping" err="found 2 allOf subschema(s), schema composition is currently not supported"
	// This means that a resource has not been generated
	//
	// this section will merge the related attribute manually
	for _, schemaPath := range [][]string{
		{"components", "schemas", "AWSStorage"},
		{"components", "schemas", "GCPStorage"},
		{"components", "schemas", "SwiftStorage"},
	} {
		err = utils.SwaggerMergeAllOf(openApiData, schemaPath)
		if err != nil {
			return err
		}
	}

	// Add OnCreateVisibility and OnCreateTeam parameters for catalog repo

	// write to destination
	fmt.Println("writing output file", c.OpenApiDestFile)
	file, err := os.Create(c.OpenApiDestFile)
	if err != nil {
		return errors.Wrapf(err, "failed to open or create output file '%s'", c.OpenApiDestFile)
	}

	defer file.Close()

	encoder := yaml.NewEncoder(file)
	defer encoder.Close()
	encoder.SetIndent(2)
	return encoder.Encode(openApiData)
}
