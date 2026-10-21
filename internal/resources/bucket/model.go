package bucket

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type BucketResourceModel struct {
	Name               types.String `tfsdk:"name"`
	LocationConstraint types.String `tfsdk:"location_constraint"`
	VersioningEnabled  types.Bool   `tfsdk:"versioning_enabled"`
	ObjectLockEnabled  types.Bool   `tfsdk:"object_lock_enabled"`
	AccountName        types.String `tfsdk:"account_name"`
	ForceDestroy       types.Bool   `tfsdk:"force_destroy"`
}
