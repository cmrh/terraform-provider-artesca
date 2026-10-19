package user

import (
	"context"
	"fmt"
	"strings"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/cmrh/terraform-provider-artesca/internal/creds"
	validators "github.com/cmrh/terraform-provider-artesca/internal/validators"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                 = &UserResource{}
	_ resource.ResourceWithImportState  = &UserResource{}
	_ resource.ResourceWithUpgradeState = &UserResource{}
)

type UserResource struct {
	accounts  *client.AccountCredentialSource
	iamClient *client.IAMClient
}

func NewUserResource() resource.Resource {
	return &UserResource{}
}

func (r *UserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *UserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "Manages an IAM user within an ARTESCA account.",
		Attributes: map[string]schema.Attribute{
			creds.AttrAccountName: creds.ResourceAttribute(),
			"force_destroy": schema.BoolAttribute{
				Description: "Detach the user's policies and remove it from its groups when it is destroyed, including ones not managed by this configuration. Defaults to false. Set it and apply before destroying.",
				Optional:    true,
			},
			"username": schema.StringAttribute{
				Description: "The name of the IAM user. Must be 1–64 characters, alphanumeric and +=,.@-.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validators.IAMUsername(),
				},
			},
			"user_id": schema.StringAttribute{
				Description: "The unique ID of the user.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"arn": schema.StringAttribute{
				Description: "The ARN of the user.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"path": schema.StringAttribute{
				Description: "The path of the user.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *UserResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *UserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan UserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	acctCreds, err := r.accounts.For(ctx, plan.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting account credentials", err.Error())
		return
	}

	username := plan.Username.ValueString()

	tflog.Debug(ctx, "Creating IAM user", map[string]any{"username": username})

	user, err := r.iamClient.CreateUser(ctx, acctCreds, username)
	if err != nil {
		resp.Diagnostics.AddError("Error creating IAM user", err.Error())
		return
	}

	plan.UserID = types.StringValue(user.UserId)
	plan.ARN = types.StringValue(user.Arn)
	plan.Path = types.StringValue(user.Path)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state UserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	acctCreds, err := r.accounts.For(ctx, state.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting account credentials", err.Error())
		return
	}

	username := state.Username.ValueString()

	user, err := r.iamClient.GetUser(ctx, acctCreds, username)
	if err != nil {
		resp.Diagnostics.AddError("Error reading IAM user", err.Error())
		return
	}
	if user == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.UserID = types.StringValue(user.UserId)
	state.ARN = types.StringValue(user.Arn)
	state.Path = types.StringValue(user.Path)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *UserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Only force_destroy changes in place, and it is local; the other
	// attributes force replacement.
	var plan UserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state UserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	acctCreds, err := r.accounts.For(ctx, state.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting account credentials", err.Error())
		return
	}

	username := state.Username.ValueString()

	tflog.Debug(ctx, "Deleting IAM user", map[string]any{"username": username})

	// ARTESCA removes a user's access keys and inline policies with it;
	// attached policies and group memberships block the delete.
	if state.ForceDestroy.ValueBool() {
		if err := r.removeUserDependents(ctx, acctCreds, username); err != nil {
			resp.Diagnostics.AddError("Error removing the user's policies and groups", err.Error())
			return
		}
	}

	err = r.iamClient.DeleteUser(ctx, acctCreds, username)
	if err != nil && strings.Contains(err.Error(), "DeleteConflict") {
		policies, _ := r.iamClient.ListAttachedUserPolicies(ctx, acctCreds, username)
		groups, _ := r.iamClient.ListGroupsForUser(ctx, acctCreds, username)
		resp.Diagnostics.AddError("User has dependents",
			fmt.Sprintf("User %q still has attached policies %v and group memberships %v. Remove them, or set force_destroy = true and apply before destroying.", username, policies, groups))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error deleting IAM user", err.Error())
		return
	}
}

func (r *UserResource) removeUserDependents(ctx context.Context, acctCreds client.Credentials, username string) error {
	policies, err := r.iamClient.ListAttachedUserPolicies(ctx, acctCreds, username)
	if err != nil {
		return err
	}
	for _, arn := range policies {
		if err := r.iamClient.DetachUserPolicy(ctx, acctCreds, username, arn); err != nil {
			return err
		}
	}
	groups, err := r.iamClient.ListGroupsForUser(ctx, acctCreds, username)
	if err != nil {
		return err
	}
	for _, g := range groups {
		if err := r.iamClient.RemoveUserFromGroup(ctx, acctCreds, g, username); err != nil {
			return err
		}
	}
	return nil
}

func (r *UserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	creds.ImportByID(ctx, "username", req, resp)
}

// UpgradeState migrates v0 state, which held the account's access key pair,
// to account_name.
func (r *UserResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	return creds.UpgradeFromAccountKeys(r, func() *client.AccountCredentialSource { return r.accounts })
}
