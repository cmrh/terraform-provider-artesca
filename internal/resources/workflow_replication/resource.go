package workflowreplication

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/cmrh/terraform-provider-artesca/internal/creds"
	validators "github.com/cmrh/terraform-provider-artesca/internal/validators"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &WorkflowReplicationResource{}
	_ resource.ResourceWithImportState = &WorkflowReplicationResource{}
)

type WorkflowReplicationResource struct {
	accounts *client.AccountCredentialSource
	s3       *client.S3Client
}

func NewWorkflowReplicationResource() resource.Resource {
	return &WorkflowReplicationResource{}
}

func (r *WorkflowReplicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket_workflow_replication"
}

func (r *WorkflowReplicationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a bucket replication rule in ARTESCA via the S3 API. Replicates objects from bucket_name to destination_bucket_name.",
		Attributes: map[string]schema.Attribute{
			creds.AttrAccountName: creds.ResourceAttribute(),
			"bucket_name": schema.StringAttribute{
				Description: "The source bucket. Versioning must be enabled. Must be 3–63 characters, lowercase letters, numbers, hyphens, and periods.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validators.BucketName{},
				},
			},
			"rule_id": schema.StringAttribute{
				Description: "The replication rule ID. Auto-generated if not set.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the replication rule is enabled.",
				Required:    true,
			},
			"destination_bucket_name": schema.StringAttribute{
				Description: "The bucket objects are replicated to. Versioning must be enabled. Must be 3–63 characters, lowercase letters, numbers, hyphens, and periods.",
				Required:    true,
				Validators: []validator.String{
					validators.BucketName{},
				},
			},
		},
		Blocks: map[string]schema.Block{
			"filter": schema.SingleNestedBlock{
				Description: "Filter to scope which objects this rule replicates.",
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

func (r *WorkflowReplicationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WorkflowReplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan WorkflowReplicationResourceModel
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
		b := make([]byte, 16)
		_, _ = rand.Read(b)
		ruleID = fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	}

	r.s3.LockReplication()
	defer r.s3.UnlockReplication()

	cfg, err := r.s3.GetBucketReplication(ctx, acctCreds, bucket)
	if err != nil {
		resp.Diagnostics.AddError("Error reading existing replication rules", err.Error())
		return
	}
	if cfg == nil {
		cfg = &client.BucketReplication{Role: client.DefaultReplicationRole}
	}
	for _, rule := range cfg.Rules {
		if rule.ID == ruleID {
			resp.Diagnostics.AddError("Replication rule already exists",
				fmt.Sprintf("Bucket %q already has a replication rule with ID %q.", bucket, ruleID))
			return
		}
	}
	cfg.Rules = append(cfg.Rules, modelToReplicationRule(&plan, ruleID))

	tflog.Debug(ctx, "Creating replication rule", map[string]any{"bucket": bucket, "rule_id": ruleID})

	if err := r.s3.PutBucketReplication(ctx, acctCreds, bucket, *cfg); err != nil {
		resp.Diagnostics.AddError("Error creating replication rule", err.Error())
		return
	}

	plan.RuleID = types.StringValue(ruleID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WorkflowReplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state WorkflowReplicationResourceModel
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

	cfg, err := r.s3.GetBucketReplication(ctx, acctCreds, bucket)
	if err != nil {
		resp.Diagnostics.AddError("Error reading replication rules", err.Error())
		return
	}

	var found *client.ReplicationRule
	if cfg != nil {
		for i := range cfg.Rules {
			if cfg.Rules[i].ID == ruleID {
				found = &cfg.Rules[i]
				break
			}
		}
	}
	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	replicationRuleToModel(found, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *WorkflowReplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan WorkflowReplicationResourceModel
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

	r.s3.LockReplication()
	defer r.s3.UnlockReplication()

	cfg, err := r.s3.GetBucketReplication(ctx, acctCreds, bucket)
	if err != nil {
		resp.Diagnostics.AddError("Error reading existing replication rules", err.Error())
		return
	}
	replaced := false
	if cfg != nil {
		for i := range cfg.Rules {
			if cfg.Rules[i].ID == ruleID {
				cfg.Rules[i] = modelToReplicationRule(&plan, ruleID)
				replaced = true
			}
		}
	}
	if !replaced {
		resp.Diagnostics.AddError("Replication rule not found",
			fmt.Sprintf("Bucket %q has no replication rule with ID %q.", bucket, ruleID))
		return
	}

	tflog.Debug(ctx, "Updating replication rule", map[string]any{"bucket": bucket, "rule_id": ruleID})

	if err := r.s3.PutBucketReplication(ctx, acctCreds, bucket, *cfg); err != nil {
		resp.Diagnostics.AddError("Error updating replication rule", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WorkflowReplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state WorkflowReplicationResourceModel
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

	r.s3.LockReplication()
	defer r.s3.UnlockReplication()

	cfg, err := r.s3.GetBucketReplication(ctx, acctCreds, bucket)
	if err != nil {
		resp.Diagnostics.AddError("Error reading existing replication rules", err.Error())
		return
	}
	if cfg == nil {
		return
	}

	remaining := make([]client.ReplicationRule, 0, len(cfg.Rules))
	for _, rule := range cfg.Rules {
		if rule.ID != ruleID {
			remaining = append(remaining, rule)
		}
	}

	tflog.Debug(ctx, "Deleting replication rule", map[string]any{"bucket": bucket, "rule_id": ruleID})

	if len(remaining) == 0 {
		if err := r.s3.DeleteBucketReplication(ctx, acctCreds, bucket); err != nil {
			resp.Diagnostics.AddError("Error deleting replication configuration", err.Error())
		}
		return
	}
	cfg.Rules = remaining
	if err := r.s3.PutBucketReplication(ctx, acctCreds, bucket, *cfg); err != nil {
		resp.Diagnostics.AddError("Error updating replication configuration", err.Error())
	}
}

func (r *WorkflowReplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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

func modelToReplicationRule(model *WorkflowReplicationResourceModel, ruleID string) client.ReplicationRule {
	status := "Disabled"
	if model.Enabled.ValueBool() {
		status = "Enabled"
	}

	rule := client.ReplicationRule{
		ID:                ruleID,
		Status:            status,
		DestinationBucket: model.DestinationBucketName.ValueString(),
	}
	if model.Filter != nil && !model.Filter.ObjectKeyPrefix.IsNull() {
		rule.Prefix = model.Filter.ObjectKeyPrefix.ValueString()
	}
	return rule
}

func replicationRuleToModel(rule *client.ReplicationRule, model *WorkflowReplicationResourceModel) {
	model.Enabled = types.BoolValue(rule.Status == "Enabled")
	model.DestinationBucketName = types.StringValue(rule.DestinationBucket)
	switch {
	case model.Filter != nil:
		// An unset prefix is stored as ""; keep it null so it doesn't diff.
		if !model.Filter.ObjectKeyPrefix.IsNull() || rule.Prefix != "" {
			model.Filter.ObjectKeyPrefix = types.StringValue(rule.Prefix)
		}
	case rule.Prefix != "":
		// Surface a server-side prefix even when the config has no filter
		// block, so a prefix added outside Terraform shows up as drift.
		model.Filter = &WorkflowFilterModel{ObjectKeyPrefix: types.StringValue(rule.Prefix)}
	}
}
