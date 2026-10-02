package account

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AccountDataSourceModel struct {
	Name        types.String `tfsdk:"name"`
	ID          types.String `tfsdk:"id"`
	CanonicalID types.String `tfsdk:"canonical_id"`
	ARN         types.String `tfsdk:"arn"`
}
