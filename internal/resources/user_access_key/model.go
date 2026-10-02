package useraccesskey

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type UserAccessKeyResourceModel struct {
	AccountName     types.String `tfsdk:"account_name"`
	Username        types.String `tfsdk:"username"`
	AccessKeyID     types.String `tfsdk:"access_key_id"`
	SecretAccessKey types.String `tfsdk:"secret_access_key"`
	Status          types.String `tfsdk:"status"`
}
