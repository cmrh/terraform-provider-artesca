package grouppolicy

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type GroupPolicyResourceModel struct {
	AccountName    types.String `tfsdk:"account_name"`
	GroupName      types.String `tfsdk:"group_name"`
	PolicyName     types.String `tfsdk:"policy_name"`
	PolicyDocument types.String `tfsdk:"policy_document"`
}
