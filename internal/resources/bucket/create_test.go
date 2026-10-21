package bucket

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// newTestBucketResource wires a BucketResource to mock OIDC, IAM, STS and S3
// servers. s3 handles the S3 requests.
func newTestBucketResource(t *testing.T, s3 http.HandlerFunc) *BucketResource {
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

	return &BucketResource{
		accounts: client.NewAccountCredentialSource(
			client.NewIAMClient(iam.URL, "us-east-1", false),
			client.NewSTSClient(sts.URL, "us-east-1", false),
			client.NewOIDCTokenSource(oidc.URL, "realm", "client", "openid", "user", "pass", false),
		),
		s3Client: client.NewS3Client(s3Server.URL, "us-east-1", false),
	}
}

// A failure after the bucket is created must leave the bucket in state, so
// the next apply replaces it instead of failing on an orphaned bucket.
func TestCreateSavesBucketWhenVersioningFails(t *testing.T) {
	r := newTestBucketResource(t, func(w http.ResponseWriter, req *http.Request) {
		if req.URL.RawQuery == "versioning=" {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`<Error><Code>InternalError</Code><Message>boom</Message></Error>`))
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	s := schemaResp.Schema
	objType := s.Type().TerraformType(ctx)

	plan := tftypes.NewValue(objType, map[string]tftypes.Value{
		"name":                tftypes.NewValue(tftypes.String, "bkt"),
		"location_constraint": tftypes.NewValue(tftypes.String, nil),
		"versioning_enabled":  tftypes.NewValue(tftypes.Bool, true),
		"object_lock_enabled": tftypes.NewValue(tftypes.Bool, false),
		"account_name":        tftypes.NewValue(tftypes.String, "app"),
		"force_destroy":       tftypes.NewValue(tftypes.Bool, nil),
	})
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(objType, nil)}}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: s, Raw: plan}}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error from the failed versioning call")
	}
	var got BucketResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("state not saved: %v", diags)
	}
	if got.Name.ValueString() != "bkt" || got.VersioningEnabled.ValueBool() {
		t.Errorf("state = %+v, want bucket bkt with versioning false", got)
	}
}

// fakeBucketS3 serves one bucket holding object versions; deleting the bucket
// fails with BucketNotEmpty until every version is deleted.
func fakeBucketS3(t *testing.T, versions map[string]bool) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.Method == http.MethodGet && req.URL.Query().Has("versions"):
			body := `<ListVersionsResult><IsTruncated>false</IsTruncated>`
			for v := range versions {
				body += `<Version><Key>obj</Key><VersionId>` + v + `</VersionId></Version>`
			}
			_, _ = w.Write([]byte(body + `</ListVersionsResult>`))
		case req.Method == http.MethodDelete && req.URL.Path == "/bkt/obj":
			if req.Header.Get("X-Amz-Bypass-Governance-Retention") != "" {
				t.Errorf("bypass header sent on a bucket without object lock")
			}
			delete(versions, req.URL.Query().Get("versionId"))
			w.WriteHeader(http.StatusNoContent)
		case req.Method == http.MethodDelete && req.URL.Path == "/bkt":
			if len(versions) > 0 {
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`<Error><Code>BucketNotEmpty</Code><Message>The bucket you tried to delete is not empty.</Message></Error>`))
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s %s", req.Method, req.URL)
		}
	}
}

func deleteBucket(t *testing.T, r *BucketResource, forceDestroy, objectLock bool) *resource.DeleteResponse {
	t.Helper()
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	s := schemaResp.Schema
	state := tftypes.NewValue(s.Type().TerraformType(ctx), map[string]tftypes.Value{
		"name":                tftypes.NewValue(tftypes.String, "bkt"),
		"location_constraint": tftypes.NewValue(tftypes.String, nil),
		"versioning_enabled":  tftypes.NewValue(tftypes.Bool, true),
		"object_lock_enabled": tftypes.NewValue(tftypes.Bool, objectLock),
		"account_name":        tftypes.NewValue(tftypes.String, "app"),
		"force_destroy":       tftypes.NewValue(tftypes.Bool, forceDestroy),
	})
	resp := &resource.DeleteResponse{}
	r.Delete(ctx, resource.DeleteRequest{State: tfsdk.State{Schema: s, Raw: state}}, resp)
	return resp
}

func TestDeleteNonEmptyBucket(t *testing.T) {
	versions := map[string]bool{"v1": true, "v2": true}
	resp := deleteBucket(t, newTestBucketResource(t, fakeBucketS3(t, versions)), false, false)
	if !resp.Diagnostics.HasError() || resp.Diagnostics.Errors()[0].Summary() != "Bucket not empty" {
		t.Fatalf("diagnostics = %v, want Bucket not empty", resp.Diagnostics)
	}
	if len(versions) != 2 {
		t.Errorf("versions deleted without force_destroy: %v", versions)
	}
}

func TestDeleteForceDestroyEmptiesBucket(t *testing.T) {
	versions := map[string]bool{"v1": true, "v2": true}
	resp := deleteBucket(t, newTestBucketResource(t, fakeBucketS3(t, versions)), true, false)
	if resp.Diagnostics.HasError() {
		t.Fatalf("diagnostics = %v", resp.Diagnostics)
	}
	if len(versions) != 0 {
		t.Errorf("versions left: %v", versions)
	}
}
