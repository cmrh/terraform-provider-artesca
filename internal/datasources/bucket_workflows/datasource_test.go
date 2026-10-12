package bucket_workflows

import (
	"testing"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
)

func TestReplicationToSummaryLocationRule(t *testing.T) {
	// Shape returned by workflow search for an S3 replication rule with a
	// destination location (StorageClass).
	s := &client.ReplicationStream{
		StreamID: "r1",
		Enabled:  true,
		Source:   &client.ReplicationSource{BucketName: "src"},
		Destination: &client.ReplicationDest{
			BucketName: "src",
			Locations:  []client.ReplicationDestLocation{{Name: "lab1"}},
		},
	}
	got := replicationToSummary(s)
	if got.DestinationLocation.ValueString() != "lab1" {
		t.Errorf("destination_location = %s, want lab1", got.DestinationLocation)
	}
	if got.DestinationBucketName.ValueString() != "" {
		t.Errorf("destination_bucket_name = %s, want empty for the source-bucket placeholder", got.DestinationBucketName)
	}

	// A CRR rule names a real remote bucket alongside the location.
	s.Destination.BucketName = "remote-dst"
	if got := replicationToSummary(s); got.DestinationBucketName.ValueString() != "remote-dst" {
		t.Errorf("destination_bucket_name = %s, want remote-dst", got.DestinationBucketName)
	}
}
