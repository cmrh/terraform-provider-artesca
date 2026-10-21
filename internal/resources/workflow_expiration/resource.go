package workflowexpiration

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/cmrh/terraform-provider-artesca/internal/creds"
	"github.com/cmrh/terraform-provider-artesca/internal/resources/lifecycle"
	validators "github.com/cmrh/terraform-provider-artesca/internal/validators"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                   = &WorkflowExpirationResource{}
	_ resource.ResourceWithImportState    = &WorkflowExpirationResource{}
	_ resource.ResourceWithUpgradeState   = &WorkflowExpirationResource{}
	_ resource.ResourceWithValidateConfig = &WorkflowExpirationResource{}
)

type WorkflowExpirationResource struct {
	accounts *client.AccountCredentialSource
	s3       *client.S3Client
}

func NewWorkflowExpirationResource() resource.Resource {
	return &WorkflowExpirationResource{}
}

func (r *WorkflowExpirationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket_workflow_expiration"
}

func (r *WorkflowExpirationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "Manages a bucket expiration lifecycle rule in ARTESCA via the S3 API.",
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
			"current_version_trigger_delay_days": schema.Int64Attribute{
				Description: "Number of days after which current versions expire. Must be at least 1. Conflicts with `current_version_trigger_delay_date` and `expire_delete_markers`.",
				Optional:    true,
				Validators: []validator.Int64{
					validators.Int64AtLeast{Min: 1},
				},
			},
			"current_version_trigger_delay_date": schema.StringAttribute{
				Description: "Date (YYYY-MM-DD, midnight UTC) on which current versions expire. Conflicts with `current_version_trigger_delay_days` and `expire_delete_markers`.",
				Optional:    true,
				Validators: []validator.String{
					validators.Date{},
				},
			},
			"expire_delete_markers": schema.BoolAttribute{
				Description: "Remove delete markers that have no noncurrent versions left behind them. Conflicts with the current-version days or date, and with `filter.object_tags`.",
				Optional:    true,
			},
			"previous_version_trigger_delay_days": schema.Int64Attribute{
				Description: "Number of days after an object version becomes noncurrent before it expires. Must be at least 1.",
				Optional:    true,
				Validators: []validator.Int64{
					validators.Int64AtLeast{Min: 1},
				},
			},
			"incomplete_multipart_upload_trigger_delay_days": schema.Int64Attribute{
				Description: "Number of days after a multipart upload starts before an incomplete upload is aborted and its parts deleted. Must be at least 1. Conflicts with `filter.object_tags`.",
				Optional:    true,
				Validators: []validator.Int64{
					validators.Int64AtLeast{Min: 1},
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

// ValidateConfig rejects rules ARTESCA refuses: a rule without an action, two
// timings for current versions, and tag filters with actions that don't
// support them.
func (r *WorkflowExpirationResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config WorkflowExpirationResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	days := config.CurrentVersionTriggerDelayDays
	date := config.CurrentVersionTriggerDelayDate
	markers := config.ExpireDeleteMarkers
	previous := config.PreviousVersionTriggerDelayDays
	multipart := config.IncompleteMultipartUploadTriggerDelayDays
	var tags types.Map
	if config.Filter != nil {
		tags = config.Filter.ObjectTags
	}

	set := func(v attr.Value) bool { return !v.IsNull() && !v.IsUnknown() }
	markersOn := set(markers) && markers.ValueBool()

	if set(days) && set(date) {
		resp.Diagnostics.AddAttributeError(path.Root("current_version_trigger_delay_date"), "Conflicting expiration",
			"Set current_version_trigger_delay_days or current_version_trigger_delay_date, not both.")
	}
	if markersOn && (set(days) || set(date)) {
		resp.Diagnostics.AddAttributeError(path.Root("expire_delete_markers"), "Conflicting expiration",
			"expire_delete_markers can't be combined with current_version_trigger_delay_days or current_version_trigger_delay_date.")
	}
	if set(tags) && markersOn {
		resp.Diagnostics.AddAttributeError(path.Root("filter").AtName("object_tags"), "Tag filter not supported",
			"A rule with filter.object_tags can't set expire_delete_markers.")
	}
	if set(tags) && set(multipart) {
		resp.Diagnostics.AddAttributeError(path.Root("filter").AtName("object_tags"), "Tag filter not supported",
			"A rule with filter.object_tags can't set incomplete_multipart_upload_trigger_delay_days.")
	}

	// A value unknown at validation may still provide the action.
	for _, v := range []attr.Value{days, date, markers, previous, multipart} {
		if v.IsUnknown() {
			return
		}
	}
	if !set(days) && !set(date) && !markersOn && !set(previous) && !set(multipart) {
		resp.Diagnostics.AddError("Missing expiration action",
			"Set at least one of current_version_trigger_delay_days, current_version_trigger_delay_date, expire_delete_markers = true, previous_version_trigger_delay_days or incomplete_multipart_upload_trigger_delay_days.")
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
	r.accounts = providerData.Accounts
}

func (r *WorkflowExpirationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan WorkflowExpirationResourceModel
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

	tflog.Debug(ctx, "Creating expiration lifecycle rule", map[string]any{"bucket": bucket, "rule_id": ruleID})

	if err := r.s3.PutBucketLifecycle(ctx, acctCreds, bucket, rules); err != nil {
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

func (r *WorkflowExpirationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan WorkflowExpirationResourceModel
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
	rules := make([]client.LifecycleRule, 0, len(existing))
	replaced := false
	for _, rule := range existing {
		if rule.ID == ruleID {
			// Keep the parts of the rule this resource doesn't model.
			merged, err := rule.Updated(updated, client.ScopeExpiration)
			if err != nil {
				resp.Diagnostics.AddError("Error updating expiration lifecycle rule", err.Error())
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

	tflog.Debug(ctx, "Updating expiration lifecycle rule", map[string]any{"rule_id": ruleID})

	if err := r.s3.PutBucketLifecycle(ctx, acctCreds, bucket, rules); err != nil {
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

	tflog.Debug(ctx, "Deleting expiration lifecycle rule", map[string]any{"rule_id": ruleID})

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

func (r *WorkflowExpirationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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

func modelToLifecycleRule(ctx context.Context, model *WorkflowExpirationResourceModel, ruleID string) (client.LifecycleRule, diag.Diagnostics) {
	status := "Disabled"
	if model.Enabled.ValueBool() {
		status = "Enabled"
	}

	rule := client.LifecycleRule{
		ID:                        ruleID,
		Status:                    status,
		ExpirationDays:            int(model.CurrentVersionTriggerDelayDays.ValueInt64()),
		ExpirationDate:            model.CurrentVersionTriggerDelayDate.ValueString(),
		ExpiredObjectDeleteMarker: model.ExpireDeleteMarkers.ValueBool(),
		NoncurrentExpirationDays:  int(model.PreviousVersionTriggerDelayDays.ValueInt64()),
		AbortMultipartDays:        int(model.IncompleteMultipartUploadTriggerDelayDays.ValueInt64()),
	}

	var diags diag.Diagnostics
	if model.Filter != nil {
		rule.Prefix = model.Filter.ObjectKeyPrefix.ValueString()
		rule.Tags, diags = lifecycle.TagsFromMap(ctx, model.Filter.ObjectTags)
	}
	return rule, diags
}

func lifecycleRuleToModel(ctx context.Context, rule *client.LifecycleRule, model *WorkflowExpirationResourceModel) diag.Diagnostics {
	model.Enabled = types.BoolValue(rule.Status == "Enabled")
	model.CurrentVersionTriggerDelayDays = lifecycle.PositiveInt64(rule.ExpirationDays)
	model.CurrentVersionTriggerDelayDate = lifecycle.NonEmptyString(rule.ExpirationDate)
	model.PreviousVersionTriggerDelayDays = lifecycle.PositiveInt64(rule.NoncurrentExpirationDays)
	model.IncompleteMultipartUploadTriggerDelayDays = lifecycle.PositiveInt64(rule.AbortMultipartDays)
	// An absent delete-marker action matches a configured false.
	if rule.ExpiredObjectDeleteMarker || model.ExpireDeleteMarkers.ValueBool() {
		model.ExpireDeleteMarkers = types.BoolValue(rule.ExpiredObjectDeleteMarker)
	}

	filter, diags := lifecycle.FilterToModel(ctx, rule, model.Filter.toCommon())
	model.Filter = fromCommon(filter)
	return diags
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
func (r *WorkflowExpirationResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	return creds.UpgradeFromAccountKeys(r, func() *client.AccountCredentialSource { return r.accounts })
}
