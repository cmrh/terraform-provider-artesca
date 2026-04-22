package workflowtransition

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

var _ resource.Resource = &WorkflowTransitionResource{}

type WorkflowTransitionResource struct {
	client *client.ManagementClient
}

func NewWorkflowTransitionResource() resource.Resource {
	return &WorkflowTransitionResource{}
}

func (r *WorkflowTransitionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket_workflow_transition"
}

func (r *WorkflowTransitionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a bucket transition lifecycle workflow in ARTESCA (v2 schema).",
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
			"location_name": schema.StringAttribute{
				Description: "The destination location name for transitioning objects. Must be 3–63 characters, lowercase letters, numbers, hyphens, and periods.",
				Required:    true,
				Validators: []validator.String{
					validators.BucketName{},
				},
			},
			"apply_to_version": schema.StringAttribute{
				Description: "Which object versions to apply the transition to. Must be 'current' or 'noncurrent'.",
				Required:    true,
			},
			"trigger_delay_date": schema.StringAttribute{
				Description: "Date after which objects are transitioned (format: YYYY-MM-DD).",
				Optional:    true,
			},
			"trigger_delay_days": schema.Int64Attribute{
				Description: "Number of days after which objects are transitioned.",
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
	r.client = providerData.Management
}

func (r *WorkflowTransitionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan WorkflowTransitionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instanceID := r.resolveInstanceID(&plan)
	accountID := plan.AccountID.ValueString()
	bucketName := plan.BucketName.ValueString()

	apiWf := modelToAPITransition(&plan)

	tflog.Debug(ctx, "Creating transition workflow", map[string]any{
		"bucket":   bucketName,
		"location": plan.LocationName.ValueString(),
	})

	created, err := r.client.CreateBucketWorkflowTransition(ctx, instanceID, accountID, bucketName, apiWf)
	if err != nil {
		resp.Diagnostics.AddError("Error creating transition workflow", err.Error())
		return
	}

	plan.InstanceID = types.StringValue(instanceID)
	apiTransitionToModel(created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WorkflowTransitionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state WorkflowTransitionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Workflow reads are not available via the overlay — preserve state as-is.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *WorkflowTransitionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan WorkflowTransitionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state WorkflowTransitionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instanceID := r.resolveInstanceID(&plan)
	accountID := plan.AccountID.ValueString()
	bucketName := plan.BucketName.ValueString()
	workflowID := state.WorkflowID.ValueString()

	apiWf := modelToAPITransition(&plan)

	tflog.Debug(ctx, "Updating transition workflow", map[string]any{"workflow_id": workflowID})

	updated, err := r.client.UpdateBucketWorkflowTransition(ctx, instanceID, accountID, bucketName, workflowID, apiWf)
	if err != nil {
		resp.Diagnostics.AddError("Error updating transition workflow", err.Error())
		return
	}

	plan.InstanceID = types.StringValue(instanceID)
	apiTransitionToModel(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WorkflowTransitionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state WorkflowTransitionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instanceID := r.resolveInstanceID(&state)
	accountID := state.AccountID.ValueString()
	bucketName := state.BucketName.ValueString()
	workflowID := state.WorkflowID.ValueString()

	tflog.Debug(ctx, "Deleting transition workflow", map[string]any{"workflow_id": workflowID})

	err := r.client.DeleteBucketWorkflowTransition(ctx, instanceID, accountID, bucketName, workflowID)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting transition workflow", err.Error())
		return
	}
}

func (r *WorkflowTransitionResource) resolveInstanceID(model *WorkflowTransitionResourceModel) string {
	if !model.InstanceID.IsNull() && !model.InstanceID.IsUnknown() && model.InstanceID.ValueString() != "" {
		return model.InstanceID.ValueString()
	}
	return r.client.InstanceID
}

// --- Conversion helpers ---

func modelToAPITransition(model *WorkflowTransitionResourceModel) *client.BucketWorkflowTransition {
	wf := &client.BucketWorkflowTransition{
		Enabled:        model.Enabled.ValueBool(),
		BucketName:     model.BucketName.ValueString(),
		Type:           "bucket-workflow-transition-v2",
		LocationName:   model.LocationName.ValueString(),
		ApplyToVersion: model.ApplyToVersion.ValueString(),
	}

	if !model.Name.IsNull() && !model.Name.IsUnknown() {
		wf.Name = model.Name.ValueString()
	}
	if !model.TriggerDelayDate.IsNull() && !model.TriggerDelayDate.IsUnknown() {
		wf.TriggerDelayDate = model.TriggerDelayDate.ValueString()
	}
	if !model.TriggerDelayDays.IsNull() && !model.TriggerDelayDays.IsUnknown() {
		v := model.TriggerDelayDays.ValueInt64()
		wf.TriggerDelayDays = &v
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

func apiTransitionToModel(wf *client.BucketWorkflowTransition, model *WorkflowTransitionResourceModel) {
	if wf.WorkflowID != "" {
		model.WorkflowID = types.StringValue(wf.WorkflowID)
	}
	if wf.Name != "" {
		model.Name = types.StringValue(wf.Name)
	}
	model.Enabled = types.BoolValue(wf.Enabled)
	model.BucketName = types.StringValue(wf.BucketName)
	model.LocationName = types.StringValue(wf.LocationName)
	model.ApplyToVersion = types.StringValue(wf.ApplyToVersion)

	if wf.TriggerDelayDate != "" {
		model.TriggerDelayDate = types.StringValue(wf.TriggerDelayDate)
	}
	if wf.TriggerDelayDays != nil {
		model.TriggerDelayDays = types.Int64Value(*wf.TriggerDelayDays)
	}
}
