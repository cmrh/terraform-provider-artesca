package workflowexpiration

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestValidateConfigActions(t *testing.T) {
	r := NewWorkflowExpirationResource().(*WorkflowExpirationResource)
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	typ := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	filterType := typ.AttributeTypes["filter"].(tftypes.Object)
	tagged := tftypes.NewValue(filterType, map[string]tftypes.Value{
		"object_key_prefix": tftypes.NewValue(tftypes.String, nil),
		"object_tags":       tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, map[string]tftypes.Value{"k": tftypes.NewValue(tftypes.String, "v")}),
	})

	for _, tc := range []struct {
		name    string
		set     map[string]tftypes.Value
		wantErr bool
	}{
		{"days", map[string]tftypes.Value{"current_version_trigger_delay_days": tftypes.NewValue(tftypes.Number, 30)}, false},
		{"date", map[string]tftypes.Value{"current_version_trigger_delay_date": tftypes.NewValue(tftypes.String, "2027-01-01")}, false},
		{"noncurrent only", map[string]tftypes.Value{"previous_version_trigger_delay_days": tftypes.NewValue(tftypes.Number, 30)}, false},
		{"multipart only", map[string]tftypes.Value{"incomplete_multipart_upload_trigger_delay_days": tftypes.NewValue(tftypes.Number, 7)}, false},
		{"delete markers only", map[string]tftypes.Value{"expire_delete_markers": tftypes.NewValue(tftypes.Bool, true)}, false},
		{"no action", map[string]tftypes.Value{}, true},
		{"delete markers false is no action", map[string]tftypes.Value{"expire_delete_markers": tftypes.NewValue(tftypes.Bool, false)}, true},
		{"days and date", map[string]tftypes.Value{
			"current_version_trigger_delay_days": tftypes.NewValue(tftypes.Number, 30),
			"current_version_trigger_delay_date": tftypes.NewValue(tftypes.String, "2027-01-01")}, true},
		{"delete markers and days", map[string]tftypes.Value{
			"expire_delete_markers":              tftypes.NewValue(tftypes.Bool, true),
			"current_version_trigger_delay_days": tftypes.NewValue(tftypes.Number, 30)}, true},
		{"tags and delete markers", map[string]tftypes.Value{
			"expire_delete_markers": tftypes.NewValue(tftypes.Bool, true), "filter": tagged}, true},
		{"tags and multipart", map[string]tftypes.Value{
			"incomplete_multipart_upload_trigger_delay_days": tftypes.NewValue(tftypes.Number, 7), "filter": tagged}, true},
		{"tags and noncurrent", map[string]tftypes.Value{
			"previous_version_trigger_delay_days": tftypes.NewValue(tftypes.Number, 7), "filter": tagged}, false},
		{"unknown action", map[string]tftypes.Value{"current_version_trigger_delay_days": tftypes.NewValue(tftypes.Number, tftypes.UnknownValue)}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vals := map[string]tftypes.Value{}
			for name, at := range typ.AttributeTypes {
				vals[name] = tftypes.NewValue(at, nil)
			}
			for k, v := range tc.set {
				vals[k] = v
			}
			resp := &resource.ValidateConfigResponse{}
			r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(typ, vals)}}, resp)
			if resp.Diagnostics.HasError() != tc.wantErr {
				t.Errorf("diagnostics = %v, want error %v", resp.Diagnostics, tc.wantErr)
			}
		})
	}
}
