// Package creds holds the account_name attribute shared by account-scoped
// resources and data sources, and the "<account_name>/<id>" import ID format.
// The provider turns account_name into temporary credentials via
// client.AccountCredentialSource.
package creds

import (
	"context"
	"fmt"
	"strings"

	validators "github.com/cmrh/terraform-provider-artesca/internal/validators"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

const AttrAccountName = "account_name"

// ResourceAttribute is the account_name attribute for account-scoped resources.
func ResourceAttribute() schema.StringAttribute {
	return schema.StringAttribute{
		Description: "The name of the account that owns this resource. The provider obtains temporary credentials for it from the provider's OIDC login.",
		Required:    true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		},
		Validators: []validator.String{
			validators.AccountName{},
		},
	}
}

// DataSourceAttribute is the account_name attribute for account-scoped data sources.
func DataSourceAttribute() dsschema.StringAttribute {
	return dsschema.StringAttribute{
		Description: "The name of the account to read from. The provider obtains temporary credentials for it from the provider's OIDC login.",
		Required:    true,
		Validators: []validator.String{
			validators.AccountName{},
		},
	}
}

// SplitImportID splits an import ID of the form "<account_name>/<rest>".
// Account names cannot contain "/", so the first slash is the separator and
// rest may itself contain slashes (e.g. policy ARNs).
func SplitImportID(id string) (accountName, rest string, err error) {
	accountName, rest, ok := strings.Cut(id, "/")
	if !ok || accountName == "" || rest == "" {
		return "", "", fmt.Errorf("expected import ID of the form <account_name>/<id>, got %q", id)
	}
	return accountName, rest, nil
}

// ImportAccount sets account_name from the import ID and returns the rest of
// the ID for the resource to parse. ok is false if diagnostics were added.
func ImportAccount(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) (rest string, ok bool) {
	accountName, rest, err := SplitImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return "", false
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(AttrAccountName), accountName)...)
	return rest, !resp.Diagnostics.HasError()
}

// ImportByID handles "<account_name>/<value>" for resources identified by a
// single attribute.
func ImportByID(ctx context.Context, idAttr string, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	rest, ok := ImportAccount(ctx, req, resp)
	if !ok {
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(idAttr), rest)...)
}
