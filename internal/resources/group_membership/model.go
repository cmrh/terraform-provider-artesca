package groupmembership

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type GroupMembershipResourceModel struct {
	AccountName types.String `tfsdk:"account_name"`
	GroupName   types.String `tfsdk:"group_name"`
	Username    types.String `tfsdk:"username"`
}
