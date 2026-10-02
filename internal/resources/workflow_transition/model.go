package workflowtransition

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type WorkflowTransitionResourceModel struct {
	AccountName      types.String         `tfsdk:"account_name"`
	BucketName       types.String         `tfsdk:"bucket_name"`
	RuleID           types.String         `tfsdk:"rule_id"`
	Enabled          types.Bool           `tfsdk:"enabled"`
	LocationName     types.String         `tfsdk:"location_name"`
	TriggerDelayDays types.Int64          `tfsdk:"trigger_delay_days"`
	Filter           *WorkflowFilterModel `tfsdk:"filter"`
}

type WorkflowFilterModel struct {
	ObjectKeyPrefix types.String `tfsdk:"object_key_prefix"`
}
