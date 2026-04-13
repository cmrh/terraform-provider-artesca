package workflowtransition

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type WorkflowTransitionResourceModel struct {
	InstanceID       types.String         `tfsdk:"instance_id"`
	AccountID        types.String         `tfsdk:"account_id"`
	BucketName       types.String         `tfsdk:"bucket_name"`
	WorkflowID       types.String         `tfsdk:"workflow_id"`
	Name             types.String         `tfsdk:"name"`
	Enabled          types.Bool           `tfsdk:"enabled"`
	LocationName     types.String         `tfsdk:"location_name"`
	ApplyToVersion   types.String         `tfsdk:"apply_to_version"`
	TriggerDelayDate types.String         `tfsdk:"trigger_delay_date"`
	TriggerDelayDays types.Int64          `tfsdk:"trigger_delay_days"`
	Filter           *WorkflowFilterModel `tfsdk:"filter"`
}

type WorkflowFilterModel struct {
	ObjectKeyPrefix types.String       `tfsdk:"object_key_prefix"`
	ObjectTags      []WorkflowTagModel `tfsdk:"object_tags"`
}

type WorkflowTagModel struct {
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}
