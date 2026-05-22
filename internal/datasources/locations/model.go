package locations

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type LocationsDataSourceModel struct {
	Locations []LocationSummary `tfsdk:"locations"`
}

type LocationSummary struct {
	Name              types.String `tfsdk:"name"`
	LocationType      types.String `tfsdk:"location_type"`
	IsBuiltin         types.Bool   `tfsdk:"is_builtin"`
	IsTransient       types.Bool   `tfsdk:"is_transient"`
	LegacyAwsBehavior types.Bool   `tfsdk:"legacy_aws_behavior"`
	SizeLimitGB       types.Int64  `tfsdk:"size_limit_gb"`
	ObjectID          types.String `tfsdk:"object_id"`
}
