package workflowreplication

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

var _ resource.Resource = &WorkflowReplicationResource{}

type WorkflowReplicationResource struct {
	client *client.ManagementClient
}

func NewWorkflowReplicationResource() resource.Resource {
	return &WorkflowReplicationResource{}
}

func (r *WorkflowReplicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket_workflow_replication"
}

func (r *WorkflowReplicationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a bucket replication workflow in ARTESCA.",
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
				Description: "The name of the replication workflow.",
				Required:    true,
			},
			"version": schema.Int64Attribute{
				Description: "The version of the replication workflow.",
				Required:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the replication workflow is enabled.",
				Required:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"source": schema.SingleNestedBlock{
				Description: "Source configuration for replication.",
				Attributes: map[string]schema.Attribute{
					"bucket_name": schema.StringAttribute{
						Description: "Source bucket name. Must be 3–63 characters, lowercase letters, numbers, hyphens, and periods.",
						Required:    true,
						Validators: []validator.String{
							validators.BucketName{},
						},
					},
					"prefix": schema.StringAttribute{
						Description: "Object key prefix filter for replication.",
						Required:    true,
					},
					"location": schema.StringAttribute{
						Description: "Source location name.",
						Optional:    true,
					},
				},
			},
			"destination": schema.SingleNestedBlock{
				Description: "Destination configuration for replication.",
				Attributes: map[string]schema.Attribute{
					"bucket_name": schema.StringAttribute{
						Description: "Destination bucket name. Must be 3–63 characters, lowercase letters, numbers, hyphens, and periods.",
						Optional:    true,
						Validators: []validator.String{
							validators.BucketName{},
						},
					},
					"location": schema.StringAttribute{
						Description: "Destination location name.",
						Optional:    true,
					},
					"preferred_read_location": schema.StringAttribute{
						Description: "Preferred read location.",
						Optional:    true,
					},
					"role": schema.StringAttribute{
						Description: "IAM role for replication.",
						Optional:    true,
					},
				},
				Blocks: map[string]schema.Block{
					"locations": schema.ListNestedBlock{
						Description: "Destination locations with storage class.",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"name": schema.StringAttribute{
									Description: "Destination location name.",
									Required:    true,
								},
								"storage_class": schema.StringAttribute{
									Description: "Storage class at the destination location.",
									Optional:    true,
								},
							},
						},
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
	r.client = providerData.Management
}

func (r *WorkflowReplicationResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config WorkflowReplicationResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.Destination != nil {
		if !config.Destination.Location.IsNull() && !config.Destination.Location.IsUnknown() {
			resp.Diagnostics.AddError(
				"Invalid destination configuration",
				"The per-bucket workflow replication API does not support destination.location. "+
					"Use destination.bucket_name only. For location-based replication, use the artesca_replication resource instead.",
			)
		}
		if len(config.Destination.Locations) > 0 {
			resp.Diagnostics.AddError(
				"Invalid destination configuration",
				"The per-bucket workflow replication API does not support destination.locations. "+
					"Use destination.bucket_name only. For multi-backend replication, use the artesca_replication resource instead.",
			)
		}
	}
}

func (r *WorkflowReplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan WorkflowReplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instanceID := r.resolveInstanceID(&plan)
	accountID := plan.AccountID.ValueString()
	bucketName := plan.BucketName.ValueString()

	apiStream := modelToAPIReplication(&plan)

	tflog.Debug(ctx, "Creating bucket workflow replication", map[string]any{
		"bucket": bucketName,
		"name":   plan.Name.ValueString(),
	})

	created, err := r.client.CreateBucketWorkflowReplication(ctx, instanceID, accountID, bucketName, apiStream)
	if err != nil {
		resp.Diagnostics.AddError("Error creating bucket workflow replication", err.Error())
		return
	}

	plan.InstanceID = types.StringValue(instanceID)
	apiReplicationToModel(created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WorkflowReplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state WorkflowReplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Workflow reads are not available via a direct endpoint — preserve state as-is.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *WorkflowReplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan WorkflowReplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state WorkflowReplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instanceID := r.resolveInstanceID(&plan)
	accountID := plan.AccountID.ValueString()
	bucketName := plan.BucketName.ValueString()
	workflowID := state.WorkflowID.ValueString()

	apiStream := modelToAPIReplication(&plan)

	tflog.Debug(ctx, "Updating bucket workflow replication", map[string]any{"workflow_id": workflowID})

	updated, err := r.client.UpdateBucketWorkflowReplication(ctx, instanceID, accountID, bucketName, workflowID, apiStream)
	if err != nil {
		resp.Diagnostics.AddError("Error updating bucket workflow replication", err.Error())
		return
	}

	plan.InstanceID = types.StringValue(instanceID)
	apiReplicationToModel(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WorkflowReplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state WorkflowReplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instanceID := r.resolveInstanceID(&state)
	accountID := state.AccountID.ValueString()
	bucketName := state.BucketName.ValueString()
	workflowID := state.WorkflowID.ValueString()

	tflog.Debug(ctx, "Deleting bucket workflow replication", map[string]any{"workflow_id": workflowID})

	err := r.client.DeleteBucketWorkflowReplication(ctx, instanceID, accountID, bucketName, workflowID)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting bucket workflow replication", err.Error())
		return
	}
}

func (r *WorkflowReplicationResource) resolveInstanceID(model *WorkflowReplicationResourceModel) string {
	if !model.InstanceID.IsNull() && !model.InstanceID.IsUnknown() && model.InstanceID.ValueString() != "" {
		return model.InstanceID.ValueString()
	}
	return r.client.InstanceID
}

// --- Conversion helpers ---

func modelToAPIReplication(model *WorkflowReplicationResourceModel) *client.ReplicationStream {
	stream := &client.ReplicationStream{
		Name:    model.Name.ValueString(),
		Version: model.Version.ValueInt64(),
		Enabled: model.Enabled.ValueBool(),
	}

	if model.Source != nil {
		stream.Source = &client.ReplicationSource{
			BucketName: model.Source.BucketName.ValueString(),
			Prefix:     model.Source.Prefix.ValueString(),
		}
		if !model.Source.Location.IsNull() && !model.Source.Location.IsUnknown() {
			stream.Source.Location = model.Source.Location.ValueString()
		}
	}

	if model.Destination != nil {
		stream.Destination = &client.ReplicationDest{}
		if !model.Destination.BucketName.IsNull() && !model.Destination.BucketName.IsUnknown() {
			stream.Destination.BucketName = model.Destination.BucketName.ValueString()
		}
		if !model.Destination.Location.IsNull() && !model.Destination.Location.IsUnknown() {
			stream.Destination.Location = model.Destination.Location.ValueString()
		}
		if !model.Destination.PreferredReadLocation.IsNull() && !model.Destination.PreferredReadLocation.IsUnknown() {
			stream.Destination.PreferredReadLocation = model.Destination.PreferredReadLocation.ValueString()
		}
		if !model.Destination.Role.IsNull() && !model.Destination.Role.IsUnknown() {
			stream.Destination.Role = model.Destination.Role.ValueString()
		}
		for _, loc := range model.Destination.Locations {
			destLoc := client.ReplicationDestLocation{
				Name: loc.Name.ValueString(),
			}
			if !loc.StorageClass.IsNull() && !loc.StorageClass.IsUnknown() {
				destLoc.StorageClass = loc.StorageClass.ValueString()
			}
			stream.Destination.Locations = append(stream.Destination.Locations, destLoc)
		}
	}

	return stream
}

func apiReplicationToModel(stream *client.ReplicationStream, model *WorkflowReplicationResourceModel) {
	if stream.StreamID != "" {
		model.WorkflowID = types.StringValue(stream.StreamID)
	}
	model.Name = types.StringValue(stream.Name)
	model.Version = types.Int64Value(stream.Version)
	model.Enabled = types.BoolValue(stream.Enabled)

	if stream.Source != nil {
		model.Source = &WorkflowReplicationSourceModel{
			BucketName: types.StringValue(stream.Source.BucketName),
			Prefix:     types.StringValue(stream.Source.Prefix),
		}
		if stream.Source.Location != "" {
			model.Source.Location = types.StringValue(stream.Source.Location)
		} else {
			model.Source.Location = types.StringNull()
		}
	}

	if stream.Destination != nil {
		model.Destination = &WorkflowReplicationDestModel{}
		if stream.Destination.BucketName != "" {
			model.Destination.BucketName = types.StringValue(stream.Destination.BucketName)
		} else {
			model.Destination.BucketName = types.StringNull()
		}
		if stream.Destination.Location != "" {
			model.Destination.Location = types.StringValue(stream.Destination.Location)
		} else {
			model.Destination.Location = types.StringNull()
		}
		if stream.Destination.PreferredReadLocation != "" {
			model.Destination.PreferredReadLocation = types.StringValue(stream.Destination.PreferredReadLocation)
		} else {
			model.Destination.PreferredReadLocation = types.StringNull()
		}
		if stream.Destination.Role != "" {
			model.Destination.Role = types.StringValue(stream.Destination.Role)
		} else {
			model.Destination.Role = types.StringNull()
		}
		if len(stream.Destination.Locations) > 0 {
			model.Destination.Locations = make([]WorkflowReplicationDestLocModel, len(stream.Destination.Locations))
			for i, loc := range stream.Destination.Locations {
				model.Destination.Locations[i] = WorkflowReplicationDestLocModel{
					Name: types.StringValue(loc.Name),
				}
				if loc.StorageClass != "" {
					model.Destination.Locations[i].StorageClass = types.StringValue(loc.StorageClass)
				} else {
					model.Destination.Locations[i].StorageClass = types.StringNull()
				}
			}
		}
	}
}
