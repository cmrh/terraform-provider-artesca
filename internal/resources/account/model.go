package account

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AccountResourceModel struct {
	Name        types.String `tfsdk:"name"`
	Email       types.String `tfsdk:"email"`
	AccessKey   types.String `tfsdk:"access_key"`
	SecretKey   types.String `tfsdk:"secret_key"`
	ARN         types.String `tfsdk:"arn"`
	CanonicalID types.String `tfsdk:"canonical_id"`
	ID          types.String `tfsdk:"id"`
}
