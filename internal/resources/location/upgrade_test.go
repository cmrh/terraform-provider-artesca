package location

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

func upgradeV0(t *testing.T, state map[string]any) map[string]any {
	t.Helper()
	ctx := context.Background()
	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	r := &LocationResource{}
	resp := &resource.UpgradeStateResponse{}
	r.UpgradeState(ctx)[0].StateUpgrader(ctx, resource.UpgradeStateRequest{RawState: &tfprotov6.RawState{JSON: raw}}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}

	// The result must decode against the current schema.
	schemaResp := resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if _, err := resp.DynamicValue.Unmarshal(schemaResp.Schema.Type().TerraformType(ctx)); err != nil {
		t.Fatalf("upgraded state does not match the schema: %v", err)
	}

	out := map[string]any{}
	if err := json.Unmarshal(resp.DynamicValue.JSON, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestUpgradeStateV0BucketMatch(t *testing.T) {
	for _, tc := range []struct {
		name  string
		match any
		want  any
	}{
		{"unset becomes false", nil, false},
		{"true is kept", true, true},
		{"false is kept", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := upgradeV0(t, map[string]any{
				"name":          "loc",
				"location_type": "location-scality-ring-s3-v1",
				"details":       map[string]any{"endpoint": "http://ring:8080", "bucket_match": tc.match},
			})
			details := out["details"].(map[string]any)
			if details["bucket_match"] != tc.want {
				t.Errorf("bucket_match = %v, want %v", details["bucket_match"], tc.want)
			}
			if details["endpoint"] != "http://ring:8080" {
				t.Errorf("endpoint = %v", details["endpoint"])
			}
		})
	}
}

func TestUpgradeStateV0NoDetails(t *testing.T) {
	out := upgradeV0(t, map[string]any{"name": "loc", "location_type": "location-mem-v1", "details": nil})
	if out["details"] != nil {
		t.Errorf("details = %v, want null", out["details"])
	}
}
