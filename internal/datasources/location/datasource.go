package location

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/scality/terraform-provider-scality-artesca/internal/client"
)

var _ datasource.DataSource = &LocationDataSource{}

type LocationDataSource struct {
	client *client.ManagementClient
}

func NewLocationDataSource() datasource.DataSource {
	return &LocationDataSource{}
}

func (d *LocationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_location"
}

func (d *LocationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up an existing ARTESCA storage location by name. Sensitive details (secret_key, password) are not returned -- the overlay view masks them.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the location to look up.",
				Required:    true,
			},
			"location_type": schema.StringAttribute{
				Description: "Backend type (e.g. location-aws-s3-v1).",
				Computed:    true,
			},
			"is_transient": schema.BoolAttribute{
				Description: "Whether the location is transient.",
				Computed:    true,
			},
			"is_builtin": schema.BoolAttribute{
				Description: "Whether the location is a built-in default.",
				Computed:    true,
			},
			"legacy_aws_behavior": schema.BoolAttribute{
				Description: "Whether legacy AWS behavior is enabled.",
				Computed:    true,
			},
			"size_limit_gb": schema.Int64Attribute{
				Description: "Storage size limit in gigabytes.",
				Computed:    true,
			},
			"object_id": schema.StringAttribute{
				Description: "Internal object identifier.",
				Computed:    true,
			},
			"details": schema.SingleNestedAttribute{
				Description: "Backend-specific configuration. Sensitive fields are blank.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"access_key":             schema.StringAttribute{Computed: true, Sensitive: true},
					"secret_key":             schema.StringAttribute{Computed: true, Sensitive: true},
					"bucket_name":            schema.StringAttribute{Computed: true},
					"bucket_match":           schema.BoolAttribute{Computed: true},
					"endpoint":               schema.StringAttribute{Computed: true},
					"region":                 schema.StringAttribute{Computed: true},
					"server_side_encryption": schema.BoolAttribute{Computed: true},
					"storage_class":          schema.StringAttribute{Computed: true},
					"mpu_bucket_name":        schema.StringAttribute{Computed: true},
					"username":               schema.StringAttribute{Computed: true},
					"password":               schema.StringAttribute{Computed: true, Sensitive: true},
					"tenant_name":            schema.StringAttribute{Computed: true},
					"subscription_id":        schema.StringAttribute{Computed: true},
					"resource_group":         schema.StringAttribute{Computed: true},
					"storage_account_name":   schema.StringAttribute{Computed: true},
					"storage_container_name": schema.StringAttribute{Computed: true},
					"ns_id":                  schema.StringAttribute{Computed: true},
					"repo_id":                schema.ListAttribute{Computed: true, ElementType: types.StringType},
					"proxy_path":             schema.StringAttribute{Computed: true},
					"bootstrap_list":         schema.ListAttribute{Computed: true, ElementType: types.StringType},
					"chord_cos":              schema.Int64Attribute{Computed: true},
					"coding_parts":           schema.Int64Attribute{Computed: true},
					"data_parts":             schema.Int64Attribute{Computed: true},
					"gcp_endpoint":           schema.StringAttribute{Computed: true},
					"bucket_prefix":          schema.StringAttribute{Computed: true},
				},
			},
		},
	}
}

func (d *LocationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	providerData, ok := req.ProviderData.(*client.ProviderClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.ProviderClients, got: %T", req.ProviderData),
		)
		return
	}
	d.client = providerData.Management
}

func (d *LocationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data LocationDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	loc, err := d.client.GetLocation(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading location", err.Error())
		return
	}
	if loc == nil {
		resp.Diagnostics.AddError(
			"Location not found",
			fmt.Sprintf("No location exists with name %q.", data.Name.ValueString()),
		)
		return
	}

	data.LocationType = types.StringValue(loc.LocationType)
	data.IsTransient = types.BoolValue(loc.IsTransient)
	data.IsBuiltin = types.BoolValue(loc.IsBuiltin)
	data.LegacyAwsBehavior = types.BoolValue(loc.LegacyAwsBehavior)
	data.SizeLimitGB = types.Int64Value(loc.SizeLimitGB)
	data.ObjectID = types.StringValue(loc.ObjectID)
	data.Details = apiDetailsToDataModel(ctx, loc.Details)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func apiDetailsToDataModel(ctx context.Context, d *client.LocationDetails) *LocationDetailsDataModel {
	if d == nil {
		return &LocationDetailsDataModel{
			RepoID:        types.ListValueMust(types.StringType, nil),
			BootstrapList: types.ListValueMust(types.StringType, nil),
		}
	}

	stringOrNull := func(s string) types.String {
		if s == "" {
			return types.StringNull()
		}
		return types.StringValue(s)
	}
	boolOrNull := func(b *bool) types.Bool {
		if b == nil {
			return types.BoolNull()
		}
		return types.BoolValue(*b)
	}
	int64OrNull := func(i *int64) types.Int64 {
		if i == nil {
			return types.Int64Null()
		}
		return types.Int64Value(*i)
	}
	listOrEmpty := func(s []string) types.List {
		if s == nil {
			return types.ListValueMust(types.StringType, nil)
		}
		l, _ := types.ListValueFrom(ctx, types.StringType, s)
		return l
	}

	return &LocationDetailsDataModel{
		AccessKey:            stringOrNull(d.AccessKey),
		SecretKey:            stringOrNull(d.SecretKey),
		BucketName:           stringOrNull(d.BucketName),
		BucketMatch:          boolOrNull(d.BucketMatch),
		Endpoint:             stringOrNull(d.Endpoint),
		Region:               stringOrNull(d.Region),
		ServerSideEncryption: boolOrNull(d.ServerSideEncryption),
		StorageClass:         stringOrNull(d.StorageClass),
		MpuBucketName:        stringOrNull(d.MpuBucketName),
		Username:             stringOrNull(d.Username),
		Password:             stringOrNull(d.Password),
		TenantName:           stringOrNull(d.TenantName),
		SubscriptionID:       stringOrNull(d.SubscriptionID),
		ResourceGroup:        stringOrNull(d.ResourceGroup),
		StorageAccountName:   stringOrNull(d.StorageAccountName),
		StorageContainerName: stringOrNull(d.StorageContainerName),
		NsID:                 stringOrNull(d.NsID),
		RepoID:               listOrEmpty(d.RepoID),
		ProxyPath:            stringOrNull(d.ProxyPath),
		BootstrapList:        listOrEmpty(d.BootstrapList),
		ChordCos:             int64OrNull(d.ChordCos),
		CodingParts:          int64OrNull(d.CodingParts),
		DataParts:            int64OrNull(d.DataParts),
		GcpEndpoint:          stringOrNull(d.GcpEndpoint),
		BucketPrefix:         stringOrNull(d.BucketPrefix),
	}
}
