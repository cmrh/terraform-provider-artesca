package bucketencryption

import (
	"context"
	"fmt"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/cmrh/terraform-provider-artesca/internal/creds"
	validators "github.com/cmrh/terraform-provider-artesca/internal/validators"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                 = &BucketEncryptionResource{}
	_ resource.ResourceWithImportState  = &BucketEncryptionResource{}
	_ resource.ResourceWithUpgradeState = &BucketEncryptionResource{}
)

type BucketEncryptionResource struct {
	accounts *client.AccountCredentialSource
	s3Client *client.S3Client
}

func NewBucketEncryptionResource() resource.Resource {
	return &BucketEncryptionResource{}
}

func (r *BucketEncryptionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket_encryption"
}

func (r *BucketEncryptionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "Manages the server-side encryption configuration of an ARTESCA bucket.",
		Attributes: map[string]schema.Attribute{
			creds.AttrAccountName: creds.ResourceAttribute(),
			"bucket_name": schema.StringAttribute{
				Description: "The name of the bucket to configure encryption on.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validators.BucketName{},
				},
			},
			"sse_algorithm": schema.StringAttribute{
				Description: "The server-side encryption algorithm. Currently only \"AES256\" (SSE-S3) is supported by ARTESCA.",
				Required:    true,
				Validators: []validator.String{
					validators.SSEAlgorithm{},
				},
			},
			"bucket_key_enabled": schema.BoolAttribute{
				Description: "Whether to use an S3 Bucket Key. Defaults to false. Computed from the server when not set.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *BucketEncryptionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.s3Client = providerData.S3
	r.accounts = providerData.Accounts
}

func (r *BucketEncryptionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BucketEncryptionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg := client.BucketEncryptionConfig{
		SSEAlgorithm:     plan.SSEAlgorithm.ValueString(),
		BucketKeyEnabled: plan.BucketKeyEnabled.ValueBool(),
	}
	tflog.Debug(ctx, "Putting bucket encryption", map[string]any{"bucket": plan.BucketName.ValueString(), "sse": cfg.SSEAlgorithm})

	acctCreds, err := r.accounts.For(ctx, plan.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting account credentials", err.Error())
		return
	}

	unlock := r.s3Client.LockBucket(plan.BucketName.ValueString())
	defer unlock()

	if err := r.s3Client.PutBucketEncryption(ctx,
		acctCreds,
		plan.BucketName.ValueString(),
		cfg,
	); err != nil {
		resp.Diagnostics.AddError("Error putting bucket encryption", err.Error())
		return
	}

	plan.BucketKeyEnabled = types.BoolValue(cfg.BucketKeyEnabled)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BucketEncryptionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state BucketEncryptionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	acctCreds, err := r.accounts.For(ctx, state.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting account credentials", err.Error())
		return
	}

	cfg, err := r.s3Client.GetBucketEncryption(ctx,
		acctCreds,
		state.BucketName.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error reading bucket encryption", err.Error())
		return
	}
	if cfg == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.SSEAlgorithm = types.StringValue(cfg.SSEAlgorithm)
	state.BucketKeyEnabled = types.BoolValue(cfg.BucketKeyEnabled)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *BucketEncryptionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan BucketEncryptionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg := client.BucketEncryptionConfig{
		SSEAlgorithm:     plan.SSEAlgorithm.ValueString(),
		BucketKeyEnabled: plan.BucketKeyEnabled.ValueBool(),
	}
	tflog.Debug(ctx, "Updating bucket encryption", map[string]any{"bucket": plan.BucketName.ValueString(), "sse": cfg.SSEAlgorithm})

	acctCreds, err := r.accounts.For(ctx, plan.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting account credentials", err.Error())
		return
	}

	unlock := r.s3Client.LockBucket(plan.BucketName.ValueString())
	defer unlock()

	if err := r.s3Client.PutBucketEncryption(ctx,
		acctCreds,
		plan.BucketName.ValueString(),
		cfg,
	); err != nil {
		resp.Diagnostics.AddError("Error updating bucket encryption", err.Error())
		return
	}

	plan.BucketKeyEnabled = types.BoolValue(cfg.BucketKeyEnabled)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BucketEncryptionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state BucketEncryptionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting bucket encryption", map[string]any{"bucket": state.BucketName.ValueString()})

	acctCreds, err := r.accounts.For(ctx, state.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting account credentials", err.Error())
		return
	}

	unlock := r.s3Client.LockBucket(state.BucketName.ValueString())
	defer unlock()

	if err := r.s3Client.DeleteBucketEncryption(ctx,
		acctCreds,
		state.BucketName.ValueString(),
	); err != nil {
		resp.Diagnostics.AddError("Error deleting bucket encryption", err.Error())
	}
}

func (r *BucketEncryptionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	creds.ImportByID(ctx, "bucket_name", req, resp)
}

// UpgradeState migrates v0 state, which held the account's access key pair,
// to account_name.
func (r *BucketEncryptionResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	return creds.UpgradeFromAccountKeys(r, func() *client.AccountCredentialSource { return r.accounts })
}
