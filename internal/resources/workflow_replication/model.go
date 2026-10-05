package workflowreplication

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type WorkflowReplicationResourceModel struct {
	AccountName           types.String         `tfsdk:"account_name"`
	BucketName            types.String         `tfsdk:"bucket_name"`
	RuleID                types.String         `tfsdk:"rule_id"`
	Enabled               types.Bool           `tfsdk:"enabled"`
	DestinationBucketName types.String         `tfsdk:"destination_bucket_name"`
	Filter                *WorkflowFilterModel `tfsdk:"filter"`
}

type WorkflowFilterModel struct {
	ObjectKeyPrefix types.String `tfsdk:"object_key_prefix"`
}
