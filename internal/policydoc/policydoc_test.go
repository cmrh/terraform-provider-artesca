package policydoc

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	doc        = `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:GetObject","Resource":"*"}]}`
	docSpaced  = "{\n  \"Statement\": [{\"Resource\": \"*\", \"Action\": \"s3:GetObject\", \"Effect\": \"Allow\"}],\n  \"Version\": \"2012-10-17\"\n}"
	docChanged = `{"Version":"2012-10-17","Statement":[{"Effect":"Deny","Action":"s3:GetObject","Resource":"*"}]}`
)

func TestEquivalent(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want bool
	}{
		{doc, docSpaced, true},
		{doc, docChanged, false},
		{doc, "not json", false},
		{"", doc, false},
	} {
		if got := Equivalent(tc.a, tc.b); got != tc.want {
			t.Errorf("Equivalent(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestRefresh(t *testing.T) {
	current := types.StringValue(docSpaced)
	if got := Refresh(current, doc); got != current {
		t.Errorf("equivalent remote: got %s, want state kept", got)
	}
	if got := Refresh(current, docChanged); got.ValueString() != docChanged {
		t.Errorf("changed remote: got %s, want %s", got, docChanged)
	}
	if got := Refresh(types.StringNull(), doc); got.ValueString() != doc {
		t.Errorf("import (null state): got %s, want %s", got, doc)
	}
}

func TestEquivalenceModifier(t *testing.T) {
	ctx := context.Background()
	m := EquivalenceModifier()
	for _, tc := range []struct {
		name        string
		state, plan types.String
		want        types.String
	}{
		{"formatting only keeps state", types.StringValue(doc), types.StringValue(docSpaced), types.StringValue(doc)},
		{"real change keeps plan", types.StringValue(doc), types.StringValue(docChanged), types.StringValue(docChanged)},
		{"create (null state)", types.StringNull(), types.StringValue(doc), types.StringValue(doc)},
		{"unknown plan", types.StringValue(doc), types.StringUnknown(), types.StringUnknown()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := &planmodifier.StringResponse{PlanValue: tc.plan}
			m.PlanModifyString(ctx, planmodifier.StringRequest{StateValue: tc.state, PlanValue: tc.plan}, resp)
			if !resp.PlanValue.Equal(tc.want) {
				t.Errorf("plan = %s, want %s", resp.PlanValue, tc.want)
			}
		})
	}
}
