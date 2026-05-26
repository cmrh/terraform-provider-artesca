package user

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type UserDataSourceModel struct {
	AccountAccessKey types.String `tfsdk:"account_access_key"`
	AccountSecretKey types.String `tfsdk:"account_secret_key"`
	Username         types.String `tfsdk:"username"`
	UserID           types.String `tfsdk:"user_id"`
	ARN              types.String `tfsdk:"arn"`
	Path             types.String `tfsdk:"path"`
}
