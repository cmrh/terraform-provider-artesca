package workflowexpiration

import (
	"context"
	"testing"

	"github.com/cmrh/terraform-provider-artesca/internal/validators"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestSchema_Validators(t *testing.T) {
	r := NewWorkflowExpirationResource()
	ctx := context.Background()
	resp := resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, &resp)

	t.Run("bucket_name has BucketName validator", func(t *testing.T) {
		attr := resp.Schema.Attributes["bucket_name"].(schema.StringAttribute)
		if len(attr.Validators) != 1 {
			t.Fatalf("expected 1 validator, got %d", len(attr.Validators))
		}
		if _, ok := attr.Validators[0].(validators.BucketName); !ok {
			t.Errorf("expected BucketName validator, got %T", attr.Validators[0])
		}
	})
}
