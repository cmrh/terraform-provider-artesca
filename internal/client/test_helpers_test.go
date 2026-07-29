package client

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// zeroOverlayRetryDelays disables the overlay-lookup backoff during unit tests
// so a "not found" run doesn't burn ~3.5s per test on retries that will never
// change outcome (mock responses are deterministic).
func zeroOverlayRetryDelays(t *testing.T) {
	t.Helper()
	orig := overlayLookupDelays
	overlayLookupDelays = []time.Duration{0}
	t.Cleanup(func() { overlayLookupDelays = orig })
}

func xmlErrorResponse(code, message string) string {
	return `<ErrorResponse><Error><Code>` + code + `</Code><Message>` + message + `</Message></Error></ErrorResponse>`
}

func newMockOIDCServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"access_token":"mock-token","expires_in":3600,"token_type":"bearer"}`))
	}))
}

func newTestManagementClient(t *testing.T, apiServer *httptest.Server) (*ManagementClient, func()) {
	t.Helper()
	zeroOverlayRetryDelays(t)
	oidcServer := newMockOIDCServer(t)
	ts := NewOIDCTokenSource(oidcServer.URL, "test-realm", "test-client", "openid", "user", "pass", false)

	client := NewManagementClient(apiServer.URL, "test-instance-id", ts, false)

	cleanup := func() {
		oidcServer.Close()
	}
	return client, cleanup
}
