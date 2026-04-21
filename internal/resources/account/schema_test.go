package account

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/scality/terraform-provider-scality-artesca/internal/validators"
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
