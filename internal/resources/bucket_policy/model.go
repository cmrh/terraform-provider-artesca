package bucketpolicy

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type BucketPolicyResourceModel struct {
	AccountName types.String `tfsdk:"account_name"`
	BucketName  types.String `tfsdk:"bucket_name"`
	Policy      types.String `tfsdk:"policy"`
}
