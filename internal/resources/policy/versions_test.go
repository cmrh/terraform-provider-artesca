package policy

import (
	"context"
	"testing"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestOldestNonDefault(t *testing.T) {
	versions := []client.PolicyVersion{
		{VersionID: "v5", IsDefault: true, CreateDate: "2026-10-19T10:05:00Z"},
		{VersionID: "v3", CreateDate: "2026-10-19T10:03:00Z"},
		{VersionID: "v2", CreateDate: "2026-10-19T10:02:00Z"},
		{VersionID: "v4", CreateDate: "2026-10-19T10:04:00Z"},
	}
	if got := oldestNonDefault(versions); got != "v2" {
		t.Errorf("oldestNonDefault = %q, want v2", got)
	}
	if got := oldestNonDefault(versions[:1]); got != "" {
		t.Errorf("only the default version: got %q, want \"\"", got)
	}
}

func TestDefaultVersionModifier(t *testing.T) {
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	(&PolicyResource{}).Schema(ctx, resource.SchemaRequest{}, schemaResp)
	s := schemaResp.Schema
	obj := func(doc tftypes.Value) tftypes.Value {
		vals := map[string]tftypes.Value{}
		for name := range s.Attributes {
			vals[name] = tftypes.NewValue(tftypes.String, nil)
		}
		vals["policy_document"] = doc
		vals["default_version_id"] = tftypes.NewValue(tftypes.String, "v1")
		return tftypes.NewValue(s.Type().TerraformType(ctx), vals)
	}
	str := func(v string) tftypes.Value { return tftypes.NewValue(tftypes.String, v) }
	current := `{"Version":"2012-10-17","Statement":[]}`

	for _, tc := range []struct {
		name        string
		planned     tftypes.Value
		wantUnknown bool
	}{
		{"same document", str(current), false},
		{"reformatted document", str(`{ "Statement": [], "Version": "2012-10-17" }`), false},
		{"changed document", str(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`), true},
		{"unknown document", tftypes.NewValue(tftypes.String, tftypes.UnknownValue), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := planmodifier.StringRequest{
				StateValue: types.StringValue("v1"),
				PlanValue:  types.StringUnknown(),
				State:      tfsdk.State{Schema: s, Raw: obj(str(current))},
				Plan:       tfsdk.Plan{Schema: s, Raw: obj(tc.planned)},
			}
			resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
			defaultVersionModifier{}.PlanModifyString(ctx, req, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("diagnostics: %v", resp.Diagnostics)
			}
			if resp.PlanValue.IsUnknown() != tc.wantUnknown {
				t.Errorf("plan = %s, want unknown = %v", resp.PlanValue, tc.wantUnknown)
			}
		})
	}
}
