package bucket

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type BucketResourceModel struct {
	Name               types.String `tfsdk:"name"`
	LocationConstraint types.String `tfsdk:"location_constraint"`
	AccountAccessKey   types.String `tfsdk:"account_access_key"`
	AccountSecretKey   types.String `tfsdk:"account_secret_key"`
}
