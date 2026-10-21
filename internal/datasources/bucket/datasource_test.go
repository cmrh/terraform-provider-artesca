package bucket

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func newTestDataSource(t *testing.T, s3 http.HandlerFunc) *BucketDataSource {
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
	return &BucketDataSource{
		accounts: client.NewAccountCredentialSource(
			client.NewIAMClient(iam.URL, "us-east-1", false),
			client.NewSTSClient(sts.URL, "us-east-1", false),
			client.NewOIDCTokenSource(oidc.URL, "realm", "client", "openid", "user", "pass", false),
		),
		s3Client: client.NewS3Client(s3Server.URL, "us-east-1", false),
	}
}

func read(t *testing.T, d *BucketDataSource) (*datasource.ReadResponse, BucketDataSourceModel) {
	t.Helper()
	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	d.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	typ := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	vals := map[string]tftypes.Value{}
	for name, attrType := range typ.AttributeTypes {
		vals[name] = tftypes.NewValue(attrType, nil)
	}
	vals["account_name"] = tftypes.NewValue(tftypes.String, "app")
	vals["name"] = tftypes.NewValue(tftypes.String, "bkt")
	config := tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(typ, vals)}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(typ, nil)}}
	d.Read(ctx, datasource.ReadRequest{Config: config}, resp)
	var got BucketDataSourceModel
	if !resp.Diagnostics.HasError() {
		resp.State.Get(ctx, &got)
	}
	return resp, got
}

func TestReadBucket(t *testing.T) {
	for _, tc := range []struct {
		name       string
		location   string
		tagging    string
		taggingErr bool
		wantLoc    string
		wantTags   map[string]string
	}{
		{"default location, no tags", "", "", true, "us-east-1", map[string]string{}},
		{"named location, tags", "ring", `<Tagging><TagSet><Tag><Key>team</Key><Value>data</Value></Tag></TagSet></Tagging>`, false, "ring", map[string]string{"team": "data"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := newTestDataSource(t, func(w http.ResponseWriter, req *http.Request) {
				switch req.URL.RawQuery {
				case "":
					w.WriteHeader(http.StatusOK) // HeadBucket
				case "location=":
					_, _ = w.Write([]byte(`<LocationConstraint>` + tc.location + `</LocationConstraint>`))
				case "versioning=":
					_, _ = w.Write([]byte(`<VersioningConfiguration><Status>Enabled</Status></VersioningConfiguration>`))
				case "object-lock=":
					_, _ = w.Write([]byte(`<ObjectLockConfiguration><ObjectLockEnabled>Enabled</ObjectLockEnabled></ObjectLockConfiguration>`))
				case "tagging=":
					if tc.taggingErr {
						w.WriteHeader(http.StatusNotFound)
						_, _ = w.Write([]byte(`<Error><Code>NoSuchTagSet</Code><Message>The TagSet does not exist</Message></Error>`))
						return
					}
					_, _ = w.Write([]byte(tc.tagging))
				default:
					t.Errorf("unexpected %s %s", req.Method, req.URL)
				}
			})
			resp, got := read(t, d)
			if resp.Diagnostics.HasError() {
				t.Fatalf("diagnostics = %v", resp.Diagnostics)
			}
			if got.ARN.ValueString() != "arn:aws:s3:::bkt" || got.LocationConstraint.ValueString() != tc.wantLoc ||
				!got.VersioningEnabled.ValueBool() || !got.ObjectLockEnabled.ValueBool() {
				t.Errorf("got %+v", got)
			}
			tags := map[string]string{}
			got.Tags.ElementsAs(context.Background(), &tags, false)
			if got.Tags.IsNull() || len(tags) != len(tc.wantTags) || tags["team"] != tc.wantTags["team"] {
				t.Errorf("tags = %v, want %v", got.Tags, tc.wantTags)
			}
		})
	}
}

func TestReadBucketNotFound(t *testing.T) {
	d := newTestDataSource(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	resp, _ := read(t, d)
	if !resp.Diagnostics.HasError() || resp.Diagnostics.Errors()[0].Summary() != "Bucket not found" {
		t.Fatalf("diagnostics = %v, want Bucket not found", resp.Diagnostics)
	}
}
