package replication

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
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

var (
	_ resource.Resource                = &ReplicationResource{}
	_ resource.ResourceWithImportState = &ReplicationResource{}
)

type ReplicationResource struct {
	client *client.ManagementClient
}

func NewReplicationResource() resource.Resource {
	return &ReplicationResource{}
}

func (r *ReplicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_replication"
}

func (r *ReplicationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an ARTESCA replication stream between two buckets.",
		Attributes: map[string]schema.Attribute{
			"stream_id": schema.StringAttribute{
				Description: "The unique ID of the replication stream, assigned by ARTESCA.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the replication stream.",
				Required:    true,
			},
			"version": schema.Int64Attribute{
				Description: "The version of the replication stream. Auto-incremented by the server on each update.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					serverManagedVersion{},
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the replication stream is enabled.",
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

func (r *ReplicationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ReplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ReplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiStream := modelToAPIReplication(&plan)

	tflog.Debug(ctx, "Creating replication stream", map[string]any{"name": plan.Name.ValueString()})

	created, err := r.client.CreateReplicationStream(ctx, apiStream)
	if err != nil {
		resp.Diagnostics.AddError("Error creating replication stream", err.Error())
		return
	}

	apiReplicationToModel(created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ReplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ReplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	stream, err := r.client.GetReplicationStream(ctx, state.StreamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading replication stream", err.Error())
		return
	}
	if stream == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	apiReplicationToModel(stream, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ReplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ReplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ReplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiStream := modelToAPIReplication(&plan)
	apiStream.StreamID = state.StreamID.ValueString()

	tflog.Debug(ctx, "Updating replication stream", map[string]any{"stream_id": state.StreamID.ValueString()})

	updated, err := r.client.UpdateReplicationStream(ctx, state.StreamID.ValueString(), apiStream)
	if err != nil {
		resp.Diagnostics.AddError("Error updating replication stream", err.Error())
		return
	}

	apiReplicationToModel(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ReplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ReplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting replication stream", map[string]any{"stream_id": state.StreamID.ValueString()})

	err := r.client.DeleteReplicationStream(ctx, state.StreamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting replication stream", err.Error())
		return
	}
}

func (r *ReplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("stream_id"), req, resp)
}

// serverManagedVersion marks version as unknown during updates since the API auto-increments it.
type serverManagedVersion struct{}

func (m serverManagedVersion) Description(_ context.Context) string {
	return "Server auto-increments version on each update."
}

func (m serverManagedVersion) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m serverManagedVersion) PlanModifyInt64(_ context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	if !req.Plan.Raw.Equal(req.State.Raw) {
		resp.PlanValue = types.Int64Unknown()
	}
}

// --- Conversion helpers ---

func modelToAPIReplication(model *ReplicationResourceModel) *client.ReplicationStream {
	version := model.Version.ValueInt64()
	if model.Version.IsNull() || model.Version.IsUnknown() {
		version = 1
	}
	stream := &client.ReplicationStream{
		Name:    model.Name.ValueString(),
		Version: version,
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
			dl := client.ReplicationDestLocation{
				Name: loc.Name.ValueString(),
			}
			if !loc.StorageClass.IsNull() && !loc.StorageClass.IsUnknown() {
				dl.StorageClass = loc.StorageClass.ValueString()
			}
			stream.Destination.Locations = append(stream.Destination.Locations, dl)
		}
	}

	return stream
}

func apiReplicationToModel(stream *client.ReplicationStream, model *ReplicationResourceModel) {
	model.StreamID = types.StringValue(stream.StreamID)
	model.Name = types.StringValue(stream.Name)
	model.Version = types.Int64Value(stream.Version)
	model.Enabled = types.BoolValue(stream.Enabled)

	if stream.Source != nil {
		model.Source = &ReplicationSourceModel{
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
		model.Destination = &ReplicationDestModel{}
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
		model.Destination.Locations = nil
		for _, loc := range stream.Destination.Locations {
			dl := ReplicationDestLocationModel{
				Name: types.StringValue(loc.Name),
			}
			if loc.StorageClass != "" {
				dl.StorageClass = types.StringValue(loc.StorageClass)
			} else {
				dl.StorageClass = types.StringNull()
			}
			model.Destination.Locations = append(model.Destination.Locations, dl)
		}
	}
}
