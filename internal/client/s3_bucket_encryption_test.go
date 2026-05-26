package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPutBucketEncryption(t *testing.T) {
	var requestBody string
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/my-bucket" {
			t.Errorf("path = %q, want /my-bucket", r.URL.Path)
		}
		if !strings.Contains(r.URL.RawQuery, "encryption") {
			t.Errorf("query = %q, want encryption", r.URL.RawQuery)
		}
		if r.Header.Get("Authorization") == "" {
			t.Error("missing Authorization header")
		}
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		requestBody = string(buf[:n])
		w.WriteHeader(200)
	}))
	defer apiServer.Close()

	c := NewS3Client(apiServer.URL, "us-east-1", false)
	err := c.PutBucketEncryption(context.Background(), "AKID", "secret", "my-bucket", BucketEncryptionConfig{
		SSEAlgorithm:     "AES256",
		BucketKeyEnabled: true,
	})
	if err != nil {
		t.Fatalf("PutBucketEncryption returned error: %v", err)
	}
	if !strings.Contains(requestBody, "<SSEAlgorithm>AES256</SSEAlgorithm>") {
		t.Errorf("body missing SSEAlgorithm: %q", requestBody)
	}
	if !strings.Contains(requestBody, "<BucketKeyEnabled>true</BucketKeyEnabled>") {
		t.Errorf("body missing BucketKeyEnabled=true: %q", requestBody)
	}
}

func TestGetBucketEncryption(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if !strings.Contains(r.URL.RawQuery, "encryption") {
			t.Errorf("query = %q, want encryption", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<ServerSideEncryptionConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Rule>
    <ApplyServerSideEncryptionByDefault>
      <SSEAlgorithm>AES256</SSEAlgorithm>
    </ApplyServerSideEncryptionByDefault>
    <BucketKeyEnabled>false</BucketKeyEnabled>
  </Rule>
</ServerSideEncryptionConfiguration>`))
	}))
	defer apiServer.Close()

	c := NewS3Client(apiServer.URL, "us-east-1", false)
	cfg, err := c.GetBucketEncryption(context.Background(), "AKID", "secret", "my-bucket")
	if err != nil {
		t.Fatalf("GetBucketEncryption returned error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.SSEAlgorithm != "AES256" {
		t.Errorf("SSEAlgorithm = %q, want AES256", cfg.SSEAlgorithm)
	}
	if cfg.BucketKeyEnabled {
		t.Errorf("BucketKeyEnabled = true, want false")
	}
}

func TestGetBucketEncryptionNotConfigured(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<Error>
  <Code>ServerSideEncryptionConfigurationNotFoundError</Code>
  <Message>The server side encryption configuration was not found</Message>
</Error>`))
	}))
	defer apiServer.Close()

	c := NewS3Client(apiServer.URL, "us-east-1", false)
	cfg, err := c.GetBucketEncryption(context.Background(), "AKID", "secret", "my-bucket")
	if err != nil {
		t.Fatalf("GetBucketEncryption returned error: %v", err)
	}
	if cfg != nil {
		t.Errorf("expected nil config, got %+v", cfg)
	}
}

func TestDeleteBucketEncryption(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if !strings.Contains(r.URL.RawQuery, "encryption") {
			t.Errorf("query = %q, want encryption", r.URL.RawQuery)
		}
		w.WriteHeader(204)
	}))
	defer apiServer.Close()

	c := NewS3Client(apiServer.URL, "us-east-1", false)
	if err := c.DeleteBucketEncryption(context.Background(), "AKID", "secret", "my-bucket"); err != nil {
		t.Fatalf("DeleteBucketEncryption returned error: %v", err)
	}
}

func TestDeleteBucketEncryptionAlreadyAbsent(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<Error>
  <Code>ServerSideEncryptionConfigurationNotFoundError</Code>
  <Message>The server side encryption configuration was not found</Message>
</Error>`))
	}))
	defer apiServer.Close()

	c := NewS3Client(apiServer.URL, "us-east-1", false)
	if err := c.DeleteBucketEncryption(context.Background(), "AKID", "secret", "my-bucket"); err != nil {
		t.Fatalf("DeleteBucketEncryption should treat 404 as success, got: %v", err)
	}
}
