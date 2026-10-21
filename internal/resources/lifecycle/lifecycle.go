// Package lifecycle holds the filter and value mapping shared by the
// expiration and transition lifecycle resources.
package lifecycle

import (
	"context"
	"sort"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Filter is the filter block of a lifecycle resource.
type Filter struct {
	ObjectKeyPrefix types.String
	ObjectTags      types.Map
}

// TagsFromMap converts an object_tags map to filter tags, sorted by key so
// the request body is stable.
func TagsFromMap(ctx context.Context, m types.Map) ([]client.BucketTag, diag.Diagnostics) {
	if m.IsNull() || m.IsUnknown() {
		return nil, nil
	}
	raw := map[string]string{}
	if diags := m.ElementsAs(ctx, &raw, false); diags.HasError() {
		return nil, diags
	}
	tags := make([]client.BucketTag, 0, len(raw))
	for k, v := range raw {
		tags = append(tags, client.BucketTag{Key: k, Value: v})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })
	return tags, nil
}

// FilterToModel returns the filter block for a rule read from the server,
// starting from the block in state (f, which may be nil).
func FilterToModel(ctx context.Context, rule *client.LifecycleRule, f *Filter) (*Filter, diag.Diagnostics) {
	tags := types.MapNull(types.StringType)
	if len(rule.Tags) > 0 {
		elems := make(map[string]string, len(rule.Tags))
		for _, t := range rule.Tags {
			elems[t.Key] = t.Value
		}
		var diags diag.Diagnostics
		tags, diags = types.MapValueFrom(ctx, types.StringType, elems)
		if diags.HasError() {
			return f, diags
		}
	}

	switch {
	case f != nil:
		// An unset prefix is stored as ""; keep it null so it doesn't diff.
		if !f.ObjectKeyPrefix.IsNull() || rule.Prefix != "" {
			f.ObjectKeyPrefix = types.StringValue(rule.Prefix)
		}
		f.ObjectTags = tags
	case rule.Prefix != "" || len(rule.Tags) > 0:
		// No filter block in state (e.g. after import): surface the server's
		// filter so it round-trips and out-of-band changes show as drift.
		f = &Filter{ObjectKeyPrefix: NonEmptyString(rule.Prefix), ObjectTags: tags}
	}
	return f, nil
}

// PositiveInt64 is n, or null when n is 0 (the action is not set).
func PositiveInt64(n int) types.Int64 {
	if n > 0 {
		return types.Int64Value(int64(n))
	}
	return types.Int64Null()
}

// NonEmptyString is s, or null when s is empty.
func NonEmptyString(s string) types.String {
	if s != "" {
		return types.StringValue(s)
	}
	return types.StringNull()
}
