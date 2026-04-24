package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDeriveS3Endpoint(t *testing.T) {
	got, err := DeriveS3Endpoint("https://management.artesca.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://s3.artesca.example.com" {
		t.Errorf("got %q, want https://s3.artesca.example.com", got)
	}
}

func TestDeriveS3EndpointWithPort(t *testing.T) {
	got, err := DeriveS3Endpoint("https://management.artesca.example.com:8443")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://s3.artesca.example.com:8443" {
		t.Errorf("got %q, want https://s3.artesca.example.com:8443", got)
	}
}

func TestDeriveS3EndpointHTTP(t *testing.T) {
	got, err := DeriveS3Endpoint("http://management.artesca.local:8080")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "http://s3.artesca.local:8080" {
		t.Errorf("got %q, want http://s3.artesca.local:8080", got)
	}
}

func TestDeriveS3EndpointStripsPath(t *testing.T) {
	got, err := DeriveS3Endpoint("https://management.artesca.example.com/api/v1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://s3.artesca.example.com" {
		t.Errorf("got %q, want https://s3.artesca.example.com", got)
	}
}

func TestDeriveS3EndpointNonManagement(t *testing.T) {
	_, err := DeriveS3Endpoint("https://api.artesca.example.com")
	if err == nil {
		t.Fatal("expected error for non-management hostname")
	}
	if !strings.Contains(err.Error(), "does not start with 'management.'") {
		t.Errorf("error = %q, want 'does not start with management.'", err.Error())
	}
}

func TestDeriveS3EndpointInvalidURL(t *testing.T) {
	_, err := DeriveS3Endpoint("://bad")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestCreateBucket(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/my-bucket" {
			t.Errorf("path = %q, want /my-bucket", r.URL.Path)
		}
		if r.Header.Get("Authorization") == "" {
			t.Error("missing Authorization header")
		}
		if r.Header.Get("X-Amz-Content-Sha256") == "" {
			t.Error("missing X-Amz-Content-Sha256 header")
		}
		w.WriteHeader(200)
	}))
	defer apiServer.Close()

	client := NewS3Client(apiServer.URL, "us-east-1", false)
	err := client.CreateBucket(context.Background(), "AKID", "secret", "my-bucket", "us-east-1")
	if err != nil {
		t.Fatalf("CreateBucket returned error: %v", err)
	}
}

func TestCreateBucketWithLocationConstraint(t *testing.T) {
	var requestBody string
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		requestBody = string(buf[:n])
		w.WriteHeader(200)
	}))
	defer apiServer.Close()

	client := NewS3Client(apiServer.URL, "us-east-1", false)
	err := client.CreateBucket(context.Background(), "AKID", "secret", "my-bucket", "my-location")
	if err != nil {
		t.Fatalf("CreateBucket returned error: %v", err)
	}
	if !strings.Contains(requestBody, "<LocationConstraint>my-location</LocationConstraint>") {
		t.Errorf("body = %q, want LocationConstraint", requestBody)
	}
}

func TestCreateBucketError(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(409)
		_, _ = w.Write([]byte(`<Error><Code>BucketAlreadyExists</Code><Message>bucket exists</Message></Error>`))
	}))
	defer apiServer.Close()

	client := NewS3Client(apiServer.URL, "us-east-1", false)
	err := client.CreateBucket(context.Background(), "AKID", "secret", "my-bucket", "")
	if err == nil {
		t.Fatal("expected error for 409 response")
	}
	if !strings.Contains(err.Error(), "BucketAlreadyExists") {
		t.Errorf("error = %q, want BucketAlreadyExists", err.Error())
	}
}

func TestHeadBucketExists(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Errorf("method = %s, want HEAD", r.Method)
		}
		if r.URL.Path != "/my-bucket" {
			t.Errorf("path = %q, want /my-bucket", r.URL.Path)
		}
		w.WriteHeader(200)
	}))
	defer apiServer.Close()

	client := NewS3Client(apiServer.URL, "us-east-1", false)
	exists, err := client.HeadBucket(context.Background(), "AKID", "secret", "my-bucket")
	if err != nil {
		t.Fatalf("HeadBucket returned error: %v", err)
	}
	if !exists {
		t.Error("expected exists=true")
	}
}

func TestHeadBucketNotFound(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer apiServer.Close()

	client := NewS3Client(apiServer.URL, "us-east-1", false)
	exists, err := client.HeadBucket(context.Background(), "AKID", "secret", "my-bucket")
	if err != nil {
		t.Fatalf("HeadBucket returned error: %v", err)
	}
	if exists {
		t.Error("expected exists=false for 404")
	}
}

func TestHeadBucketServerError(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`<Error><Code>InternalError</Code><Message>fail</Message></Error>`))
	}))
	defer apiServer.Close()

	client := NewS3Client(apiServer.URL, "us-east-1", false)
	_, err := client.HeadBucket(context.Background(), "AKID", "secret", "my-bucket")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestDeleteBucket(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/my-bucket" {
			t.Errorf("path = %q, want /my-bucket", r.URL.Path)
		}
		w.WriteHeader(204)
	}))
	defer apiServer.Close()

	client := NewS3Client(apiServer.URL, "us-east-1", false)
	err := client.DeleteBucket(context.Background(), "AKID", "secret", "my-bucket")
	if err != nil {
		t.Fatalf("DeleteBucket returned error: %v", err)
	}
}

func TestDeleteBucketNoSuchBucket(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`<Error><Code>NoSuchBucket</Code><Message>not found</Message></Error>`))
	}))
	defer apiServer.Close()

	client := NewS3Client(apiServer.URL, "us-east-1", false)
	err := client.DeleteBucket(context.Background(), "AKID", "secret", "my-bucket")
	if err != nil {
		t.Fatalf("DeleteBucket should not error for NoSuchBucket, got: %v", err)
	}
}

func TestDeleteBucketError(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(409)
		_, _ = w.Write([]byte(`<Error><Code>BucketNotEmpty</Code><Message>not empty</Message></Error>`))
	}))
	defer apiServer.Close()

	client := NewS3Client(apiServer.URL, "us-east-1", false)
	err := client.DeleteBucket(context.Background(), "AKID", "secret", "my-bucket")
	if err == nil {
		t.Fatal("expected error for BucketNotEmpty")
	}
	if !strings.Contains(err.Error(), "BucketNotEmpty") {
		t.Errorf("error = %q, want BucketNotEmpty", err.Error())
	}
}

func TestS3ClientTrailingSlash(t *testing.T) {
	client := NewS3Client("https://s3.example.com/", "us-east-1", false)
	if client.endpoint != "https://s3.example.com" {
		t.Errorf("endpoint = %q, want trailing slash stripped", client.endpoint)
	}
}
