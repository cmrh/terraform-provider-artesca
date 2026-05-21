package grouppolicy

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/scality/terraform-provider-scality-artesca/internal/validators"
)

func TestSchema_Validators(t *testing.T) {
	r := NewGroupPolicyResource()
	resp := resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	pn := resp.Schema.Attributes["policy_name"].(schema.StringAttribute)
	v, ok := pn.Validators[0].(validators.IAMName)
	if !ok {
		t.Fatalf("policy_name: expected IAMName, got %T", pn.Validators[0])
	}
	if v.MaxLength != 128 {
		t.Errorf("expected MaxLength 128, got %d", v.MaxLength)
	}

	pd := resp.Schema.Attributes["policy_document"].(schema.StringAttribute)
	if _, ok := pd.Validators[0].(validators.JSONDocument); !ok {
		t.Errorf("policy_document: expected JSONDocument, got %T", pd.Validators[0])
	}
}
