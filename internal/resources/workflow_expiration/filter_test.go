package workflowexpiration

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
		{"tags removed on the server", &WorkflowFilterModel{ObjectKeyPrefix: types.StringNull(), ObjectTags: tagMap}, "", nil, &WorkflowFilterModel{ObjectKeyPrefix: types.StringNull(), ObjectTags: nullTags}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := &WorkflowExpirationResourceModel{Filter: tc.filter}
			if diags := lifecycleRuleToModel(context.Background(), &client.LifecycleRule{Status: "Enabled", Prefix: tc.rulePrefix, Tags: tc.ruleTags, ExpirationDays: 1}, m); diags.HasError() {
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
