package caller_identity

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type CallerIdentityDataSourceModel struct {
	AccessKey    types.String `tfsdk:"access_key"`
	SecretKey    types.String `tfsdk:"secret_key"`
	SessionToken types.String `tfsdk:"session_token"`
	UserID       types.String `tfsdk:"user_id"`
	Account      types.String `tfsdk:"account"`
	ARN          types.String `tfsdk:"arn"`
}
