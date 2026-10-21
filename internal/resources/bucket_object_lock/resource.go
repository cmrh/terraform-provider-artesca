package bucketobjectlock

import (
	"context"
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
	_ resource.Resource                   = &BucketObjectLockResource{}
	_ resource.ResourceWithImportState    = &BucketObjectLockResource{}
	_ resource.ResourceWithValidateConfig = &BucketObjectLockResource{}
)

type BucketObjectLockResource struct {
	accounts *client.AccountCredentialSource
	s3Client *client.S3Client
}

func NewBucketObjectLockResource() resource.Resource {
	return &BucketObjectLockResource{}
}

func (r *BucketObjectLockResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket_object_lock"
}

func (r *BucketObjectLockResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the default retention of an object-lock bucket: every new object version is locked for this period. The bucket must be created with `object_lock_enabled = true`. Destroying this resource removes the default retention; object lock stays enabled and existing retention is unchanged.",
		Attributes: map[string]schema.Attribute{
			creds.AttrAccountName: creds.ResourceAttribute(),
			"bucket_name": schema.StringAttribute{
				Description: "The name of the object-lock bucket.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validators.BucketName{},
				},
			},
			"retention_mode": schema.StringAttribute{
				Description: "Default retention mode: `GOVERNANCE` (users with the s3:BypassGovernanceRetention permission can delete or shorten it) or `COMPLIANCE` (nobody can delete a locked version until its retention ends).",
				Required:    true,
				Validators: []validator.String{
					validators.RetentionMode{},
				},
			},
			"retention_days": schema.Int64Attribute{
				Description: "Default retention period in days. Exactly one of `retention_days` and `retention_years` must be set.",
				Optional:    true,
				Validators: []validator.Int64{
					validators.Int64AtLeast{Min: 1},
				},
			},
			"retention_years": schema.Int64Attribute{
				Description: "Default retention period in years. Exactly one of `retention_days` and `retention_years` must be set.",
				Optional:    true,
				Validators: []validator.Int64{
					validators.Int64AtLeast{Min: 1},
				},
			},
		},
	}
}

// ValidateConfig requires exactly one of retention_days and retention_years.
func (r *BucketObjectLockResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config BucketObjectLockResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// An unknown value may turn out set or null.
	if config.RetentionDays.IsUnknown() || config.RetentionYears.IsUnknown() {
		return
	}
	daysSet, yearsSet := !config.RetentionDays.IsNull(), !config.RetentionYears.IsNull()
	switch {
	case daysSet && yearsSet:
		resp.Diagnostics.AddAttributeError(path.Root("retention_years"), "Conflicting retention period",
			"Set exactly one of retention_days or retention_years, not both.")
	case !daysSet && !yearsSet:
		resp.Diagnostics.AddAttributeError(path.Root("retention_days"), "Missing retention period",
			"Set exactly one of retention_days or retention_years.")
	}
}

func (r *BucketObjectLockResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"The s3_endpoint must be set in the provider configuration or via the ARTESCA_S3_ENDPOINT environment variable to use bucket resources.",
		)
		return
	}
	r.s3Client = providerData.S3
	r.accounts = providerData.Accounts
}

func (r *BucketObjectLockResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BucketObjectLockResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.put(ctx, &plan, "Error setting bucket default retention", resp.Diagnostics.AddError) {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	}
}

func (r *BucketObjectLockResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state BucketObjectLockResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	acctCreds, err := r.accounts.For(ctx, state.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting account credentials", err.Error())
		return
	}

	_, retention, err := r.s3Client.GetObjectLockConfiguration(ctx, acctCreds, state.BucketName.ValueString())
	if client.IsNoSuchBucket(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading bucket default retention", err.Error())
		return
	}
	if retention == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.RetentionMode = types.StringValue(retention.Mode)
	state.RetentionDays = types.Int64Null()
	state.RetentionYears = types.Int64Null()
	if retention.Days > 0 {
		state.RetentionDays = types.Int64Value(retention.Days)
	}
	if retention.Years > 0 {
		state.RetentionYears = types.Int64Value(retention.Years)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *BucketObjectLockResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan BucketObjectLockResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.put(ctx, &plan, "Error updating bucket default retention", resp.Diagnostics.AddError) {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	}
}

func (r *BucketObjectLockResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state BucketObjectLockResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Removing bucket default retention", map[string]any{"bucket": state.BucketName.ValueString()})

	acctCreds, err := r.accounts.For(ctx, state.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting account credentials", err.Error())
		return
	}

	unlock := r.s3Client.LockBucket(state.BucketName.ValueString())
	defer unlock()

	if err := r.s3Client.PutObjectLockConfiguration(ctx, acctCreds, state.BucketName.ValueString(), nil); err != nil {
		if client.IsNoSuchBucket(err) {
			return // the bucket, and its configuration, is gone
		}
		resp.Diagnostics.AddError("Error removing bucket default retention", err.Error())
	}
}

func (r *BucketObjectLockResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	creds.ImportByID(ctx, "bucket_name", req, resp)
}

// put writes the planned default retention, reporting failures with summary.
func (r *BucketObjectLockResource) put(ctx context.Context, plan *BucketObjectLockResourceModel, summary string, addError func(string, string)) bool {
	bucket := plan.BucketName.ValueString()
	retention := &client.ObjectLockRetention{
		Mode:  plan.RetentionMode.ValueString(),
		Days:  plan.RetentionDays.ValueInt64(),
		Years: plan.RetentionYears.ValueInt64(),
	}

	tflog.Debug(ctx, "Setting bucket default retention", map[string]any{
		"bucket": bucket, "mode": retention.Mode, "days": retention.Days, "years": retention.Years,
	})

	acctCreds, err := r.accounts.For(ctx, plan.AccountName.ValueString())
	if err != nil {
		addError("Error getting account credentials", err.Error())
		return false
	}

	unlock := r.s3Client.LockBucket(bucket)
	defer unlock()

	err = r.s3Client.PutObjectLockConfiguration(ctx, acctCreds, bucket, retention)
	if err != nil && strings.Contains(err.Error(), "InvalidBucketState:") {
		addError(summary, fmt.Sprintf("Bucket %q does not have object lock enabled. Object lock can only be enabled when the bucket is created: set object_lock_enabled = true on the artesca_bucket, which replaces it.\n\n%s", bucket, err))
		return false
	}
	if err != nil {
		addError(summary, err.Error())
		return false
	}
	return true
}
