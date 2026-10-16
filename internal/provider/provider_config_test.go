package provider

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestEnvOrValue(t *testing.T) {
	t.Setenv("ARTESCA_TEST_VAR", "from-env")
	for _, tc := range []struct {
		name string
		val  types.String
		want string
	}{
		{"configured", types.StringValue("set"), "set"},
		{"null falls back", types.StringNull(), "from-env"},
		{"empty falls back", types.StringValue(""), "from-env"},
	} {
		if got := envOrValue(tc.val, "ARTESCA_TEST_VAR"); got != tc.want {
			t.Errorf("%s: envOrValue = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestEnvOrDefault(t *testing.T) {
	for _, tc := range []struct {
		name string
		env  string
		val  types.String
		want string
	}{
		{"configured", "from-env", types.StringValue("set"), "set"},
		{"null uses env", "from-env", types.StringNull(), "from-env"},
		{"empty uses env", "from-env", types.StringValue(""), "from-env"},
		{"empty without env uses default", "", types.StringValue(""), "default"},
		{"null without env uses default", "", types.StringNull(), "default"},
	} {
		t.Setenv("ARTESCA_TEST_VAR", tc.env)
		if got := envOrDefault(tc.val, "ARTESCA_TEST_VAR", "default"); got != tc.want {
			t.Errorf("%s: envOrDefault = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestCheckUnknown(t *testing.T) {
	known := ArtescaProviderModel{
		ManagementEndpoint: types.StringValue("https://management.example.com"),
		InstanceID:         types.StringNull(),
		OIDCUrl:            types.StringValue("https://oidc.example.com"),
		OIDCRealm:          types.StringNull(),
		ClientID:           types.StringNull(),
		Username:           types.StringValue("u"),
		Password:           types.StringValue("p"),
		InsecureSkipVerify: types.BoolNull(),
		IAMRegion:          types.StringNull(),
		S3Endpoint:         types.StringNull(),
	}
	if diags := checkUnknown(known); diags.HasError() {
		t.Fatalf("known/null config: unexpected diagnostics %v", diags)
	}

	unknown := known
	unknown.ManagementEndpoint = types.StringUnknown()
	unknown.InsecureSkipVerify = types.BoolUnknown()
	diags := checkUnknown(unknown)
	if diags.ErrorsCount() != 2 {
		t.Fatalf("want 2 errors, got %v", diags)
	}
	for _, env := range []string{"ARTESCA_MANAGEMENT_ENDPOINT", "ARTESCA_INSECURE_SKIP_VERIFY"} {
		found := false
		for _, d := range diags.Errors() {
			if strings.Contains(d.Detail(), env) {
				found = true
			}
		}
		if !found {
			t.Errorf("no error mentions %s: %v", env, diags)
		}
	}
}
