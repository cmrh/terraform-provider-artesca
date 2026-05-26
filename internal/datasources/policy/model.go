package policy

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PolicyDataSourceModel struct {
	AccountAccessKey types.String `tfsdk:"account_access_key"`
	AccountSecretKey types.String `tfsdk:"account_secret_key"`
	ARN              types.String `tfsdk:"arn"`
	Name             types.String `tfsdk:"name"`
	PolicyID         types.String `tfsdk:"policy_id"`
	Path             types.String `tfsdk:"path"`
	DefaultVersionID types.String `tfsdk:"default_version_id"`
	Description      types.String `tfsdk:"description"`
	PolicyDocument   types.String `tfsdk:"policy_document"`
}
