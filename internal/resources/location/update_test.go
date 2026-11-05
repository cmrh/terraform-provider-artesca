package location

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestCRRFlag(t *testing.T) {
	if got := crrFlag("location-scality-crr-v1"); got == nil || !*got {
		t.Errorf("crr-v1: crrFlag = %v, want true", got)
	}
	for _, typ := range []string{"location-scality-ring-s3-v1", "location-scality-sproxyd-v1", "location-aws-s3-v1"} {
		if got := crrFlag(typ); got != nil {
			t.Errorf("%s: crrFlag = %v, want unset", typ, *got)
		}
	}
}

// Update sends isCRR: true for a CRR location, which ARTESCA requires, and
// leaves it out for other types.
func TestUpdateSendsCRRFlag(t *testing.T) {
	for _, tc := range []struct {
		locationType string
		want         string
	}{
		{"location-scality-crr-v1", `"isCRR":true`},
		{"location-scality-ring-s3-v1", ""},
	} {
		t.Run(tc.locationType, func(t *testing.T) {
			var put string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, "/protocol/openid-connect/token") {
					_, _ = w.Write([]byte(`{"access_token":"tok","expires_in":300,"token_type":"bearer"}`))
					return
				}
				if r.Method != http.MethodPut {
					t.Errorf("unexpected %s %s", r.Method, r.URL)
					return
				}
				b, _ := io.ReadAll(r.Body)
				put = string(b)
				var loc client.Location
				_ = json.Unmarshal(b, &loc)
				_ = json.NewEncoder(w).Encode(loc)
			}))
			defer srv.Close()

			tokens := client.NewOIDCTokenSource(srv.URL, "realm", "client", "openid", "user", "pass", false)
			r := &LocationResource{client: client.NewManagementClient(srv.URL, "inst", tokens, false)}

			ctx := context.Background()
			schemaResp := &resource.SchemaResponse{}
			r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
			model := LocationResourceModel{
				Name:         types.StringValue("loc"),
				LocationType: types.StringValue(tc.locationType),
				ObjectID:     types.StringValue("obj-1"),
				Details: &LocationDetailsModel{
					Endpoint:      types.StringValue("http://ring.example.com/"),
					AccessKey:     types.StringValue("ak"),
					SecretKey:     types.StringValue("sk"),
					RepoID:        types.ListNull(types.StringType),
					BootstrapList: types.ListNull(types.StringType),
				},
			}
			plan := tfsdk.State{Schema: schemaResp.Schema}
			if diags := plan.Set(ctx, &model); diags.HasError() {
				t.Fatalf("building plan: %v", diags)
			}
			resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			r.Update(ctx, resource.UpdateRequest{Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: plan.Raw}}, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("diagnostics = %v", resp.Diagnostics)
			}
			switch {
			case tc.want != "" && !strings.Contains(put, tc.want):
				t.Errorf("PUT body = %s, want it to contain %s", put, tc.want)
			case tc.want == "" && strings.Contains(put, "isCRR"):
				t.Errorf("PUT body = %s, want no isCRR", put)
			}
		})
	}
}
