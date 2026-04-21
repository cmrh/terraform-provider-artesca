package useraccesskey

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
	validators "github.com/scality/terraform-provider-scality-artesca/internal/validators"
)

var _ resource.Resource = &UserAccessKeyResource{}

type UserAccessKeyResource struct {
	iamClient *client.IAMClient
}

func NewUserAccessKeyResource() resource.Resource {
	return &UserAccessKeyResource{}
}

func (r *UserAccessKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_access_key"
}

func (r *UserAccessKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates an IAM access key for a user within an ARTESCA account.",
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
				Description: "The IAM user to create the access key for. Must be 1–64 characters, alphanumeric and +=,.@-.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validators.IAMUsername(),
				},
			},
			"access_key_id": schema.StringAttribute{
				Description: "The generated access key ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"secret_access_key": schema.StringAttribute{
				Description: "The generated secret access key. Only available at creation time.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Description: "The status of the access key (Active or Inactive).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *UserAccessKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *UserAccessKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan UserAccessKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating access key for user", map[string]any{
		"username": plan.Username.ValueString(),
	})

	ak, err := r.iamClient.CreateAccessKey(ctx,
		plan.AccountAccessKey.ValueString(),
		plan.AccountSecretKey.ValueString(),
		plan.Username.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error creating access key", err.Error())
		return
	}

	plan.AccessKeyID = types.StringValue(ak.AccessKeyId)
	plan.SecretAccessKey = types.StringValue(ak.SecretAccessKey)
	plan.Status = types.StringValue(ak.Status)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserAccessKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state UserAccessKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	keys, err := r.iamClient.ListAccessKeys(ctx,
		state.AccountAccessKey.ValueString(),
		state.AccountSecretKey.ValueString(),
		state.Username.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error listing access keys", err.Error())
		return
	}

	// Find our key in the list.
	found := false
	for _, k := range keys {
		if k.AccessKeyId == state.AccessKeyID.ValueString() {
			state.Status = types.StringValue(k.Status)
			found = true
			break
		}
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	// secret_access_key is preserved from state (only available at creation).
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *UserAccessKeyResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All fields are ForceNew, so Update should never be called.
	resp.Diagnostics.AddError("Update not supported", "Access keys cannot be updated in place.")
}

func (r *UserAccessKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state UserAccessKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting access key", map[string]any{
		"username":      state.Username.ValueString(),
		"access_key_id": state.AccessKeyID.ValueString(),
	})

	err := r.iamClient.DeleteAccessKey(ctx,
		state.AccountAccessKey.ValueString(),
		state.AccountSecretKey.ValueString(),
		state.Username.ValueString(),
		state.AccessKeyID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting access key", err.Error())
		return
	}
}
