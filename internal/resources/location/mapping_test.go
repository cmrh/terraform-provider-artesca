package location

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestStsEndpointMapping(t *testing.T) {
	ctx := context.Background()
	model := &LocationResourceModel{
		Name:         types.StringValue("crr-loc"),
		LocationType: types.StringValue("location-scality-crr-v1"),
		Details: &LocationDetailsModel{
			Endpoint:    types.StringValue("https://s3.dst.example.com"),
			StsEndpoint: types.StringValue("https://sts.dst.example.com"),
		},
	}

	body, err := json.Marshal(modelToAPILocation(ctx, model))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"stsEndpoint":"https://sts.dst.example.com"`) {
		t.Errorf("request body missing stsEndpoint: %s", body)
	}

	apiDetailsToModel(ctx, &client.LocationDetails{StsEndpoint: "https://sts2.dst.example.com"}, model.Details, false)
	if got := model.Details.StsEndpoint.ValueString(); got != "https://sts2.dst.example.com" {
		t.Errorf("sts_endpoint read back as %q", got)
	}
}

func TestApiDetailsToModelImported(t *testing.T) {
	ctx := context.Background()
	matchFalse := false
	sseFalse := false
	d := &client.LocationDetails{
		AccessKey:            "AKIAEXAMPLE",
		BucketName:           "target",
		BucketMatch:          &matchFalse,
		Endpoint:             "http://ring.example.com:8080",
		Region:               "us-east-1",
		ServerSideEncryption: &sseFalse,
	}

	normal := &LocationDetailsModel{}
	apiDetailsToModel(ctx, d, normal, false)
	if !normal.Endpoint.IsNull() || !normal.BucketName.IsNull() {
		t.Errorf("outside import, unset fields should stay null: endpoint=%s bucket_name=%s", normal.Endpoint, normal.BucketName)
	}

	imported := &LocationDetailsModel{}
	apiDetailsToModel(ctx, d, imported, true)
	for name, got := range map[string]types.String{
		"access_key":  imported.AccessKey,
		"bucket_name": imported.BucketName,
		"endpoint":    imported.Endpoint,
		"region":      imported.Region,
	} {
		if got.IsNull() || got.ValueString() == "" {
			t.Errorf("after import, %s is empty", name)
		}
	}
	if imported.BucketMatch.IsNull() || imported.BucketMatch.ValueBool() {
		t.Errorf("after import, bucket_match = %s, want false", imported.BucketMatch)
	}
	if !imported.ServerSideEncryption.IsNull() {
		t.Errorf("server_side_encryption false should stay null, got %s", imported.ServerSideEncryption)
	}
	if !imported.SecretKey.IsNull() {
		t.Errorf("secret_key should stay null, got %s", imported.SecretKey)
	}
}
