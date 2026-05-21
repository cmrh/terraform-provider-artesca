package role

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/scality/terraform-provider-scality-artesca/internal/client"
	"github.com/scality/terraform-provider-scality-artesca/internal/validators"
)

var _ resource.Resource = &RoleResource{}

type RoleResource struct {
	iamClient *client.IAMClient
}

func NewRoleResource() resource.Resource {
	return &RoleResource{}
}

func (r *RoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (r *RoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an IAM role within an ARTESCA account. The role's trust policy " +
			"(assume_role_policy_document) cannot be updated in place — changing it forces replacement.",
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
			"name": schema.StringAttribute{
				Description: "The name of the role. Must be 1–64 characters, alphanumeric and +=,.@-.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validators.IAMName{MaxLength: 64, FieldName: "IAM role name"},
				},
			},
			"assume_role_policy_document": schema.StringAttribute{
				Description: "The trust policy JSON. ARTESCA does not implement UpdateAssumeRolePolicy, " +
					"so changing this forces the role to be replaced.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validators.JSONDocument{},
				},
			},
			"description": schema.StringAttribute{
				Description: "An optional description for the role.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role_id": schema.StringAttribute{
				Description: "The unique ID of the role.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"arn": schema.StringAttribute{
				Description: "The ARN of the role.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"path": schema.StringAttribute{
				Description: "The path of the role.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *RoleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating IAM role", map[string]any{"name": plan.Name.ValueString()})

	role, err := r.iamClient.CreateRole(ctx,
		plan.AccountAccessKey.ValueString(),
		plan.AccountSecretKey.ValueString(),
		plan.Name.ValueString(),
		plan.AssumeRolePolicyDocument.ValueString(),
		plan.Description.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error creating IAM role", err.Error())
		return
	}

	plan.RoleID = types.StringValue(role.RoleId)
	plan.ARN = types.StringValue(role.Arn)
	plan.Path = types.StringValue(role.Path)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := r.iamClient.GetRole(ctx,
		state.AccountAccessKey.ValueString(),
		state.AccountSecretKey.ValueString(),
		state.Name.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error reading IAM role", err.Error())
		return
	}
	if role == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.RoleID = types.StringValue(role.RoleId)
	state.ARN = types.StringValue(role.Arn)
	state.Path = types.StringValue(role.Path)
	// assume_role_policy_document and description are preserved from state — the
	// API response may re-format JSON whitespace and we don't want spurious drift.

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RoleResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "IAM roles cannot be updated in place — all fields are ForceNew.")
}

func (r *RoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting IAM role", map[string]any{"name": state.Name.ValueString()})

	err := r.iamClient.DeleteRole(ctx,
		state.AccountAccessKey.ValueString(),
		state.AccountSecretKey.ValueString(),
		state.Name.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting IAM role", err.Error())
		return
	}
}
