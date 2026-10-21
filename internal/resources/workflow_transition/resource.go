package workflowtransition

import (
	"context"
	"fmt"
	"strings"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/cmrh/terraform-provider-artesca/internal/creds"
	"github.com/cmrh/terraform-provider-artesca/internal/resources/lifecycle"
	validators "github.com/cmrh/terraform-provider-artesca/internal/validators"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                   = &WorkflowTransitionResource{}
	_ resource.ResourceWithImportState    = &WorkflowTransitionResource{}
	_ resource.ResourceWithUpgradeState   = &WorkflowTransitionResource{}
	_ resource.ResourceWithValidateConfig = &WorkflowTransitionResource{}
)

const (
	applyCurrent    = "current"
	applyNoncurrent = "noncurrent"
)

type WorkflowTransitionResource struct {
	accounts *client.AccountCredentialSource
	s3       *client.S3Client
}

func NewWorkflowTransitionResource() resource.Resource {
	return &WorkflowTransitionResource{}
}

func (r *WorkflowTransitionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket_workflow_transition"
}

func (r *WorkflowTransitionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "Manages a bucket transition lifecycle rule in ARTESCA via the S3 API.",
		Attributes: map[string]schema.Attribute{
			creds.AttrAccountName: creds.ResourceAttribute(),
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
				Validators: []validator.String{
					validators.RuleID{},
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the lifecycle rule is enabled.",
				Required:    true,
			},
			"apply_to_version": schema.StringAttribute{
				Description: "Which object versions the rule transitions: `current` (default) or `noncurrent`. Changing it replaces the rule.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(applyCurrent),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIf(applyToVersionChanged,
						"Changing apply_to_version replaces the rule.",
						"Changing apply_to_version replaces the rule."),
				},
				Validators: []validator.String{
					validators.OneOf{Values: []string{applyCurrent, applyNoncurrent}},
				},
			},
			"location_name": schema.StringAttribute{
				Description: "The destination location (storage class) for transitioning objects.",
				Required:    true,
			},
			"trigger_delay_days": schema.Int64Attribute{
				Description: "Number of days after which objects are transitioned; for `noncurrent`, days after a version becomes noncurrent. Must be at least 0. Exactly one of `trigger_delay_days` and `trigger_delay_date` must be set; `noncurrent` requires `trigger_delay_days`.",
				Optional:    true,
				Validators: []validator.Int64{
					validators.Int64AtLeast{Min: 0},
				},
			},
			"trigger_delay_date": schema.StringAttribute{
				Description: "Date (YYYY-MM-DD, midnight UTC) on which current versions are transitioned. Only with `apply_to_version = \"current\"`.",
				Optional:    true,
				Validators: []validator.String{
					validators.Date{},
				},
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
					"object_tags": schema.MapAttribute{
						Description: "Object tags filter: the rule applies to objects that have all of these tags.",
						Optional:    true,
						ElementType: types.StringType,
						Validators: []validator.Map{
							validators.MapNotEmpty{},
						},
					},
				},
			},
		},
	}
}

// ValidateConfig requires exactly one timing: days or a date for current
// versions, days for noncurrent versions.
func (r *WorkflowTransitionResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config WorkflowTransitionResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if config.TriggerDelayDays.IsUnknown() || config.TriggerDelayDate.IsUnknown() || config.ApplyToVersion.IsUnknown() {
		return
	}
	daysSet, dateSet := !config.TriggerDelayDays.IsNull(), !config.TriggerDelayDate.IsNull()
	switch {
	case config.ApplyToVersion.ValueString() == applyNoncurrent && dateSet:
		resp.Diagnostics.AddAttributeError(path.Root("trigger_delay_date"), "Date not supported for noncurrent versions",
			"A noncurrent-version transition takes trigger_delay_days, the days after a version becomes noncurrent.")
	case daysSet && dateSet:
		resp.Diagnostics.AddAttributeError(path.Root("trigger_delay_date"), "Conflicting transition timing",
			"Set trigger_delay_days or trigger_delay_date, not both.")
	case !daysSet && !dateSet:
		resp.Diagnostics.AddAttributeError(path.Root("trigger_delay_days"), "Missing transition timing",
			"Set trigger_delay_days or trigger_delay_date.")
	}
}

func (r *WorkflowTransitionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.accounts = providerData.Accounts
}

func (r *WorkflowTransitionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan WorkflowTransitionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	acctCreds, err := r.accounts.For(ctx, plan.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting account credentials", err.Error())
		return
	}

	bucket := plan.BucketName.ValueString()

	ruleID := plan.RuleID.ValueString()
	if ruleID == "" {
		ruleID = client.NewRuleID()
	}

	newRule, diags := modelToLifecycleRule(ctx, &plan, ruleID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	unlock := r.s3.LockBucket(bucket)
	defer unlock()

	existing, err := r.s3.GetBucketLifecycle(ctx, acctCreds, bucket)
	if err != nil {
		resp.Diagnostics.AddError("Error reading existing lifecycle rules", err.Error())
		return
	}

	for _, rule := range existing {
		if rule.ID == ruleID {
			resp.Diagnostics.AddError("Lifecycle rule already exists",
				fmt.Sprintf("Bucket %q already has a lifecycle rule with ID %q.", bucket, ruleID))
			return
		}
	}
	rules := append(existing, newRule)

	tflog.Debug(ctx, "Creating transition lifecycle rule", map[string]any{"bucket": bucket, "rule_id": ruleID})

	if err := r.s3.PutBucketLifecycle(ctx, acctCreds, bucket, rules); err != nil {
		resp.Diagnostics.AddError("Error creating transition lifecycle rule", err.Error())
		return
	}

	plan.RuleID = types.StringValue(ruleID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WorkflowTransitionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state WorkflowTransitionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	acctCreds, err := r.accounts.For(ctx, state.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting account credentials", err.Error())
		return
	}

	bucket := state.BucketName.ValueString()
	ruleID := state.RuleID.ValueString()

	rules, err := r.s3.GetBucketLifecycle(ctx, acctCreds, bucket)
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

	resp.Diagnostics.Append(lifecycleRuleToModel(ctx, found, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *WorkflowTransitionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan WorkflowTransitionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	acctCreds, err := r.accounts.For(ctx, plan.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting account credentials", err.Error())
		return
	}

	bucket := plan.BucketName.ValueString()
	ruleID := plan.RuleID.ValueString()

	unlock := r.s3.LockBucket(bucket)
	defer unlock()

	existing, err := r.s3.GetBucketLifecycle(ctx, acctCreds, bucket)
	if err != nil {
		resp.Diagnostics.AddError("Error reading existing lifecycle rules", err.Error())
		return
	}

	updated, diags := modelToLifecycleRule(ctx, &plan, ruleID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	scope := client.ScopeTransition
	if plan.ApplyToVersion.ValueString() == applyNoncurrent {
		scope = client.ScopeNoncurrentTransition
	}
	rules := make([]client.LifecycleRule, 0, len(existing))
	replaced := false
	for _, rule := range existing {
		if rule.ID == ruleID {
			// Keep the parts of the rule this resource doesn't model.
			merged, err := rule.Updated(updated, scope)
			if err != nil {
				resp.Diagnostics.AddError("Error updating transition lifecycle rule", err.Error())
				return
			}
			rules = append(rules, merged)
			replaced = true
		} else {
			rules = append(rules, rule)
		}
	}
	if !replaced {
		resp.Diagnostics.AddError("Lifecycle rule not found",
			fmt.Sprintf("Bucket %q has no lifecycle rule with ID %q.", bucket, ruleID))
		return
	}

	tflog.Debug(ctx, "Updating transition lifecycle rule", map[string]any{"rule_id": ruleID})

	if err := r.s3.PutBucketLifecycle(ctx, acctCreds, bucket, rules); err != nil {
		resp.Diagnostics.AddError("Error updating transition lifecycle rule", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WorkflowTransitionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state WorkflowTransitionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	acctCreds, err := r.accounts.For(ctx, state.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting account credentials", err.Error())
		return
	}

	bucket := state.BucketName.ValueString()
	ruleID := state.RuleID.ValueString()

	unlock := r.s3.LockBucket(bucket)
	defer unlock()

	existing, err := r.s3.GetBucketLifecycle(ctx, acctCreds, bucket)
	if err != nil {
		if client.IsNoSuchBucket(err) {
			return // the bucket, and its configuration, is gone
		}
		resp.Diagnostics.AddError("Error reading existing lifecycle rules", err.Error())
		return
	}

	remaining := make([]client.LifecycleRule, 0, len(existing))
	for _, rule := range existing {
		if rule.ID != ruleID {
			remaining = append(remaining, rule)
		}
	}

	tflog.Debug(ctx, "Deleting transition lifecycle rule", map[string]any{"rule_id": ruleID})

	if len(remaining) == 0 {
		if err := r.s3.DeleteBucketLifecycle(ctx, acctCreds, bucket); err != nil {
			if client.IsNoSuchBucket(err) {
				return // the bucket, and its configuration, is gone
			}
			resp.Diagnostics.AddError("Error deleting lifecycle configuration", err.Error())
			return
		}
	} else {
		if err := r.s3.PutBucketLifecycle(ctx, acctCreds, bucket, remaining); err != nil {
			if client.IsNoSuchBucket(err) {
				return // the bucket, and its configuration, is gone
			}
			resp.Diagnostics.AddError("Error updating lifecycle configuration", err.Error())
			return
		}
	}
}

func (r *WorkflowTransitionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	rest, ok := creds.ImportAccount(ctx, req, resp)
	if !ok {
		return
	}
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format <account_name>/bucket_name/rule_id, got %q", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bucket_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("rule_id"), parts[1])...)
}

func modelToLifecycleRule(ctx context.Context, model *WorkflowTransitionResourceModel, ruleID string) (client.LifecycleRule, diag.Diagnostics) {
	status := "Disabled"
	if model.Enabled.ValueBool() {
		status = "Enabled"
	}

	rule := client.LifecycleRule{ID: ruleID, Status: status}
	days := int(model.TriggerDelayDays.ValueInt64())
	if model.ApplyToVersion.ValueString() == applyNoncurrent {
		rule.NoncurrentTransitionDays = days
		rule.NoncurrentTransitionLocation = model.LocationName.ValueString()
	} else {
		rule.TransitionDays = days
		rule.TransitionDate = model.TriggerDelayDate.ValueString()
		rule.TransitionLocation = model.LocationName.ValueString()
	}

	var diags diag.Diagnostics
	if model.Filter != nil {
		rule.Prefix = model.Filter.ObjectKeyPrefix.ValueString()
		rule.Tags, diags = lifecycle.TagsFromMap(ctx, model.Filter.ObjectTags)
	}
	return rule, diags
}

func lifecycleRuleToModel(ctx context.Context, rule *client.LifecycleRule, model *WorkflowTransitionResourceModel) diag.Diagnostics {
	model.Enabled = types.BoolValue(rule.Status == "Enabled")

	// State without apply_to_version (an import, or state from before the
	// attribute existed) takes the rule's transition: the current-version
	// one when it has both.
	if model.ApplyToVersion.IsNull() || model.ApplyToVersion.IsUnknown() {
		apply := applyCurrent
		if rule.TransitionLocation == "" && rule.NoncurrentTransitionLocation != "" {
			apply = applyNoncurrent
		}
		model.ApplyToVersion = types.StringValue(apply)
	}

	if model.ApplyToVersion.ValueString() == applyNoncurrent {
		model.LocationName = types.StringValue(rule.NoncurrentTransitionLocation)
		model.TriggerDelayDays = types.Int64Value(int64(rule.NoncurrentTransitionDays))
		model.TriggerDelayDate = types.StringNull()
	} else {
		model.LocationName = types.StringValue(rule.TransitionLocation)
		if rule.TransitionDate != "" {
			model.TriggerDelayDays = types.Int64Null()
			model.TriggerDelayDate = types.StringValue(rule.TransitionDate)
		} else {
			model.TriggerDelayDays = types.Int64Value(int64(rule.TransitionDays))
			model.TriggerDelayDate = types.StringNull()
		}
	}

	filter, diags := lifecycle.FilterToModel(ctx, rule, model.Filter.toCommon())
	model.Filter = fromCommon(filter)
	return diags
}

// applyToVersionChanged replaces the rule when apply_to_version changes, but
// not when state predates the attribute (null): every such rule transitions
// current versions.
func applyToVersionChanged(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
	resp.RequiresReplace = !req.StateValue.IsNull()
}

func (f *WorkflowFilterModel) toCommon() *lifecycle.Filter {
	if f == nil {
		return nil
	}
	return &lifecycle.Filter{ObjectKeyPrefix: f.ObjectKeyPrefix, ObjectTags: f.ObjectTags}
}

func fromCommon(f *lifecycle.Filter) *WorkflowFilterModel {
	if f == nil {
		return nil
	}
	return &WorkflowFilterModel{ObjectKeyPrefix: f.ObjectKeyPrefix, ObjectTags: f.ObjectTags}
}

// UpgradeState migrates v0 state, which held the account's access key pair,
// to account_name.
func (r *WorkflowTransitionResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	return creds.UpgradeFromAccountKeys(r, func() *client.AccountCredentialSource { return r.accounts })
}
