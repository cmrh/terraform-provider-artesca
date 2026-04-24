package bucket

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/scality/terraform-provider-scality-artesca/internal/validators"
)

func TestSchema_Validators(t *testing.T) {
	r := NewBucketResource()
	ctx := context.Background()
	resp := resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, &resp)

	t.Run("name has BucketName validator", func(t *testing.T) {
		attr := resp.Schema.Attributes["name"].(schema.StringAttribute)
		if len(attr.Validators) != 1 {
			t.Fatalf("expected 1 validator, got %d", len(attr.Validators))
		}
		if _, ok := attr.Validators[0].(validators.BucketName); !ok {
			t.Errorf("expected BucketName validator, got %T", attr.Validators[0])
		}
	})
}

func TestSchema_Attributes(t *testing.T) {
	r := NewBucketResource()
	ctx := context.Background()
	resp := resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, &resp)

	requiredAttrs := []string{"name", "account_access_key", "account_secret_key"}
	for _, name := range requiredAttrs {
		t.Run(name+" is required", func(t *testing.T) {
			attr, ok := resp.Schema.Attributes[name]
			if !ok {
				t.Fatalf("attribute %q not found", name)
			}
			if !attr.IsRequired() {
				t.Errorf("attribute %q should be required", name)
			}
		})
	}

	sensitiveAttrs := []string{"account_access_key", "account_secret_key"}
	for _, name := range sensitiveAttrs {
		t.Run(name+" is sensitive", func(t *testing.T) {
			attr := resp.Schema.Attributes[name].(schema.StringAttribute)
			if !attr.Sensitive {
				t.Errorf("attribute %q should be sensitive", name)
			}
		})
	}
}

func TestSchema_Metadata(t *testing.T) {
	r := NewBucketResource()
	ctx := context.Background()
	resp := resource.MetadataResponse{}
	r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "artesca"}, &resp)

	if resp.TypeName != "artesca_bucket" {
		t.Errorf("TypeName = %q, want artesca_bucket", resp.TypeName)
	}
}
