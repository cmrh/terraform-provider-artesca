package role

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type RoleDataSourceModel struct {
	AccountName              types.String `tfsdk:"account_name"`
	Name                     types.String `tfsdk:"name"`
	RoleID                   types.String `tfsdk:"role_id"`
	ARN                      types.String `tfsdk:"arn"`
	Path                     types.String `tfsdk:"path"`
	AssumeRolePolicyDocument types.String `tfsdk:"assume_role_policy_document"`
	Description              types.String `tfsdk:"description"`
}
