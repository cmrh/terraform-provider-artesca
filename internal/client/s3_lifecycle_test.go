package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPutBucketLifecycleZeroDayTransition(t *testing.T) {
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		body = string(b)
	}))
	defer server.Close()

	rules := []LifecycleRule{{ID: "r1", Status: "Enabled", TransitionDays: 0, TransitionLocation: "cold"}}
	if err := NewS3Client(server.URL, "us-east-1", false).PutBucketLifecycle(context.Background(), testCreds, "bkt", rules); err != nil {
		t.Fatalf("PutBucketLifecycle returned error: %v", err)
	}
	if want := "<Transition><Days>0</Days><StorageClass>cold</StorageClass></Transition>"; !strings.Contains(body, want) {
		t.Errorf("body = %s, want it to contain %s", body, want)
	}
}
