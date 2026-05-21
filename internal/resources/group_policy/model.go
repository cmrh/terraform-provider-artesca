package grouppolicy

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type GroupPolicyResourceModel struct {
	AccountAccessKey types.String `tfsdk:"account_access_key"`
	AccountSecretKey types.String `tfsdk:"account_secret_key"`
	GroupName        types.String `tfsdk:"group_name"`
	PolicyName       types.String `tfsdk:"policy_name"`
	PolicyDocument   types.String `tfsdk:"policy_document"`
}
