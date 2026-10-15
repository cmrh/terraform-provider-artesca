package account

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// present is a non-null resource object, standing in for existing state or a plan.
var present = tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{}}, map[string]tftypes.Value{})

func TestEmailPlanModifier(t *testing.T) {
	null, str := types.StringNull(), types.StringValue
	for _, tc := range []struct {
		name          string
		create        bool
		state, config types.String
		wantPlan      types.String
		wantReplace   bool
		wantError     bool
	}{
		{"create without email", true, null, null, null, false, true},
		{"create with email", true, null, str("a@x.io"), str("a@x.io"), false, false},
		{"unchanged", false, str("a@x.io"), str("a@x.io"), str("a@x.io"), false, false},
		{"removed from config keeps the value", false, str("a@x.io"), null, str("a@x.io"), false, false},
		{"changed replaces", false, str("a@x.io"), str("b@x.io"), str("b@x.io"), true, false},
		{"set on an imported account is adopted", false, null, str("a@x.io"), str("a@x.io"), false, false},
		{"imported account still unset", false, null, null, null, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := planmodifier.StringRequest{
				StateValue:  tc.state,
				ConfigValue: tc.config,
				PlanValue:   tc.config,
				Plan:        tfsdk.Plan{Raw: present},
			}
			if !tc.create {
				req.State = tfsdk.State{Raw: present}
			}
			if tc.config.IsNull() && !tc.create {
				req.PlanValue = types.StringUnknown() // computed attribute with no config
			}
			resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
			emailPlanModifier{}.PlanModifyString(context.Background(), req, resp)
			if resp.Diagnostics.HasError() != tc.wantError {
				t.Fatalf("diagnostics: %v; want error = %v", resp.Diagnostics, tc.wantError)
			}
			if !resp.PlanValue.Equal(tc.wantPlan) || resp.RequiresReplace != tc.wantReplace {
				t.Errorf("plan = %s, replace = %v; want %s, %v", resp.PlanValue, resp.RequiresReplace, tc.wantPlan, tc.wantReplace)
			}
		})
	}
}
