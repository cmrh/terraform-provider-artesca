package accounts

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AccountsDataSourceModel struct {
	Accounts []AccountSummary `tfsdk:"accounts"`
}

type AccountSummary struct {
	Name        types.String `tfsdk:"name"`
	ID          types.String `tfsdk:"id"`
	CanonicalID types.String `tfsdk:"canonical_id"`
	ARN         types.String `tfsdk:"arn"`
	Email       types.String `tfsdk:"email"`
}
