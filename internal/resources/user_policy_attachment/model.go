package userpolicyattachment

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type UserPolicyAttachmentResourceModel struct {
	AccountName types.String `tfsdk:"account_name"`
	Username    types.String `tfsdk:"username"`
	PolicyArn   types.String `tfsdk:"policy_arn"`
}
