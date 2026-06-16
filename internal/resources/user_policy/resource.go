package userpolicy

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
	_ resource.Resource                = &UserPolicyResource{}
	_ resource.ResourceWithImportState = &UserPolicyResource{}
)

type UserPolicyResource struct {
	iamClient *client.IAMClient
}

func NewUserPolicyResource() resource.Resource {
	return &UserPolicyResource{}
}

func (r *UserPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_policy"
}

func (r *UserPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Attaches an inline IAM policy to a user within an ARTESCA account.",
		Attributes: map[string]schema.Attribute{
			"account_access_key": schema.StringAttribute{
				Description: "The access key of the account this user belongs to.",
				Required:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"account_secret_key": schema.StringAttribute{
				Description: "The secret key of the account this user belongs to.",
				Required:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"username": schema.StringAttribute{
				Description: "The IAM user to attach the policy to. Must be 1–64 characters, alphanumeric and +=,.@-.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validators.IAMUsername(),
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
				Validators: []validator.String{
					validators.JSONDocument{},
				},
			},
		},
	}
}

func (r *UserPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *UserPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan UserPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Attaching user policy", map[string]any{
		"username":    plan.Username.ValueString(),
		"policy_name": plan.PolicyName.ValueString(),
	})

	err := r.iamClient.PutUserPolicy(ctx,
		plan.AccountAccessKey.ValueString(),
		plan.AccountSecretKey.ValueString(),
		plan.Username.ValueString(),
		plan.PolicyName.ValueString(),
		plan.PolicyDocument.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error attaching user policy", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state UserPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policyDoc, err := r.iamClient.GetUserPolicy(ctx,
		creds.Resolve(state.AccountAccessKey, creds.EnvAccessKey),
		creds.Resolve(state.AccountSecretKey, creds.EnvSecretKey),
		state.Username.ValueString(),
		state.PolicyName.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error reading user policy", err.Error())
		return
	}
	if policyDoc == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	// On import, state.PolicyDocument is empty — populate from the API so
	// the resource isn't half-constructed. Otherwise keep state as canonical
	// to avoid JSON-formatting drift.
	if state.PolicyDocument.IsNull() || state.PolicyDocument.ValueString() == "" {
		state.PolicyDocument = types.StringValue(policyDoc)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *UserPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Only policy_document can change (all other fields are ForceNew).
	var plan UserPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating user policy", map[string]any{
		"username":    plan.Username.ValueString(),
		"policy_name": plan.PolicyName.ValueString(),
	})

	err := r.iamClient.PutUserPolicy(ctx,
		plan.AccountAccessKey.ValueString(),
		plan.AccountSecretKey.ValueString(),
		plan.Username.ValueString(),
		plan.PolicyName.ValueString(),
		plan.PolicyDocument.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error updating user policy", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state UserPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting user policy", map[string]any{
		"username":    state.Username.ValueString(),
		"policy_name": state.PolicyName.ValueString(),
	})

	err := r.iamClient.DeleteUserPolicy(ctx,
		creds.Resolve(state.AccountAccessKey, creds.EnvAccessKey),
		creds.Resolve(state.AccountSecretKey, creds.EnvSecretKey),
		state.Username.ValueString(),
		state.PolicyName.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting user policy", err.Error())
		return
	}
}

func (r *UserPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format username/policy_name, got %q", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("username"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("policy_name"), parts[1])...)
	creds.WriteImport(ctx, &resp.State, &resp.Diagnostics)
}
