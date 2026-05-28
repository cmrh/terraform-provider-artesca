package bucketpolicy

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/scality/terraform-provider-artesca/internal/client"
	validators "github.com/scality/terraform-provider-artesca/internal/validators"
)

var _ resource.Resource = &BucketPolicyResource{}

type BucketPolicyResource struct {
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
		Description: "Attaches an S3 bucket policy to an ARTESCA bucket. ARTESCA validates the policy server-side; Resource ARNs that don't match the bucket are rejected with MalformedPolicy.",
		Attributes: map[string]schema.Attribute{
			"account_access_key": schema.StringAttribute{
				Description: "The access key of the account that owns the bucket.",
				Required:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"account_secret_key": schema.StringAttribute{
				Description: "The secret key of the account that owns the bucket.",
				Required:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
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
					policyEquivalenceModifier{},
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
}

func (r *BucketPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BucketPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Attaching bucket policy", map[string]any{"bucket": plan.BucketName.ValueString()})

	if err := r.s3Client.PutBucketPolicy(ctx,
		plan.AccountAccessKey.ValueString(),
		plan.AccountSecretKey.ValueString(),
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

	remote, err := r.s3Client.GetBucketPolicy(ctx,
		state.AccountAccessKey.ValueString(),
		state.AccountSecretKey.ValueString(),
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

	if !jsonEquivalent(state.Policy.ValueString(), remote) {
		state.Policy = types.StringValue(remote)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *BucketPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan BucketPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating bucket policy", map[string]any{"bucket": plan.BucketName.ValueString()})

	if err := r.s3Client.PutBucketPolicy(ctx,
		plan.AccountAccessKey.ValueString(),
		plan.AccountSecretKey.ValueString(),
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

	if err := r.s3Client.DeleteBucketPolicy(ctx,
		state.AccountAccessKey.ValueString(),
		state.AccountSecretKey.ValueString(),
		state.BucketName.ValueString(),
	); err != nil {
		resp.Diagnostics.AddError("Error deleting bucket policy", err.Error())
	}
}

// jsonEquivalent reports whether two JSON documents represent the same value,
// ignoring whitespace and key ordering. Returns false on parse error.
func jsonEquivalent(a, b string) bool {
	var av, bv any
	if err := json.Unmarshal([]byte(a), &av); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &bv); err != nil {
		return false
	}
	return reflect.DeepEqual(av, bv)
}
