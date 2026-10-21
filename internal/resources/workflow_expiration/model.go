package workflowexpiration

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type WorkflowExpirationResourceModel struct {
	AccountName types.String         `tfsdk:"account_name"`
	BucketName  types.String         `tfsdk:"bucket_name"`
	RuleID      types.String         `tfsdk:"rule_id"`
	Enabled     types.Bool           `tfsdk:"enabled"`
	Filter      *WorkflowFilterModel `tfsdk:"filter"`

	CurrentVersionTriggerDelayDays            types.Int64  `tfsdk:"current_version_trigger_delay_days"`
	CurrentVersionTriggerDelayDate            types.String `tfsdk:"current_version_trigger_delay_date"`
	ExpireDeleteMarkers                       types.Bool   `tfsdk:"expire_delete_markers"`
	PreviousVersionTriggerDelayDays           types.Int64  `tfsdk:"previous_version_trigger_delay_days"`
	IncompleteMultipartUploadTriggerDelayDays types.Int64  `tfsdk:"incomplete_multipart_upload_trigger_delay_days"`
}

type WorkflowFilterModel struct {
	ObjectKeyPrefix types.String `tfsdk:"object_key_prefix"`
	ObjectTags      types.Map    `tfsdk:"object_tags"`
}
