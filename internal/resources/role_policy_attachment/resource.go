package rolepolicyattachment

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
	_ resource.Resource                 = &RolePolicyAttachmentResource{}
	_ resource.ResourceWithImportState  = &RolePolicyAttachmentResource{}
	_ resource.ResourceWithUpgradeState = &RolePolicyAttachmentResource{}
)

type RolePolicyAttachmentResource struct {
	accounts  *client.AccountCredentialSource
	iamClient *client.IAMClient
}

func NewRolePolicyAttachmentResource() resource.Resource {
	return &RolePolicyAttachmentResource{}
}

func (r *RolePolicyAttachmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role_policy_attachment"
}

func (r *RolePolicyAttachmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version: 1,
		Description: "Attaches a managed IAM policy to a role within an ARTESCA account. " +
			"This is the only way to grant permissions to a role — ARTESCA does not implement " +
			"inline role policies.",
		Attributes: map[string]schema.Attribute{
			creds.AttrAccountName: creds.ResourceAttribute(),
			"role_name": schema.StringAttribute{
				Description: "The IAM role to attach the policy to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validators.IAMName{MaxLength: 64, FieldName: "IAM role name"},
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

func (r *RolePolicyAttachmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RolePolicyAttachmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RolePolicyAttachmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Attaching managed policy to role", map[string]any{
		"role":       plan.RoleName.ValueString(),
		"policy_arn": plan.PolicyArn.ValueString(),
	})

	acctCreds, err := r.accounts.For(ctx, plan.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting account credentials", err.Error())
		return
	}

	err = r.iamClient.AttachRolePolicy(ctx,
		acctCreds,
		plan.RoleName.ValueString(),
		plan.PolicyArn.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error attaching policy to role", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RolePolicyAttachmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RolePolicyAttachmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	acctCreds, err := r.accounts.For(ctx, state.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting account credentials", err.Error())
		return
	}

	arns, err := r.iamClient.ListAttachedRolePolicies(ctx,
		acctCreds,
		state.RoleName.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error listing attached role policies", err.Error())
		return
	}

	if slices.Contains(arns, state.PolicyArn.ValueString()) {
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *RolePolicyAttachmentResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "Role policy attachments cannot be updated in place.")
}

func (r *RolePolicyAttachmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RolePolicyAttachmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	acctCreds, err := r.accounts.For(ctx, state.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting account credentials", err.Error())
		return
	}

	err = r.iamClient.DetachRolePolicy(ctx,
		acctCreds,
		state.RoleName.ValueString(),
		state.PolicyArn.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error detaching policy from role", err.Error())
		return
	}
}

func (r *RolePolicyAttachmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	rest, ok := creds.ImportAccount(ctx, req, resp)
	if !ok {
		return
	}
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 || parts[0] == "" || !strings.HasPrefix(parts[1], "arn:") {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format <account_name>/role_name/policy_arn, got %q", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("role_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("policy_arn"), parts[1])...)
}

// UpgradeState migrates v0 state, which held the account's access key pair,
// to account_name.
func (r *RolePolicyAttachmentResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	return creds.UpgradeFromAccountKeys(r, func() *client.AccountCredentialSource { return r.accounts })
}
