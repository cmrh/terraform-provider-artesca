package bucket_workflows

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type BucketWorkflowsDataSourceModel struct {
	InstanceID   types.String         `tfsdk:"instance_id"`
	AccountID    types.String         `tfsdk:"account_id"`
	BucketName   types.String         `tfsdk:"bucket_name"`
	Replications []ReplicationSummary `tfsdk:"replications"`
	Expirations  []ExpirationSummary  `tfsdk:"expirations"`
	Transitions  []TransitionSummary  `tfsdk:"transitions"`
}

type ReplicationSummary struct {
	WorkflowID                       types.String `tfsdk:"workflow_id"`
	Enabled                          types.Bool   `tfsdk:"enabled"`
	SourceBucketName                 types.String `tfsdk:"source_bucket_name"`
	SourcePrefix                     types.String `tfsdk:"source_prefix"`
	SourceLocation                   types.String `tfsdk:"source_location"`
	DestinationBucketName            types.String `tfsdk:"destination_bucket_name"`
	DestinationLocation              types.String `tfsdk:"destination_location"`
	DestinationPreferredReadLocation types.String `tfsdk:"destination_preferred_read_location"`
	DestinationRole                  types.String `tfsdk:"destination_role"`
}

type ExpirationSummary struct {
	WorkflowID types.String `tfsdk:"workflow_id"`
	Name       types.String `tfsdk:"name"`
	BucketName types.String `tfsdk:"bucket_name"`
	Type       types.String `tfsdk:"type"`
	Enabled    types.Bool   `tfsdk:"enabled"`
}

type TransitionSummary struct {
	WorkflowID     types.String `tfsdk:"workflow_id"`
	Name           types.String `tfsdk:"name"`
	BucketName     types.String `tfsdk:"bucket_name"`
	Type           types.String `tfsdk:"type"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	LocationName   types.String `tfsdk:"location_name"`
	ApplyToVersion types.String `tfsdk:"apply_to_version"`
}
