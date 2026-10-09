package endpoint

import (
	"context"
	"testing"

	"github.com/cmrh/terraform-provider-artesca/internal/validators"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestSchema_Validators(t *testing.T) {
	r := NewEndpointResource()
	ctx := context.Background()
	resp := resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, &resp)

	t.Run("hostname has Hostname validator", func(t *testing.T) {
		attr := resp.Schema.Attributes["hostname"].(schema.StringAttribute)
		if len(attr.Validators) != 1 {
			t.Fatalf("expected 1 validator, got %d", len(attr.Validators))
		}
		if _, ok := attr.Validators[0].(validators.Hostname); !ok {
			t.Errorf("expected Hostname validator, got %T", attr.Validators[0])
		}
	})

	t.Run("location_name has LocationName validator", func(t *testing.T) {
		attr := resp.Schema.Attributes["location_name"].(schema.StringAttribute)
		if len(attr.Validators) != 1 {
			t.Fatalf("expected 1 validator, got %d", len(attr.Validators))
		}
		if _, ok := attr.Validators[0].(validators.LocationName); !ok {
			t.Errorf("expected LocationName validator, got %T", attr.Validators[0])
		}
	})
}
