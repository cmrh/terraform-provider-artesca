package policy

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PolicyResourceModel struct {
	AccountAccessKey types.String `tfsdk:"account_access_key"`
	AccountSecretKey types.String `tfsdk:"account_secret_key"`
	Name             types.String `tfsdk:"name"`
	PolicyDocument   types.String `tfsdk:"policy_document"`
	Description      types.String `tfsdk:"description"`
	PolicyID         types.String `tfsdk:"policy_id"`
	ARN              types.String `tfsdk:"arn"`
	Path             types.String `tfsdk:"path"`
	DefaultVersionID types.String `tfsdk:"default_version_id"`
}
