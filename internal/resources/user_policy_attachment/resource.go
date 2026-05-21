package userpolicyattachment

import (
	"context"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/scality/terraform-provider-scality-artesca/internal/client"
	"github.com/scality/terraform-provider-scality-artesca/internal/validators"
)

var _ resource.Resource = &UserPolicyAttachmentResource{}

type UserPolicyAttachmentResource struct {
	iamClient *client.IAMClient
}

func NewUserPolicyAttachmentResource() resource.Resource {
	return &UserPolicyAttachmentResource{}
}

func (r *UserPolicyAttachmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_policy_attachment"
}

func (r *UserPolicyAttachmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Attaches a managed IAM policy to a user within an ARTESCA account.",
		Attributes: map[string]schema.Attribute{
			"account_access_key": schema.StringAttribute{
				Description: "The access key of the account.",
				Required:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"account_secret_key": schema.StringAttribute{
				Description: "The secret key of the account.",
				Required:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"username": schema.StringAttribute{
				Description: "The IAM user to attach the policy to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validators.IAMUsername(),
				},
			},
			"policy_arn": schema.StringAttribute{
				Description: "The ARN of the managed policy to attach.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *UserPolicyAttachmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.iamClient = providerData.IAM
}

func (r *UserPolicyAttachmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan UserPolicyAttachmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Attaching managed policy to user", map[string]any{
		"username":   plan.Username.ValueString(),
		"policy_arn": plan.PolicyArn.ValueString(),
	})

	err := r.iamClient.AttachUserPolicy(ctx,
		plan.AccountAccessKey.ValueString(),
		plan.AccountSecretKey.ValueString(),
		plan.Username.ValueString(),
		plan.PolicyArn.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error attaching policy to user", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserPolicyAttachmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state UserPolicyAttachmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	arns, err := r.iamClient.ListAttachedUserPolicies(ctx,
		state.AccountAccessKey.ValueString(),
		state.AccountSecretKey.ValueString(),
		state.Username.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error listing attached user policies", err.Error())
		return
	}

	if slices.Contains(arns, state.PolicyArn.ValueString()) {
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *UserPolicyAttachmentResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "User policy attachments cannot be updated in place.")
}

func (r *UserPolicyAttachmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state UserPolicyAttachmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.iamClient.DetachUserPolicy(ctx,
		state.AccountAccessKey.ValueString(),
		state.AccountSecretKey.ValueString(),
		state.Username.ValueString(),
		state.PolicyArn.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error detaching policy from user", err.Error())
		return
	}
}
