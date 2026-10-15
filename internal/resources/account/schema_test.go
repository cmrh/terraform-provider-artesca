package account

import (
	"context"
	"testing"

	"github.com/cmrh/terraform-provider-artesca/internal/validators"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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

func TestSchema_EmailPlanModifier(t *testing.T) {
	r := NewAccountResource()
	resp := resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	attr := resp.Schema.Attributes["email"].(schema.StringAttribute)
	if !attr.Optional || !attr.Computed {
		t.Errorf("email: Optional = %v, Computed = %v; want both", attr.Optional, attr.Computed)
	}
	if len(attr.PlanModifiers) != 1 {
		t.Fatalf("expected 1 plan modifier on email, got %d", len(attr.PlanModifiers))
	}
	if _, ok := attr.PlanModifiers[0].(emailPlanModifier); !ok {
		t.Errorf("expected emailPlanModifier, got %T", attr.PlanModifiers[0])
	}
}
