package endpoint

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type EndpointResourceModel struct {
	Hostname     types.String `tfsdk:"hostname"`
	LocationName types.String `tfsdk:"location_name"`
	IsBuiltin    types.Bool   `tfsdk:"is_builtin"`
}
