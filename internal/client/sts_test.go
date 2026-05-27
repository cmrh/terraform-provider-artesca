package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// DeriveSTSEndpoint
// ---------------------------------------------------------------------------

func TestDeriveSTSEndpoint(t *testing.T) {
	got, err := DeriveSTSEndpoint("https://s3.artesca.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://sts.artesca.example.com" {
		t.Errorf("got %q, want https://sts.artesca.example.com", got)
	}
}

func TestDeriveSTSEndpointWithPort(t *testing.T) {
	got, err := DeriveSTSEndpoint("https://s3.artesca.example.com:8443")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://sts.artesca.example.com:8443" {
		t.Errorf("got %q, want https://sts.artesca.example.com:8443", got)
	}
}

func TestDeriveSTSEndpointStripsPath(t *testing.T) {
	got, err := DeriveSTSEndpoint("https://s3.artesca.example.com/v1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://sts.artesca.example.com" {
		t.Errorf("got %q, want https://sts.artesca.example.com", got)
	}
}

func TestDeriveSTSEndpointNotS3(t *testing.T) {
	_, err := DeriveSTSEndpoint("https://api.artesca.example.com")
	if err == nil {
		t.Fatal("expected error for non-s3 hostname")
	}
	if !strings.Contains(err.Error(), "does not start with 's3.'") {
		t.Errorf("error = %q, want mention of 's3.'", err.Error())
	}
}

func TestDeriveSTSEndpointInvalidURL(t *testing.T) {
	_, err := DeriveSTSEndpoint("://bad")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

// ---------------------------------------------------------------------------
// AssumeRole
// ---------------------------------------------------------------------------

const assumeRoleSuccessXML = `<AssumeRoleResponse>
	<AssumeRoleResult>
		<Credentials>
			<AccessKeyId>AKIATEST</AccessKeyId>
			<SecretAccessKey>SECRET123</SecretAccessKey>
			<SessionToken>TOKEN_BLOB</SessionToken>
			<Expiration>2030-01-01T12:34:56Z</Expiration>
		</Credentials>
		<AssumedRoleUser>
			<AssumedRoleId>AROAEXAMPLE:test-session</AssumedRoleId>
			<Arn>arn:aws:sts::123:assumed-role/myrole/test-session</Arn>
		</AssumedRoleUser>
	</AssumeRoleResult>
</AssumeRoleResponse>`

func TestAssumeRole(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.Form.Get("Action"); got != "AssumeRole" {
			t.Errorf("Action = %q, want AssumeRole", got)
		}
		if got := r.Form.Get("RoleArn"); got != "arn:aws:iam::123:role/myrole" {
			t.Errorf("RoleArn = %q", got)
		}
		if got := r.Form.Get("RoleSessionName"); got != "test-session" {
			t.Errorf("RoleSessionName = %q", got)
		}
		if got := r.Form.Get("Version"); got != "2011-06-15" {
			t.Errorf("Version = %q, want 2011-06-15", got)
		}
		if _, set := r.Form["DurationSeconds"]; set {
			t.Errorf("DurationSeconds should not be set when zero")
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(assumeRoleSuccessXML))
	}))
	defer server.Close()

	client := NewSTSClient(server.URL, "us-east-1", false)
	creds, err := client.AssumeRole(context.Background(), "ak", "sk", "arn:aws:iam::123:role/myrole", "test-session", AssumeRoleOptions{})
	if err != nil {
		t.Fatalf("AssumeRole: %v", err)
	}
	if creds.AccessKeyID != "AKIATEST" {
		t.Errorf("AccessKeyID = %q", creds.AccessKeyID)
	}
	if creds.SecretAccessKey != "SECRET123" {
		t.Errorf("SecretAccessKey = %q", creds.SecretAccessKey)
	}
	if creds.SessionToken != "TOKEN_BLOB" {
		t.Errorf("SessionToken = %q", creds.SessionToken)
	}
	wantExp, _ := time.Parse(time.RFC3339, "2030-01-01T12:34:56Z")
	if !creds.Expiration.Equal(wantExp) {
		t.Errorf("Expiration = %v, want %v", creds.Expiration, wantExp)
	}
	if creds.AssumedRoleID != "AROAEXAMPLE:test-session" {
		t.Errorf("AssumedRoleID = %q", creds.AssumedRoleID)
	}
	if creds.AssumedRoleArn != "arn:aws:sts::123:assumed-role/myrole/test-session" {
		t.Errorf("AssumedRoleArn = %q", creds.AssumedRoleArn)
	}
}

func TestAssumeRoleWithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if r.Form.Get("DurationSeconds") != "7200" {
			t.Errorf("DurationSeconds = %q", r.Form.Get("DurationSeconds"))
		}
		if r.Form.Get("ExternalId") != "ext-1" {
			t.Errorf("ExternalId = %q", r.Form.Get("ExternalId"))
		}
		if r.Form.Get("Policy") != "{}" {
			t.Errorf("Policy = %q", r.Form.Get("Policy"))
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(assumeRoleSuccessXML))
	}))
	defer server.Close()

	client := NewSTSClient(server.URL, "us-east-1", false)
	_, err := client.AssumeRole(context.Background(), "ak", "sk", "arn:aws:iam::123:role/myrole", "test-session",
		AssumeRoleOptions{DurationSeconds: 7200, ExternalID: "ext-1", Policy: "{}"})
	if err != nil {
		t.Fatalf("AssumeRole: %v", err)
	}
}

func TestAssumeRoleAccessDenied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		_, _ = w.Write([]byte(xmlErrorResponse("AccessDenied", "not authorized")))
	}))
	defer server.Close()

	client := NewSTSClient(server.URL, "us-east-1", false)
	_, err := client.AssumeRole(context.Background(), "ak", "sk", "arn:aws:iam::123:role/myrole", "test", AssumeRoleOptions{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "AccessDenied") {
		t.Errorf("error = %q, want mention of AccessDenied", err.Error())
	}
}

func TestAssumeRoleBadExpiration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`<AssumeRoleResponse><AssumeRoleResult><Credentials><AccessKeyId>X</AccessKeyId><SecretAccessKey>Y</SecretAccessKey><SessionToken>Z</SessionToken><Expiration>NOT-A-TIME</Expiration></Credentials><AssumedRoleUser><AssumedRoleId>A</AssumedRoleId><Arn>B</Arn></AssumedRoleUser></AssumeRoleResult></AssumeRoleResponse>`))
	}))
	defer server.Close()

	client := NewSTSClient(server.URL, "us-east-1", false)
	_, err := client.AssumeRole(context.Background(), "ak", "sk", "arn", "name", AssumeRoleOptions{})
	if err == nil {
		t.Fatal("expected parse error for bad expiration")
	}
}

// ---------------------------------------------------------------------------
// GetCallerIdentity
// ---------------------------------------------------------------------------

func TestGetCallerIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if r.Form.Get("Action") != "GetCallerIdentity" {
			t.Errorf("Action = %q", r.Form.Get("Action"))
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`<GetCallerIdentityResponse><GetCallerIdentityResult><UserId>UID1</UserId><Account>123</Account><Arn>arn:aws:iam::123:user/alice</Arn></GetCallerIdentityResult></GetCallerIdentityResponse>`))
	}))
	defer server.Close()

	client := NewSTSClient(server.URL, "us-east-1", false)
	id, err := client.GetCallerIdentity(context.Background(), "ak", "sk", "")
	if err != nil {
		t.Fatalf("GetCallerIdentity: %v", err)
	}
	if id.UserID != "UID1" || id.Account != "123" || id.Arn != "arn:aws:iam::123:user/alice" {
		t.Errorf("unexpected identity: %+v", id)
	}
}

func TestGetCallerIdentityServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := NewSTSClient(server.URL, "us-east-1", false)
	_, err := client.GetCallerIdentity(context.Background(), "ak", "sk", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "status 500") {
		t.Errorf("error = %q, want status 500 mention", err.Error())
	}
}

func TestGetCallerIdentityWithSessionToken(t *testing.T) {
	var gotSecurityToken string
	var gotAuthHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSecurityToken = r.Header.Get("X-Amz-Security-Token")
		gotAuthHeader = r.Header.Get("Authorization")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`<GetCallerIdentityResponse><GetCallerIdentityResult><UserId>SESSION:alice</UserId><Account>123</Account><Arn>arn:aws:sts::123:assumed-role/writer/alice</Arn></GetCallerIdentityResult></GetCallerIdentityResponse>`))
	}))
	defer server.Close()

	client := NewSTSClient(server.URL, "us-east-1", false)
	id, err := client.GetCallerIdentity(context.Background(), "AKID", "secret", "TOKEN-abc")
	if err != nil {
		t.Fatalf("GetCallerIdentity: %v", err)
	}
	if id.Arn != "arn:aws:sts::123:assumed-role/writer/alice" {
		t.Errorf("Arn = %q", id.Arn)
	}
	if gotSecurityToken != "TOKEN-abc" {
		t.Errorf("X-Amz-Security-Token header = %q, want TOKEN-abc", gotSecurityToken)
	}
	if !strings.Contains(gotAuthHeader, "SignedHeaders=host;x-amz-date;x-amz-security-token") {
		t.Errorf("Authorization header missing x-amz-security-token in SignedHeaders: %q", gotAuthHeader)
	}
}
