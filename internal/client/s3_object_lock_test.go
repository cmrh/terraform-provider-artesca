package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPutObjectLockConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name      string
		retention *ObjectLockRetention
		want      string
	}{
		{"days", &ObjectLockRetention{Mode: "GOVERNANCE", Days: 30}, `<ObjectLockEnabled>Enabled</ObjectLockEnabled><Rule><DefaultRetention><Mode>GOVERNANCE</Mode><Days>30</Days></DefaultRetention></Rule>`},
		{"years", &ObjectLockRetention{Mode: "COMPLIANCE", Years: 1}, `<ObjectLockEnabled>Enabled</ObjectLockEnabled><Rule><DefaultRetention><Mode>COMPLIANCE</Mode><Years>1</Years></DefaultRetention></Rule>`},
		{"remove", nil, `<ObjectLockEnabled>Enabled</ObjectLockEnabled></ObjectLockConfiguration>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var body string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPut || r.URL.RawQuery != "object-lock=" {
					t.Errorf("request = %s %s", r.Method, r.URL)
				}
				b, _ := io.ReadAll(r.Body)
				body = string(b)
			}))
			defer srv.Close()
			c := NewS3Client(srv.URL, "us-east-1", false)
			if err := c.PutObjectLockConfiguration(context.Background(), Credentials{AccessKey: "AKID", SecretKey: "secret"}, "bkt", tc.retention); err != nil {
				t.Fatalf("PutObjectLockConfiguration returned error: %v", err)
			}
			if !strings.Contains(body, tc.want) {
				t.Errorf("body = %s, want it to contain %s", body, tc.want)
			}
		})
	}
}

func TestGetObjectLockConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name        string
		code        int
		body        string
		wantEnabled bool
		want        *ObjectLockRetention
	}{
		{"no object lock", http.StatusNotFound, `<Error><Code>ObjectLockConfigurationNotFoundError</Code><Message>The object lock configuration was not found</Message></Error>`, false, nil},
		{"no default retention", http.StatusOK, `<ObjectLockConfiguration><ObjectLockEnabled>Enabled</ObjectLockEnabled></ObjectLockConfiguration>`, true, nil},
		{"days", http.StatusOK, `<ObjectLockConfiguration><ObjectLockEnabled>Enabled</ObjectLockEnabled><Rule><DefaultRetention><Mode>GOVERNANCE</Mode><Days>1</Days></DefaultRetention></Rule></ObjectLockConfiguration>`, true, &ObjectLockRetention{Mode: "GOVERNANCE", Days: 1}},
		{"years", http.StatusOK, `<ObjectLockConfiguration><ObjectLockEnabled>Enabled</ObjectLockEnabled><Rule><DefaultRetention><Mode>COMPLIANCE</Mode><Years>2</Years></DefaultRetention></Rule></ObjectLockConfiguration>`, true, &ObjectLockRetention{Mode: "COMPLIANCE", Years: 2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.code)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			c := NewS3Client(srv.URL, "us-east-1", false)
			enabled, retention, err := c.GetObjectLockConfiguration(context.Background(), Credentials{AccessKey: "AKID", SecretKey: "secret"}, "bkt")
			if err != nil {
				t.Fatalf("GetObjectLockConfiguration returned error: %v", err)
			}
			if enabled != tc.wantEnabled {
				t.Errorf("enabled = %v, want %v", enabled, tc.wantEnabled)
			}
			if (retention == nil) != (tc.want == nil) || (retention != nil && *retention != *tc.want) {
				t.Errorf("retention = %+v, want %+v", retention, tc.want)
			}
		})
	}
}

func TestSignedExtraHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.Contains(auth, "SignedHeaders=host;x-amz-bypass-governance-retention;x-amz-content-sha256;x-amz-date;x-amz-security-token,") {
			t.Errorf("Authorization = %s, want the bypass header signed in sorted order", auth)
		}
		if r.Header.Get("X-Amz-Bypass-Governance-Retention") != "true" {
			t.Errorf("bypass header not sent")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	c := NewS3Client(srv.URL, "us-east-1", false)
	if err := c.DeleteObjectVersion(context.Background(), Credentials{AccessKey: "AKID", SecretKey: "secret", SessionToken: "tok"}, "bkt", "k", "v1", true); err != nil {
		t.Fatalf("DeleteObjectVersion returned error: %v", err)
	}
}
