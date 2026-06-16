package bucket

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/scality/terraform-provider-artesca/internal/client"
	"github.com/scality/terraform-provider-artesca/internal/creds"
	validators "github.com/scality/terraform-provider-artesca/internal/validators"
)

var (
	_ resource.Resource                = &BucketResource{}
	_ resource.ResourceWithImportState = &BucketResource{}
)

type BucketResource struct {
	s3Client *client.S3Client
}

func NewBucketResource() resource.Resource {
	return &BucketResource{}
}

func (r *BucketResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket"
}

func (r *BucketResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an S3 bucket on the ARTESCA S3 endpoint.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the S3 bucket.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validators.BucketName{},
				},
			},
			"location_constraint": schema.StringAttribute{
				Description: "The ARTESCA location name to use as the storage backend for this bucket.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"versioning_enabled": schema.BoolAttribute{
				Description: "Whether versioning is enabled on the ARTESCA S3 bucket. Required for replication workflows.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"account_access_key": schema.StringAttribute{
				Description: "The access key of the account that owns this bucket.",
				Required:    true,
				Sensitive:   true,
			},
			"account_secret_key": schema.StringAttribute{
				Description: "The secret key of the account that owns this bucket.",
				Required:    true,
				Sensitive:   true,
			},
		},
	}
}

func (r *BucketResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	if providerData.S3 == nil {
		resp.Diagnostics.AddError(
			"S3 Client Not Configured",
			"The s3_endpoint must be set in the provider configuration or via the ARTESCA_S3_ENDPOINT environment variable to use bucket resources.",
		)
		return
	}
	r.s3Client = providerData.S3
}

func (r *BucketResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BucketResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := plan.Name.ValueString()
	accessKey := plan.AccountAccessKey.ValueString()
	secretKey := plan.AccountSecretKey.ValueString()
	locationConstraint := plan.LocationConstraint.ValueString()

	tflog.Debug(ctx, "Creating bucket", map[string]any{
		"bucket":   bucketName,
		"location": locationConstraint,
	})

	err := r.s3Client.CreateBucket(ctx, accessKey, secretKey, bucketName, locationConstraint)
	if err != nil {
		if strings.Contains(err.Error(), "BucketAlreadyOwnedByYou") {
			tflog.Debug(ctx, "Bucket already exists and is owned by this account", map[string]any{"bucket": bucketName})
		} else {
			resp.Diagnostics.AddError("Error creating bucket", err.Error())
			return
		}
	}

	if plan.VersioningEnabled.ValueBool() {
		if err := r.s3Client.PutBucketVersioning(ctx, accessKey, secretKey, bucketName, true); err != nil {
			resp.Diagnostics.AddError("Error enabling bucket versioning", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BucketResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state BucketResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := state.Name.ValueString()
	accessKey := creds.Resolve(state.AccountAccessKey, creds.EnvAccessKey)
	secretKey := creds.Resolve(state.AccountSecretKey, creds.EnvSecretKey)

	exists, err := r.s3Client.HeadBucket(ctx, accessKey, secretKey, bucketName)
	if err != nil {
		resp.Diagnostics.AddError("Error reading bucket", err.Error())
		return
	}
	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}

	if !state.LocationConstraint.IsNull() {
		loc, err := r.s3Client.GetBucketLocation(ctx, accessKey, secretKey, bucketName)
		if err != nil {
			resp.Diagnostics.AddError("Error reading bucket location", err.Error())
			return
		}
		if loc != "" {
			state.LocationConstraint = types.StringValue(loc)
		}
	}

	versioning, err := r.s3Client.GetBucketVersioning(ctx, accessKey, secretKey, bucketName)
	if err != nil {
		resp.Diagnostics.AddError("Error reading bucket versioning", err.Error())
		return
	}
	state.VersioningEnabled = types.BoolValue(versioning)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *BucketResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan BucketResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state BucketResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.VersioningEnabled.Equal(state.VersioningEnabled) {
		bucketName := plan.Name.ValueString()
		accessKey := plan.AccountAccessKey.ValueString()
		secretKey := plan.AccountSecretKey.ValueString()
		if err := r.s3Client.PutBucketVersioning(ctx, accessKey, secretKey, bucketName, plan.VersioningEnabled.ValueBool()); err != nil {
			resp.Diagnostics.AddError("Error updating bucket versioning", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BucketResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state BucketResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := state.Name.ValueString()
	accessKey := creds.Resolve(state.AccountAccessKey, creds.EnvAccessKey)
	secretKey := creds.Resolve(state.AccountSecretKey, creds.EnvSecretKey)

	tflog.Debug(ctx, "Deleting bucket", map[string]any{"bucket": bucketName})

	err := r.s3Client.DeleteBucket(ctx, accessKey, secretKey, bucketName)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting bucket", err.Error())
		return
	}
}

func (r *BucketResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	creds.ImportByID(ctx, "name", req, resp)
}
