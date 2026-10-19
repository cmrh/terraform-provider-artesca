package location

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
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

	apiLoc, diags := modelToAPILocation(ctx, model)
	if diags.HasError() {
		t.Fatal(diags)
	}
	body, err := json.Marshal(apiLoc)
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

// A configured field the API no longer returns was cleared outside
// Terraform and must read as null, so the next plan restores it.
func TestApiDetailsToModelClearedFields(t *testing.T) {
	ctx := context.Background()
	model := &LocationDetailsModel{
		AccessKey:            types.StringValue("AK"),
		SecretKey:            types.StringValue("SK"),
		BucketName:           types.StringValue("target"),
		Endpoint:             types.StringValue("http://ring.example.com"),
		Region:               types.StringValue("eu-west-1"),
		ProxyPath:            types.StringValue("/proxy/"),
		ChordCos:             types.Int64Value(2),
		StsEndpoint:          types.StringValue("https://sts.example.com"),
		ServerSideEncryption: types.BoolValue(true),
	}
	apiDetailsToModel(ctx, &client.LocationDetails{}, model, false)

	for name, got := range map[string]types.String{
		"access_key": model.AccessKey, "bucket_name": model.BucketName, "endpoint": model.Endpoint,
		"region": model.Region, "proxy_path": model.ProxyPath,
	} {
		if !got.IsNull() {
			t.Errorf("%s = %s, want null", name, got)
		}
	}
	if !model.ChordCos.IsNull() {
		t.Errorf("chord_cos = %s, want null", model.ChordCos)
	}
	if model.ServerSideEncryption.ValueBool() {
		t.Errorf("server_side_encryption = %s, want false", model.ServerSideEncryption)
	}
	// Never returned by the API: kept.
	if model.SecretKey.ValueString() != "SK" {
		t.Errorf("secret_key = %s, want kept", model.SecretKey)
	}
	// Not verified to round-trip: kept.
	if model.StsEndpoint.ValueString() != "https://sts.example.com" {
		t.Errorf("sts_endpoint = %s, want kept", model.StsEndpoint)
	}
}

func TestApiLocationToModelSizeLimit(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name        string
		state       types.Int64
		api         int64
		wantNull    bool
		wantValueGB int64
	}{
		{"cleared outside Terraform", types.Int64Value(100), 0, true, 0},
		{"configured 0 kept", types.Int64Value(0), 0, false, 0},
		{"unset stays unset", types.Int64Null(), 0, true, 0},
		{"changed outside Terraform", types.Int64Value(100), 50, false, 50},
	} {
		t.Run(tc.name, func(t *testing.T) {
			model := &LocationResourceModel{SizeLimitGB: tc.state}
			apiLocationToModel(ctx, &client.Location{Name: "l", LocationType: "t", SizeLimitGB: tc.api}, model, false)
			if model.SizeLimitGB.IsNull() != tc.wantNull || (!tc.wantNull && model.SizeLimitGB.ValueInt64() != tc.wantValueGB) {
				t.Errorf("size_limit_gb = %s", model.SizeLimitGB)
			}
		})
	}
}

func TestModelToAPIDetailsReportsListErrors(t *testing.T) {
	ctx := context.Background()
	d := &LocationDetailsModel{
		BootstrapList: types.ListValueMust(types.StringType, []attr.Value{types.StringNull()}),
	}
	if _, diags := modelToAPIDetails(ctx, d); !diags.HasError() {
		t.Error("expected a diagnostic for a null list element")
	}
}

func TestRequiredDetailsHdclientV2(t *testing.T) {
	if got := requiredDetailsByType["location-scality-hdclient-v2"]; len(got) != 1 || got[0] != "bootstrap_list" {
		t.Errorf("hdclient-v2 required details = %v, want [bootstrap_list]", got)
	}
}
