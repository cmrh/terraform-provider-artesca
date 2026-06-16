package grouppolicyattachment

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/scality/terraform-provider-artesca/internal/client"
	"github.com/scality/terraform-provider-artesca/internal/creds"
	"github.com/scality/terraform-provider-artesca/internal/validators"
)

var (
	_ resource.Resource                = &GroupPolicyAttachmentResource{}
	_ resource.ResourceWithImportState = &GroupPolicyAttachmentResource{}
)

type GroupPolicyAttachmentResource struct {
	iamClient *client.IAMClient
}

func NewGroupPolicyAttachmentResource() resource.Resource {
	return &GroupPolicyAttachmentResource{}
}

func (r *GroupPolicyAttachmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_policy_attachment"
}

func (r *GroupPolicyAttachmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Attaches a managed IAM policy to a group within an ARTESCA account.",
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

func (r *GroupPolicyAttachmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *GroupPolicyAttachmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan GroupPolicyAttachmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Attaching managed policy to group", map[string]any{
		"group":      plan.GroupName.ValueString(),
		"policy_arn": plan.PolicyArn.ValueString(),
	})

	err := r.iamClient.AttachGroupPolicy(ctx,
		plan.AccountAccessKey.ValueString(),
		plan.AccountSecretKey.ValueString(),
		plan.GroupName.ValueString(),
		plan.PolicyArn.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error attaching policy to group", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GroupPolicyAttachmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state GroupPolicyAttachmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	arns, err := r.iamClient.ListAttachedGroupPolicies(ctx,
		creds.Resolve(state.AccountAccessKey, creds.EnvAccessKey),
		creds.Resolve(state.AccountSecretKey, creds.EnvSecretKey),
		state.GroupName.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error listing attached group policies", err.Error())
		return
	}

	if slices.Contains(arns, state.PolicyArn.ValueString()) {
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *GroupPolicyAttachmentResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "Group policy attachments cannot be updated in place.")
}

func (r *GroupPolicyAttachmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state GroupPolicyAttachmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.iamClient.DetachGroupPolicy(ctx,
		creds.Resolve(state.AccountAccessKey, creds.EnvAccessKey),
		creds.Resolve(state.AccountSecretKey, creds.EnvSecretKey),
		state.GroupName.ValueString(),
		state.PolicyArn.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error detaching policy from group", err.Error())
		return
	}
}

func (r *GroupPolicyAttachmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || !strings.HasPrefix(parts[1], "arn:") {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format group_name/policy_arn, got %q", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("policy_arn"), parts[1])...)
	creds.WriteImport(ctx, &resp.State, &resp.Diagnostics)
}
