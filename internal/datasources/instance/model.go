package instance

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type InstanceDataSourceModel struct {
	InstanceID                  types.String `tfsdk:"instance_id"`
	CreatedAt                   types.String `tfsdk:"created_at"`
	State                       types.String `tfsdk:"state"`
	PublicKey                   types.String `tfsdk:"public_key"`
	IPAddress                   types.String `tfsdk:"ip_address"`
	LastSeen                    types.String `tfsdk:"last_seen"`
	RunningConfigurationVersion types.Int64  `tfsdk:"running_configuration_version"`
	ServerVersion               types.String `tfsdk:"server_version"`
}
