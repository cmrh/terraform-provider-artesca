package assumedrolecredentials

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AssumedRoleCredentialsModel struct {
	// Caller credentials (the user/role calling AssumeRole).
	AccessKey types.String `tfsdk:"access_key"`
	SecretKey types.String `tfsdk:"secret_key"`

	// AssumeRole parameters.
	RoleArn         types.String `tfsdk:"role_arn"`
	RoleSessionName types.String `tfsdk:"role_session_name"`
	DurationSeconds types.Int64  `tfsdk:"duration_seconds"`
	ExternalID      types.String `tfsdk:"external_id"`
	Policy          types.String `tfsdk:"policy"`

	// Returned credentials.
	AccessKeyID     types.String `tfsdk:"access_key_id"`
	SecretAccessKey types.String `tfsdk:"secret_access_key"`
	SessionToken    types.String `tfsdk:"session_token"`
	Expiration      types.String `tfsdk:"expiration"`

	// Returned identity.
	AssumedRoleID  types.String `tfsdk:"assumed_role_id"`
	AssumedRoleArn types.String `tfsdk:"assumed_role_arn"`
}
