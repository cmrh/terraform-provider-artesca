package rolepolicyattachment

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type RolePolicyAttachmentResourceModel struct {
	AccountName types.String `tfsdk:"account_name"`
	RoleName    types.String `tfsdk:"role_name"`
	PolicyArn   types.String `tfsdk:"policy_arn"`
}
