package workflowreplication

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type WorkflowReplicationResourceModel struct {
	InstanceID  types.String                    `tfsdk:"instance_id"`
	AccountID   types.String                    `tfsdk:"account_id"`
	BucketName  types.String                    `tfsdk:"bucket_name"`
	WorkflowID  types.String                    `tfsdk:"workflow_id"`
	Name        types.String                    `tfsdk:"name"`
	Version     types.Int64                     `tfsdk:"version"`
	Enabled     types.Bool                      `tfsdk:"enabled"`
	Source      *WorkflowReplicationSourceModel `tfsdk:"source"`
	Destination *WorkflowReplicationDestModel   `tfsdk:"destination"`
}

type WorkflowReplicationSourceModel struct {
	BucketName types.String `tfsdk:"bucket_name"`
	Prefix     types.String `tfsdk:"prefix"`
	Location   types.String `tfsdk:"location"`
}

type WorkflowReplicationDestModel struct {
	BucketName            types.String                         `tfsdk:"bucket_name"`
	Location              types.String                         `tfsdk:"location"`
	Locations             []WorkflowReplicationDestLocModel    `tfsdk:"locations"`
	PreferredReadLocation types.String                         `tfsdk:"preferred_read_location"`
	Role                  types.String                         `tfsdk:"role"`
}

type WorkflowReplicationDestLocModel struct {
	Name         types.String `tfsdk:"name"`
	StorageClass types.String `tfsdk:"storage_class"`
}
