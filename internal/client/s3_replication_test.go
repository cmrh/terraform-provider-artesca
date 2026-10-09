package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var testCreds = Credentials{AccessKey: "ak", SecretKey: "sk", SessionToken: "tok"}

func TestGetBucketReplication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.RawQuery != "replication=" {
			t.Errorf("got %s ?%s, want GET ?replication=", r.Method, r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><ReplicationConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">` +
			`<Rule><ID>r1</ID><Prefix/><Status>Enabled</Status><Destination><Bucket>arn:aws:s3:::dst</Bucket></Destination></Rule>` +
			`<Rule><ID>r2</ID><Prefix>logs/</Prefix><Status>Disabled</Status><Destination><Bucket>arn:aws:s3:::dst2</Bucket></Destination></Rule>` +
			`<Role>arn:aws:iam::root:role/s3-replication-role</Role></ReplicationConfiguration>`))
	}))
	defer server.Close()

	cfg, err := NewS3Client(server.URL, "us-east-1", false).GetBucketReplication(context.Background(), testCreds, "src")
	if err != nil {
		t.Fatalf("GetBucketReplication returned error: %v", err)
	}
	if cfg.Role != DefaultReplicationRole || len(cfg.Rules) != 2 {
		t.Fatalf("got %+v", cfg)
	}
	want := []ReplicationRule{
		{ID: "r1", Status: "Enabled", Prefix: "", DestinationBucket: "dst"},
		{ID: "r2", Status: "Disabled", Prefix: "logs/", DestinationBucket: "dst2"},
	}
	for i := range want {
		got := cfg.Rules[i]
		if got.raw == "" {
			t.Errorf("rule %d: raw XML not kept", i)
		}
		got.raw = ""
		if got != want[i] {
			t.Errorf("rule %d = %+v, want %+v", i, got, want[i])
		}
	}
}

// Rules read from the server are written back unchanged, including elements
// the provider doesn't model; new rules are encoded from their fields.
func TestPutBucketReplicationKeepsUnmodelledElements(t *testing.T) {
	const uiRule = `<ID>ui</ID><Prefix>ui/</Prefix><Status>Enabled</Status><Destination><Bucket>arn:aws:s3:::dst</Bucket><StorageClass>cold-loc</StorageClass></Destination>`
	var put string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`<ReplicationConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Rule>` + uiRule +
				`</Rule><Role>arn:aws:iam::root:role/s3-replication-role</Role></ReplicationConfiguration>`))
			return
		}
		body, _ := io.ReadAll(r.Body)
		put = string(body)
	}))
	defer server.Close()

	c := NewS3Client(server.URL, "us-east-1", false)
	cfg, err := c.GetBucketReplication(context.Background(), testCreds, "src")
	if err != nil {
		t.Fatalf("GetBucketReplication returned error: %v", err)
	}
	cfg.Rules = append(cfg.Rules, ReplicationRule{ID: "tf", Status: "Enabled", Prefix: "tf/", DestinationBucket: "dst"})
	if err := c.PutBucketReplication(context.Background(), testCreds, "src", *cfg); err != nil {
		t.Fatalf("PutBucketReplication returned error: %v", err)
	}
	for _, want := range []string{
		"<Rule>" + uiRule + "</Rule>",
		`<Rule><ID>tf</ID><Prefix>tf/</Prefix><Status>Enabled</Status><Destination><Bucket>arn:aws:s3:::dst</Bucket></Destination></Rule>`,
	} {
		if !strings.Contains(put, want) {
			t.Errorf("body missing %s:\n%s", want, put)
		}
	}
}

func TestGetBucketReplicationNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`<Error><Code>ReplicationConfigurationNotFoundError</Code><Message>The replication configuration was not found</Message></Error>`))
	}))
	defer server.Close()

	cfg, err := NewS3Client(server.URL, "us-east-1", false).GetBucketReplication(context.Background(), testCreds, "src")
	if err != nil || cfg != nil {
		t.Fatalf("got (%+v, %v), want (nil, nil)", cfg, err)
	}
}

func TestGetBucketReplicationNoSuchBucket(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`<Error><Code>NoSuchBucket</Code><Message>The specified bucket does not exist</Message></Error>`))
	}))
	defer server.Close()

	_, err := NewS3Client(server.URL, "us-east-1", false).GetBucketReplication(context.Background(), testCreds, "src")
	if err == nil || !strings.Contains(err.Error(), "NoSuchBucket") {
		t.Fatalf("err = %v, want NoSuchBucket", err)
	}
}

func TestPutBucketReplication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.RawQuery != "replication=" {
			t.Errorf("got %s ?%s, want PUT ?replication=", r.Method, r.URL.RawQuery)
		}
		body, _ := io.ReadAll(r.Body)
		for _, want := range []string{
			`<Role>arn:aws:iam::root:role/s3-replication-role</Role>`,
			`<ID>r1</ID><Prefix>logs/</Prefix><Status>Enabled</Status><Destination><Bucket>arn:aws:s3:::dst</Bucket></Destination>`,
		} {
			if !strings.Contains(string(body), want) {
				t.Errorf("body missing %s:\n%s", want, body)
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := NewS3Client(server.URL, "us-east-1", false).PutBucketReplication(context.Background(), testCreds, "src", BucketReplication{
		Role:  DefaultReplicationRole,
		Rules: []ReplicationRule{{ID: "r1", Status: "Enabled", Prefix: "logs/", DestinationBucket: "dst"}},
	})
	if err != nil {
		t.Fatalf("PutBucketReplication returned error: %v", err)
	}
}

func TestDeleteBucketReplication(t *testing.T) {
	for _, status := range []int{http.StatusNoContent, http.StatusNotFound} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete || r.URL.RawQuery != "replication=" {
				t.Errorf("got %s ?%s, want DELETE ?replication=", r.Method, r.URL.RawQuery)
			}
			w.WriteHeader(status)
		}))
		err := NewS3Client(server.URL, "us-east-1", false).DeleteBucketReplication(context.Background(), testCreds, "src")
		server.Close()
		if err != nil {
			t.Errorf("status %d: DeleteBucketReplication returned error: %v", status, err)
		}
	}
}
