package bucketencryption

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type BucketEncryptionResourceModel struct {
	AccountAccessKey types.String `tfsdk:"account_access_key"`
	AccountSecretKey types.String `tfsdk:"account_secret_key"`
	BucketName       types.String `tfsdk:"bucket_name"`
	SSEAlgorithm     types.String `tfsdk:"sse_algorithm"`
	BucketKeyEnabled types.Bool   `tfsdk:"bucket_key_enabled"`
}
