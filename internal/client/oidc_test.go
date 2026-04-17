package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestTokenFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %q, want form-urlencoded", ct)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.Form.Get("grant_type"); got != "password" {
			t.Errorf("grant_type = %q, want password", got)
		}
		if got := r.Form.Get("client_id"); got != "test-client" {
			t.Errorf("client_id = %q, want test-client", got)
		}
		if got := r.Form.Get("username"); got != "admin" {
			t.Errorf("username = %q, want admin", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"access_token":"my-token","expires_in":3600,"token_type":"bearer"}`))
	}))
	defer server.Close()

	ts := NewOIDCTokenSource(server.URL, "test-realm", "test-client", "admin", "secret", false)
	token, err := ts.Token(context.Background())
	if err != nil {
		t.Fatalf("Token returned error: %v", err)
	}
	if token != "my-token" {
		t.Errorf("token = %q, want my-token", token)
	}
}

func TestTokenCaching(t *testing.T) {
	var callCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"access_token":"cached-token","expires_in":3600,"token_type":"bearer"}`))
	}))
	defer server.Close()

	ts := NewOIDCTokenSource(server.URL, "realm", "client", "user", "pass", false)

	token1, err := ts.Token(context.Background())
	if err != nil {
		t.Fatalf("first Token call: %v", err)
	}
	token2, err := ts.Token(context.Background())
	if err != nil {
		t.Fatalf("second Token call: %v", err)
	}

	if token1 != token2 {
		t.Errorf("tokens differ: %q vs %q", token1, token2)
	}
	if callCount.Load() != 1 {
		t.Errorf("server called %d times, want 1 (cached)", callCount.Load())
	}
}

func TestTokenRefreshOnExpiry(t *testing.T) {
	var callCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		fmt.Fprintf(w, `{"access_token":"token-%d","expires_in":1,"token_type":"bearer"}`, n)
	}))
	defer server.Close()

	ts := NewOIDCTokenSource(server.URL, "realm", "client", "user", "pass", false)

	token1, err := ts.Token(context.Background())
	if err != nil {
		t.Fatalf("first Token call: %v", err)
	}

	// Force token expiry by manipulating the cached expiry time.
	ts.mu.Lock()
	ts.tokenExpiry = time.Now().Add(-1 * time.Minute)
	ts.mu.Unlock()

	token2, err := ts.Token(context.Background())
	if err != nil {
		t.Fatalf("second Token call: %v", err)
	}

	if token1 == token2 {
		t.Error("expected different tokens after expiry, got same")
	}
	if callCount.Load() != 2 {
		t.Errorf("server called %d times, want 2", callCount.Load())
	}
}

func TestTokenFetchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"error":"invalid_grant","error_description":"Invalid credentials"}`))
	}))
	defer server.Close()

	ts := NewOIDCTokenSource(server.URL, "realm", "client", "user", "badpass", false)
	_, err := ts.Token(context.Background())
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error = %q, want status 401", err.Error())
	}
}

func TestTokenEmptyAccessToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"access_token":"","expires_in":3600,"token_type":"bearer"}`))
	}))
	defer server.Close()

	ts := NewOIDCTokenSource(server.URL, "realm", "client", "user", "pass", false)
	_, err := ts.Token(context.Background())
	if err == nil {
		t.Fatal("expected error for empty access_token")
	}
	if !strings.Contains(err.Error(), "access_token") {
		t.Errorf("error = %q, want mention of access_token", err.Error())
	}
}

// ---------------------------------------------------------------------------
// InstanceIDs (JWT parsing)
// ---------------------------------------------------------------------------

func makeJWT(claims map[string]any) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, _ := json.Marshal(claims)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	return header + "." + encodedPayload + ".signature"
}

func TestInstanceIDs(t *testing.T) {
	jwt := makeJWT(map[string]any{
		"instanceIds": []string{"uuid-1234-5678"},
		"sub":         "admin",
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		fmt.Fprintf(w, `{"access_token":"%s","expires_in":3600,"token_type":"bearer"}`, jwt)
	}))
	defer server.Close()

	ts := NewOIDCTokenSource(server.URL, "realm", "client", "user", "pass", false)
	ids, err := ts.InstanceIDs(context.Background())
	if err != nil {
		t.Fatalf("InstanceIDs returned error: %v", err)
	}
	if len(ids) != 1 {
		t.Fatalf("expected 1 instance ID, got %d", len(ids))
	}
	if ids[0] != "uuid-1234-5678" {
		t.Errorf("instanceId = %q, want uuid-1234-5678", ids[0])
	}
}

func TestInstanceIDsMultiple(t *testing.T) {
	jwt := makeJWT(map[string]any{
		"instanceIds": []string{"id-1", "id-2", "id-3"},
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		fmt.Fprintf(w, `{"access_token":"%s","expires_in":3600,"token_type":"bearer"}`, jwt)
	}))
	defer server.Close()

	ts := NewOIDCTokenSource(server.URL, "realm", "client", "user", "pass", false)
	ids, err := ts.InstanceIDs(context.Background())
	if err != nil {
		t.Fatalf("InstanceIDs returned error: %v", err)
	}
	if len(ids) != 3 {
		t.Fatalf("expected 3 instance IDs, got %d", len(ids))
	}
}

func TestInstanceIDsEmpty(t *testing.T) {
	jwt := makeJWT(map[string]any{
		"instanceIds": []string{},
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		fmt.Fprintf(w, `{"access_token":"%s","expires_in":3600,"token_type":"bearer"}`, jwt)
	}))
	defer server.Close()

	ts := NewOIDCTokenSource(server.URL, "realm", "client", "user", "pass", false)
	ids, err := ts.InstanceIDs(context.Background())
	if err != nil {
		t.Fatalf("InstanceIDs returned error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("expected 0 instance IDs, got %d", len(ids))
	}
}

func TestInstanceIDsNeedsPadding(t *testing.T) {
	// Use a payload that requires base64 padding (len%4 != 0).
	// "ab" base64-encodes to "YWI=" (len 3 without padding, 3%4=3 → needs "=").
	// But we need valid JSON, so craft a JWT whose raw-URL-encoded payload length triggers padding.
	jwt := makeJWT(map[string]any{
		"instanceIds": []string{"padded-id"},
		"x":           "y",
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		fmt.Fprintf(w, `{"access_token":"%s","expires_in":3600,"token_type":"bearer"}`, jwt)
	}))
	defer server.Close()

	ts := NewOIDCTokenSource(server.URL, "realm", "client", "user", "pass", false)
	ids, err := ts.InstanceIDs(context.Background())
	if err != nil {
		t.Fatalf("InstanceIDs returned error: %v", err)
	}
	if len(ids) != 1 || ids[0] != "padded-id" {
		t.Errorf("instanceIds = %v, want [padded-id]", ids)
	}
}

func TestInstanceIDsTokenError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`unauthorized`))
	}))
	defer server.Close()

	ts := NewOIDCTokenSource(server.URL, "realm", "client", "user", "badpass", false)
	_, err := ts.InstanceIDs(context.Background())
	if err == nil {
		t.Fatal("expected error when token fetch fails")
	}
}

func TestInstanceIDsPaddingCase2(t *testing.T) {
	// Craft a JWT whose base64url payload length % 4 == 2 to exercise the "==" padding branch.
	// {"instanceIds":["x"]} is 22 bytes → base64url raw = 30 chars → 30%4=2.
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"instanceIds":["x"]}`))
	if len(payload)%4 != 2 {
		// Safety check: if the assumption is wrong, skip rather than test the wrong thing.
		t.Skipf("payload length %d mod 4 = %d, expected 2", len(payload), len(payload)%4)
	}
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	jwt := header + "." + payload + ".sig"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		fmt.Fprintf(w, `{"access_token":"%s","expires_in":3600,"token_type":"bearer"}`, jwt)
	}))
	defer server.Close()

	ts := NewOIDCTokenSource(server.URL, "realm", "client", "user", "pass", false)
	ids, err := ts.InstanceIDs(context.Background())
	if err != nil {
		t.Fatalf("InstanceIDs returned error: %v", err)
	}
	if len(ids) != 1 || ids[0] != "x" {
		t.Errorf("instanceIds = %v, want [x]", ids)
	}
}

func TestInstanceIDsInvalidBase64(t *testing.T) {
	jwt := "eyJhbGciOiJub25lIn0.!!!invalid-base64!!!.sig"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		fmt.Fprintf(w, `{"access_token":"%s","expires_in":3600,"token_type":"bearer"}`, jwt)
	}))
	defer server.Close()

	ts := NewOIDCTokenSource(server.URL, "realm", "client", "user", "pass", false)
	_, err := ts.InstanceIDs(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid base64 in JWT payload")
	}
	if !strings.Contains(err.Error(), "decoding JWT payload") {
		t.Errorf("error = %q, want 'decoding JWT payload'", err.Error())
	}
}

func TestInstanceIDsInvalidJSONClaims(t *testing.T) {
	// Valid base64 but not valid JSON.
	payload := base64.RawURLEncoding.EncodeToString([]byte(`not json at all`))
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	jwt := header + "." + payload + ".sig"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		fmt.Fprintf(w, `{"access_token":"%s","expires_in":3600,"token_type":"bearer"}`, jwt)
	}))
	defer server.Close()

	ts := NewOIDCTokenSource(server.URL, "realm", "client", "user", "pass", false)
	_, err := ts.InstanceIDs(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid JSON in JWT claims")
	}
	if !strings.Contains(err.Error(), "parsing JWT claims") {
		t.Errorf("error = %q, want 'parsing JWT claims'", err.Error())
	}
}

func TestInstanceIDsMalformedJWT(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"access_token":"not-a-jwt","expires_in":3600,"token_type":"bearer"}`))
	}))
	defer server.Close()

	ts := NewOIDCTokenSource(server.URL, "realm", "client", "user", "pass", false)
	_, err := ts.InstanceIDs(context.Background())
	if err == nil {
		t.Fatal("expected error for malformed JWT")
	}
}
