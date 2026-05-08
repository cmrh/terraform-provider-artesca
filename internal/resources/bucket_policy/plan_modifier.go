package bucketpolicy

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// policyEquivalenceModifier suppresses planned changes when the prior state
// and the new config differ only in JSON formatting (whitespace, key order)
// but represent the same policy.
type policyEquivalenceModifier struct{}

func (m policyEquivalenceModifier) Description(_ context.Context) string {
	return "Suppress diff when JSON policy documents are semantically equivalent."
}

func (m policyEquivalenceModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m policyEquivalenceModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}
	if jsonEquivalent(req.StateValue.ValueString(), req.PlanValue.ValueString()) {
		resp.PlanValue = req.StateValue
	}
}
