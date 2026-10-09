package grouppolicy

import (
	"context"
	"fmt"
	"strings"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/cmrh/terraform-provider-artesca/internal/creds"
	"github.com/cmrh/terraform-provider-artesca/internal/policydoc"
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
	_ resource.Resource                 = &GroupPolicyResource{}
	_ resource.ResourceWithImportState  = &GroupPolicyResource{}
	_ resource.ResourceWithUpgradeState = &GroupPolicyResource{}
)

type GroupPolicyResource struct {
	accounts  *client.AccountCredentialSource
	iamClient *client.IAMClient
}

func NewGroupPolicyResource() resource.Resource {
	return &GroupPolicyResource{}
}

func (r *GroupPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_policy"
}

func (r *GroupPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "Attaches an inline IAM policy to a group within an ARTESCA account.",
		Attributes: map[string]schema.Attribute{
			creds.AttrAccountName: creds.ResourceAttribute(),
			"group_name": schema.StringAttribute{
				Description: "The IAM group to attach the policy to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validators.IAMName{MaxLength: 128, FieldName: "IAM group name"},
				},
			},
			"policy_name": schema.StringAttribute{
				Description: "The name of the policy. Must be 1–128 characters, alphanumeric and +=,.@-.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validators.IAMPolicyName(),
				},
			},
			"policy_document": schema.StringAttribute{
				Description: "The JSON policy document. Can be provided via file(), jsonencode(), or as a raw JSON string.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					policydoc.EquivalenceModifier(),
				},
				Validators: []validator.String{
					validators.JSONDocument{},
				},
			},
		},
	}
}

func (r *GroupPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *GroupPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan GroupPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Attaching group policy", map[string]any{
		"group":  plan.GroupName.ValueString(),
		"policy": plan.PolicyName.ValueString(),
	})

	acctCreds, err := r.accounts.For(ctx, plan.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting account credentials", err.Error())
		return
	}

	err = r.iamClient.PutGroupPolicy(ctx,
		acctCreds,
		plan.GroupName.ValueString(),
		plan.PolicyName.ValueString(),
		plan.PolicyDocument.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error attaching group policy", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GroupPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state GroupPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	acctCreds, err := r.accounts.For(ctx, state.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting account credentials", err.Error())
		return
	}

	doc, err := r.iamClient.GetGroupPolicy(ctx,
		acctCreds,
		state.GroupName.ValueString(),
		state.PolicyName.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error reading group policy", err.Error())
		return
	}
	if doc == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	state.PolicyDocument = policydoc.Refresh(state.PolicyDocument, doc)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GroupPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan GroupPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	acctCreds, err := r.accounts.For(ctx, plan.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting account credentials", err.Error())
		return
	}

	err = r.iamClient.PutGroupPolicy(ctx,
		acctCreds,
		plan.GroupName.ValueString(),
		plan.PolicyName.ValueString(),
		plan.PolicyDocument.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error updating group policy", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GroupPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state GroupPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	acctCreds, err := r.accounts.For(ctx, state.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting account credentials", err.Error())
		return
	}

	err = r.iamClient.DeleteGroupPolicy(ctx,
		acctCreds,
		state.GroupName.ValueString(),
		state.PolicyName.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting group policy", err.Error())
		return
	}
}

func (r *GroupPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	rest, ok := creds.ImportAccount(ctx, req, resp)
	if !ok {
		return
	}
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format <account_name>/group_name/policy_name, got %q", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("policy_name"), parts[1])...)
}

// UpgradeState migrates v0 state, which held the account's access key pair,
// to account_name.
func (r *GroupPolicyResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	return creds.UpgradeFromAccountKeys(r, func() *client.AccountCredentialSource { return r.accounts })
}
