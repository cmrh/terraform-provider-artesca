// Package creds resolves per-account credentials for IAM/S3 resources, with
// an environment-variable fallback used during `tofu import` when state
// attributes are empty.
package creds

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	EnvAccessKey = "ARTESCA_ACCOUNT_ACCESS_KEY"
	EnvSecretKey = "ARTESCA_ACCOUNT_SECRET_KEY"

	AttrAccessKey = "account_access_key"
	AttrSecretKey = "account_secret_key"
)

// Resolve returns the attribute's string value if set, otherwise the value
// of envVar. An attribute that is null, unknown, or an empty string falls
// through to the environment.
func Resolve(attr types.String, envVar string) string {
	if !attr.IsNull() && !attr.IsUnknown() {
		if v := attr.ValueString(); v != "" {
			return v
		}
	}
	return os.Getenv(envVar)
}

// WriteImport copies account_access_key and account_secret_key from the
// fallback env vars into the given state during `tofu import`, so the
// subsequent Read can authenticate and so the first plan doesn't show a
// spurious diff for the credential attributes.
func WriteImport(ctx context.Context, state *tfsdk.State, diags *diag.Diagnostics) {
	if ak := os.Getenv(EnvAccessKey); ak != "" {
		diags.Append(state.SetAttribute(ctx, path.Root(AttrAccessKey), ak)...)
	}
	if sk := os.Getenv(EnvSecretKey); sk != "" {
		diags.Append(state.SetAttribute(ctx, path.Root(AttrSecretKey), sk)...)
	}
}

// ImportByID is the standard ImportState shape for resources with a single
// natural-ID attribute: passthrough the ID, then populate creds from env.
func ImportByID(ctx context.Context, idAttr string, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root(idAttr), req, resp)
	WriteImport(ctx, &resp.State, &resp.Diagnostics)
}
