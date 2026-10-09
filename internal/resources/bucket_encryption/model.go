package bucketencryption

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type BucketEncryptionResourceModel struct {
	AccountName  types.String `tfsdk:"account_name"`
	BucketName   types.String `tfsdk:"bucket_name"`
	SSEAlgorithm types.String `tfsdk:"sse_algorithm"`
}
