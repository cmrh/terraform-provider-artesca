package replication

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestValidateConfigRequiresBlocks(t *testing.T) {
	ctx := context.Background()
	r := &ReplicationResource{}
	schemaResp := resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)

	// config builds a configuration with every attribute null except the given blocks.
	config := func(blocks map[string]tftypes.Value) tfsdk.Config {
		vals := map[string]tftypes.Value{}
		for name, typ := range objType.AttributeTypes {
			vals[name] = tftypes.NewValue(typ, nil)
		}
		for name, v := range blocks {
			vals[name] = v
		}
		return tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, vals)}
	}
	// present returns a non-null block value with all its attributes null.
	present := func(name string) tftypes.Value {
		bt := objType.AttributeTypes[name].(tftypes.Object)
		vals := map[string]tftypes.Value{}
		for n, typ := range bt.AttributeTypes {
			vals[n] = tftypes.NewValue(typ, nil)
		}
		return tftypes.NewValue(bt, vals)
	}

	for _, tc := range []struct {
		name   string
		blocks map[string]tftypes.Value
		errors int
	}{
		{"both present", map[string]tftypes.Value{"source": present("source"), "destination": present("destination")}, 0},
		{"source missing", map[string]tftypes.Value{"destination": present("destination")}, 1},
		{"destination missing", map[string]tftypes.Value{"source": present("source")}, 1},
		{"both missing", nil, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := &resource.ValidateConfigResponse{}
			r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: config(tc.blocks)}, resp)
			if got := resp.Diagnostics.ErrorsCount(); got != tc.errors {
				t.Errorf("errors = %d, want %d: %v", got, tc.errors, resp.Diagnostics)
			}
		})
	}
}
