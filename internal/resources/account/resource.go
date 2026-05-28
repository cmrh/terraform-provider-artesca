package account

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
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

var (
	_ resource.Resource                = &AccountResource{}
	_ resource.ResourceWithImportState = &AccountResource{}
)

type AccountResource struct {
	client *client.ManagementClient
}

func NewAccountResource() resource.Resource {
	return &AccountResource{}
}

func (r *AccountResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_account"
}

func (r *AccountResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an ARTESCA account (S3 user) via the management API.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The account name. Must be 1–128 characters, ASCII alphanumeric and hyphens only.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validators.AccountName{},
				},
			},
			"email": schema.StringAttribute{
				Description: "The email address associated with the account. Cannot be changed after creation.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validators.Email{},
				},
			},
			"access_key": schema.StringAttribute{
				Description: "The access key for the account. Generated on creation.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"secret_key": schema.StringAttribute{
				Description: "The secret key for the account. Generated on creation.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"arn": schema.StringAttribute{
				Description: "The ARN of the account.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"canonical_id": schema.StringAttribute{
				Description: "The canonical ID of the account.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"id": schema.StringAttribute{
				Description: "The unique ID of the account.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *AccountResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.client = providerData.Management
}

func (r *AccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating account", map[string]any{"name": plan.Name.ValueString()})

	user, err := r.client.CreateAccount(ctx, plan.Name.ValueString(), plan.Email.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error creating account", err.Error())
		return
	}

	apiUserToModel(user, &plan)

	// Save state now so Terraform can track the account even if key generation fails.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If the create response didn't include keys, generate them.
	if plan.AccessKey.IsNull() || plan.AccessKey.ValueString() == "" {
		keyUser, err := r.client.GenerateAccountKey(ctx, plan.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error generating account key",
				fmt.Sprintf("Account was created but key generation failed: %s", err))
			return
		}
		if keyUser.AccessKey != "" {
			plan.AccessKey = types.StringValue(keyUser.AccessKey)
		}
		if keyUser.SecretKey != "" {
			plan.SecretKey = types.StringValue(keyUser.SecretKey)
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	}

}

func (r *AccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := r.client.GetAccount(ctx, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading account", err.Error())
		return
	}
	if user == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Preserve sensitive fields from state since the API may not return them.
	accessKey := state.AccessKey
	secretKey := state.SecretKey

	apiUserToModel(user, &state)

	// Restore sensitive fields if the API returned empty values.
	if state.AccessKey.IsNull() || state.AccessKey.ValueString() == "" {
		state.AccessKey = accessKey
	}
	if state.SecretKey.IsNull() || state.SecretKey.ValueString() == "" {
		state.SecretKey = secretKey
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Name is ForceNew, so the only updatable field is email.
	// The management API may not support email updates directly.
	// For now, just sync state.
	var plan AccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve computed fields from state.
	var state AccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.AccessKey = state.AccessKey
	plan.SecretKey = state.SecretKey
	plan.ARN = state.ARN
	plan.CanonicalID = state.CanonicalID
	plan.ID = state.ID

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting account", map[string]any{"name": state.Name.ValueString()})

	err := r.client.DeleteAccount(ctx, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting account", err.Error())
		return
	}
}

func (r *AccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

func apiUserToModel(user *client.User, model *AccountResourceModel) {
	if user.AccountName != "" {
		model.Name = types.StringValue(user.AccountName)
	} else if user.UserName != "" {
		model.Name = types.StringValue(user.UserName)
	}

	if user.Email != "" {
		model.Email = types.StringValue(user.Email)
	}
	if user.AccessKey != "" {
		model.AccessKey = types.StringValue(user.AccessKey)
	}
	if user.SecretKey != "" {
		model.SecretKey = types.StringValue(user.SecretKey)
	}
	if user.ARN != "" {
		model.ARN = types.StringValue(user.ARN)
	}
	if user.CanonicalID != "" {
		model.CanonicalID = types.StringValue(user.CanonicalID)
	}
	if user.ID != "" {
		model.ID = types.StringValue(user.ID)
	}
}
