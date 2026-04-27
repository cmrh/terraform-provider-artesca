package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetEndpoint(t *testing.T) {
	overlay := ConfigOverlay{
		Endpoints: []Endpoint{
			{Hostname: "s3.example.com", LocationName: "us-east-1", IsBuiltin: false},
			{Hostname: "built-in.example.com", LocationName: "default", IsBuiltin: true},
		},
	}
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(overlay)
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	ep, err := client.GetEndpoint(context.Background(), "s3.example.com")
	if err != nil {
		t.Fatalf("GetEndpoint returned error: %v", err)
	}
	if ep == nil {
		t.Fatal("GetEndpoint returned nil")
	}
	if ep.LocationName != "us-east-1" {
		t.Errorf("LocationName = %q, want us-east-1", ep.LocationName)
	}
	if ep.IsBuiltin {
		t.Error("expected IsBuiltin=false")
	}
}

func TestGetEndpointNotFound(t *testing.T) {
	overlay := ConfigOverlay{
		Endpoints: []Endpoint{
			{Hostname: "other.example.com"},
		},
	}
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(overlay)
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	ep, err := client.GetEndpoint(context.Background(), "missing.example.com")
	if err != nil {
		t.Fatalf("GetEndpoint returned error: %v", err)
	}
	if ep != nil {
		t.Errorf("expected nil for missing endpoint, got: %+v", ep)
	}
}

func TestCreateEndpoint(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		var ep Endpoint
		if err := json.NewDecoder(r.Body).Decode(&ep); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if ep.Hostname != "new.example.com" {
			t.Errorf("Hostname = %q, want new.example.com", ep.Hostname)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(ep)
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	created, err := client.CreateEndpoint(context.Background(), &Endpoint{
		Hostname:     "new.example.com",
		LocationName: "us-east-1",
	})
	if err != nil {
		t.Fatalf("CreateEndpoint returned error: %v", err)
	}
	if created.Hostname != "new.example.com" {
		t.Errorf("Hostname = %q, want new.example.com", created.Hostname)
	}
}

func TestCreateEndpointError(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`bad request`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	_, err := client.CreateEndpoint(context.Background(), &Endpoint{Hostname: "bad"})
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
}

func TestDeleteEndpoint(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/endpoint/s3.example.com") {
			t.Errorf("path = %q, want /endpoint/s3.example.com", r.URL.Path)
		}
		w.WriteHeader(204)
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	err := client.DeleteEndpoint(context.Background(), "s3.example.com")
	if err != nil {
		t.Fatalf("DeleteEndpoint returned error: %v", err)
	}
}

func TestDeleteEndpointNotFound(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`not found`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	err := client.DeleteEndpoint(context.Background(), "gone.example.com")
	if err != nil {
		t.Fatalf("expected 404 to be treated as success, got error: %v", err)
	}
}

func TestDeleteEndpointError(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`error`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	err := client.DeleteEndpoint(context.Background(), "s3.example.com")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}
