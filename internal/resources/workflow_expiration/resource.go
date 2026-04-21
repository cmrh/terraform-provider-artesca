package workflowexpiration

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

var _ resource.Resource = &WorkflowExpirationResource{}

type WorkflowExpirationResource struct {
	client *client.ManagementClient
}

func NewWorkflowExpirationResource() resource.Resource {
	return &WorkflowExpirationResource{}
}

func (r *WorkflowExpirationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket_workflow_expiration"
}

func (r *WorkflowExpirationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a bucket expiration lifecycle workflow in ARTESCA.",
		Attributes: map[string]schema.Attribute{
			"instance_id": schema.StringAttribute{
				Description: "The instance ID. Defaults to the provider's instance_id if omitted.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"account_id": schema.StringAttribute{
				Description: "The account ID that owns the bucket.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"bucket_name": schema.StringAttribute{
				Description: "The name of the bucket this workflow applies to. Must be 3–63 characters, lowercase letters, numbers, hyphens, and periods.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validators.BucketName{},
				},
			},
			"workflow_id": schema.StringAttribute{
				Description: "The workflow ID assigned by ARTESCA.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the workflow.",
				Optional:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the workflow is enabled.",
				Required:    true,
			},
			"current_version_trigger_delay_date": schema.StringAttribute{
				Description: "Date after which current version objects expire (format: YYYY-MM-DD).",
				Optional:    true,
			},
			"current_version_trigger_delay_days": schema.Int64Attribute{
				Description: "Number of days after which current version objects expire.",
				Optional:    true,
			},
			"expire_delete_markers_trigger": schema.BoolAttribute{
				Description: "Whether to expire delete markers.",
				Optional:    true,
			},
			"incomplete_multipart_upload_trigger_delay_days": schema.Int64Attribute{
				Description: "Number of days after which incomplete multipart uploads are cleaned up.",
				Optional:    true,
			},
			"previous_version_trigger_delay_days": schema.Int64Attribute{
				Description: "Number of days after which previous version objects expire.",
				Optional:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"filter": schema.SingleNestedBlock{
				Description: "Filter to scope which objects this workflow applies to.",
				Attributes: map[string]schema.Attribute{
					"object_key_prefix": schema.StringAttribute{
						Description: "Object key prefix filter.",
						Optional:    true,
					},
				},
				Blocks: map[string]schema.Block{
					"object_tags": schema.ListNestedBlock{
						Description: "Object tag filters.",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"key": schema.StringAttribute{
									Description: "Tag key.",
									Required:    true,
								},
								"value": schema.StringAttribute{
									Description: "Tag value.",
									Required:    true,
								},
							},
						},
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
	r.client = providerData.Management
}

func (r *WorkflowExpirationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan WorkflowExpirationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instanceID := r.resolveInstanceID(&plan)
	accountID := plan.AccountID.ValueString()
	bucketName := plan.BucketName.ValueString()

	apiWf := modelToAPIExpiration(&plan)

	tflog.Debug(ctx, "Creating expiration workflow", map[string]any{
		"bucket": bucketName,
		"name":   plan.Name.ValueString(),
	})

	created, err := r.client.CreateBucketWorkflowExpiration(ctx, instanceID, accountID, bucketName, apiWf)
	if err != nil {
		resp.Diagnostics.AddError("Error creating expiration workflow", err.Error())
		return
	}

	plan.InstanceID = types.StringValue(instanceID)
	apiExpirationToModel(created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WorkflowExpirationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state WorkflowExpirationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Workflow reads are not available via the overlay — we preserve state as-is.
	// The workflow was created successfully, so we trust the stored state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *WorkflowExpirationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan WorkflowExpirationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state WorkflowExpirationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instanceID := r.resolveInstanceID(&plan)
	accountID := plan.AccountID.ValueString()
	bucketName := plan.BucketName.ValueString()
	workflowID := state.WorkflowID.ValueString()

	apiWf := modelToAPIExpiration(&plan)

	tflog.Debug(ctx, "Updating expiration workflow", map[string]any{"workflow_id": workflowID})

	updated, err := r.client.UpdateBucketWorkflowExpiration(ctx, instanceID, accountID, bucketName, workflowID, apiWf)
	if err != nil {
		resp.Diagnostics.AddError("Error updating expiration workflow", err.Error())
		return
	}

	plan.InstanceID = types.StringValue(instanceID)
	apiExpirationToModel(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WorkflowExpirationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state WorkflowExpirationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instanceID := r.resolveInstanceID(&state)
	accountID := state.AccountID.ValueString()
	bucketName := state.BucketName.ValueString()
	workflowID := state.WorkflowID.ValueString()

	tflog.Debug(ctx, "Deleting expiration workflow", map[string]any{"workflow_id": workflowID})

	err := r.client.DeleteBucketWorkflowExpiration(ctx, instanceID, accountID, bucketName, workflowID)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting expiration workflow", err.Error())
		return
	}
}

func (r *WorkflowExpirationResource) resolveInstanceID(model *WorkflowExpirationResourceModel) string {
	if !model.InstanceID.IsNull() && !model.InstanceID.IsUnknown() && model.InstanceID.ValueString() != "" {
		return model.InstanceID.ValueString()
	}
	return r.client.InstanceID
}

// --- Conversion helpers ---

func modelToAPIExpiration(model *WorkflowExpirationResourceModel) *client.BucketWorkflowExpiration {
	wf := &client.BucketWorkflowExpiration{
		Enabled:    model.Enabled.ValueBool(),
		BucketName: model.BucketName.ValueString(),
		Type:       "bucket-workflow-expiration-v1",
	}

	if !model.Name.IsNull() && !model.Name.IsUnknown() {
		wf.Name = model.Name.ValueString()
	}
	if !model.CurrentVersionTriggerDelayDate.IsNull() && !model.CurrentVersionTriggerDelayDate.IsUnknown() {
		wf.CurrentVersionTriggerDelayDate = model.CurrentVersionTriggerDelayDate.ValueString()
	}
	if !model.CurrentVersionTriggerDelayDays.IsNull() && !model.CurrentVersionTriggerDelayDays.IsUnknown() {
		v := int(model.CurrentVersionTriggerDelayDays.ValueInt64())
		wf.CurrentVersionTriggerDelayDays = &v
	}
	if !model.ExpireDeleteMarkersTrigger.IsNull() && !model.ExpireDeleteMarkersTrigger.IsUnknown() {
		v := model.ExpireDeleteMarkersTrigger.ValueBool()
		wf.ExpireDeleteMarkersTrigger = &v
	}
	if !model.IncompleteMultipartUploadTriggerDelayDays.IsNull() && !model.IncompleteMultipartUploadTriggerDelayDays.IsUnknown() {
		v := int(model.IncompleteMultipartUploadTriggerDelayDays.ValueInt64())
		wf.IncompleteMultipartUploadTriggerDelayDays = &v
	}
	if !model.PreviousVersionTriggerDelayDays.IsNull() && !model.PreviousVersionTriggerDelayDays.IsUnknown() {
		v := int(model.PreviousVersionTriggerDelayDays.ValueInt64())
		wf.PreviousVersionTriggerDelayDays = &v
	}

	if model.Filter != nil {
		wf.Filter = &client.WorkflowFilter{}
		if !model.Filter.ObjectKeyPrefix.IsNull() && !model.Filter.ObjectKeyPrefix.IsUnknown() {
			wf.Filter.ObjectKeyPrefix = model.Filter.ObjectKeyPrefix.ValueString()
		}
		for _, tag := range model.Filter.ObjectTags {
			wf.Filter.ObjectTags = append(wf.Filter.ObjectTags, client.WorkflowTag{
				Key:   tag.Key.ValueString(),
				Value: tag.Value.ValueString(),
			})
		}
	}

	return wf
}

func apiExpirationToModel(wf *client.BucketWorkflowExpiration, model *WorkflowExpirationResourceModel) {
	if wf.WorkflowID != "" {
		model.WorkflowID = types.StringValue(wf.WorkflowID)
	}
	if wf.Name != "" {
		model.Name = types.StringValue(wf.Name)
	}
	model.Enabled = types.BoolValue(wf.Enabled)
	model.BucketName = types.StringValue(wf.BucketName)

	if wf.CurrentVersionTriggerDelayDate != "" {
		model.CurrentVersionTriggerDelayDate = types.StringValue(wf.CurrentVersionTriggerDelayDate)
	}
	if wf.CurrentVersionTriggerDelayDays != nil {
		model.CurrentVersionTriggerDelayDays = types.Int64Value(int64(*wf.CurrentVersionTriggerDelayDays))
	}
	if wf.ExpireDeleteMarkersTrigger != nil {
		model.ExpireDeleteMarkersTrigger = types.BoolValue(*wf.ExpireDeleteMarkersTrigger)
	}
	if wf.IncompleteMultipartUploadTriggerDelayDays != nil {
		model.IncompleteMultipartUploadTriggerDelayDays = types.Int64Value(int64(*wf.IncompleteMultipartUploadTriggerDelayDays))
	}
	if wf.PreviousVersionTriggerDelayDays != nil {
		model.PreviousVersionTriggerDelayDays = types.Int64Value(int64(*wf.PreviousVersionTriggerDelayDays))
	}
}
