package replication

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ReplicationResourceModel struct {
	StreamID    types.String            `tfsdk:"stream_id"`
	Name        types.String            `tfsdk:"name"`
	Version     types.Int64             `tfsdk:"version"`
	Enabled     types.Bool              `tfsdk:"enabled"`
	Source      *ReplicationSourceModel `tfsdk:"source"`
	Destination *ReplicationDestModel   `tfsdk:"destination"`
}

type ReplicationSourceModel struct {
	BucketName types.String `tfsdk:"bucket_name"`
	Prefix     types.String `tfsdk:"prefix"`
	Location   types.String `tfsdk:"location"`
}

type ReplicationDestModel struct {
	BucketName            types.String                   `tfsdk:"bucket_name"`
	Location              types.String                   `tfsdk:"location"`
	Locations             []ReplicationDestLocationModel `tfsdk:"locations"`
	PreferredReadLocation types.String                   `tfsdk:"preferred_read_location"`
	Role                  types.String                   `tfsdk:"role"`
}

type ReplicationDestLocationModel struct {
	Name         types.String `tfsdk:"name"`
	StorageClass types.String `tfsdk:"storage_class"`
}
