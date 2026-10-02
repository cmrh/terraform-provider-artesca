package grouppolicyattachment

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type GroupPolicyAttachmentResourceModel struct {
	AccountName types.String `tfsdk:"account_name"`
	GroupName   types.String `tfsdk:"group_name"`
	PolicyArn   types.String `tfsdk:"policy_arn"`
}
