package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// failFirst answers status to the first request and ok to the rest.
func failFirst(t *testing.T, status int, ok string) (*httptest.Server, *int32) {
	t.Helper()
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&attempts, 1) == 1 {
			w.WriteHeader(status)
			return
		}
		_, _ = w.Write([]byte(ok))
	}))
	t.Cleanup(srv.Close)
	return srv, &attempts
}

func TestIAMRetriesTransientGatewayStatus(t *testing.T) {
	for _, status := range []int{http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout} {
		srv, attempts := failFirst(t, status, `<ListAccessKeysResponse><ListAccessKeysResult><AccessKeyMetadata></AccessKeyMetadata></ListAccessKeysResult></ListAccessKeysResponse>`)
		c := NewIAMClient(srv.URL, "us-east-1", false)
		c.transientBackoffOverride = time.Millisecond
		if _, err := c.ListAccessKeys(context.Background(), Credentials{AccessKey: "AKID", SecretKey: "secret"}, "u"); err != nil {
			t.Fatalf("%d: ListAccessKeys returned error: %v", status, err)
		}
		if got := atomic.LoadInt32(attempts); got != 2 {
			t.Errorf("%d: attempts = %d, want 2", status, got)
		}
	}
}

func TestIAMNoRetryOn500(t *testing.T) {
	srv, attempts := failFirst(t, http.StatusInternalServerError, "")
	c := NewIAMClient(srv.URL, "us-east-1", false)
	c.transientBackoffOverride = time.Millisecond
	if _, err := c.ListAccessKeys(context.Background(), Credentials{AccessKey: "AKID", SecretKey: "secret"}, "u"); err == nil {
		t.Fatal("expected error for 500")
	}
	if got := atomic.LoadInt32(attempts); got != 1 {
		t.Errorf("attempts = %d, want 1", got)
	}
}

func TestSTSRetriesTransientGatewayStatus(t *testing.T) {
	srv, attempts := failFirst(t, http.StatusGatewayTimeout, `<GetCallerIdentityResponse><GetCallerIdentityResult><UserId>UID1</UserId><Account>123</Account><Arn>arn:aws:iam::123:user/alice</Arn></GetCallerIdentityResult></GetCallerIdentityResponse>`)
	c := NewSTSClient(srv.URL, "us-east-1", false)
	c.transientBackoffOverride = time.Millisecond
	if _, err := c.GetCallerIdentity(context.Background(), "AKID", "secret", ""); err != nil {
		t.Fatalf("GetCallerIdentity returned error: %v", err)
	}
	if got := atomic.LoadInt32(attempts); got != 2 {
		t.Errorf("attempts = %d, want 2", got)
	}
}

func TestSTSWebIdentityRetriesTransientGatewayStatus(t *testing.T) {
	srv, attempts := failFirst(t, http.StatusServiceUnavailable, `<AssumeRoleWithWebIdentityResponse><AssumeRoleWithWebIdentityResult><Credentials><AccessKeyId>AK</AccessKeyId><SecretAccessKey>SK</SecretAccessKey><SessionToken>T</SessionToken><Expiration>2030-01-01T00:00:00Z</Expiration></Credentials></AssumeRoleWithWebIdentityResult></AssumeRoleWithWebIdentityResponse>`)
	c := NewSTSClient(srv.URL, "us-east-1", false)
	c.transientBackoffOverride = time.Millisecond
	if _, err := c.AssumeRoleWithWebIdentity(context.Background(), "token", "arn:aws:iam::123:role/r", "s", 0); err != nil {
		t.Fatalf("AssumeRoleWithWebIdentity returned error: %v", err)
	}
	if got := atomic.LoadInt32(attempts); got != 2 {
		t.Errorf("attempts = %d, want 2", got)
	}
}

func TestOIDCRetriesTransientGatewayStatus(t *testing.T) {
	for _, status := range []int{http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout} {
		var attempts int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = r.ParseForm()
			if r.Form.Get("password") != "pass" {
				t.Errorf("%d: attempt %d sent password %q, want the form resent", status, atomic.LoadInt32(&attempts)+1, r.Form.Get("password"))
			}
			if atomic.AddInt32(&attempts, 1) == 1 {
				w.WriteHeader(status)
				return
			}
			_, _ = w.Write([]byte(`{"access_token":"tok","expires_in":300,"token_type":"bearer"}`))
		}))
		ts := NewOIDCTokenSource(srv.URL, "realm", "client", "openid", "user", "pass", false)
		ts.transientBackoffOverride = time.Millisecond
		token, err := ts.Token(context.Background())
		srv.Close()
		if err != nil {
			t.Fatalf("%d: Token returned error: %v", status, err)
		}
		if token != "tok" {
			t.Errorf("%d: token = %q, want tok", status, token)
		}
		if got := atomic.LoadInt32(&attempts); got != 2 {
			t.Errorf("%d: attempts = %d, want 2", status, got)
		}
	}
}

func TestOIDCNoRetryOnAuthError(t *testing.T) {
	srv, attempts := failFirst(t, http.StatusUnauthorized, "")
	ts := NewOIDCTokenSource(srv.URL, "realm", "client", "openid", "user", "badpass", false)
	ts.transientBackoffOverride = time.Millisecond
	if _, err := ts.Token(context.Background()); err == nil {
		t.Fatal("expected error for 401")
	}
	if got := atomic.LoadInt32(attempts); got != 1 {
		t.Errorf("attempts = %d, want 1", got)
	}
}

func TestSendWithTransientRetryHonoursContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGatewayTimeout)
	}))
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := sendWithTransientRetry(ctx, srv.Client(), time.Hour, func() (*http.Request, error) {
		return http.NewRequest(http.MethodGet, srv.URL, nil)
	})
	if err != context.Canceled {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestIAMListFollowsMarker(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("Marker") == "" {
			_, _ = w.Write([]byte(`<ListAttachedUserPoliciesResponse><ListAttachedUserPoliciesResult><AttachedPolicies><member><PolicyArn>arn:1</PolicyArn></member></AttachedPolicies><IsTruncated>true</IsTruncated><Marker>m1</Marker></ListAttachedUserPoliciesResult></ListAttachedUserPoliciesResponse>`))
			return
		}
		if r.Form.Get("Marker") != "m1" {
			t.Errorf("Marker = %q, want m1", r.Form.Get("Marker"))
		}
		_, _ = w.Write([]byte(`<ListAttachedUserPoliciesResponse><ListAttachedUserPoliciesResult><AttachedPolicies><member><PolicyArn>arn:2</PolicyArn></member></AttachedPolicies><IsTruncated>false</IsTruncated></ListAttachedUserPoliciesResult></ListAttachedUserPoliciesResponse>`))
	}))
	defer srv.Close()

	c := NewIAMClient(srv.URL, "us-east-1", false)
	arns, err := c.ListAttachedUserPolicies(context.Background(), Credentials{AccessKey: "AKID", SecretKey: "secret"}, "u")
	if err != nil {
		t.Fatalf("ListAttachedUserPolicies returned error: %v", err)
	}
	if len(arns) != 2 || arns[0] != "arn:1" || arns[1] != "arn:2" {
		t.Errorf("arns = %v, want [arn:1 arn:2]", arns)
	}
}

func TestIAMListRejectsRepeatedMarker(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<ListAccessKeysResponse><ListAccessKeysResult><AccessKeyMetadata></AccessKeyMetadata><IsTruncated>true</IsTruncated><Marker>same</Marker></ListAccessKeysResult></ListAccessKeysResponse>`))
	}))
	defer srv.Close()

	c := NewIAMClient(srv.URL, "us-east-1", false)
	if _, err := c.ListAccessKeys(context.Background(), Credentials{AccessKey: "AKID", SecretKey: "secret"}, "u"); err == nil {
		t.Fatal("expected error for a repeated Marker")
	}
}
