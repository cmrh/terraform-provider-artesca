package groupmembership

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/scality/terraform-provider-artesca/internal/validators"
)

func TestSchema_Validators(t *testing.T) {
	r := NewGroupMembershipResource()
	resp := resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	g := resp.Schema.Attributes["group_name"].(schema.StringAttribute)
	if _, ok := g.Validators[0].(validators.IAMName); !ok {
		t.Errorf("group_name: expected IAMName validator, got %T", g.Validators[0])
	}

	u := resp.Schema.Attributes["username"].(schema.StringAttribute)
	if _, ok := u.Validators[0].(validators.IAMName); !ok {
		t.Errorf("username: expected IAMName validator, got %T", u.Validators[0])
	}
}
