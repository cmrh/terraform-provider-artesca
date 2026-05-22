package endpoints

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type EndpointsDataSourceModel struct {
	Endpoints []EndpointSummary `tfsdk:"endpoints"`
}

type EndpointSummary struct {
	Hostname     types.String `tfsdk:"hostname"`
	LocationName types.String `tfsdk:"location_name"`
	IsBuiltin    types.Bool   `tfsdk:"is_builtin"`
}
