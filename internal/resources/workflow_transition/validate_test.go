package workflowtransition

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestValidateConfigTiming(t *testing.T) {
	r := NewWorkflowTransitionResource().(*WorkflowTransitionResource)
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	typ := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)

	for _, tc := range []struct {
		name       string
		apply      any
		days, date any
		wantErr    bool
	}{
		{"current days", "current", 10, nil, false},
		{"current date", nil, nil, "2027-01-01", false},
		{"both", "current", 10, "2027-01-01", true},
		{"neither", "current", nil, nil, true},
		{"noncurrent days", "noncurrent", 0, nil, false},
		{"noncurrent date", "noncurrent", nil, "2027-01-01", true},
		{"unknown days", "current", tftypes.UnknownValue, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vals := map[string]tftypes.Value{}
			for name, at := range typ.AttributeTypes {
				vals[name] = tftypes.NewValue(at, nil)
			}
			vals["apply_to_version"] = tftypes.NewValue(tftypes.String, tc.apply)
			vals["trigger_delay_days"] = tftypes.NewValue(tftypes.Number, tc.days)
			vals["trigger_delay_date"] = tftypes.NewValue(tftypes.String, tc.date)
			resp := &resource.ValidateConfigResponse{}
			r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(typ, vals)}}, resp)
			if resp.Diagnostics.HasError() != tc.wantErr {
				t.Errorf("diagnostics = %v, want error %v", resp.Diagnostics, tc.wantErr)
			}
		})
	}
}
