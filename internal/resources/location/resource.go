package location

import (
	"context"
	"fmt"
	"strings"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	validators "github.com/cmrh/terraform-provider-artesca/internal/validators"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                   = &LocationResource{}
	_ resource.ResourceWithImportState    = &LocationResource{}
	_ resource.ResourceWithValidateConfig = &LocationResource{}
)

// requiredDetailsByType maps each known location_type to the details fields
// the ARTESCA API requires. Sourced from the swagger location-*-v1 schemas.
// Types not in this map (location-mem-v1, location-file-v1, location-b2-v1,
// location-scality-hdclient-v1, and any future types) skip client-side
// validation and rely on the API to reject incomplete config.
var requiredDetailsByType = map[string][]string{
	"location-aws-s3-v1":             {"access_key", "secret_key", "bucket_name"},
	"location-gcp-v1":                {"access_key", "secret_key", "bucket_name"},
	"location-aws-glacier-v1":        {"access_key", "secret_key", "bucket_name"},
	"location-azure-v1":              {"endpoint", "bucket_name"},
	"location-azure-archive-v1":      {"endpoint", "bucket_name"},
	"location-wasabi-v1":             {"endpoint", "access_key", "secret_key", "bucket_name"},
	"location-do-spaces-v1":          {"endpoint", "access_key", "secret_key", "bucket_name"},
	"location-scality-ring-s3-v1":    {"endpoint", "access_key", "secret_key", "bucket_name"},
	"location-scality-artesca-s3-v1": {"endpoint", "access_key", "secret_key", "bucket_name"},
	"location-ceph-radosgw-s3-v1":    {"endpoint", "access_key", "secret_key", "bucket_name"},
	"location-scality-sproxyd-v1":    {"bootstrap_list", "chord_cos", "proxy_path"},
	"location-scality-hdclient-v2":   {"bootstrap_list"},
	"location-nfs-mount-v1":          {"endpoint"},
	"location-dmf-v1":                {"endpoint", "username", "password", "repo_id", "ns_id"},
	"location-miria-v1":              {"endpoint", "username", "password", "repo_id"},
	"location-scality-crr-v1":        {"endpoint", "access_key", "secret_key"},
}

type LocationResource struct {
	client *client.ManagementClient
}

func NewLocationResource() resource.Resource {
	return &LocationResource{}
}

func (r *LocationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_location"
}

func (r *LocationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an ARTESCA storage location.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the location. Must be 3–63 characters, lowercase letters, numbers, hyphens, and periods.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validators.BucketName{},
				},
			},
			"location_type": schema.StringAttribute{
				Description: "The type of location (e.g., location-aws-s3-v1, location-azure-v1, location-gcp-v1, location-scality-ring-s3-v1, etc.).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"is_transient": schema.BoolAttribute{
				Description: "Whether the location is transient.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"legacy_aws_behavior": schema.BoolAttribute{
				Description: "Whether to use legacy AWS behavior.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"size_limit_gb": schema.Int64Attribute{
				Description: "Size limit in GB for the location.",
				Optional:    true,
			},
			"object_id": schema.StringAttribute{
				Description: "The object ID assigned by ARTESCA.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"is_builtin": schema.BoolAttribute{
				Description: "Whether the location is a built-in location.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"details": schema.SingleNestedBlock{
				Description: "Location-specific configuration details. Required fields depend on the location_type.",
				Attributes: map[string]schema.Attribute{
					"access_key": schema.StringAttribute{
						Description: "Access key for cloud storage authentication.",
						Optional:    true,
						Sensitive:   true,
					},
					"secret_key": schema.StringAttribute{
						Description: "Secret key for cloud storage authentication.",
						Optional:    true,
						Sensitive:   true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"bucket_name": schema.StringAttribute{
						Description: "Name of the target bucket. Must be 3–63 characters, lowercase letters, numbers, hyphens, and periods.",
						Optional:    true,
						Validators: []validator.String{
							validators.BucketName{},
						},
					},
					"bucket_match": schema.BoolAttribute{
						Description: "Whether the bucket name must match exactly.",
						Optional:    true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.RequiresReplace(),
						},
					},
					"endpoint": schema.StringAttribute{
						Description: "Custom endpoint URL for the storage service.",
						Optional:    true,
					},
					"region": schema.StringAttribute{
						Description: "Region for the storage location.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"server_side_encryption": schema.BoolAttribute{
						Description: "Whether to enable server-side encryption.",
						Optional:    true,
					},
					"storage_class": schema.StringAttribute{
						Description: "Storage class to use.",
						Optional:    true,
					},
					"mpu_bucket_name": schema.StringAttribute{
						Description: "Bucket name for multipart uploads.",
						Optional:    true,
					},
					"username": schema.StringAttribute{
						Description: "Username for authentication.",
						Optional:    true,
					},
					"password": schema.StringAttribute{
						Description: "Password for authentication.",
						Optional:    true,
						Sensitive:   true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"tenant_name": schema.StringAttribute{
						Description: "Azure tenant name.",
						Optional:    true,
					},
					"subscription_id": schema.StringAttribute{
						Description: "Azure subscription ID.",
						Optional:    true,
					},
					"resource_group": schema.StringAttribute{
						Description: "Azure resource group.",
						Optional:    true,
					},
					"storage_account_name": schema.StringAttribute{
						Description: "Azure storage account name.",
						Optional:    true,
					},
					"storage_container_name": schema.StringAttribute{
						Description: "Azure storage container name.",
						Optional:    true,
					},
					"ns_id": schema.StringAttribute{
						Description: "Namespace ID for Scality RING.",
						Optional:    true,
					},
					"repo_id": schema.ListAttribute{
						Description: "Repository IDs for Scality RING.",
						Optional:    true,
						ElementType: types.StringType,
					},
					"proxy_path": schema.StringAttribute{
						Description: "Proxy path for NFS/RING.",
						Optional:    true,
					},
					"bootstrap_list": schema.ListAttribute{
						Description: "Bootstrap list for Scality RING sproxyd.",
						Optional:    true,
						ElementType: types.StringType,
					},
					"chord_cos": schema.Int64Attribute{
						Description: "Chord COS for sproxyd.",
						Optional:    true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"coding_parts": schema.Int64Attribute{
						Description: "Number of coding parts for erasure coding.",
						Optional:    true,
					},
					"data_parts": schema.Int64Attribute{
						Description: "Number of data parts for erasure coding.",
						Optional:    true,
					},
					"gcp_endpoint": schema.StringAttribute{
						Description: "GCP endpoint URL.",
						Optional:    true,
					},
					"bucket_prefix": schema.StringAttribute{
						Description: "Bucket prefix.",
						Optional:    true,
					},
				},
			},
		},
	}
}

func (r *LocationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	providerData, ok := req.ProviderData.(*client.ProviderClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.ProviderClients, got: %T", req.ProviderData),
		)
		return
	}
	r.client = providerData.Management
}

func (r *LocationResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config LocationResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if config.LocationType.IsNull() || config.LocationType.IsUnknown() {
		return
	}
	required, ok := requiredDetailsByType[config.LocationType.ValueString()]
	if !ok {
		return
	}
	if len(required) == 0 {
		return
	}
	if config.Details == nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("details"),
			"Missing details block",
			fmt.Sprintf("location_type %q requires a details block with: %s.",
				config.LocationType.ValueString(), strings.Join(required, ", ")),
		)
		return
	}
	for _, field := range required {
		if isLocationDetailsFieldEmpty(config.Details, field) {
			resp.Diagnostics.AddAttributeError(
				path.Root("details").AtName(field),
				"Missing required field",
				fmt.Sprintf("details.%s is required when location_type is %q.",
					field, config.LocationType.ValueString()),
			)
		}
	}
}

// isLocationDetailsFieldEmpty reports whether a required details field is
// effectively unset. Unknown values (e.g. references to other resources) are
// treated as set, since they will be resolved at apply time.
func isLocationDetailsFieldEmpty(d *LocationDetailsModel, field string) bool {
	if d == nil {
		return true
	}
	emptyString := func(s types.String) bool {
		if s.IsUnknown() {
			return false
		}
		return s.IsNull() || s.ValueString() == ""
	}
	emptyInt := func(i types.Int64) bool {
		if i.IsUnknown() {
			return false
		}
		return i.IsNull()
	}
	emptyList := func(l types.List) bool {
		if l.IsUnknown() {
			return false
		}
		return l.IsNull() || len(l.Elements()) == 0
	}
	switch field {
	case "access_key":
		return emptyString(d.AccessKey)
	case "secret_key":
		return emptyString(d.SecretKey)
	case "bucket_name":
		return emptyString(d.BucketName)
	case "endpoint":
		return emptyString(d.Endpoint)
	case "username":
		return emptyString(d.Username)
	case "password":
		return emptyString(d.Password)
	case "ns_id":
		return emptyString(d.NsID)
	case "proxy_path":
		return emptyString(d.ProxyPath)
	case "chord_cos":
		return emptyInt(d.ChordCos)
	case "repo_id":
		return emptyList(d.RepoID)
	case "bootstrap_list":
		return emptyList(d.BootstrapList)
	}
	return false
}

func (r *LocationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan LocationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiLoc := modelToAPILocation(ctx, &plan)

	tflog.Debug(ctx, "Creating location", map[string]any{"name": plan.Name.ValueString()})

	var planSecretKey, planPassword types.String
	if plan.Details != nil {
		planSecretKey = plan.Details.SecretKey
		planPassword = plan.Details.Password
	}

	created, err := r.client.CreateLocation(ctx, apiLoc)
	if err != nil {
		resp.Diagnostics.AddError("Error creating location", err.Error())
		return
	}

	apiLocationToModel(ctx, created, &plan)

	if plan.Details != nil {
		if !planSecretKey.IsNull() && !planSecretKey.IsUnknown() {
			plan.Details.SecretKey = planSecretKey
		}
		if !planPassword.IsNull() && !planPassword.IsUnknown() {
			plan.Details.Password = planPassword
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LocationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state LocationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var stateSecretKey, statePassword types.String
	if state.Details != nil {
		stateSecretKey = state.Details.SecretKey
		statePassword = state.Details.Password
	}

	loc, err := r.client.GetLocation(ctx, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading location", err.Error())
		return
	}
	if loc == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	apiLocationToModel(ctx, loc, &state)

	if state.Details != nil {
		if !stateSecretKey.IsNull() && !stateSecretKey.IsUnknown() {
			state.Details.SecretKey = stateSecretKey
		}
		if !statePassword.IsNull() && !statePassword.IsUnknown() {
			state.Details.Password = statePassword
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *LocationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan LocationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiLoc := modelToAPILocation(ctx, &plan)

	tflog.Debug(ctx, "Updating location", map[string]any{"name": plan.Name.ValueString()})

	updated, err := r.client.UpdateLocation(ctx, plan.Name.ValueString(), apiLoc)
	if err != nil {
		resp.Diagnostics.AddError("Error updating location", err.Error())
		return
	}

	apiLocationToModel(ctx, updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LocationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state LocationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting location", map[string]any{"name": state.Name.ValueString()})

	err := r.client.DeleteLocation(ctx, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting location", err.Error())
		return
	}
}

func (r *LocationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

// --- Conversion helpers ---

func modelToAPILocation(ctx context.Context, model *LocationResourceModel) *client.Location {
	loc := &client.Location{
		Name:              model.Name.ValueString(),
		LocationType:      model.LocationType.ValueString(),
		IsTransient:       model.IsTransient.ValueBool(),
		LegacyAwsBehavior: model.LegacyAwsBehavior.ValueBool(),
	}

	if !model.ObjectID.IsNull() && !model.ObjectID.IsUnknown() {
		loc.ObjectID = model.ObjectID.ValueString()
	}

	if !model.SizeLimitGB.IsNull() && !model.SizeLimitGB.IsUnknown() {
		loc.SizeLimitGB = model.SizeLimitGB.ValueInt64()
	}

	if model.Details != nil {
		loc.Details = modelToAPIDetails(ctx, model.Details)
	}

	return loc
}

func modelToAPIDetails(ctx context.Context, d *LocationDetailsModel) *client.LocationDetails {
	details := &client.LocationDetails{}

	if !d.AccessKey.IsNull() && !d.AccessKey.IsUnknown() {
		details.AccessKey = d.AccessKey.ValueString()
	}
	if !d.SecretKey.IsNull() && !d.SecretKey.IsUnknown() {
		details.SecretKey = d.SecretKey.ValueString()
	}
	if !d.BucketName.IsNull() && !d.BucketName.IsUnknown() {
		details.BucketName = d.BucketName.ValueString()
	}
	if !d.BucketMatch.IsNull() && !d.BucketMatch.IsUnknown() {
		v := d.BucketMatch.ValueBool()
		details.BucketMatch = &v
	}
	if !d.Endpoint.IsNull() && !d.Endpoint.IsUnknown() {
		details.Endpoint = d.Endpoint.ValueString()
	}
	if !d.Region.IsNull() && !d.Region.IsUnknown() {
		details.Region = d.Region.ValueString()
	}
	if !d.ServerSideEncryption.IsNull() && !d.ServerSideEncryption.IsUnknown() {
		v := d.ServerSideEncryption.ValueBool()
		details.ServerSideEncryption = &v
	}
	if !d.StorageClass.IsNull() && !d.StorageClass.IsUnknown() {
		details.StorageClass = d.StorageClass.ValueString()
	}
	if !d.MpuBucketName.IsNull() && !d.MpuBucketName.IsUnknown() {
		details.MpuBucketName = d.MpuBucketName.ValueString()
	}
	if !d.Username.IsNull() && !d.Username.IsUnknown() {
		details.Username = d.Username.ValueString()
	}
	if !d.Password.IsNull() && !d.Password.IsUnknown() {
		details.Password = d.Password.ValueString()
	}
	if !d.TenantName.IsNull() && !d.TenantName.IsUnknown() {
		details.TenantName = d.TenantName.ValueString()
	}
	if !d.SubscriptionID.IsNull() && !d.SubscriptionID.IsUnknown() {
		details.SubscriptionID = d.SubscriptionID.ValueString()
	}
	if !d.ResourceGroup.IsNull() && !d.ResourceGroup.IsUnknown() {
		details.ResourceGroup = d.ResourceGroup.ValueString()
	}
	if !d.StorageAccountName.IsNull() && !d.StorageAccountName.IsUnknown() {
		details.StorageAccountName = d.StorageAccountName.ValueString()
	}
	if !d.StorageContainerName.IsNull() && !d.StorageContainerName.IsUnknown() {
		details.StorageContainerName = d.StorageContainerName.ValueString()
	}
	if !d.NsID.IsNull() && !d.NsID.IsUnknown() {
		details.NsID = d.NsID.ValueString()
	}
	if !d.ProxyPath.IsNull() && !d.ProxyPath.IsUnknown() {
		details.ProxyPath = d.ProxyPath.ValueString()
	}
	if !d.ChordCos.IsNull() && !d.ChordCos.IsUnknown() {
		v := d.ChordCos.ValueInt64()
		details.ChordCos = &v
	}
	if !d.CodingParts.IsNull() && !d.CodingParts.IsUnknown() {
		v := d.CodingParts.ValueInt64()
		details.CodingParts = &v
	}
	if !d.DataParts.IsNull() && !d.DataParts.IsUnknown() {
		v := d.DataParts.ValueInt64()
		details.DataParts = &v
	}
	if !d.GcpEndpoint.IsNull() && !d.GcpEndpoint.IsUnknown() {
		details.GcpEndpoint = d.GcpEndpoint.ValueString()
	}
	if !d.BucketPrefix.IsNull() && !d.BucketPrefix.IsUnknown() {
		details.BucketPrefix = d.BucketPrefix.ValueString()
	}

	if !d.RepoID.IsNull() && !d.RepoID.IsUnknown() {
		var repoIDs []string
		if diags := d.RepoID.ElementsAs(ctx, &repoIDs, false); !diags.HasError() {
			details.RepoID = repoIDs
		}
	}
	if !d.BootstrapList.IsNull() && !d.BootstrapList.IsUnknown() {
		var bootstrapList []string
		if diags := d.BootstrapList.ElementsAs(ctx, &bootstrapList, false); !diags.HasError() {
			details.BootstrapList = bootstrapList
		}
	}

	return details
}

func apiLocationToModel(ctx context.Context, loc *client.Location, model *LocationResourceModel) {
	model.Name = types.StringValue(loc.Name)
	model.LocationType = types.StringValue(loc.LocationType)
	model.IsTransient = types.BoolValue(loc.IsTransient)
	model.LegacyAwsBehavior = types.BoolValue(loc.LegacyAwsBehavior)
	model.IsBuiltin = types.BoolValue(loc.IsBuiltin)

	if loc.ObjectID != "" {
		model.ObjectID = types.StringValue(loc.ObjectID)
	}

	if loc.SizeLimitGB > 0 {
		model.SizeLimitGB = types.Int64Value(loc.SizeLimitGB)
	}

	if loc.Details != nil {
		if model.Details == nil {
			model.Details = &LocationDetailsModel{}
		}
		apiDetailsToModel(ctx, loc.Details, model.Details)
	}
}

func apiDetailsToModel(ctx context.Context, d *client.LocationDetails, model *LocationDetailsModel) {
	// Only update a field from the API if the user configured it (non-null in state)
	// or the value is non-empty from the API. This avoids null→value drift for
	// API-defaulted fields the user never set.
	setIfConfigured := func(target *types.String, value string) {
		if value != "" && !target.IsNull() {
			*target = types.StringValue(value)
		}
	}

	setIfConfigured(&model.AccessKey, d.AccessKey)
	// Sensitive fields: preserve state value, API may return redacted data
	setIfConfigured(&model.BucketName, d.BucketName)
	if d.BucketMatch != nil {
		model.BucketMatch = types.BoolValue(*d.BucketMatch)
	}
	setIfConfigured(&model.Endpoint, d.Endpoint)
	setIfConfigured(&model.Region, d.Region)
	if d.ServerSideEncryption != nil {
		model.ServerSideEncryption = types.BoolValue(*d.ServerSideEncryption)
	}
	setIfConfigured(&model.StorageClass, d.StorageClass)
	setIfConfigured(&model.MpuBucketName, d.MpuBucketName)
	setIfConfigured(&model.Username, d.Username)
	setIfConfigured(&model.TenantName, d.TenantName)
	setIfConfigured(&model.SubscriptionID, d.SubscriptionID)
	setIfConfigured(&model.ResourceGroup, d.ResourceGroup)
	setIfConfigured(&model.StorageAccountName, d.StorageAccountName)
	setIfConfigured(&model.StorageContainerName, d.StorageContainerName)
	setIfConfigured(&model.NsID, d.NsID)
	setIfConfigured(&model.ProxyPath, d.ProxyPath)
	setIfConfigured(&model.GcpEndpoint, d.GcpEndpoint)
	setIfConfigured(&model.BucketPrefix, d.BucketPrefix)

	if d.ChordCos != nil {
		model.ChordCos = types.Int64Value(*d.ChordCos)
	}
	if d.CodingParts != nil {
		model.CodingParts = types.Int64Value(*d.CodingParts)
	}
	if d.DataParts != nil {
		model.DataParts = types.Int64Value(*d.DataParts)
	}

	if len(d.RepoID) > 0 {
		repoIDs, _ := types.ListValueFrom(ctx, types.StringType, d.RepoID)
		model.RepoID = repoIDs
	} else {
		model.RepoID = types.ListNull(types.StringType)
	}
	if len(d.BootstrapList) > 0 {
		bootstrapList, _ := types.ListValueFrom(ctx, types.StringType, d.BootstrapList)
		model.BootstrapList = bootstrapList
	} else {
		model.BootstrapList = types.ListNull(types.StringType)
	}
}
