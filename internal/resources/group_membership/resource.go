package groupmembership

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
	_ resource.Resource                = &GroupMembershipResource{}
	_ resource.ResourceWithImportState = &GroupMembershipResource{}
)

type GroupMembershipResource struct {
	iamClient *client.IAMClient
}

func NewGroupMembershipResource() resource.Resource {
	return &GroupMembershipResource{}
}

func (r *GroupMembershipResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_membership"
}

func (r *GroupMembershipResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Attaches an IAM user to a group within an ARTESCA account.",
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
			"group_name": schema.StringAttribute{
				Description: "The IAM group to add the user to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validators.IAMName{MaxLength: 128, FieldName: "IAM group name"},
				},
			},
			"username": schema.StringAttribute{
				Description: "The IAM user to attach to the group.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validators.IAMUsername(),
				},
			},
		},
	}
}

func (r *GroupMembershipResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *GroupMembershipResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan GroupMembershipResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Adding user to group", map[string]any{
		"group":    plan.GroupName.ValueString(),
		"username": plan.Username.ValueString(),
	})

	err := r.iamClient.AddUserToGroup(ctx,
		plan.AccountAccessKey.ValueString(),
		plan.AccountSecretKey.ValueString(),
		plan.GroupName.ValueString(),
		plan.Username.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error adding user to group", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GroupMembershipResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state GroupMembershipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groups, err := r.iamClient.ListGroupsForUser(ctx,
		creds.Resolve(state.AccountAccessKey, creds.EnvAccessKey),
		creds.Resolve(state.AccountSecretKey, creds.EnvSecretKey),
		state.Username.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error reading group membership", err.Error())
		return
	}

	if slices.Contains(groups, state.GroupName.ValueString()) {
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *GroupMembershipResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "Group memberships cannot be updated in place.")
}

func (r *GroupMembershipResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state GroupMembershipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.iamClient.RemoveUserFromGroup(ctx,
		creds.Resolve(state.AccountAccessKey, creds.EnvAccessKey),
		creds.Resolve(state.AccountSecretKey, creds.EnvSecretKey),
		state.GroupName.ValueString(),
		state.Username.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error removing user from group", err.Error())
		return
	}
}

func (r *GroupMembershipResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format group_name/username, got %q", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("username"), parts[1])...)
	creds.WriteImport(ctx, &resp.State, &resp.Diagnostics)
}
