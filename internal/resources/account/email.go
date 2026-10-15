package account

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// emailPlanModifier plans email, which ARTESCA requires at creation and never
// returns. A null email in state therefore means an imported account:
//   - create: email is required;
//   - removed from config: keep the last known value (it can't be cleared);
//   - set on an imported account: adopt it without replacing the account;
//   - changed: replace the account.
type emailPlanModifier struct{}

func (m emailPlanModifier) Description(_ context.Context) string {
	return "Changing email replaces the account; removing it keeps the last known value."
}

func (m emailPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m emailPlanModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.Plan.Raw.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if req.State.Raw.IsNull() {
		if req.ConfigValue.IsNull() {
			resp.Diagnostics.AddAttributeError(req.Path, "Missing email", "email is required to create an account.")
		}
		return
	}
	if req.ConfigValue.IsNull() {
		resp.PlanValue = req.StateValue
		return
	}
	if req.ConfigValue.Equal(req.StateValue) || req.StateValue.IsNull() {
		return
	}
	resp.RequiresReplace = true
}

var _ planmodifier.String = emailPlanModifier{}
