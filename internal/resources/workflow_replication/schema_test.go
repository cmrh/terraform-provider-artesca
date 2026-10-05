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
		Enabled:               types.BoolValue(false),
		DestinationBucketName: types.StringValue("dst"),
		Filter:                &WorkflowFilterModel{ObjectKeyPrefix: types.StringValue("logs/")},
	}
	got := modelToReplicationRule(m, "rule-1")
	want := client.ReplicationRule{ID: "rule-1", Status: "Disabled", Prefix: "logs/", DestinationBucket: "dst"}
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
