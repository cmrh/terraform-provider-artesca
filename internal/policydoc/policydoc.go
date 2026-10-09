// Package policydoc compares JSON policy documents by meaning rather than
// formatting, so whitespace and key order never show up as drift.
package policydoc

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Equivalent reports whether two JSON documents represent the same value,
// ignoring whitespace and key order. It returns false if either fails to parse.
func Equivalent(a, b string) bool {
	var av, bv any
	if err := json.Unmarshal([]byte(a), &av); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &bv); err != nil {
		return false
	}
	return reflect.DeepEqual(av, bv)
}

// Refresh returns the value Read should keep in state: the current value when
// it is equivalent to the document read from the API, otherwise the API's.
func Refresh(current types.String, remote string) types.String {
	if Equivalent(current.ValueString(), remote) {
		return current
	}
	return types.StringValue(remote)
}

// EquivalenceModifier keeps the state value when the planned document differs
// from it only in formatting. List it before RequiresReplace.
func EquivalenceModifier() planmodifier.String {
	return equivalenceModifier{}
}

type equivalenceModifier struct{}

func (m equivalenceModifier) Description(_ context.Context) string {
	return "Suppress diff when JSON policy documents are semantically equivalent."
}

func (m equivalenceModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m equivalenceModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}
	if Equivalent(req.StateValue.ValueString(), req.PlanValue.ValueString()) {
		resp.PlanValue = req.StateValue
	}
}
