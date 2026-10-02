package buckettagging

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type BucketTaggingResourceModel struct {
	AccountName types.String `tfsdk:"account_name"`
	BucketName  types.String `tfsdk:"bucket_name"`
	Tags        types.Map    `tfsdk:"tags"`
}
