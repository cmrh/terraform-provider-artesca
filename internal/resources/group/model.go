package group

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type GroupResourceModel struct {
	AccountName types.String `tfsdk:"account_name"`
	Name        types.String `tfsdk:"name"`
	GroupID     types.String `tfsdk:"group_id"`
	ARN         types.String `tfsdk:"arn"`
	Path        types.String `tfsdk:"path"`
}
