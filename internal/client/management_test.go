package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// doRequest
// ---------------------------------------------------------------------------

func TestDoRequestSetsHeaders(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Authentication-Token"); got != "mock-token" {
			t.Errorf("X-Authentication-Token = %q, want mock-token", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q, want application/json", got)
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	body, status, err := client.doRequest(context.Background(), http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("doRequest returned error: %v", err)
	}
	if status != 200 {
		t.Errorf("status = %d, want 200", status)
	}
	if !strings.Contains(string(body), "ok") {
		t.Errorf("body = %q, want 'ok'", string(body))
	}
}

func TestDoRequestMarshalsBody(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if payload["name"] != "test" {
			t.Errorf("name = %q, want test", payload["name"])
		}
		w.WriteHeader(201)
		w.Write([]byte(`{"created":true}`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	reqBody := map[string]string{"name": "test"}
	_, status, err := client.doRequest(context.Background(), http.MethodPost, "/create", reqBody)
	if err != nil {
		t.Fatalf("doRequest returned error: %v", err)
	}
	if status != 201 {
		t.Errorf("status = %d, want 201", status)
	}
}

func TestDoRequestNilBody(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > 0 {
			t.Error("expected no body for nil payload")
		}
		w.WriteHeader(200)
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	_, status, err := client.doRequest(context.Background(), http.MethodGet, "/empty", nil)
	if err != nil {
		t.Fatalf("doRequest returned error: %v", err)
	}
	if status != 200 {
		t.Errorf("status = %d, want 200", status)
	}
}

func TestDoRequestReturnsStatusCode(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"error":"not found"}`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	body, status, err := client.doRequest(context.Background(), http.MethodGet, "/missing", nil)
	if err != nil {
		t.Fatalf("doRequest returned error: %v", err)
	}
	if status != 404 {
		t.Errorf("status = %d, want 404", status)
	}
	if !strings.Contains(string(body), "not found") {
		t.Errorf("body = %q, want 'not found'", string(body))
	}
}

// ---------------------------------------------------------------------------
// GetOverlay
// ---------------------------------------------------------------------------

func TestGetOverlay(t *testing.T) {
	overlay := ConfigOverlay{
		InstanceID: "test-instance-id",
		Locations: map[string]Location{
			"us-east-1": {Name: "us-east-1", LocationType: "location-aws-s3-v1"},
		},
		Users: []User{
			{AccountName: "admin", UserName: "admin"},
		},
		Endpoints: []Endpoint{
			{Hostname: "s3.example.com", LocationName: "us-east-1"},
		},
		Version: 42,
	}

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/config/overlay/view/test-instance-id") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(overlay)
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	got, err := client.GetOverlay(context.Background())
	if err != nil {
		t.Fatalf("GetOverlay returned error: %v", err)
	}
	if got.InstanceID != "test-instance-id" {
		t.Errorf("InstanceID = %q, want test-instance-id", got.InstanceID)
	}
	if len(got.Locations) != 1 {
		t.Fatalf("expected 1 location, got %d", len(got.Locations))
	}
	if got.Locations["us-east-1"].LocationType != "location-aws-s3-v1" {
		t.Errorf("location type = %q, want location-aws-s3-v1", got.Locations["us-east-1"].LocationType)
	}
	if len(got.Users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(got.Users))
	}
	if got.Users[0].AccountName != "admin" {
		t.Errorf("user account name = %q, want admin", got.Users[0].AccountName)
	}
	if len(got.Endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(got.Endpoints))
	}
	if got.Version != 42 {
		t.Errorf("version = %d, want 42", got.Version)
	}
}

func TestGetOverlayError(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`internal server error`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	_, err := client.GetOverlay(context.Background())
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error = %q, want mention of 500", err.Error())
	}
}

func TestGetOverlayEmptyOverlay(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"instanceId":"test","locations":{},"users":[],"endpoints":[],"replicationStreams":[],"version":1}`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	got, err := client.GetOverlay(context.Background())
	if err != nil {
		t.Fatalf("GetOverlay returned error: %v", err)
	}
	if len(got.Locations) != 0 {
		t.Errorf("expected 0 locations, got %d", len(got.Locations))
	}
	if len(got.Users) != 0 {
		t.Errorf("expected 0 users, got %d", len(got.Users))
	}
}
