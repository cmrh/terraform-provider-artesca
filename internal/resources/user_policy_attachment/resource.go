package userpolicyattachment

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/cmrh/terraform-provider-artesca/internal/creds"
	"github.com/cmrh/terraform-provider-artesca/internal/validators"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &UserPolicyAttachmentResource{}
	_ resource.ResourceWithImportState = &UserPolicyAttachmentResource{}
)

type UserPolicyAttachmentResource struct {
	accounts  *client.AccountCredentialSource
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
			creds.AttrAccountName: creds.ResourceAttribute(),
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
	r.accounts = providerData.Accounts
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

	acctCreds, err := r.accounts.For(ctx, plan.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting account credentials", err.Error())
		return
	}

	err = r.iamClient.AttachUserPolicy(ctx,
		acctCreds,
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

	acctCreds, err := r.accounts.For(ctx, state.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting account credentials", err.Error())
		return
	}

	arns, err := r.iamClient.ListAttachedUserPolicies(ctx,
		acctCreds,
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

	acctCreds, err := r.accounts.For(ctx, state.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting account credentials", err.Error())
		return
	}

	err = r.iamClient.DetachUserPolicy(ctx,
		acctCreds,
		state.Username.ValueString(),
		state.PolicyArn.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error detaching policy from user", err.Error())
		return
	}
}

func (r *UserPolicyAttachmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	rest, ok := creds.ImportAccount(ctx, req, resp)
	if !ok {
		return
	}
	// Format: <account_name>/username/policy_arn. ARN contains slashes but starts
	// with "arn:", so SplitN with n=2 on the rest gives username and the full ARN.
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 || parts[0] == "" || !strings.HasPrefix(parts[1], "arn:") {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format <account_name>/username/policy_arn, got %q", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("username"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("policy_arn"), parts[1])...)
}
