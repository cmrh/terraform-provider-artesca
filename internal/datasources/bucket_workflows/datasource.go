package bucket_workflows

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/scality/terraform-provider-scality-artesca/internal/client"
)

var _ datasource.DataSource = &BucketWorkflowsDataSource{}

type BucketWorkflowsDataSource struct {
	client *client.ManagementClient
}

func NewBucketWorkflowsDataSource() datasource.DataSource {
	return &BucketWorkflowsDataSource{}
}

func (d *BucketWorkflowsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket_workflows"
}

func (d *BucketWorkflowsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the workflows (replication, lifecycle expiration, transition) configured on an ARTESCA bucket. Useful for asserting on or iterating over the workflow set without managing each one individually.",
		Attributes: map[string]schema.Attribute{
			"instance_id": schema.StringAttribute{
				Description: "The instance ID. Defaults to the provider's instance_id if omitted.",
				Optional:    true,
				Computed:    true,
			},
			"account_id": schema.StringAttribute{
				Description: "The account ID that owns the bucket.",
				Required:    true,
			},
			"bucket_name": schema.StringAttribute{
				Description: "The bucket whose workflows should be listed.",
				Required:    true,
			},
			"replications": schema.ListNestedAttribute{
				Description: "Replication workflows configured on the bucket. Note: name and version are not returned by the workflow-search endpoint — use the artesca_bucket_workflow_replication resource if you need those.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"workflow_id":                         schema.StringAttribute{Description: "Workflow (stream) ID.", Computed: true},
						"enabled":                             schema.BoolAttribute{Description: "Whether the workflow is enabled.", Computed: true},
						"source_bucket_name":                  schema.StringAttribute{Description: "Source bucket name.", Computed: true},
						"source_prefix":                       schema.StringAttribute{Description: "Object key prefix filter.", Computed: true},
						"source_location":                     schema.StringAttribute{Description: "Source location name (if set).", Computed: true},
						"destination_bucket_name":             schema.StringAttribute{Description: "Destination bucket name (if set).", Computed: true},
						"destination_location":                schema.StringAttribute{Description: "Destination location name (if set).", Computed: true},
						"destination_preferred_read_location": schema.StringAttribute{Description: "Preferred read location (if set).", Computed: true},
						"destination_role":                    schema.StringAttribute{Description: "IAM role for replication (if set).", Computed: true},
					},
				},
			},
			"expirations": schema.ListNestedAttribute{
				Description: "Lifecycle expiration workflows configured on the bucket.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"workflow_id": schema.StringAttribute{Description: "Workflow ID.", Computed: true},
						"name":        schema.StringAttribute{Description: "Workflow name.", Computed: true},
						"bucket_name": schema.StringAttribute{Description: "Bucket name.", Computed: true},
						"type":        schema.StringAttribute{Description: "Workflow type.", Computed: true},
						"enabled":     schema.BoolAttribute{Description: "Whether the workflow is enabled.", Computed: true},
					},
				},
			},
			"transitions": schema.ListNestedAttribute{
				Description: "Lifecycle transition workflows configured on the bucket.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"workflow_id":      schema.StringAttribute{Description: "Workflow ID.", Computed: true},
						"name":             schema.StringAttribute{Description: "Workflow name.", Computed: true},
						"bucket_name":      schema.StringAttribute{Description: "Bucket name.", Computed: true},
						"type":             schema.StringAttribute{Description: "Workflow type.", Computed: true},
						"enabled":          schema.BoolAttribute{Description: "Whether the workflow is enabled.", Computed: true},
						"location_name":    schema.StringAttribute{Description: "Destination location for the transition.", Computed: true},
						"apply_to_version": schema.StringAttribute{Description: "\"current\" or \"noncurrent\".", Computed: true},
					},
				},
			},
		},
	}
}

func (d *BucketWorkflowsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	providerData, ok := req.ProviderData.(*client.ProviderClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.ProviderClients, got: %T", req.ProviderData),
		)
		return
	}
	d.client = providerData.Management
}

func (d *BucketWorkflowsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data BucketWorkflowsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instanceID := data.InstanceID.ValueString()
	if instanceID == "" {
		instanceID = d.client.InstanceID
	}
	accountID := data.AccountID.ValueString()
	bucketName := data.BucketName.ValueString()

	results, err := d.client.SearchWorkflows(ctx, instanceID, accountID, []string{bucketName})
	if err != nil {
		resp.Diagnostics.AddError("Error listing bucket workflows", err.Error())
		return
	}

	out := BucketWorkflowsDataSourceModel{
		InstanceID:   types.StringValue(instanceID),
		AccountID:    data.AccountID,
		BucketName:   data.BucketName,
		Replications: make([]ReplicationSummary, 0),
		Expirations:  make([]ExpirationSummary, 0),
		Transitions:  make([]TransitionSummary, 0),
	}

	for _, item := range results {
		switch {
		case item.Replication != nil:
			out.Replications = append(out.Replications, replicationToSummary(item.Replication))
		case item.Expiration != nil:
			out.Expirations = append(out.Expirations, expirationToSummary(item.Expiration))
		case item.Transition != nil:
			out.Transitions = append(out.Transitions, transitionToSummary(item.Transition))
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}

func replicationToSummary(s *client.ReplicationStream) ReplicationSummary {
	r := ReplicationSummary{
		WorkflowID: types.StringValue(s.StreamID),
		Enabled:    types.BoolValue(s.Enabled),
	}
	if s.Source != nil {
		r.SourceBucketName = types.StringValue(s.Source.BucketName)
		r.SourcePrefix = types.StringValue(s.Source.Prefix)
		r.SourceLocation = types.StringValue(s.Source.Location)
	} else {
		r.SourceBucketName = types.StringValue("")
		r.SourcePrefix = types.StringValue("")
		r.SourceLocation = types.StringValue("")
	}
	if s.Destination != nil {
		r.DestinationBucketName = types.StringValue(s.Destination.BucketName)
		r.DestinationLocation = types.StringValue(s.Destination.Location)
		r.DestinationPreferredReadLocation = types.StringValue(s.Destination.PreferredReadLocation)
		r.DestinationRole = types.StringValue(s.Destination.Role)
	} else {
		r.DestinationBucketName = types.StringValue("")
		r.DestinationLocation = types.StringValue("")
		r.DestinationPreferredReadLocation = types.StringValue("")
		r.DestinationRole = types.StringValue("")
	}
	return r
}

func expirationToSummary(e *client.BucketWorkflowExpiration) ExpirationSummary {
	return ExpirationSummary{
		WorkflowID: types.StringValue(e.WorkflowID),
		Name:       types.StringValue(e.Name),
		BucketName: types.StringValue(e.BucketName),
		Type:       types.StringValue(e.Type),
		Enabled:    types.BoolValue(e.Enabled),
	}
}

func transitionToSummary(t *client.BucketWorkflowTransition) TransitionSummary {
	return TransitionSummary{
		WorkflowID:     types.StringValue(t.WorkflowID),
		Name:           types.StringValue(t.Name),
		BucketName:     types.StringValue(t.BucketName),
		Type:           types.StringValue(t.Type),
		Enabled:        types.BoolValue(t.Enabled),
		LocationName:   types.StringValue(t.LocationName),
		ApplyToVersion: types.StringValue(t.ApplyToVersion),
	}
}
