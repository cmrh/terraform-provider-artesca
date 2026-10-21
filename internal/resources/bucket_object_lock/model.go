package bucketobjectlock

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type BucketObjectLockResourceModel struct {
	AccountName    types.String `tfsdk:"account_name"`
	BucketName     types.String `tfsdk:"bucket_name"`
	RetentionMode  types.String `tfsdk:"retention_mode"`
	RetentionDays  types.Int64  `tfsdk:"retention_days"`
	RetentionYears types.Int64  `tfsdk:"retention_years"`
}
