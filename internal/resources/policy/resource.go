package policy

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

var _ resource.Resource = &PolicyResource{}

type PolicyResource struct {
	iamClient *client.IAMClient
}

func NewPolicyResource() resource.Resource {
	return &PolicyResource{}
}

func (r *PolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy"
}

func (r *PolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an IAM managed policy within an ARTESCA account. " +
			"Managed policies can be attached to users, groups, and roles via the " +
			"*_policy_attachment resources.",
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
				Description: "The JSON policy document.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validators.JSONDocument{},
				},
			},
			"description": schema.StringAttribute{
				Description: "An optional description for the policy.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"policy_id": schema.StringAttribute{
				Description: "The unique ID of the policy.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"arn": schema.StringAttribute{
				Description: "The ARN of the policy. Use this when attaching the policy to users, groups, or roles.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"path": schema.StringAttribute{
				Description: "The path of the policy.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"default_version_id": schema.StringAttribute{
				Description: "The default policy version ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *PolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating managed policy", map[string]any{"name": plan.Name.ValueString()})

	pol, err := r.iamClient.CreatePolicy(ctx,
		plan.AccountAccessKey.ValueString(),
		plan.AccountSecretKey.ValueString(),
		plan.Name.ValueString(),
		plan.PolicyDocument.ValueString(),
		plan.Description.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error creating managed policy", err.Error())
		return
	}

	plan.PolicyID = types.StringValue(pol.PolicyId)
	plan.ARN = types.StringValue(pol.Arn)
	plan.Path = types.StringValue(pol.Path)
	plan.DefaultVersionID = types.StringValue(pol.DefaultVersionId)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pol, err := r.iamClient.GetPolicy(ctx,
		state.AccountAccessKey.ValueString(),
		state.AccountSecretKey.ValueString(),
		state.ARN.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error reading managed policy", err.Error())
		return
	}
	if pol == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.PolicyID = types.StringValue(pol.PolicyId)
	state.Path = types.StringValue(pol.Path)
	state.DefaultVersionID = types.StringValue(pol.DefaultVersionId)
	// policy_document is preserved from state — round-tripping through the API
	// can re-format JSON whitespace and we don't want spurious drift.

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *PolicyResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "Managed policies cannot be updated in place — all fields are ForceNew.")
}

func (r *PolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting managed policy", map[string]any{"arn": state.ARN.ValueString()})

	err := r.iamClient.DeletePolicy(ctx,
		state.AccountAccessKey.ValueString(),
		state.AccountSecretKey.ValueString(),
		state.ARN.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting managed policy", err.Error())
		return
	}
}
