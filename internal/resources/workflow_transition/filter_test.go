package workflowtransition

import (
	"testing"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestLifecycleRuleToModelFilter(t *testing.T) {
	cases := []struct {
		name       string
		filter     *WorkflowFilterModel
		rulePrefix string
		wantFilter *WorkflowFilterModel
	}{
		{"no filter, no prefix", nil, "", nil},
		{"no filter (import), server prefix surfaces", nil, "logs/", &WorkflowFilterModel{ObjectKeyPrefix: types.StringValue("logs/")}},
		{"filter with null prefix stays null", &WorkflowFilterModel{ObjectKeyPrefix: types.StringNull()}, "", &WorkflowFilterModel{ObjectKeyPrefix: types.StringNull()}},
		{"filter prefix updated from server", &WorkflowFilterModel{ObjectKeyPrefix: types.StringValue("a/")}, "b/", &WorkflowFilterModel{ObjectKeyPrefix: types.StringValue("b/")}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := &WorkflowTransitionResourceModel{Filter: tc.filter}
			lifecycleRuleToModel(&client.LifecycleRule{Status: "Enabled", Prefix: tc.rulePrefix}, m)
			switch {
			case tc.wantFilter == nil && m.Filter != nil:
				t.Errorf("filter = %+v, want nil", m.Filter)
			case tc.wantFilter != nil && (m.Filter == nil || !m.Filter.ObjectKeyPrefix.Equal(tc.wantFilter.ObjectKeyPrefix)):
				t.Errorf("filter = %+v, want %+v", m.Filter, tc.wantFilter)
			}
		})
	}
}
