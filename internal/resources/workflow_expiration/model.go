package workflowexpiration

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type WorkflowExpirationResourceModel struct {
	InstanceID  types.String       `tfsdk:"instance_id"`
	AccountID   types.String       `tfsdk:"account_id"`
	BucketName  types.String       `tfsdk:"bucket_name"`
	WorkflowID  types.String       `tfsdk:"workflow_id"`
	Name        types.String       `tfsdk:"name"`
	Enabled     types.Bool         `tfsdk:"enabled"`
	Filter      *WorkflowFilterModel `tfsdk:"filter"`

	CurrentVersionTriggerDelayDate             types.String `tfsdk:"current_version_trigger_delay_date"`
	CurrentVersionTriggerDelayDays             types.Int64  `tfsdk:"current_version_trigger_delay_days"`
	ExpireDeleteMarkersTrigger                 types.Bool   `tfsdk:"expire_delete_markers_trigger"`
	IncompleteMultipartUploadTriggerDelayDays  types.Int64  `tfsdk:"incomplete_multipart_upload_trigger_delay_days"`
	PreviousVersionTriggerDelayDays            types.Int64  `tfsdk:"previous_version_trigger_delay_days"`
}

type WorkflowFilterModel struct {
	ObjectKeyPrefix types.String       `tfsdk:"object_key_prefix"`
	ObjectTags      []WorkflowTagModel `tfsdk:"object_tags"`
}

type WorkflowTagModel struct {
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}
