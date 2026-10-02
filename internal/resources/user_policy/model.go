package userpolicy

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type UserPolicyResourceModel struct {
	AccountName    types.String `tfsdk:"account_name"`
	Username       types.String `tfsdk:"username"`
	PolicyName     types.String `tfsdk:"policy_name"`
	PolicyDocument types.String `tfsdk:"policy_document"`
}
