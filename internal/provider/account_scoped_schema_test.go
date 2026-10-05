package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// Account-scoped resources and data sources authenticate via account_name;
// none may still take account root keys.
var accountScopedResources = []string{
	"artesca_bucket", "artesca_bucket_encryption", "artesca_bucket_policy", "artesca_bucket_tagging",
	"artesca_group", "artesca_group_membership", "artesca_group_policy", "artesca_group_policy_attachment",
	"artesca_policy", "artesca_role", "artesca_role_policy_attachment",
	"artesca_user", "artesca_user_access_key", "artesca_user_policy", "artesca_user_policy_attachment",
	"artesca_bucket_workflow_expiration", "artesca_bucket_workflow_transition", "artesca_bucket_workflow_replication",
}

var accountScopedDataSources = []string{
	"artesca_group", "artesca_policy", "artesca_role", "artesca_user",
}

func TestAccountScopedResourceSchemas(t *testing.T) {
	ctx := context.Background()
	p := &ArtescaProvider{}
	byName := map[string]resource.Resource{}
	for _, f := range p.Resources(ctx) {
		r := f()
		meta := resource.MetadataResponse{}
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "artesca"}, &meta)
		byName[meta.TypeName] = r

		s := resource.SchemaResponse{}
		r.Schema(ctx, resource.SchemaRequest{}, &s)
		for _, old := range []string{"account_access_key", "account_secret_key"} {
			if _, ok := s.Schema.Attributes[old]; ok {
				t.Errorf("%s still has %s", meta.TypeName, old)
			}
		}
	}

	for _, name := range accountScopedResources {
		t.Run(name, func(t *testing.T) {
			r, ok := byName[name]
			if !ok {
				t.Fatalf("resource %s not registered", name)
			}
			s := resource.SchemaResponse{}
			r.Schema(ctx, resource.SchemaRequest{}, &s)
			attr, ok := s.Schema.Attributes["account_name"].(schema.StringAttribute)
			if !ok {
				t.Fatal("missing account_name string attribute")
			}
			if !attr.Required {
				t.Error("account_name must be required")
			}
			if len(attr.PlanModifiers) == 0 {
				t.Error("account_name must force replacement")
			}
		})
	}
}

func TestAccountScopedDataSourceSchemas(t *testing.T) {
	ctx := context.Background()
	p := &ArtescaProvider{}
	byName := map[string]datasource.DataSource{}
	for _, f := range p.DataSources(ctx) {
		d := f()
		meta := datasource.MetadataResponse{}
		d.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "artesca"}, &meta)
		byName[meta.TypeName] = d

		s := datasource.SchemaResponse{}
		d.Schema(ctx, datasource.SchemaRequest{}, &s)
		for _, old := range []string{"account_access_key", "account_secret_key"} {
			if _, ok := s.Schema.Attributes[old]; ok {
				t.Errorf("data source %s still has %s", meta.TypeName, old)
			}
		}
	}

	for _, name := range accountScopedDataSources {
		t.Run(name, func(t *testing.T) {
			d, ok := byName[name]
			if !ok {
				t.Fatalf("data source %s not registered", name)
			}
			s := datasource.SchemaResponse{}
			d.Schema(ctx, datasource.SchemaRequest{}, &s)
			attr, ok := s.Schema.Attributes["account_name"]
			if !ok {
				t.Fatal("missing account_name attribute")
			}
			if !attr.IsRequired() {
				t.Error("account_name must be required")
			}
		})
	}
}
