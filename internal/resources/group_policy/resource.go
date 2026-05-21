package grouppolicy

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/scality/terraform-provider-scality-artesca/internal/client"
	"github.com/scality/terraform-provider-scality-artesca/internal/validators"
)

var _ resource.Resource = &GroupPolicyResource{}

type GroupPolicyResource struct {
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
		Description: "Attaches an inline IAM policy to a group within an ARTESCA account.",
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

	err := r.iamClient.PutGroupPolicy(ctx,
		plan.AccountAccessKey.ValueString(),
		plan.AccountSecretKey.ValueString(),
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

	doc, err := r.iamClient.GetGroupPolicy(ctx,
		state.AccountAccessKey.ValueString(),
		state.AccountSecretKey.ValueString(),
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

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GroupPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan GroupPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.iamClient.PutGroupPolicy(ctx,
		plan.AccountAccessKey.ValueString(),
		plan.AccountSecretKey.ValueString(),
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

	err := r.iamClient.DeleteGroupPolicy(ctx,
		state.AccountAccessKey.ValueString(),
		state.AccountSecretKey.ValueString(),
		state.GroupName.ValueString(),
		state.PolicyName.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting group policy", err.Error())
		return
	}
}
