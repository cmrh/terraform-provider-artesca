package creds

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestResolve(t *testing.T) {
	const envVar = "ARTESCA_TEST_RESOLVE"
	t.Setenv(envVar, "from-env")

	cases := []struct {
		name string
		attr types.String
		want string
	}{
		{"null falls through to env", types.StringNull(), "from-env"},
		{"unknown falls through to env", types.StringUnknown(), "from-env"},
		{"empty string falls through to env", types.StringValue(""), "from-env"},
		{"set value wins", types.StringValue("from-attr"), "from-attr"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Resolve(tc.attr, envVar); got != tc.want {
				t.Errorf("Resolve = %q, want %q", got, tc.want)
			}
		})
	}

	t.Run("env unset returns empty", func(t *testing.T) {
		t.Setenv(envVar, "")
		if got := Resolve(types.StringNull(), envVar); got != "" {
			t.Errorf("Resolve = %q, want empty", got)
		}
	})
}
