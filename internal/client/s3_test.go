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

func TestDeleteBucketNotFound(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`<Error><Code>NoSuchBucket</Code><Message>not found</Message></Error>`))
	}))
	defer apiServer.Close()

	client := NewS3Client(apiServer.URL, "us-east-1", false)
	err := client.DeleteBucket(context.Background(), "AKID", "secret", "my-bucket")
	if err != nil {
		t.Fatalf("expected 404 to be treated as success, got error: %v", err)
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

func TestPutBucketVersioningEnabled(t *testing.T) {
	var requestBody string
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/my-bucket" {
			t.Errorf("path = %q, want /my-bucket", r.URL.Path)
		}
		if r.URL.RawQuery != "versioning=" {
			t.Errorf("query = %q, want versioning=", r.URL.RawQuery)
		}
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		requestBody = string(buf[:n])
		w.WriteHeader(200)
	}))
	defer apiServer.Close()

	client := NewS3Client(apiServer.URL, "us-east-1", false)
	err := client.PutBucketVersioning(context.Background(), "AKID", "secret", "my-bucket", true)
	if err != nil {
		t.Fatalf("PutBucketVersioning returned error: %v", err)
	}
	if !strings.Contains(requestBody, "<Status>Enabled</Status>") {
		t.Errorf("body = %q, want Enabled status", requestBody)
	}
}

func TestPutBucketVersioningSuspended(t *testing.T) {
	var requestBody string
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		requestBody = string(buf[:n])
		w.WriteHeader(200)
	}))
	defer apiServer.Close()

	client := NewS3Client(apiServer.URL, "us-east-1", false)
	err := client.PutBucketVersioning(context.Background(), "AKID", "secret", "my-bucket", false)
	if err != nil {
		t.Fatalf("PutBucketVersioning returned error: %v", err)
	}
	if !strings.Contains(requestBody, "<Status>Suspended</Status>") {
		t.Errorf("body = %q, want Suspended status", requestBody)
	}
}

func TestPutBucketVersioningError(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`<Error><Code>InternalError</Code><Message>fail</Message></Error>`))
	}))
	defer apiServer.Close()

	client := NewS3Client(apiServer.URL, "us-east-1", false)
	err := client.PutBucketVersioning(context.Background(), "AKID", "secret", "my-bucket", true)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestGetBucketVersioningEnabled(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.RawQuery != "versioning=" {
			t.Errorf("query = %q, want versioning=", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`<VersioningConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Status>Enabled</Status></VersioningConfiguration>`))
	}))
	defer apiServer.Close()

	client := NewS3Client(apiServer.URL, "us-east-1", false)
	enabled, err := client.GetBucketVersioning(context.Background(), "AKID", "secret", "my-bucket")
	if err != nil {
		t.Fatalf("GetBucketVersioning returned error: %v", err)
	}
	if !enabled {
		t.Error("expected enabled=true")
	}
}

func TestGetBucketVersioningDisabled(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`<VersioningConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"/>`))
	}))
	defer apiServer.Close()

	client := NewS3Client(apiServer.URL, "us-east-1", false)
	enabled, err := client.GetBucketVersioning(context.Background(), "AKID", "secret", "my-bucket")
	if err != nil {
		t.Fatalf("GetBucketVersioning returned error: %v", err)
	}
	if enabled {
		t.Error("expected enabled=false for empty versioning config")
	}
}

func TestGetBucketVersioningSuspended(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`<VersioningConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Status>Suspended</Status></VersioningConfiguration>`))
	}))
	defer apiServer.Close()

	client := NewS3Client(apiServer.URL, "us-east-1", false)
	enabled, err := client.GetBucketVersioning(context.Background(), "AKID", "secret", "my-bucket")
	if err != nil {
		t.Fatalf("GetBucketVersioning returned error: %v", err)
	}
	if enabled {
		t.Error("expected enabled=false for Suspended status")
	}
}

func TestS3ClientTrailingSlash(t *testing.T) {
	client := NewS3Client("https://s3.example.com/", "us-east-1", false)
	if client.endpoint != "https://s3.example.com" {
		t.Errorf("endpoint = %q, want trailing slash stripped", client.endpoint)
	}
}

func TestDoSignedRequestRetriesTransientGatewayStatus(t *testing.T) {
	cases := []struct {
		name   string
		status int
	}{
		{"502", http.StatusBadGateway},
		{"503", http.StatusServiceUnavailable},
		{"504", http.StatusGatewayTimeout},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var attempts int32
			apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if atomic.AddInt32(&attempts, 1) == 1 {
					w.WriteHeader(tc.status)
					return
				}
				w.WriteHeader(200)
			}))
			defer apiServer.Close()

			client := NewS3Client(apiServer.URL, "us-east-1", false)
			client.transientBackoffOverride = time.Millisecond

			err := client.CreateBucket(context.Background(), "AKID", "secret", "my-bucket", "")
			if err != nil {
				t.Fatalf("CreateBucket returned error: %v", err)
			}
			if got := atomic.LoadInt32(&attempts); got != 2 {
				t.Errorf("attempts = %d, want 2", got)
			}
		})
	}
}

func TestDoSignedRequestNoRetryOn500(t *testing.T) {
	var attempts int32
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`<Error><Code>InternalError</Code><Message>fail</Message></Error>`))
	}))
	defer apiServer.Close()

	client := NewS3Client(apiServer.URL, "us-east-1", false)
	client.transientBackoffOverride = time.Millisecond

	err := client.CreateBucket(context.Background(), "AKID", "secret", "my-bucket", "")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Errorf("attempts = %d, want 1 (500 must not retry)", got)
	}
}

func TestDoSignedRequestGivesUpAfterMaxAttempts(t *testing.T) {
	var attempts int32
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer apiServer.Close()

	client := NewS3Client(apiServer.URL, "us-east-1", false)
	client.transientBackoffOverride = time.Millisecond

	err := client.CreateBucket(context.Background(), "AKID", "secret", "my-bucket", "")
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if !strings.Contains(err.Error(), "502") {
		t.Errorf("error = %q, want status 502 surfaced", err.Error())
	}
	if got := atomic.LoadInt32(&attempts); got != transientGatewayMaxAttempts {
		t.Errorf("attempts = %d, want %d", got, transientGatewayMaxAttempts)
	}
}
