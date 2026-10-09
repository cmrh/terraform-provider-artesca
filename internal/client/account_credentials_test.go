package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func assumeRoleWithWebIdentityXML(expiration time.Time) string {
	return `<AssumeRoleWithWebIdentityResponse xmlns="https://iam.amazonaws.com/doc/2010-05-08/"><AssumeRoleWithWebIdentityResult>` +
		`<AssumedRoleUser><Arn>arn:aws:sts::111111111111:assumed-role/storage-manager-role/terraform-provider-artesca</Arn><AssumedRoleId>RID:terraform-provider-artesca</AssumedRoleId></AssumedRoleUser>` +
		`<Credentials><SecretAccessKey>tmp-sk</SecretAccessKey><AccessKeyId>tmp-ak</AccessKeyId><SessionToken>tmp-token</SessionToken><Expiration>` +
		expiration.UTC().Format(time.RFC3339) + `</Expiration></Credentials></AssumeRoleWithWebIdentityResult></AssumeRoleWithWebIdentityResponse>`
}

func TestAssumeRoleWithWebIdentity(t *testing.T) {
	exp := time.Date(2026, 10, 2, 19, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("request must not be SigV4-signed")
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("parsing form: %v", err)
			return
		}
		want := map[string]string{
			"Action":           "AssumeRoleWithWebIdentity",
			"Version":          "2011-06-15",
			"RoleArn":          "arn:aws:iam::111111111111:role/scality-internal/storage-manager-role",
			"RoleSessionName":  "sess",
			"WebIdentityToken": "tok",
			"DurationSeconds":  "3600",
		}
		for k, v := range want {
			if got := r.PostForm.Get(k); got != v {
				t.Errorf("%s = %q, want %q", k, got, v)
			}
		}
		_, _ = w.Write([]byte(assumeRoleWithWebIdentityXML(exp)))
	}))
	defer server.Close()

	sts := NewSTSClient(server.URL, "us-east-1", false)
	got, err := sts.AssumeRoleWithWebIdentity(context.Background(), "tok",
		"arn:aws:iam::111111111111:role/scality-internal/storage-manager-role", "sess", 3600)
	if err != nil {
		t.Fatalf("AssumeRoleWithWebIdentity returned error: %v", err)
	}
	if got.AccessKeyID != "tmp-ak" || got.SecretAccessKey != "tmp-sk" || got.SessionToken != "tmp-token" || !got.Expiration.Equal(exp) {
		t.Errorf("got %+v", got)
	}
}

func TestAssumeRoleWithWebIdentityError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(xmlErrorResponse("AccessDenied", "Not authorized")))
	}))
	defer server.Close()

	sts := NewSTSClient(server.URL, "us-east-1", false)
	_, err := sts.AssumeRoleWithWebIdentity(context.Background(), "tok", "arn:aws:iam::1:role/r", "sess", 0)
	if err == nil || !strings.Contains(err.Error(), "AccessDenied") {
		t.Fatalf("err = %v, want AccessDenied", err)
	}
}

func TestDeriveSTSEndpointFromManagement(t *testing.T) {
	got, err := DeriveSTSEndpointFromManagement("https://management.artesca.example.com:8443/api/v1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://sts.artesca.example.com:8443" {
		t.Errorf("got %q", got)
	}
	if _, err := DeriveSTSEndpointFromManagement("https://api.artesca.example.com"); err == nil {
		t.Error("expected error for non-management hostname")
	}
}

// newTestAccountCredentialSource wires a credential source to mock IAM, STS
// and OIDC servers. accountsJSON is the GetRolesForWebIdentity response.
func newTestAccountCredentialSource(t *testing.T, accountsJSON string, stsCalls *int32, expiration func() time.Time) *AccountCredentialSource {
	t.Helper()
	iamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(accountsJSON))
	}))
	stsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(stsCalls, 1)
		if err := r.ParseForm(); err != nil {
			t.Errorf("parsing form: %v", err)
			return
		}
		if got := r.PostForm.Get("RoleArn"); got != "arn:aws:iam::111111111111:role/scality-internal/storage-manager-role" {
			t.Errorf("RoleArn = %q", got)
		}
		_, _ = w.Write([]byte(assumeRoleWithWebIdentityXML(expiration())))
	}))
	oidcServer := newMockOIDCServer(t)
	t.Cleanup(func() {
		iamServer.Close()
		stsServer.Close()
		oidcServer.Close()
	})

	tokens := NewOIDCTokenSource(oidcServer.URL, "realm", "client", "openid", "user", "pass", false)
	return NewAccountCredentialSource(
		NewIAMClient(iamServer.URL, "us-east-1", false),
		NewSTSClient(stsServer.URL, "us-east-1", false),
		tokens,
	)
}

const oneAccountJSON = `{"IsTruncated":false,"Accounts":[{"Name":"app","CanonicalId":"cid",
 "Roles":[{"Name":"storage-manager-role","Arn":"arn:aws:iam::111111111111:role/scality-internal/storage-manager-role"}]}]}`

func TestAccountCredentialSourceCaches(t *testing.T) {
	var calls int32
	now := time.Date(2026, 10, 2, 18, 0, 0, 0, time.UTC)
	src := newTestAccountCredentialSource(t, oneAccountJSON, &calls, func() time.Time { return now.Add(time.Hour) })
	src.now = func() time.Time { return now }

	c1, err := src.For(context.Background(), "app")
	if err != nil {
		t.Fatalf("For returned error: %v", err)
	}
	if c1 != (Credentials{AccessKey: "tmp-ak", SecretKey: "tmp-sk", SessionToken: "tmp-token"}) {
		t.Errorf("credentials = %+v", c1)
	}
	if _, err := src.For(context.Background(), "app"); err != nil {
		t.Fatalf("second For returned error: %v", err)
	}
	if calls != 1 {
		t.Errorf("STS calls = %d, want 1 (second call should hit the cache)", calls)
	}

	// Within the refresh window before expiry, credentials are re-issued.
	src.now = func() time.Time { return now.Add(time.Hour - accountCredentialRefresh + time.Second) }
	if _, err := src.For(context.Background(), "app"); err != nil {
		t.Fatalf("refresh For returned error: %v", err)
	}
	if calls != 2 {
		t.Errorf("STS calls = %d, want 2 after entering the refresh window", calls)
	}

	// Forget drops the cache.
	src.Forget("app")
	if _, err := src.For(context.Background(), "app"); err != nil {
		t.Fatalf("For after Forget returned error: %v", err)
	}
	if calls != 3 {
		t.Errorf("STS calls = %d, want 3 after Forget", calls)
	}
}

func TestAccountCredentialSourceAccountNotFound(t *testing.T) {
	var calls int32
	src := newTestAccountCredentialSource(t, oneAccountJSON, &calls, func() time.Time { return time.Now().Add(time.Hour) })

	_, err := src.For(context.Background(), "missing")
	if err == nil || !strings.Contains(err.Error(), `account "missing" not found`) {
		t.Fatalf("err = %v, want not-found error", err)
	}
	if calls != 0 {
		t.Errorf("STS calls = %d, want 0 for unknown account", calls)
	}
}

func TestAccountCredentialSourceEmptyName(t *testing.T) {
	var calls int32
	src := newTestAccountCredentialSource(t, oneAccountJSON, &calls, time.Now)
	if _, err := src.For(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty account name")
	}
}

// Session tokens must be sent and covered by the SigV4 signature.
func TestSessionTokenSigning(t *testing.T) {
	check := func(t *testing.T, r *http.Request) {
		t.Helper()
		if got := r.Header.Get("X-Amz-Security-Token"); got != "tmp-token" {
			t.Errorf("X-Amz-Security-Token = %q, want tmp-token", got)
		}
		if auth := r.Header.Get("Authorization"); !strings.Contains(auth, "x-amz-security-token") {
			t.Errorf("SignedHeaders does not include x-amz-security-token: %s", auth)
		}
	}
	creds := Credentials{AccessKey: "tmp-ak", SecretKey: "tmp-sk", SessionToken: "tmp-token"}

	t.Run("IAM", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			check(t, r)
			_, _ = w.Write([]byte(`<GetUserResponse><GetUserResult><User><UserName>u</UserName></User></GetUserResult></GetUserResponse>`))
		}))
		defer server.Close()
		if _, err := NewIAMClient(server.URL, "us-east-1", false).GetUser(context.Background(), creds, "u"); err != nil {
			t.Fatalf("GetUser returned error: %v", err)
		}
	})

	t.Run("S3", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			check(t, r)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()
		if _, err := NewS3Client(server.URL, "us-east-1", false).HeadBucket(context.Background(), creds, "b"); err != nil {
			t.Fatalf("HeadBucket returned error: %v", err)
		}
	})

	t.Run("no token, not signed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Amz-Security-Token") != "" || strings.Contains(r.Header.Get("Authorization"), "x-amz-security-token") {
				t.Error("security token header present without a session token")
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()
		static := Credentials{AccessKey: "ak", SecretKey: "sk"}
		if _, err := NewS3Client(server.URL, "us-east-1", false).HeadBucket(context.Background(), static, "b"); err != nil {
			t.Fatalf("HeadBucket returned error: %v", err)
		}
	})
}

func TestAccountCredentialSourceNameLookups(t *testing.T) {
	iamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(oneAccountJSON))
	}))
	defer iamServer.Close()
	stsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parsing form: %v", err)
			return
		}
		if r.PostForm.Get("Action") != "GetCallerIdentity" {
			t.Errorf("Action = %q, want GetCallerIdentity", r.PostForm.Get("Action"))
		}
		if !strings.Contains(r.Header.Get("Authorization"), "Credential=old-ak/") {
			t.Errorf("request not signed with the old access key: %s", r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`<GetCallerIdentityResponse><GetCallerIdentityResult><UserId>111111111111</UserId><Account>111111111111</Account><Arn>arn:aws:iam::111111111111:root</Arn></GetCallerIdentityResult></GetCallerIdentityResponse>`))
	}))
	defer stsServer.Close()
	oidcServer := newMockOIDCServer(t)
	defer oidcServer.Close()

	src := NewAccountCredentialSource(
		NewIAMClient(iamServer.URL, "us-east-1", false),
		NewSTSClient(stsServer.URL, "us-east-1", false),
		NewOIDCTokenSource(oidcServer.URL, "realm", "client", "openid", "user", "pass", false),
	)

	name, err := src.NameForAccountID(context.Background(), "111111111111")
	if err != nil || name != "app" {
		t.Errorf("NameForAccountID = (%q, %v), want (app, nil)", name, err)
	}
	if _, err := src.NameForAccountID(context.Background(), "999999999999"); err == nil {
		t.Error("expected error for unknown account ID")
	}
	name, err = src.NameForAccessKey(context.Background(), "old-ak", "old-sk")
	if err != nil || name != "app" {
		t.Errorf("NameForAccessKey = (%q, %v), want (app, nil)", name, err)
	}
}

func TestAccountCredentialSourceAccountID(t *testing.T) {
	var calls int32
	src := newTestAccountCredentialSource(t, oneAccountJSON, &calls, time.Now)

	id, err := src.AccountID(context.Background(), "app")
	if err != nil || id != "111111111111" {
		t.Fatalf("AccountID = (%q, %v), want 111111111111", id, err)
	}
	if _, err := src.AccountID(context.Background(), "missing"); err == nil || !strings.Contains(err.Error(), `account "missing" not found`) {
		t.Errorf("err = %v, want not-found error", err)
	}
	if _, err := src.AccountID(context.Background(), ""); err == nil {
		t.Error("expected error for empty account name")
	}
	if calls != 0 {
		t.Errorf("STS calls = %d, want 0 (AccountID needs no credentials)", calls)
	}
}
