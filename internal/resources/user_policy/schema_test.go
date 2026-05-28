package userpolicy

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/scality/terraform-provider-artesca/internal/validators"
)

func TestSchema_Validators(t *testing.T) {
	r := NewUserPolicyResource()
	ctx := context.Background()
	resp := resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, &resp)

	t.Run("username has IAMName validator", func(t *testing.T) {
		attr := resp.Schema.Attributes["username"].(schema.StringAttribute)
		if len(attr.Validators) != 1 {
			t.Fatalf("expected 1 validator, got %d", len(attr.Validators))
		}
		v, ok := attr.Validators[0].(validators.IAMName)
		if !ok {
			t.Fatalf("expected IAMName validator, got %T", attr.Validators[0])
		}
		if v.MaxLength != 64 {
			t.Errorf("expected MaxLength 64, got %d", v.MaxLength)
		}
	})

	t.Run("policy_name has IAMName validator", func(t *testing.T) {
		attr := resp.Schema.Attributes["policy_name"].(schema.StringAttribute)
		if len(attr.Validators) != 1 {
			t.Fatalf("expected 1 validator, got %d", len(attr.Validators))
		}
		v, ok := attr.Validators[0].(validators.IAMName)
		if !ok {
			t.Fatalf("expected IAMName validator, got %T", attr.Validators[0])
		}
		if v.MaxLength != 128 {
			t.Errorf("expected MaxLength 128, got %d", v.MaxLength)
		}
	})

	t.Run("policy_document has JSONDocument validator", func(t *testing.T) {
		attr := resp.Schema.Attributes["policy_document"].(schema.StringAttribute)
		if len(attr.Validators) != 1 {
			t.Fatalf("expected 1 validator, got %d", len(attr.Validators))
		}
		if _, ok := attr.Validators[0].(validators.JSONDocument); !ok {
			t.Errorf("expected JSONDocument validator, got %T", attr.Validators[0])
		}
	})
}
