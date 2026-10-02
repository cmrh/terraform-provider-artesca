package account

import (
	"context"
	"testing"

	"github.com/cmrh/terraform-provider-artesca/internal/validators"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestSchema_Validators(t *testing.T) {
	r := NewAccountResource()
	ctx := context.Background()
	resp := resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, &resp)

	t.Run("name has AccountName validator", func(t *testing.T) {
		attr := resp.Schema.Attributes["name"].(schema.StringAttribute)
		if len(attr.Validators) != 1 {
			t.Fatalf("expected 1 validator, got %d", len(attr.Validators))
		}
		if _, ok := attr.Validators[0].(validators.AccountName); !ok {
			t.Errorf("expected AccountName validator, got %T", attr.Validators[0])
		}
	})

	t.Run("email has Email validator", func(t *testing.T) {
		attr := resp.Schema.Attributes["email"].(schema.StringAttribute)
		if len(attr.Validators) != 1 {
			t.Fatalf("expected 1 validator, got %d", len(attr.Validators))
		}
		if _, ok := attr.Validators[0].(validators.Email); !ok {
			t.Errorf("expected Email validator, got %T", attr.Validators[0])
		}
	})
}

func TestSchema_EmailReplacement(t *testing.T) {
	r := NewAccountResource()
	ctx := context.Background()
	schemaResp := resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema

	attr := s.Attributes["email"].(schema.StringAttribute)
	if len(attr.PlanModifiers) != 1 {
		t.Fatalf("expected 1 plan modifier on email, got %d", len(attr.PlanModifiers))
	}
	modifier := attr.PlanModifiers[0]

	objType := s.Type().TerraformType(ctx)
	object := func(email tftypes.Value) tftypes.Value {
		vals := map[string]tftypes.Value{}
		for name := range s.Attributes {
			vals[name] = tftypes.NewValue(tftypes.String, nil)
		}
		vals["name"] = tftypes.NewValue(tftypes.String, "acct")
		vals["email"] = email
		return tftypes.NewValue(objType, vals)
	}

	cases := []struct {
		name        string
		state       types.String
		plan        types.String
		wantReplace bool
	}{
		{"changed known email replaces", types.StringValue("old@example.com"), types.StringValue("new@example.com"), true},
		{"removing known email replaces", types.StringValue("old@example.com"), types.StringNull(), true},
		{"null state (imported) adopts config in place", types.StringNull(), types.StringValue("new@example.com"), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stateEmail := tftypes.NewValue(tftypes.String, nil)
			if !tc.state.IsNull() {
				stateEmail = tftypes.NewValue(tftypes.String, tc.state.ValueString())
			}
			planEmail := tftypes.NewValue(tftypes.String, nil)
			if !tc.plan.IsNull() {
				planEmail = tftypes.NewValue(tftypes.String, tc.plan.ValueString())
			}

			req := planmodifier.StringRequest{
				Path:        path.Root("email"),
				StateValue:  tc.state,
				PlanValue:   tc.plan,
				ConfigValue: tc.plan,
				State:       tfsdk.State{Schema: s, Raw: object(stateEmail)},
				Plan:        tfsdk.Plan{Schema: s, Raw: object(planEmail)},
				Config:      tfsdk.Config{Schema: s, Raw: object(planEmail)},
			}
			resp := &planmodifier.StringResponse{PlanValue: tc.plan}
			modifier.PlanModifyString(ctx, req, resp)

			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}
			if resp.RequiresReplace != tc.wantReplace {
				t.Errorf("RequiresReplace = %v, want %v", resp.RequiresReplace, tc.wantReplace)
			}
		})
	}
}
