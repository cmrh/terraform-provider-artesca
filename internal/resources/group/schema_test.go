package group

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/scality/terraform-provider-artesca/internal/validators"
)

func TestSchema_Validators(t *testing.T) {
	r := NewGroupResource()
	resp := resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	attr := resp.Schema.Attributes["name"].(schema.StringAttribute)
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
}
