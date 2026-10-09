package bucketpolicy

import (
	"context"
	"fmt"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/cmrh/terraform-provider-artesca/internal/creds"
	"github.com/cmrh/terraform-provider-artesca/internal/policydoc"
	validators "github.com/cmrh/terraform-provider-artesca/internal/validators"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                 = &BucketPolicyResource{}
	_ resource.ResourceWithImportState  = &BucketPolicyResource{}
	_ resource.ResourceWithUpgradeState = &BucketPolicyResource{}
)

type BucketPolicyResource struct {
	accounts *client.AccountCredentialSource
	s3Client *client.S3Client
}

func NewBucketPolicyResource() resource.Resource {
	return &BucketPolicyResource{}
}

func (r *BucketPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket_policy"
}

func (r *BucketPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "Attaches an S3 bucket policy to an ARTESCA bucket. ARTESCA validates the policy server-side; Resource ARNs that don't match the bucket are rejected with MalformedPolicy.",
		Attributes: map[string]schema.Attribute{
			creds.AttrAccountName: creds.ResourceAttribute(),
			"bucket_name": schema.StringAttribute{
				Description: "The name of the bucket to attach the policy to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validators.BucketName{},
				},
			},
			"policy": schema.StringAttribute{
				Description: "The JSON policy document. Whitespace and key ordering differences are ignored when detecting drift.",
				Required:    true,
				Validators: []validator.String{
					validators.JSONDocument{},
				},
				PlanModifiers: []planmodifier.String{
					policydoc.EquivalenceModifier(),
				},
			},
		},
	}
}

func (r *BucketPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *BucketPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BucketPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Attaching bucket policy", map[string]any{"bucket": plan.BucketName.ValueString()})

	acctCreds, err := r.accounts.For(ctx, plan.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting account credentials", err.Error())
		return
	}

	unlock := r.s3Client.LockBucket(plan.BucketName.ValueString())
	defer unlock()

	if err := r.s3Client.PutBucketPolicy(ctx,
		acctCreds,
		plan.BucketName.ValueString(),
		plan.Policy.ValueString(),
	); err != nil {
		resp.Diagnostics.AddError("Error attaching bucket policy", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BucketPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state BucketPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	acctCreds, err := r.accounts.For(ctx, state.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting account credentials", err.Error())
		return
	}

	remote, err := r.s3Client.GetBucketPolicy(ctx,
		acctCreds,
		state.BucketName.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error reading bucket policy", err.Error())
		return
	}
	if remote == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Policy = policydoc.Refresh(state.Policy, remote)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *BucketPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan BucketPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating bucket policy", map[string]any{"bucket": plan.BucketName.ValueString()})

	acctCreds, err := r.accounts.For(ctx, plan.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting account credentials", err.Error())
		return
	}

	unlock := r.s3Client.LockBucket(plan.BucketName.ValueString())
	defer unlock()

	if err := r.s3Client.PutBucketPolicy(ctx,
		acctCreds,
		plan.BucketName.ValueString(),
		plan.Policy.ValueString(),
	); err != nil {
		resp.Diagnostics.AddError("Error updating bucket policy", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BucketPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state BucketPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting bucket policy", map[string]any{"bucket": state.BucketName.ValueString()})

	acctCreds, err := r.accounts.For(ctx, state.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting account credentials", err.Error())
		return
	}

	unlock := r.s3Client.LockBucket(state.BucketName.ValueString())
	defer unlock()

	if err := r.s3Client.DeleteBucketPolicy(ctx,
		acctCreds,
		state.BucketName.ValueString(),
	); err != nil {
		resp.Diagnostics.AddError("Error deleting bucket policy", err.Error())
	}
}

func (r *BucketPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	creds.ImportByID(ctx, "bucket_name", req, resp)
}

// UpgradeState migrates v0 state, which held the account's access key pair,
// to account_name.
func (r *BucketPolicyResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	return creds.UpgradeFromAccountKeys(r, func() *client.AccountCredentialSource { return r.accounts })
}
