package user

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type UserDataSourceModel struct {
	AccountName types.String `tfsdk:"account_name"`
	Username    types.String `tfsdk:"username"`
	UserID      types.String `tfsdk:"user_id"`
	ARN         types.String `tfsdk:"arn"`
	Path        types.String `tfsdk:"path"`
}
