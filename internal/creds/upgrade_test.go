package creds

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testResource is a minimal resource whose schema drives WriteUpgradedState.
type testResource struct{ resource.Resource }

func (testResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			AttrAccountName: schema.StringAttribute{Required: true},
			"username":      schema.StringAttribute{Required: true},
			"arn":           schema.StringAttribute{Computed: true},
		},
		Blocks: map[string]schema.Block{
			"filter": schema.SingleNestedBlock{Attributes: map[string]schema.Attribute{"prefix": schema.StringAttribute{Optional: true}}},
		},
	}
}

func upgradeRequest(t *testing.T, state map[string]any) resource.UpgradeStateRequest {
	t.Helper()
	b, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	return resource.UpgradeStateRequest{RawState: &tfprotov6.RawState{JSON: b}}
}

func decodeUpgraded(t *testing.T, resp *resource.UpgradeStateResponse) map[string]any {
	t.Helper()
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if resp.DynamicValue == nil {
		t.Fatal("DynamicValue not set")
	}
	out := map[string]any{}
	if err := json.Unmarshal(resp.DynamicValue.JSON, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestWriteUpgradedStateKeysBySchema(t *testing.T) {
	resp := &resource.UpgradeStateResponse{}
	WriteUpgradedState(context.Background(), testResource{}, map[string]any{
		AttrAccountName:      "app",
		"username":           "alice",
		"account_access_key": "old",
	}, resp)
	out := decodeUpgraded(t, resp)

	if out[AttrAccountName] != "app" || out["username"] != "alice" {
		t.Errorf("values not carried over: %v", out)
	}
	if _, ok := out["account_access_key"]; ok {
		t.Error("attribute not in schema was kept")
	}
	for _, k := range []string{"arn", "filter"} {
		if v, ok := out[k]; !ok || v != nil {
			t.Errorf("%s = %v (present=%v), want null", k, v, ok)
		}
	}
}

func TestUpgradeFromAccountKeysKeepsExistingAccountName(t *testing.T) {
	up := UpgradeFromAccountKeys(testResource{}, func() *client.AccountCredentialSource { return nil })[0]
	resp := &resource.UpgradeStateResponse{}
	up.StateUpgrader(context.Background(), upgradeRequest(t, map[string]any{AttrAccountName: "app", "username": "alice"}), resp)
	if out := decodeUpgraded(t, resp); out[AttrAccountName] != "app" {
		t.Errorf("account_name = %v, want app", out[AttrAccountName])
	}
}

// importableTestResource is testResource with import support.
type importableTestResource struct{ testResource }

func (importableTestResource) ImportState(context.Context, resource.ImportStateRequest, *resource.ImportStateResponse) {
}

func TestUpgradeFromAccountKeysUnconfiguredProvider(t *testing.T) {
	for _, tc := range []struct {
		name          string
		res           resource.Resource
		want, notWant []string
	}{
		{"importable", importableTestResource{}, []string{"tofu state rm", "tofu import", "<account_name>/<id>"}, nil},
		{"not importable", testResource{}, []string{"tofu state rm", "can't be imported", "apply again"}, []string{"tofu import"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			up := UpgradeFromAccountKeys(tc.res, func() *client.AccountCredentialSource { return nil })[0]
			resp := &resource.UpgradeStateResponse{}
			up.StateUpgrader(context.Background(), upgradeRequest(t, map[string]any{
				"account_access_key": "ak", "account_secret_key": "sk", "username": "alice",
			}), resp)
			if !resp.Diagnostics.HasError() {
				t.Fatal("expected an error when the provider is not configured")
			}
			detail := resp.Diagnostics.Errors()[0].Detail()
			for _, want := range tc.want {
				if !strings.Contains(detail, want) {
					t.Errorf("error detail missing %q: %s", want, detail)
				}
			}
			for _, notWant := range tc.notWant {
				if strings.Contains(detail, notWant) {
					t.Errorf("error detail should not contain %q: %s", notWant, detail)
				}
			}
		})
	}
}

func TestDecodeRawStateErrors(t *testing.T) {
	if _, err := DecodeRawState(resource.UpgradeStateRequest{}); err == nil {
		t.Error("expected error for missing raw state")
	}
	if _, err := DecodeRawState(resource.UpgradeStateRequest{RawState: &tfprotov6.RawState{JSON: []byte("{")}}); err == nil {
		t.Error("expected error for malformed JSON")
	}
}
