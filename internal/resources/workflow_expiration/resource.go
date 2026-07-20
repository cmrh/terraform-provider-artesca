package workflowexpiration

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"

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
	_ resource.Resource                = &WorkflowExpirationResource{}
	_ resource.ResourceWithImportState = &WorkflowExpirationResource{}
)

type WorkflowExpirationResource struct {
	s3 *client.S3Client
}

func NewWorkflowExpirationResource() resource.Resource {
	return &WorkflowExpirationResource{}
}

func (r *WorkflowExpirationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket_workflow_expiration"
}

func (r *WorkflowExpirationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a bucket expiration lifecycle rule in ARTESCA via the S3 API.",
		Attributes: map[string]schema.Attribute{
			"account_access_key": schema.StringAttribute{
				Description: "The access key for the account that owns the bucket.",
				Required:    true,
				Sensitive:   true,
			},
			"account_secret_key": schema.StringAttribute{
				Description: "The secret key for the account that owns the bucket.",
				Required:    true,
				Sensitive:   true,
			},
			"bucket_name": schema.StringAttribute{
				Description: "The name of the bucket. Must be 3–63 characters, lowercase letters, numbers, hyphens, and periods.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validators.BucketName{},
				},
			},
			"rule_id": schema.StringAttribute{
				Description: "The lifecycle rule ID. Auto-generated if not set.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the lifecycle rule is enabled.",
				Required:    true,
			},
			"current_version_trigger_delay_days": schema.Int64Attribute{
				Description: "Number of days after which current version objects expire.",
				Required:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"filter": schema.SingleNestedBlock{
				Description: "Filter to scope which objects this rule applies to.",
				Attributes: map[string]schema.Attribute{
					"object_key_prefix": schema.StringAttribute{
						Description: "Object key prefix filter.",
						Optional:    true,
					},
				},
			},
		},
	}
}

func (r *WorkflowExpirationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	if providerData.S3 == nil {
		resp.Diagnostics.AddError(
			"S3 Client Not Configured",
			"The s3_endpoint must be set in the provider configuration to use bucket workflow resources.",
		)
		return
	}
	r.s3 = providerData.S3
}

func (r *WorkflowExpirationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan WorkflowExpirationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := plan.AccountAccessKey.ValueString()
	sk := plan.AccountSecretKey.ValueString()
	bucket := plan.BucketName.ValueString()

	ruleID := plan.RuleID.ValueString()
	if ruleID == "" {
		b := make([]byte, 16)
		_, _ = rand.Read(b)
		ruleID = fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	}

	newRule := modelToLifecycleRule(&plan, ruleID)

	r.s3.LockLifecycle()
	defer r.s3.UnlockLifecycle()

	existing, err := r.s3.GetBucketLifecycle(ctx, ak, sk, bucket)
	if err != nil {
		resp.Diagnostics.AddError("Error reading existing lifecycle rules", err.Error())
		return
	}

	rules := append(existing, newRule)

	tflog.Debug(ctx, "Creating expiration lifecycle rule", map[string]any{"bucket": bucket, "rule_id": ruleID})

	if err := r.s3.PutBucketLifecycle(ctx, ak, sk, bucket, rules); err != nil {
		resp.Diagnostics.AddError("Error creating expiration lifecycle rule", err.Error())
		return
	}

	plan.RuleID = types.StringValue(ruleID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WorkflowExpirationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state WorkflowExpirationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := creds.Resolve(state.AccountAccessKey, creds.EnvAccessKey)
	sk := creds.Resolve(state.AccountSecretKey, creds.EnvSecretKey)
	bucket := state.BucketName.ValueString()
	ruleID := state.RuleID.ValueString()

	rules, err := r.s3.GetBucketLifecycle(ctx, ak, sk, bucket)
	if err != nil {
		resp.Diagnostics.AddError("Error reading lifecycle rules", err.Error())
		return
	}

	var found *client.LifecycleRule
	for _, rule := range rules {
		if rule.ID == ruleID {
			found = &rule
			break
		}
	}

	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	lifecycleRuleToModel(found, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *WorkflowExpirationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan WorkflowExpirationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := plan.AccountAccessKey.ValueString()
	sk := plan.AccountSecretKey.ValueString()
	bucket := plan.BucketName.ValueString()
	ruleID := plan.RuleID.ValueString()

	r.s3.LockLifecycle()
	defer r.s3.UnlockLifecycle()

	existing, err := r.s3.GetBucketLifecycle(ctx, ak, sk, bucket)
	if err != nil {
		resp.Diagnostics.AddError("Error reading existing lifecycle rules", err.Error())
		return
	}

	updated := modelToLifecycleRule(&plan, ruleID)
	rules := make([]client.LifecycleRule, 0, len(existing))
	for _, rule := range existing {
		if rule.ID == ruleID {
			rules = append(rules, updated)
		} else {
			rules = append(rules, rule)
		}
	}

	tflog.Debug(ctx, "Updating expiration lifecycle rule", map[string]any{"rule_id": ruleID})

	if err := r.s3.PutBucketLifecycle(ctx, ak, sk, bucket, rules); err != nil {
		resp.Diagnostics.AddError("Error updating expiration lifecycle rule", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WorkflowExpirationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state WorkflowExpirationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ak := creds.Resolve(state.AccountAccessKey, creds.EnvAccessKey)
	sk := creds.Resolve(state.AccountSecretKey, creds.EnvSecretKey)
	bucket := state.BucketName.ValueString()
	ruleID := state.RuleID.ValueString()

	r.s3.LockLifecycle()
	defer r.s3.UnlockLifecycle()

	existing, err := r.s3.GetBucketLifecycle(ctx, ak, sk, bucket)
	if err != nil {
		resp.Diagnostics.AddError("Error reading existing lifecycle rules", err.Error())
		return
	}

	remaining := make([]client.LifecycleRule, 0, len(existing))
	for _, rule := range existing {
		if rule.ID != ruleID {
			remaining = append(remaining, rule)
		}
	}

	tflog.Debug(ctx, "Deleting expiration lifecycle rule", map[string]any{"rule_id": ruleID})

	if len(remaining) == 0 {
		if err := r.s3.DeleteBucketLifecycle(ctx, ak, sk, bucket); err != nil {
			resp.Diagnostics.AddError("Error deleting lifecycle configuration", err.Error())
			return
		}
	} else {
		if err := r.s3.PutBucketLifecycle(ctx, ak, sk, bucket, remaining); err != nil {
			resp.Diagnostics.AddError("Error updating lifecycle configuration", err.Error())
			return
		}
	}
}

func (r *WorkflowExpirationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format bucket_name/rule_id, got %q", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bucket_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("rule_id"), parts[1])...)
	creds.WriteImport(ctx, &resp.State, &resp.Diagnostics)
}

func modelToLifecycleRule(model *WorkflowExpirationResourceModel, ruleID string) client.LifecycleRule {
	status := "Disabled"
	if model.Enabled.ValueBool() {
		status = "Enabled"
	}

	rule := client.LifecycleRule{
		ID:             ruleID,
		Status:         status,
		ExpirationDays: int(model.CurrentVersionTriggerDelayDays.ValueInt64()),
	}

	if model.Filter != nil && !model.Filter.ObjectKeyPrefix.IsNull() {
		rule.Prefix = model.Filter.ObjectKeyPrefix.ValueString()
	}

	return rule
}

func lifecycleRuleToModel(rule *client.LifecycleRule, model *WorkflowExpirationResourceModel) {
	model.Enabled = types.BoolValue(rule.Status == "Enabled")
	model.CurrentVersionTriggerDelayDays = types.Int64Value(int64(rule.ExpirationDays))
	if model.Filter != nil {
		model.Filter.ObjectKeyPrefix = types.StringValue(rule.Prefix)
	}
}
