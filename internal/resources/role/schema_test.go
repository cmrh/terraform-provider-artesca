package role

import (
	"context"
	"testing"

	"github.com/cmrh/terraform-provider-artesca/internal/validators"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestSchema_Validators(t *testing.T) {
	r := NewRoleResource()
	resp := resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	n := resp.Schema.Attributes["name"].(schema.StringAttribute)
	v, ok := n.Validators[0].(validators.IAMName)
	if !ok {
		t.Fatalf("name: expected IAMName, got %T", n.Validators[0])
	}
	if v.MaxLength != 64 {
		t.Errorf("expected MaxLength 64, got %d", v.MaxLength)
	}

	trust := resp.Schema.Attributes["assume_role_policy_document"].(schema.StringAttribute)
	if _, ok := trust.Validators[0].(validators.JSONDocument); !ok {
		t.Errorf("assume_role_policy_document: expected JSONDocument, got %T", trust.Validators[0])
	}
}
