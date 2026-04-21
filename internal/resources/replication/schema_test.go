package replication

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/scality/terraform-provider-scality-artesca/internal/validators"
)

func TestSchema_Validators(t *testing.T) {
	r := NewReplicationResource()
	ctx := context.Background()
	resp := resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, &resp)

	t.Run("source.bucket_name has BucketName validator", func(t *testing.T) {
		block := resp.Schema.Blocks["source"].(schema.SingleNestedBlock)
		attr := block.Attributes["bucket_name"].(schema.StringAttribute)
		if len(attr.Validators) != 1 {
			t.Fatalf("expected 1 validator, got %d", len(attr.Validators))
		}
		if _, ok := attr.Validators[0].(validators.BucketName); !ok {
			t.Errorf("expected BucketName validator, got %T", attr.Validators[0])
		}
	})

	t.Run("destination.bucket_name has BucketName validator", func(t *testing.T) {
		block := resp.Schema.Blocks["destination"].(schema.SingleNestedBlock)
		attr := block.Attributes["bucket_name"].(schema.StringAttribute)
		if len(attr.Validators) != 1 {
			t.Fatalf("expected 1 validator, got %d", len(attr.Validators))
		}
		if _, ok := attr.Validators[0].(validators.BucketName); !ok {
			t.Errorf("expected BucketName validator, got %T", attr.Validators[0])
		}
	})
}
