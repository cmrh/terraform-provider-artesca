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

	apiDetailsToModel(ctx, &client.LocationDetails{StsEndpoint: "https://sts2.dst.example.com"}, model.Details)
	if got := model.Details.StsEndpoint.ValueString(); got != "https://sts2.dst.example.com" {
		t.Errorf("sts_endpoint read back as %q", got)
	}
}
