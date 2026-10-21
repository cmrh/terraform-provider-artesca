package bucket

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type BucketDataSourceModel struct {
	AccountName        types.String `tfsdk:"account_name"`
	Name               types.String `tfsdk:"name"`
	ARN                types.String `tfsdk:"arn"`
	LocationConstraint types.String `tfsdk:"location_constraint"`
	VersioningEnabled  types.Bool   `tfsdk:"versioning_enabled"`
	ObjectLockEnabled  types.Bool   `tfsdk:"object_lock_enabled"`
	Tags               types.Map    `tfsdk:"tags"`
}
