package creds

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// Schema version 1 replaced account_access_key / account_secret_key with
// account_name on account-scoped resources.
const (
	attrOldAccessKey = "account_access_key"
	attrOldSecretKey = "account_secret_key"
)

// UpgradeFromAccountKeys returns the v0 -> v1 state upgrader for a resource
// whose v0 state held the owning account's access key pair. account_name is
// resolved from those keys; everything else carries over unchanged.
func UpgradeFromAccountKeys(res resource.Resource, accounts func() *client.AccountCredentialSource) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
			values, err := DecodeRawState(req)
			if err != nil {
				resp.Diagnostics.AddError("Cannot upgrade state", err.Error())
				return
			}

			if name, _ := values[AttrAccountName].(string); name == "" {
				accessKey, _ := values[attrOldAccessKey].(string)
				secretKey, _ := values[attrOldSecretKey].(string)
				name, err := resolveAccountName(accounts, func(src *client.AccountCredentialSource) (string, error) {
					if accessKey == "" || secretKey == "" {
						return "", fmt.Errorf("prior state has no account keys")
					}
					return src.NameForAccessKey(ctx, accessKey, secretKey)
				})
				if err != nil {
					UpgradeError(resp, res, err)
					return
				}
				values[AttrAccountName] = name
			}

			WriteUpgradedState(ctx, res, values, resp)
		}},
	}
}

// ResolveAccountNameByID resolves account_name from an account ID during a
// state upgrade.
func ResolveAccountNameByID(ctx context.Context, accounts func() *client.AccountCredentialSource, accountID string) (string, error) {
	return resolveAccountName(accounts, func(src *client.AccountCredentialSource) (string, error) {
		if accountID == "" {
			return "", fmt.Errorf("prior state has no account_id")
		}
		return src.NameForAccountID(ctx, accountID)
	})
}

func resolveAccountName(accounts func() *client.AccountCredentialSource, lookup func(*client.AccountCredentialSource) (string, error)) (string, error) {
	src := accounts()
	if src == nil {
		return "", fmt.Errorf("the provider is not configured")
	}
	return lookup(src)
}

// upgradeErrorDetail explains a failed upgrade and how to migrate manually:
// re-import when the resource supports it, otherwise recreate it.
func upgradeErrorDetail(res resource.Resource, err error) string {
	detail := fmt.Sprintf("Could not determine account_name for this resource: %s.\n\n", err)
	if _, ok := res.(resource.ResourceWithImportState); ok {
		return detail + "To migrate it manually, remove it from state with `tofu state rm <address>` and " +
			"import it with `tofu import <address> <account_name>/<id>` (see the resource's Import documentation)."
	}
	return detail + "This resource can't be imported. Remove it from state with `tofu state rm <address>` " +
		"and apply again to create it anew."
}

// UpgradeError adds the standard failed-upgrade diagnostic for res.
func UpgradeError(resp *resource.UpgradeStateResponse, res resource.Resource, err error) {
	resp.Diagnostics.AddError("Cannot upgrade state", upgradeErrorDetail(res, err))
}

// DecodeRawState decodes the prior state's JSON attributes.
func DecodeRawState(req resource.UpgradeStateRequest) (map[string]any, error) {
	if req.RawState == nil || len(req.RawState.JSON) == 0 {
		return nil, fmt.Errorf("prior state is empty or not JSON")
	}
	values := map[string]any{}
	if err := json.Unmarshal(req.RawState.JSON, &values); err != nil {
		return nil, fmt.Errorf("decoding prior state: %w", err)
	}
	return values, nil
}

// WriteUpgradedState sets the upgraded state from values, keyed by the
// resource's current schema: keys not in the schema are dropped and schema
// attributes or blocks absent from values become null.
func WriteUpgradedState(ctx context.Context, res resource.Resource, values map[string]any, resp *resource.UpgradeStateResponse) {
	schemaResp := resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

	out := map[string]any{}
	for name := range schemaResp.Schema.Attributes {
		out[name] = values[name]
	}
	for name := range schemaResp.Schema.Blocks {
		out[name] = values[name]
	}

	b, err := json.Marshal(out)
	if err != nil {
		resp.Diagnostics.AddError("Cannot upgrade state", fmt.Sprintf("encoding upgraded state: %s", err))
		return
	}
	resp.DynamicValue = &tfprotov6.DynamicValue{JSON: b}
}
