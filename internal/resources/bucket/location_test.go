package bucket

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestLocationChanged(t *testing.T) {
	null, str, unknown := types.StringNull(), types.StringValue, types.StringUnknown()
	for _, tc := range []struct {
		name        string
		state, plan types.String
		want        bool
	}{
		{"imported default to us-east-1", null, str("us-east-1"), false},
		{"imported default to empty", null, str(""), false},
		{"us-east-1 to unset", str("us-east-1"), null, false},
		{"empty to us-east-1", str(""), str("us-east-1"), false},
		{"default to a location", null, str("ring"), true},
		{"location to default", str("ring"), str("us-east-1"), true},
		{"location to another", str("ring"), str("ring2"), true},
		{"default to unknown", null, unknown, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := &stringplanmodifier.RequiresReplaceIfFuncResponse{}
			locationChanged(context.Background(), planmodifier.StringRequest{StateValue: tc.state, PlanValue: tc.plan}, resp)
			if resp.RequiresReplace != tc.want {
				t.Errorf("RequiresReplace = %v, want %v", resp.RequiresReplace, tc.want)
			}
		})
	}
}
