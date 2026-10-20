package workflowreplication

import (
	"strings"
	"testing"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const destRole = "arn:aws:iam::222222222222:role/crr-role"

// fakeCRR treats locations named "crr*" as CRR locations.
func fakeCRR(location string) (bool, error) { return strings.HasPrefix(location, "crr"), nil }

func rules(locations ...string) []client.ReplicationRule {
	out := make([]client.ReplicationRule, 0, len(locations))
	for i, loc := range locations {
		out = append(out, client.ReplicationRule{ID: string(rune('a' + i)), StorageClass: loc})
	}
	return out
}

func TestBuildRole(t *testing.T) {
	src := client.DefaultReplicationRole
	for _, tc := range []struct {
		name    string
		current string
		rules   []client.ReplicationRule
		dest    string
		want    string
		wantErr bool
	}{
		{"new configuration, RING rule", "", rules("ring"), "", src, false},
		{"new configuration, CRR rule", "", rules("crr"), destRole, src + "," + destRole, false},
		{"mixed rules", src, rules("ring", "crr"), destRole, src + "," + destRole, false},
		{"last CRR rule removed", src + "," + destRole, rules("ring"), destRole, src, false},
		{"custom source role is kept", "arn:aws:iam::1:role/src," + destRole, rules("crr"), destRole, "arn:aws:iam::1:role/src," + destRole, false},
		{"CRR rule without a destination role", src, rules("crr"), "", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := buildRole(tc.current, tc.rules, fakeCRR, tc.dest)
			if (err != nil) != tc.wantErr || got != tc.want {
				t.Errorf("buildRole = %q, %v; want %q, error %v", got, err, tc.want, tc.wantErr)
			}
		})
	}
}

func TestCheckDestinationRole(t *testing.T) {
	src := client.DefaultReplicationRole
	null, str := types.StringNull(), types.StringValue
	for _, tc := range []struct {
		name     string
		location string
		role     types.String
		cfg      client.BucketReplication
		want     string
		wantErr  string
	}{
		{"CRR rule with a role", "crr", str(destRole), client.BucketReplication{Role: src}, destRole, ""},
		{"CRR rule without a role", "crr", null, client.BucketReplication{Role: src}, "", "Missing destination_role"},
		{"RING rule with a role", "ring", str(destRole), client.BucketReplication{Role: src}, "", "Unexpected destination_role"},
		{"RING rule next to a CRR rule keeps its role", "ring", null, client.BucketReplication{Role: src + "," + destRole, Rules: rules("crr")}, destRole, ""},
		{"CRR rule with a different role than another CRR rule", "crr", str("arn:aws:iam::3:role/other"), client.BucketReplication{Role: src + "," + destRole, Rules: rules("crr")}, "", "Conflicting destination_role"},
		{"changing the role of the only CRR rule", "crr", str("arn:aws:iam::3:role/other"), client.BucketReplication{Role: src + "," + destRole, Rules: []client.ReplicationRule{{ID: "me", StorageClass: "crr"}}}, "arn:aws:iam::3:role/other", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan := &WorkflowReplicationResourceModel{
				BucketName:          str("bkt"),
				DestinationLocation: str(tc.location),
				DestinationRole:     tc.role,
			}
			var diags diag.Diagnostics
			got, ok := checkDestinationRole(plan, &tc.cfg, "me", fakeCRR, &diags)
			if tc.wantErr != "" {
				if ok || !diags.HasError() || diags.Errors()[0].Summary() != tc.wantErr {
					t.Fatalf("diagnostics = %v, want %q", diags, tc.wantErr)
				}
				return
			}
			if !ok || diags.HasError() || got != tc.want {
				t.Errorf("checkDestinationRole = %q, %v, %v; want %q", got, ok, diags, tc.want)
			}
		})
	}
}
