package role

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type RoleDataSourceModel struct {
	AccountAccessKey         types.String `tfsdk:"account_access_key"`
	AccountSecretKey         types.String `tfsdk:"account_secret_key"`
	Name                     types.String `tfsdk:"name"`
	RoleID                   types.String `tfsdk:"role_id"`
	ARN                      types.String `tfsdk:"arn"`
	Path                     types.String `tfsdk:"path"`
	AssumeRolePolicyDocument types.String `tfsdk:"assume_role_policy_document"`
	Description              types.String `tfsdk:"description"`
}
