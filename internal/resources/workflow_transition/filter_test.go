package workflowtransition

import (
	"context"
	"testing"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestLifecycleRuleToModelFilter(t *testing.T) {
	tagMap := types.MapValueMust(types.StringType, map[string]attr.Value{"k": types.StringValue("v")})
	nullTags := types.MapNull(types.StringType)
	cases := []struct {
		name       string
		filter     *WorkflowFilterModel
		rulePrefix string
		ruleTags   []client.BucketTag
		wantFilter *WorkflowFilterModel
	}{
		{"no filter, no prefix", nil, "", nil, nil},
		{"no filter (import), server prefix surfaces", nil, "logs/", nil, &WorkflowFilterModel{ObjectKeyPrefix: types.StringValue("logs/"), ObjectTags: nullTags}},
		{"no filter (import), server tags surface", nil, "", []client.BucketTag{{Key: "k", Value: "v"}}, &WorkflowFilterModel{ObjectKeyPrefix: types.StringNull(), ObjectTags: tagMap}},
		{"filter with null prefix stays null", &WorkflowFilterModel{ObjectKeyPrefix: types.StringNull(), ObjectTags: nullTags}, "", nil, &WorkflowFilterModel{ObjectKeyPrefix: types.StringNull(), ObjectTags: nullTags}},
		{"filter prefix updated from server", &WorkflowFilterModel{ObjectKeyPrefix: types.StringValue("a/"), ObjectTags: nullTags}, "b/", nil, &WorkflowFilterModel{ObjectKeyPrefix: types.StringValue("b/"), ObjectTags: nullTags}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := &WorkflowTransitionResourceModel{Filter: tc.filter, ApplyToVersion: types.StringValue(applyCurrent)}
			rule := &client.LifecycleRule{Status: "Enabled", Prefix: tc.rulePrefix, Tags: tc.ruleTags, TransitionLocation: "cold"}
			if diags := lifecycleRuleToModel(context.Background(), rule, m); diags.HasError() {
				t.Fatal(diags)
			}
			switch {
			case tc.wantFilter == nil && m.Filter != nil:
				t.Errorf("filter = %+v, want nil", m.Filter)
			case tc.wantFilter != nil && (m.Filter == nil || !m.Filter.ObjectKeyPrefix.Equal(tc.wantFilter.ObjectKeyPrefix) || !m.Filter.ObjectTags.Equal(tc.wantFilter.ObjectTags)):
				t.Errorf("filter = %+v, want %+v", m.Filter, tc.wantFilter)
			}
		})
	}
}

func TestLifecycleRuleToModelTransition(t *testing.T) {
	both := client.LifecycleRule{Status: "Enabled", TransitionDays: 10, TransitionLocation: "cold", NoncurrentTransitionDays: 3, NoncurrentTransitionLocation: "nc"}
	for _, tc := range []struct {
		name      string
		apply     types.String
		rule      client.LifecycleRule
		wantApply string
		wantLoc   string
		wantDays  types.Int64
		wantDate  types.String
	}{
		{"import picks the current transition", types.StringNull(), both, applyCurrent, "cold", types.Int64Value(10), types.StringNull()},
		{"import of a noncurrent-only rule", types.StringNull(), client.LifecycleRule{NoncurrentTransitionDays: 0, NoncurrentTransitionLocation: "nc"}, applyNoncurrent, "nc", types.Int64Value(0), types.StringNull()},
		{"noncurrent state reads the noncurrent transition", types.StringValue(applyNoncurrent), both, applyNoncurrent, "nc", types.Int64Value(3), types.StringNull()},
		{"date transition", types.StringValue(applyCurrent), client.LifecycleRule{TransitionDate: "2027-02-01", TransitionLocation: "cold"}, applyCurrent, "cold", types.Int64Null(), types.StringValue("2027-02-01")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := &WorkflowTransitionResourceModel{ApplyToVersion: tc.apply}
			if diags := lifecycleRuleToModel(context.Background(), &tc.rule, m); diags.HasError() {
				t.Fatal(diags)
			}
			if m.ApplyToVersion.ValueString() != tc.wantApply || m.LocationName.ValueString() != tc.wantLoc ||
				!m.TriggerDelayDays.Equal(tc.wantDays) || !m.TriggerDelayDate.Equal(tc.wantDate) {
				t.Errorf("model = apply %v loc %v days %v date %v", m.ApplyToVersion, m.LocationName, m.TriggerDelayDays, m.TriggerDelayDate)
			}
		})
	}
}
