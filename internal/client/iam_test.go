package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// DeriveIAMEndpoint
// ---------------------------------------------------------------------------

func TestDeriveIAMEndpoint(t *testing.T) {
	got, err := DeriveIAMEndpoint("https://management.artesca.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://iam.artesca.example.com" {
		t.Errorf("got %q, want https://iam.artesca.example.com", got)
	}
}

func TestDeriveIAMEndpointWithPort(t *testing.T) {
	got, err := DeriveIAMEndpoint("https://management.artesca.example.com:8443")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://iam.artesca.example.com:8443" {
		t.Errorf("got %q, want https://iam.artesca.example.com:8443", got)
	}
}

func TestDeriveIAMEndpointHTTP(t *testing.T) {
	got, err := DeriveIAMEndpoint("http://management.artesca.local:8080")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "http://iam.artesca.local:8080" {
		t.Errorf("got %q, want http://iam.artesca.local:8080", got)
	}
}

func TestDeriveIAMEndpointStripsPath(t *testing.T) {
	got, err := DeriveIAMEndpoint("https://management.artesca.example.com/api/v1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://iam.artesca.example.com" {
		t.Errorf("got %q, want https://iam.artesca.example.com", got)
	}
}

func TestDeriveIAMEndpointNotManagement(t *testing.T) {
	_, err := DeriveIAMEndpoint("https://api.artesca.example.com")
	if err == nil {
		t.Fatal("expected error for non-management hostname")
	}
	if !strings.Contains(err.Error(), "does not start with 'management.'") {
		t.Errorf("error = %q, want mention of 'management.'", err.Error())
	}
}

func TestDeriveIAMEndpointInvalidURL(t *testing.T) {
	_, err := DeriveIAMEndpoint("://bad-url")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

// ---------------------------------------------------------------------------
// Crypto helpers
// ---------------------------------------------------------------------------

func TestSHA256Hex(t *testing.T) {
	got := sha256Hex([]byte(""))
	want := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got != want {
		t.Errorf("sha256Hex('') = %q, want %q", got, want)
	}

	got = sha256Hex([]byte("hello"))
	want = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if got != want {
		t.Errorf("sha256Hex('hello') = %q, want %q", got, want)
	}
}

func TestHmacSHA256(t *testing.T) {
	result := hmacSHA256([]byte("key"), []byte("data"))
	if len(result) != 32 {
		t.Errorf("expected 32 bytes, got %d", len(result))
	}
}

func TestGetSignatureKey(t *testing.T) {
	key := getSignatureKey("secret", "20150830", "us-east-1", "iam")
	if len(key) != 32 {
		t.Errorf("expected 32-byte signing key, got %d bytes", len(key))
	}
}

// ---------------------------------------------------------------------------
// IAM User CRUD
// ---------------------------------------------------------------------------

func TestCreateUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.Form.Get("Action"); got != "CreateUser" {
			t.Errorf("Action = %q, want CreateUser", got)
		}
		if got := r.Form.Get("UserName"); got != "testuser" {
			t.Errorf("UserName = %q, want testuser", got)
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`<CreateUserResponse><CreateUserResult><User><UserName>testuser</UserName><UserId>AIDTEST123</UserId><Arn>arn:aws:iam::123:user/testuser</Arn><Path>/</Path></User></CreateUserResult></CreateUserResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "us-east-1", false)
	user, err := client.CreateUser(context.Background(), "test-ak", "test-sk", "testuser")
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}
	if user.UserName != "testuser" {
		t.Errorf("UserName = %q, want testuser", user.UserName)
	}
	if user.UserId != "AIDTEST123" {
		t.Errorf("UserId = %q, want AIDTEST123", user.UserId)
	}
	if user.Arn != "arn:aws:iam::123:user/testuser" {
		t.Errorf("Arn = %q, want arn:aws:iam::123:user/testuser", user.Arn)
	}
	if user.Path != "/" {
		t.Errorf("Path = %q, want /", user.Path)
	}
}

func TestGetUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.Form.Get("Action"); got != "GetUser" {
			t.Errorf("Action = %q, want GetUser", got)
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`<GetUserResponse><GetUserResult><User><UserName>testuser</UserName><UserId>AIDTEST123</UserId><Arn>arn:aws:iam::123:user/testuser</Arn><Path>/</Path></User></GetUserResult></GetUserResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "us-east-1", false)
	user, err := client.GetUser(context.Background(), "test-ak", "test-sk", "testuser")
	if err != nil {
		t.Fatalf("GetUser returned error: %v", err)
	}
	if user == nil {
		t.Fatal("GetUser returned nil")
	}
	if user.UserName != "testuser" {
		t.Errorf("UserName = %q, want testuser", user.UserName)
	}
}

func TestGetUserNoSuchEntity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(xmlErrorResponse("NoSuchEntity", "The user with name testuser cannot be found.")))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "us-east-1", false)
	user, err := client.GetUser(context.Background(), "test-ak", "test-sk", "testuser")
	if err != nil {
		t.Fatalf("GetUser with NoSuchEntity should return nil error, got: %v", err)
	}
	if user != nil {
		t.Errorf("expected nil user, got: %+v", user)
	}
}

func TestDeleteUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.Form.Get("Action"); got != "DeleteUser" {
			t.Errorf("Action = %q, want DeleteUser", got)
		}
		if got := r.Form.Get("UserName"); got != "testuser" {
			t.Errorf("UserName = %q, want testuser", got)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "us-east-1", false)
	err := client.DeleteUser(context.Background(), "test-ak", "test-sk", "testuser")
	if err != nil {
		t.Fatalf("DeleteUser returned error: %v", err)
	}
}

func TestDeleteUserNoSuchEntity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(xmlErrorResponse("NoSuchEntity", "user not found")))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "us-east-1", false)
	err := client.DeleteUser(context.Background(), "test-ak", "test-sk", "gone-user")
	if err != nil {
		t.Fatalf("expected NoSuchEntity to be treated as success, got error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// IAM Policy CRUD
// ---------------------------------------------------------------------------

func TestPutUserPolicy(t *testing.T) {
	policyDoc := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.Form.Get("Action"); got != "PutUserPolicy" {
			t.Errorf("Action = %q, want PutUserPolicy", got)
		}
		if got := r.Form.Get("UserName"); got != "testuser" {
			t.Errorf("UserName = %q, want testuser", got)
		}
		if got := r.Form.Get("PolicyName"); got != "mypolicy" {
			t.Errorf("PolicyName = %q, want mypolicy", got)
		}
		if got := r.Form.Get("PolicyDocument"); got != policyDoc {
			t.Errorf("PolicyDocument = %q, want %q", got, policyDoc)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "us-east-1", false)
	err := client.PutUserPolicy(context.Background(), "test-ak", "test-sk", "testuser", "mypolicy", policyDoc)
	if err != nil {
		t.Fatalf("PutUserPolicy returned error: %v", err)
	}
}

func TestGetUserPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`<GetUserPolicyResponse><GetUserPolicyResult><UserName>testuser</UserName><PolicyName>mypolicy</PolicyName><PolicyDocument>%7B%22Version%22%3A%222012-10-17%22%7D</PolicyDocument></GetUserPolicyResult></GetUserPolicyResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "us-east-1", false)
	doc, err := client.GetUserPolicy(context.Background(), "test-ak", "test-sk", "testuser", "mypolicy")
	if err != nil {
		t.Fatalf("GetUserPolicy returned error: %v", err)
	}
	expected := `{"Version":"2012-10-17"}`
	if doc != expected {
		t.Errorf("got %q, want %q", doc, expected)
	}
}

func TestGetUserPolicyNoSuchEntity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(xmlErrorResponse("NoSuchEntity", "not found")))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "us-east-1", false)
	doc, err := client.GetUserPolicy(context.Background(), "test-ak", "test-sk", "testuser", "mypolicy")
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if doc != "" {
		t.Errorf("expected empty string, got: %q", doc)
	}
}

func TestDeleteUserPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.Form.Get("Action"); got != "DeleteUserPolicy" {
			t.Errorf("Action = %q, want DeleteUserPolicy", got)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "us-east-1", false)
	err := client.DeleteUserPolicy(context.Background(), "test-ak", "test-sk", "testuser", "mypolicy")
	if err != nil {
		t.Fatalf("DeleteUserPolicy returned error: %v", err)
	}
}

func TestDeleteUserPolicyNoSuchEntity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(xmlErrorResponse("NoSuchEntity", "policy not found")))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "us-east-1", false)
	err := client.DeleteUserPolicy(context.Background(), "test-ak", "test-sk", "testuser", "gone-policy")
	if err != nil {
		t.Fatalf("expected NoSuchEntity to be treated as success, got error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// IAM Access Key CRUD
// ---------------------------------------------------------------------------

func TestCreateAccessKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.Form.Get("Action"); got != "CreateAccessKey" {
			t.Errorf("Action = %q, want CreateAccessKey", got)
		}
		if got := r.Form.Get("UserName"); got != "testuser" {
			t.Errorf("UserName = %q, want testuser", got)
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`<CreateAccessKeyResponse><CreateAccessKeyResult><AccessKey><UserName>testuser</UserName><AccessKeyId>AKIATEST</AccessKeyId><SecretAccessKey>secret123</SecretAccessKey><Status>Active</Status></AccessKey></CreateAccessKeyResult></CreateAccessKeyResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "us-east-1", false)
	key, err := client.CreateAccessKey(context.Background(), "test-ak", "test-sk", "testuser")
	if err != nil {
		t.Fatalf("CreateAccessKey returned error: %v", err)
	}
	if key.AccessKeyId != "AKIATEST" {
		t.Errorf("AccessKeyId = %q, want AKIATEST", key.AccessKeyId)
	}
	if key.SecretAccessKey != "secret123" {
		t.Errorf("SecretAccessKey = %q, want secret123", key.SecretAccessKey)
	}
	if key.Status != "Active" {
		t.Errorf("Status = %q, want Active", key.Status)
	}
}

func TestListAccessKeys(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`<ListAccessKeysResponse><ListAccessKeysResult><AccessKeyMetadata><member><UserName>testuser</UserName><AccessKeyId>AKIATEST1</AccessKeyId><Status>Active</Status></member><member><UserName>testuser</UserName><AccessKeyId>AKIATEST2</AccessKeyId><Status>Inactive</Status></member></AccessKeyMetadata></ListAccessKeysResult></ListAccessKeysResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "us-east-1", false)
	keys, err := client.ListAccessKeys(context.Background(), "test-ak", "test-sk", "testuser")
	if err != nil {
		t.Fatalf("ListAccessKeys returned error: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
	if keys[0].AccessKeyId != "AKIATEST1" {
		t.Errorf("keys[0].AccessKeyId = %q, want AKIATEST1", keys[0].AccessKeyId)
	}
	if keys[1].Status != "Inactive" {
		t.Errorf("keys[1].Status = %q, want Inactive", keys[1].Status)
	}
}

func TestDeleteAccessKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.Form.Get("Action"); got != "DeleteAccessKey" {
			t.Errorf("Action = %q, want DeleteAccessKey", got)
		}
		if got := r.Form.Get("AccessKeyId"); got != "AKIATEST" {
			t.Errorf("AccessKeyId = %q, want AKIATEST", got)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "us-east-1", false)
	err := client.DeleteAccessKey(context.Background(), "test-ak", "test-sk", "testuser", "AKIATEST")
	if err != nil {
		t.Fatalf("DeleteAccessKey returned error: %v", err)
	}
}

func TestDeleteAccessKeyNoSuchEntity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(xmlErrorResponse("NoSuchEntity", "access key not found")))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "us-east-1", false)
	err := client.DeleteAccessKey(context.Background(), "test-ak", "test-sk", "testuser", "AKIAGONE")
	if err != nil {
		t.Fatalf("expected NoSuchEntity to be treated as success, got error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Per-function error paths
// ---------------------------------------------------------------------------

func TestGetUserServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		_, _ = w.Write([]byte(xmlErrorResponse("AccessDenied", "forbidden")))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "us-east-1", false)
	_, err := client.GetUser(context.Background(), "ak", "sk", "testuser")
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
	if !strings.Contains(err.Error(), "AccessDenied") {
		t.Errorf("error = %q, want AccessDenied", err.Error())
	}
}

func TestDeleteUserServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		_, _ = w.Write([]byte(xmlErrorResponse("AccessDenied", "forbidden")))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "us-east-1", false)
	err := client.DeleteUser(context.Background(), "ak", "sk", "testuser")
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
	if !strings.Contains(err.Error(), "delete user") {
		t.Errorf("error = %q, want 'delete user'", err.Error())
	}
}

func TestPutUserPolicyServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(xmlErrorResponse("InternalFailure", "fail")))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "us-east-1", false)
	err := client.PutUserPolicy(context.Background(), "ak", "sk", "user", "policy", "{}")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "put user policy") {
		t.Errorf("error = %q, want 'put user policy'", err.Error())
	}
}

func TestGetUserPolicyServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		_, _ = w.Write([]byte(xmlErrorResponse("AccessDenied", "forbidden")))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "us-east-1", false)
	_, err := client.GetUserPolicy(context.Background(), "ak", "sk", "user", "policy")
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
	if !strings.Contains(err.Error(), "get user policy") {
		t.Errorf("error = %q, want 'get user policy'", err.Error())
	}
}

func TestGetUserPolicyUnescapeFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`<GetUserPolicyResponse><GetUserPolicyResult><UserName>u</UserName><PolicyName>p</PolicyName><PolicyDocument>not%ZZvalid-url-encoding</PolicyDocument></GetUserPolicyResult></GetUserPolicyResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "us-east-1", false)
	doc, err := client.GetUserPolicy(context.Background(), "ak", "sk", "u", "p")
	if err != nil {
		t.Fatalf("GetUserPolicy returned error: %v", err)
	}
	if doc != "not%ZZvalid-url-encoding" {
		t.Errorf("expected raw policy doc fallback, got %q", doc)
	}
}

func TestDeleteUserPolicyServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(xmlErrorResponse("InternalFailure", "fail")))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "us-east-1", false)
	err := client.DeleteUserPolicy(context.Background(), "ak", "sk", "user", "policy")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "delete user policy") {
		t.Errorf("error = %q, want 'delete user policy'", err.Error())
	}
}

func TestCreateAccessKeyServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(xmlErrorResponse("InternalFailure", "fail")))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "us-east-1", false)
	_, err := client.CreateAccessKey(context.Background(), "ak", "sk", "user")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "create access key") {
		t.Errorf("error = %q, want 'create access key'", err.Error())
	}
}

func TestListAccessKeysServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(xmlErrorResponse("InternalFailure", "fail")))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "us-east-1", false)
	_, err := client.ListAccessKeys(context.Background(), "ak", "sk", "user")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "list access keys") {
		t.Errorf("error = %q, want 'list access keys'", err.Error())
	}
}

func TestDeleteAccessKeyServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		_, _ = w.Write([]byte(xmlErrorResponse("AccessDenied", "forbidden")))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "us-east-1", false)
	err := client.DeleteAccessKey(context.Background(), "ak", "sk", "user", "AKIATEST")
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
	if !strings.Contains(err.Error(), "delete access key") {
		t.Errorf("error = %q, want 'delete access key'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// Error handling (general)
// ---------------------------------------------------------------------------

func TestServerErrorXML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(xmlErrorResponse("InternalFailure", "Something went wrong")))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "us-east-1", false)
	_, err := client.CreateUser(context.Background(), "test-ak", "test-sk", "testuser")
	if err == nil {
		t.Fatal("expected error from 500 response")
	}
	if !strings.Contains(err.Error(), "InternalFailure") {
		t.Errorf("error = %q, want 'InternalFailure'", err.Error())
	}
}

func TestServerErrorNonXML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
		_, _ = w.Write([]byte("Service Unavailable"))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "us-east-1", false)
	_, err := client.CreateUser(context.Background(), "test-ak", "test-sk", "testuser")
	if err == nil {
		t.Fatal("expected error from 503 response")
	}
	if !strings.Contains(err.Error(), "503") {
		t.Errorf("error = %q, want status code 503", err.Error())
	}
}

func TestRequestIsPostWithFormContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", ct)
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`<CreateUserResponse><CreateUserResult><User><UserName>x</UserName><UserId>X</UserId><Arn>arn</Arn><Path>/</Path></User></CreateUserResult></CreateUserResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "us-east-1", false)
	_, err := client.CreateUser(context.Background(), "ak", "sk", "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVersionParameterIncluded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.Form.Get("Version"); got != "2010-05-08" {
			t.Errorf("Version = %q, want 2010-05-08", got)
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`<CreateUserResponse><CreateUserResult><User><UserName>x</UserName><UserId>X</UserId><Arn>arn</Arn><Path>/</Path></User></CreateUserResult></CreateUserResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "us-east-1", false)
	_, err := client.CreateUser(context.Background(), "ak", "sk", "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSigV4AuthorizationHeaderPresent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "AWS4-HMAC-SHA256") {
			t.Errorf("Authorization header = %q, want AWS4-HMAC-SHA256 prefix", auth)
		}
		if !strings.Contains(auth, "Credential=") {
			t.Error("Authorization header missing Credential=")
		}
		if !strings.Contains(auth, "Signature=") {
			t.Error("Authorization header missing Signature=")
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`<CreateUserResponse><CreateUserResult><User><UserName>x</UserName><UserId>X</UserId><Arn>arn</Arn><Path>/</Path></User></CreateUserResult></CreateUserResponse>`))
	}))
	defer server.Close()

	client := NewIAMClient(server.URL, "us-east-1", false)
	_, err := client.CreateUser(context.Background(), "AKIAEXAMPLE", "secret", "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
