package policy

import (
	"context"
	"testing"

	"github.com/cmrh/terraform-provider-artesca/internal/validators"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestSchema_Validators(t *testing.T) {
	r := NewPolicyResource()
	resp := resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	n := resp.Schema.Attributes["name"].(schema.StringAttribute)
	v, ok := n.Validators[0].(validators.IAMName)
	if !ok {
		t.Fatalf("name: expected IAMName, got %T", n.Validators[0])
	}
	if v.MaxLength != 128 {
		t.Errorf("expected MaxLength 128, got %d", v.MaxLength)
	}

	pd := resp.Schema.Attributes["policy_document"].(schema.StringAttribute)
	if _, ok := pd.Validators[0].(validators.JSONDocument); !ok {
		t.Errorf("policy_document: expected JSONDocument, got %T", pd.Validators[0])
	}
}
