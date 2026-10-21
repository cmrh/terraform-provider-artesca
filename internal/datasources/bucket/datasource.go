package bucket

import (
	"context"
	"fmt"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/cmrh/terraform-provider-artesca/internal/creds"
	validators "github.com/cmrh/terraform-provider-artesca/internal/validators"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &BucketDataSource{}

// defaultLocation is the location of a bucket created without a location
// constraint; GetBucketLocation reports "" for it.
const defaultLocation = "us-east-1"

type BucketDataSource struct {
	accounts *client.AccountCredentialSource
	s3Client *client.S3Client
}

func NewBucketDataSource() datasource.DataSource {
	return &BucketDataSource{}
}

func (d *BucketDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket"
}

func (d *BucketDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up an existing bucket in an ARTESCA account.",
		Attributes: map[string]schema.Attribute{
			creds.AttrAccountName: creds.DataSourceAttribute(),
			"name": schema.StringAttribute{
				Description: "The name of the bucket to look up.",
				Required:    true,
				Validators: []validator.String{
					validators.BucketName{},
				},
			},
			"arn": schema.StringAttribute{
				Description: "The ARN of the bucket (arn:aws:s3:::<name>).",
				Computed:    true,
			},
			"location_constraint": schema.StringAttribute{
				Description: "The ARTESCA location the bucket stores data in; `us-east-1` for the default location.",
				Computed:    true,
			},
			"versioning_enabled": schema.BoolAttribute{
				Description: "Whether versioning is enabled.",
				Computed:    true,
			},
			"object_lock_enabled": schema.BoolAttribute{
				Description: "Whether the bucket was created with object lock.",
				Computed:    true,
			},
			"tags": schema.MapAttribute{
				Description: "The bucket's tags; empty when it has none.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *BucketDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	if providerData.S3 == nil {
		resp.Diagnostics.AddError(
			"S3 Client Not Configured",
			"The s3_endpoint must be set in the provider configuration or via the ARTESCA_S3_ENDPOINT environment variable to use bucket data sources.",
		)
		return
	}
	d.s3Client = providerData.S3
	d.accounts = providerData.Accounts
}

func (d *BucketDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data BucketDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucket := data.Name.ValueString()
	acctCreds, err := d.accounts.For(ctx, data.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting account credentials", err.Error())
		return
	}

	exists, err := d.s3Client.HeadBucket(ctx, acctCreds, bucket)
	if err != nil {
		resp.Diagnostics.AddError("Error reading bucket", err.Error())
		return
	}
	if !exists {
		resp.Diagnostics.AddError("Bucket not found",
			fmt.Sprintf("Account %q has no bucket named %q.", data.AccountName.ValueString(), bucket))
		return
	}

	loc, err := d.s3Client.GetBucketLocation(ctx, acctCreds, bucket)
	if err != nil {
		resp.Diagnostics.AddError("Error reading bucket location", err.Error())
		return
	}
	if loc == "" {
		loc = defaultLocation
	}

	versioning, err := d.s3Client.GetBucketVersioning(ctx, acctCreds, bucket)
	if err != nil {
		resp.Diagnostics.AddError("Error reading bucket versioning", err.Error())
		return
	}

	objectLock, _, err := d.s3Client.GetObjectLockConfiguration(ctx, acctCreds, bucket)
	if err != nil {
		resp.Diagnostics.AddError("Error reading bucket object lock configuration", err.Error())
		return
	}

	tags, err := d.s3Client.GetBucketTagging(ctx, acctCreds, bucket)
	if err != nil {
		resp.Diagnostics.AddError("Error reading bucket tags", err.Error())
		return
	}
	tagMap := make(map[string]string, len(tags))
	for _, t := range tags {
		tagMap[t.Key] = t.Value
	}
	tagValue, diags := types.MapValueFrom(ctx, types.StringType, tagMap)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ARN = types.StringValue("arn:aws:s3:::" + bucket)
	data.LocationConstraint = types.StringValue(loc)
	data.VersioningEnabled = types.BoolValue(versioning)
	data.ObjectLockEnabled = types.BoolValue(objectLock)
	data.Tags = tagValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
