package bucketobjectlock

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func newTestResource(t *testing.T, s3 http.HandlerFunc) *BucketObjectLockResource {
	t.Helper()
	serve := func(body string) *httptest.Server {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(body))
		}))
		t.Cleanup(srv.Close)
		return srv
	}
	oidc := serve(`{"access_token":"mock-token","expires_in":3600,"token_type":"bearer"}`)
	iam := serve(`{"IsTruncated":false,"Accounts":[{"Name":"app","CanonicalId":"cid",
 "Roles":[{"Name":"storage-manager-role","Arn":"arn:aws:iam::111111111111:role/scality-internal/storage-manager-role"}]}]}`)
	sts := serve(`<AssumeRoleWithWebIdentityResponse><AssumeRoleWithWebIdentityResult><Credentials>` +
		`<AccessKeyId>ak</AccessKeyId><SecretAccessKey>sk</SecretAccessKey><SessionToken>tok</SessionToken>` +
		`<Expiration>` + time.Now().Add(time.Hour).UTC().Format(time.RFC3339) + `</Expiration>` +
		`</Credentials></AssumeRoleWithWebIdentityResult></AssumeRoleWithWebIdentityResponse>`)
	s3Server := httptest.NewServer(s3)
	t.Cleanup(s3Server.Close)
	return &BucketObjectLockResource{
		accounts: client.NewAccountCredentialSource(
			client.NewIAMClient(iam.URL, "us-east-1", false),
			client.NewSTSClient(sts.URL, "us-east-1", false),
			client.NewOIDCTokenSource(oidc.URL, "realm", "client", "openid", "user", "pass", false),
		),
		s3Client: client.NewS3Client(s3Server.URL, "us-east-1", false),
	}
}

func value(t *testing.T, r *BucketObjectLockResource, mode string, days, years any) (tftypes.Value, tfsdk.Plan) {
	t.Helper()
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	v := tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), map[string]tftypes.Value{
		"account_name":    tftypes.NewValue(tftypes.String, "app"),
		"bucket_name":     tftypes.NewValue(tftypes.String, "bkt"),
		"retention_mode":  tftypes.NewValue(tftypes.String, mode),
		"retention_days":  tftypes.NewValue(tftypes.Number, days),
		"retention_years": tftypes.NewValue(tftypes.Number, years),
	})
	return v, tfsdk.Plan{Schema: schemaResp.Schema, Raw: v}
}

func TestValidateConfigRetentionPeriod(t *testing.T) {
	r := NewBucketObjectLockResource().(*BucketObjectLockResource)
	for _, tc := range []struct {
		name        string
		days, years any
		wantErr     bool
	}{
		{"days", 30, nil, false},
		{"years", nil, 1, false},
		{"both", 30, 1, true},
		{"neither", nil, nil, true},
		{"unknown", tftypes.UnknownValue, nil, false},
	} {
		v, plan := value(t, r, "GOVERNANCE", tc.days, tc.years)
		resp := &resource.ValidateConfigResponse{}
		r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: plan.Schema, Raw: v}}, resp)
		if resp.Diagnostics.HasError() != tc.wantErr {
			t.Errorf("%s: diagnostics = %v, want error %v", tc.name, resp.Diagnostics, tc.wantErr)
		}
	}
}

func TestReadMapsRetention(t *testing.T) {
	for _, tc := range []struct {
		name       string
		body       string
		days, yrs  int64
		wantRemove bool
	}{
		{"days", `<ObjectLockConfiguration><ObjectLockEnabled>Enabled</ObjectLockEnabled><Rule><DefaultRetention><Mode>COMPLIANCE</Mode><Days>7</Days></DefaultRetention></Rule></ObjectLockConfiguration>`, 7, 0, false},
		{"years", `<ObjectLockConfiguration><ObjectLockEnabled>Enabled</ObjectLockEnabled><Rule><DefaultRetention><Mode>COMPLIANCE</Mode><Years>2</Years></DefaultRetention></Rule></ObjectLockConfiguration>`, 0, 2, false},
		{"removed out of band", `<ObjectLockConfiguration><ObjectLockEnabled>Enabled</ObjectLockEnabled></ObjectLockConfiguration>`, 0, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newTestResource(t, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(tc.body))
			})
			v, plan := value(t, r, "GOVERNANCE", 30, nil)
			state := tfsdk.State{Schema: plan.Schema, Raw: v}
			resp := &resource.ReadResponse{State: state}
			r.Read(context.Background(), resource.ReadRequest{State: state}, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("diagnostics = %v", resp.Diagnostics)
			}
			if tc.wantRemove {
				if !resp.State.Raw.IsNull() {
					t.Error("expected the resource to be removed from state")
				}
				return
			}
			var got BucketObjectLockResourceModel
			resp.State.Get(context.Background(), &got)
			if got.RetentionMode.ValueString() != "COMPLIANCE" || got.RetentionDays.ValueInt64() != tc.days || got.RetentionYears.ValueInt64() != tc.yrs {
				t.Errorf("state = %+v", got)
			}
			if (tc.days == 0) != got.RetentionDays.IsNull() || (tc.yrs == 0) != got.RetentionYears.IsNull() {
				t.Errorf("unset period should be null: %+v", got)
			}
		})
	}
}

func TestCreateOnBucketWithoutObjectLock(t *testing.T) {
	r := newTestResource(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`<Error><Code>InvalidBucketState</Code><Message>Object Lock configuration cannot be enabled on existing buckets</Message></Error>`))
	})
	_, plan := value(t, r, "GOVERNANCE", 30, nil)
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema, Raw: tftypes.NewValue(plan.Raw.Type(), nil)}}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "object_lock_enabled = true") {
		t.Fatalf("diagnostics = %v, want a hint to set object_lock_enabled", resp.Diagnostics)
	}
}

func TestDeleteRemovesDefaultRetention(t *testing.T) {
	var body string
	r := newTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPut || req.URL.RawQuery != "object-lock=" {
			t.Errorf("request = %s %s", req.Method, req.URL)
		}
		b, _ := io.ReadAll(req.Body)
		body = string(b)
	})
	v, plan := value(t, r, "GOVERNANCE", 30, nil)
	resp := &resource.DeleteResponse{}
	r.Delete(context.Background(), resource.DeleteRequest{State: tfsdk.State{Schema: plan.Schema, Raw: v}}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("diagnostics = %v", resp.Diagnostics)
	}
	if !strings.Contains(body, "<ObjectLockEnabled>Enabled</ObjectLockEnabled>") || strings.Contains(body, "<Rule>") {
		t.Errorf("body = %s, want object lock enabled without a Rule", body)
	}
}
