package provider

import (
	"context"

	"github.com/cycloidio/cycloid-cli/client/models"
	"github.com/cycloidio/cycloid-cli/cmd/cycloid/common"
	"github.com/cycloidio/cycloid-cli/cmd/cycloid/middleware"
	"github.com/cycloidio/terraform-provider-cycloid/provider_cycloid"
	"github.com/cycloidio/terraform-provider-cycloid/resource_external_backend"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = (*externalBackendResource)(nil)

func NewExternalBackendResource() resource.Resource {
	return &externalBackendResource{}
}

type externalBackendResource struct {
	provider provider_cycloid.CycloidModel
}

type externalBackendResourceModel resource_external_backend.ExternalBackendModel

func (r *externalBackendResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_external_backend"
}

func (r *externalBackendResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_external_backend.ExternalBackendResourceSchema(ctx)
}

func (r *externalBackendResource) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	pv, ok := req.ProviderData.(provider_cycloid.CycloidModel)
	if !ok {
		tflog.Error(ctx, "Unable to prepare client")
		return
	}
	r.provider = pv
}

var enginesFromCY = map[string]string{
	"AWSStorage":   "aws_storage",
	"GCPStorage":   "gcp_storage",
	"SwiftStorage": "swift_storage",
}

var enginesToCY = map[string]string{
	"aws_storage":   "AWSStorage",
	"gcp_storage":   "GCPStorage",
	"swift_storage": "SwiftStorage",
}

func (r *externalBackendResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data externalBackendResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	api := common.NewAPI(common.WithURL(r.provider.Url.ValueString()), common.WithToken(r.provider.Jwt.ValueString()))
	mid := middleware.NewMiddleware(api)

	project := data.ProjectCanonical.ValueString()
	env := data.EnvironmentCanonical.ValueString()
	purpose := data.Purpose.ValueString()
	cred := data.CredentialCanonical.ValueString()
	def := data.Default.ValueBool()

	configuration := readEBConfiguration(ctx, resp.Diagnostics, data)
	if resp.Diagnostics.HasError() {
		return
	}

	orgCan := getOrganizationCanonical(r.provider, data.OrganizationCanonical)

	eb, err := mid.CreateExternalBackends(orgCan, project, env, purpose, cred, def, configuration)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable create external backend",
			err.Error(),
		)
		return
	}

	ebCYModelToData(ctx, resp.Diagnostics, orgCan, eb, &data)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func readEBConfiguration(ctx context.Context, diag diag.Diagnostics, data externalBackendResourceModel) models.ExternalBackendConfiguration {
	var cfg models.ExternalBackendConfiguration

	engine := enginesToCY[data.Engine.ValueString()]

	switch engine {
	case "AWSStorage":
		cfg = &models.AWSStorage{
			Bucket:           data.AwsStorage.Bucket.ValueStringPointer(),
			Endpoint:         data.AwsStorage.Endpoint.ValueString(),
			Key:              data.AwsStorage.Key.ValueString(),
			Region:           data.AwsStorage.Region.ValueStringPointer(),
			S3ForcePathStyle: data.AwsStorage.S3ForcePathStyle.ValueBool(),
			SkipVerifySsl:    data.AwsStorage.SkipVerifySsl.ValueBool(),
		}
	case "GCPStorage":
		cfg = &models.GCPStorage{
			Bucket: data.GcpStorage.Bucket.ValueStringPointer(),
			Object: data.GcpStorage.Object.ValueString(),
		}
	case "SwiftStorage":
		cfg = &models.SwiftStorage{
			Container:     data.SwiftStorage.Container.ValueStringPointer(),
			Object:        data.SwiftStorage.Object.ValueString(),
			Region:        data.SwiftStorage.Region.ValueStringPointer(),
			SkipVerifySsl: data.SwiftStorage.SkipVerifySsl.ValueBool(),
		}
	default:
		diag.AddError("Unable to read configuration", "Unknown engine")
		return nil
	}

	return cfg
}

// credentialCYModelToData converts the 'cred' into the 'credentialResourceModel'
func ebCYModelToData(ctx context.Context, diag diag.Diagnostics, org string, eb *models.ExternalBackend, data *externalBackendResourceModel) {
	engine := enginesFromCY[eb.Configuration().Engine()]

	data.OrganizationCanonical = types.StringValue(org)
	data.ProjectCanonical = types.StringValue(eb.ProjectCanonical)
	data.EnvironmentCanonical = types.StringValue(eb.EnvironmentCanonical)
	data.Purpose = types.StringPointerValue(eb.Purpose)
	data.Default = types.BoolPointerValue(eb.Default)
	data.CredentialCanonical = types.StringValue(eb.CredentialCanonical)
	data.Engine = types.StringValue(engine)
	data.ExternalBackendId = types.Int64Value(int64(eb.ID))
	data.AwsStorage = resource_external_backend.AwsStorageValue{}
	data.GcpStorage = resource_external_backend.GcpStorageValue{}
	data.SwiftStorage = resource_external_backend.SwiftStorageValue{}

	switch engine {
	case "aws_storage":
		awsStorage := eb.Configuration().(*models.AWSStorage)
		attrTypes := data.AwsStorage.AttributeTypes(ctx)

		attrValues := map[string]attr.Value{
			"bucket":              types.StringPointerValue(awsStorage.Bucket),
			"endpoint":            types.StringValue(awsStorage.Endpoint),
			"key":                 types.StringValue(awsStorage.Key),
			"region":              types.StringPointerValue(awsStorage.Region),
			"s3_force_path_style": types.BoolValue(awsStorage.S3ForcePathStyle),
			"skip_verify_ssl":     types.BoolValue(awsStorage.SkipVerifySsl),
		}
		awsStorageEB, diags := resource_external_backend.NewAwsStorageValue(attrTypes, attrValues)
		if diags.HasError() {
			diag.Append(diags...)
			return
		}
		data.AwsStorage = awsStorageEB

	case "gcp_storage":
		gcpStorage := eb.Configuration().(*models.GCPStorage)

		attrTypes := data.GcpStorage.AttributeTypes(ctx)

		attrValues := map[string]attr.Value{
			"bucket": types.StringPointerValue(gcpStorage.Bucket),
			"object": types.StringValue(gcpStorage.Object),
		}
		gcpStorageEB, diags := resource_external_backend.NewGcpStorageValue(attrTypes, attrValues)
		if diags.HasError() {
			diag.Append(diags...)
			return
		}
		data.GcpStorage = gcpStorageEB

	case "swift_storage":
		swiftStorage := eb.Configuration().(*models.SwiftStorage)

		attrTypes := data.SwiftStorage.AttributeTypes(ctx)
		attrValues := map[string]attr.Value{
			"container": types.StringPointerValue(swiftStorage.Container),
			"object":    types.StringValue(swiftStorage.Object),
			"region":    types.StringPointerValue(swiftStorage.Region),
		}
		swiftStorageEB, diags := resource_external_backend.NewSwiftStorageValue(attrTypes, attrValues)
		if diags.HasError() {
			diag.Append(diags...)
			return
		}
		data.SwiftStorage = swiftStorageEB

	default:
		diag.AddError("Unable to read configuration", "Unknown engine")
	}

	return
}

func (r *externalBackendResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data externalBackendResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read API call logic
	api := common.NewAPI(common.WithURL(r.provider.Url.ValueString()), common.WithToken(r.provider.Jwt.ValueString()))
	mid := middleware.NewMiddleware(api)

	id := data.ExternalBackendId.ValueInt64()

	orgCan := getOrganizationCanonical(r.provider, data.OrganizationCanonical)

	eb, err := mid.GetExternalBackend(orgCan, uint32(id))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable read external backend",
			err.Error(),
		)
		return
	}

	ebCYModelToData(ctx, resp.Diagnostics, orgCan, eb, &data)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *externalBackendResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data externalBackendResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Update API call logic
	// Read API call logic
	api := common.NewAPI(common.WithURL(r.provider.Url.ValueString()), common.WithToken(r.provider.Jwt.ValueString()))
	mid := middleware.NewMiddleware(api)

	orgCan := getOrganizationCanonical(r.provider, data.OrganizationCanonical)

	configuration := readEBConfiguration(ctx, resp.Diagnostics, data)
	if resp.Diagnostics.HasError() {
		return
	}

	var (
		eb  *models.ExternalBackend
		err error
	)

	eb, err = mid.GetExternalBackend(orgCan, uint32(data.ExternalBackendId.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable read external backend",
			err.Error(),
		)
		return
	}

	eb, err = mid.UpdateExternalBackend(orgCan, eb.ID, *eb.Purpose, eb.CredentialCanonical, *eb.Default, configuration)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable update external backend",
			err.Error(),
		)
		return
	}

	ebCYModelToData(ctx, resp.Diagnostics, orgCan, eb, &data)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *externalBackendResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data externalBackendResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete API call logic
	api := common.NewAPI(common.WithURL(r.provider.Url.ValueString()), common.WithToken(r.provider.Jwt.ValueString()))
	mid := middleware.NewMiddleware(api)

	orgCan := getOrganizationCanonical(r.provider, data.OrganizationCanonical)

	id := data.ExternalBackendId.ValueInt64()
	err := mid.DeleteExternalBackend(orgCan, uint32(id))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete external backend",
			err.Error(),
		)
		return
	}
}
