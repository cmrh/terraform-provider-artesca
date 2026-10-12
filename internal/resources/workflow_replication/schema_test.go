package workflowreplication

import (
	"context"
	"testing"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/cmrh/terraform-provider-artesca/internal/validators"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSchema_Validators(t *testing.T) {
	r := NewWorkflowReplicationResource()
	ctx := context.Background()
	resp := resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, &resp)

	for _, name := range []string{"bucket_name", "destination_bucket_name"} {
		t.Run(name+" has BucketName validator", func(t *testing.T) {
			attr := resp.Schema.Attributes[name].(schema.StringAttribute)
			if len(attr.Validators) != 1 {
				t.Fatalf("expected 1 validator, got %d", len(attr.Validators))
			}
			if _, ok := attr.Validators[0].(validators.BucketName); !ok {
				t.Errorf("expected BucketName validator, got %T", attr.Validators[0])
			}
		})
	}

	for _, gone := range []string{"name", "version", "account_id", "instance_id", "workflow_id"} {
		if _, ok := resp.Schema.Attributes[gone]; ok {
			t.Errorf("attribute %q should not exist", gone)
		}
	}
	for _, gone := range []string{"source", "destination"} {
		if _, ok := resp.Schema.Blocks[gone]; ok {
			t.Errorf("block %q should not exist", gone)
		}
	}
}

func TestModelToReplicationRule(t *testing.T) {
	m := &WorkflowReplicationResourceModel{
		BucketName:          types.StringValue("src"),
		Enabled:             types.BoolValue(false),
		DestinationLocation: types.StringValue("lab1"),
		Filter:              &WorkflowFilterModel{ObjectKeyPrefix: types.StringValue("logs/")},
	}
	got := modelToReplicationRule(m, "rule-1")
	want := client.ReplicationRule{ID: "rule-1", Status: "Disabled", Prefix: "logs/", DestinationBucket: "src", StorageClass: "lab1"}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}

	m.Enabled = types.BoolValue(true)
	m.Filter = nil
	got = modelToReplicationRule(m, "rule-1")
	if got.Status != "Enabled" || got.Prefix != "" {
		t.Errorf("got %+v, want Enabled with empty prefix", got)
	}
}

func TestReplicationDestinationModes(t *testing.T) {
	null := types.StringNull()
	str := types.StringValue
	for _, tc := range []struct {
		name             string
		bucket, location types.String // configuration
		wantRuleBucket   string
		wantStorageClass string
	}{
		{"location only: source bucket fills the rule", null, str("lab1"), "src", "lab1"},
		{"bucket and location (CRR)", str("remote-dst"), str("crr"), "remote-dst", "crr"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := &WorkflowReplicationResourceModel{
				BucketName:            str("src"),
				Enabled:               types.BoolValue(true),
				DestinationBucketName: tc.bucket,
				DestinationLocation:   tc.location,
			}
			rule := modelToReplicationRule(m, "r")
			if rule.DestinationBucket != tc.wantRuleBucket || rule.StorageClass != tc.wantStorageClass {
				t.Fatalf("rule = %+v, want bucket %q, storage class %q", rule, tc.wantRuleBucket, tc.wantStorageClass)
			}

			// Read after create starts from the configured values; read after
			// import starts empty. Both must reproduce the configuration.
			for _, start := range []types.String{tc.bucket, null} {
				got := &WorkflowReplicationResourceModel{BucketName: str("src"), DestinationBucketName: start}
				replicationRuleToModel(&rule, got)
				if !got.DestinationBucketName.Equal(tc.bucket) || !got.DestinationLocation.Equal(tc.location) {
					t.Errorf("starting from %s: read back bucket %s, location %s; want %s, %s",
						start, got.DestinationBucketName, got.DestinationLocation, tc.bucket, tc.location)
				}
			}
		})
	}
}

func TestReplicationRuleToModelFilter(t *testing.T) {
	cases := []struct {
		name       string
		filter     *WorkflowFilterModel
		rulePrefix string
		wantFilter *WorkflowFilterModel
	}{
		{"no filter, no prefix", nil, "", nil},
		{"no filter, server prefix surfaces as drift", nil, "logs/", &WorkflowFilterModel{ObjectKeyPrefix: types.StringValue("logs/")}},
		{"filter with null prefix stays null", &WorkflowFilterModel{ObjectKeyPrefix: types.StringNull()}, "", &WorkflowFilterModel{ObjectKeyPrefix: types.StringNull()}},
		{"filter prefix updated from server", &WorkflowFilterModel{ObjectKeyPrefix: types.StringValue("a/")}, "b/", &WorkflowFilterModel{ObjectKeyPrefix: types.StringValue("b/")}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := &WorkflowReplicationResourceModel{Filter: tc.filter}
			replicationRuleToModel(&client.ReplicationRule{Status: "Enabled", Prefix: tc.rulePrefix, DestinationBucket: "dst"}, m)
			if !m.Enabled.ValueBool() || m.DestinationBucketName.ValueString() != "dst" {
				t.Errorf("enabled/destination not mapped: %+v", m)
			}
			switch {
			case tc.wantFilter == nil && m.Filter != nil:
				t.Errorf("filter = %+v, want nil", m.Filter)
			case tc.wantFilter != nil && (m.Filter == nil || !m.Filter.ObjectKeyPrefix.Equal(tc.wantFilter.ObjectKeyPrefix)):
				t.Errorf("filter = %+v, want %+v", m.Filter, tc.wantFilter)
			}
		})
	}
}

func TestUpgradeReplicationV0(t *testing.T) {
	old := map[string]any{
		"instance_id": "inst", "account_id": "111111111111", "bucket_name": "src",
		"workflow_id": "0b3c4a37", "name": "nightly", "version": float64(1), "enabled": true,
		"source":      map[string]any{"bucket_name": "src", "prefix": "logs/", "location": nil},
		"destination": map[string]any{"bucket_name": "dst", "location": nil, "locations": []any{}},
	}
	got := upgradeReplicationV0(old)
	want := map[string]any{
		"bucket_name": "src", "rule_id": "0b3c4a37", "enabled": true,
		"destination_bucket_name": "dst",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %v, want %v", k, got[k], v)
		}
	}
	filter, ok := got["filter"].(map[string]any)
	if !ok || filter["object_key_prefix"] != "logs/" {
		t.Errorf("filter = %v, want object_key_prefix logs/", got["filter"])
	}

	old["source"] = map[string]any{"bucket_name": "src", "prefix": ""}
	if f, ok := upgradeReplicationV0(old)["filter"]; ok {
		t.Errorf("empty prefix should leave filter unset, got %v", f)
	}
}
