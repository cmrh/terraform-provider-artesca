package bucketpolicy

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type BucketPolicyResourceModel struct {
	AccountAccessKey types.String `tfsdk:"account_access_key"`
	AccountSecretKey types.String `tfsdk:"account_secret_key"`
	BucketName       types.String `tfsdk:"bucket_name"`
	Policy           types.String `tfsdk:"policy"`
}
