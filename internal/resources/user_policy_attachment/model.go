package userpolicyattachment

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type UserPolicyAttachmentResourceModel struct {
	AccountAccessKey types.String `tfsdk:"account_access_key"`
	AccountSecretKey types.String `tfsdk:"account_secret_key"`
	Username         types.String `tfsdk:"username"`
	PolicyArn        types.String `tfsdk:"policy_arn"`
}
