package group

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type GroupResourceModel struct {
	AccountAccessKey types.String `tfsdk:"account_access_key"`
	AccountSecretKey types.String `tfsdk:"account_secret_key"`
	Name             types.String `tfsdk:"name"`
	GroupID          types.String `tfsdk:"group_id"`
	ARN              types.String `tfsdk:"arn"`
	Path             types.String `tfsdk:"path"`
}
