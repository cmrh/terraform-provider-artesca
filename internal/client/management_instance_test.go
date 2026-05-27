package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetInstance(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/instance/eb766fb5-9cee-481c-919d-d95321de9835") {
			t.Errorf("path = %q, want suffix /instance/eb766fb5-...", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"instanceId": "eb766fb5-9cee-481c-919d-d95321de9835",
			"createdAt": "2026-01-14T16:25:45Z",
			"state": "confirmed",
			"publicKey": "-----BEGIN PUBLIC KEY-----\nMII...\n-----END PUBLIC KEY-----\n"
		}`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	inst, err := client.GetInstance(context.Background(), "eb766fb5-9cee-481c-919d-d95321de9835")
	if err != nil {
		t.Fatalf("GetInstance returned error: %v", err)
	}
	if inst.InstanceID != "eb766fb5-9cee-481c-919d-d95321de9835" {
		t.Errorf("InstanceID = %q", inst.InstanceID)
	}
	if inst.State != "confirmed" {
		t.Errorf("State = %q, want confirmed", inst.State)
	}
	if inst.CreatedAt != "2026-01-14T16:25:45Z" {
		t.Errorf("CreatedAt = %q", inst.CreatedAt)
	}
	if !strings.Contains(inst.PublicKey, "BEGIN PUBLIC KEY") {
		t.Errorf("PublicKey missing PEM header: %q", inst.PublicKey)
	}
}

func TestGetInstanceNotFound(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"code":"NotFound","message":"instance not found"}`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	_, err := client.GetInstance(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !strings.Contains(err.Error(), "get instance failed") {
		t.Errorf("error = %q, want 'get instance failed'", err.Error())
	}
}

func TestGetInstanceStatus(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/instance/i-abc/status") {
			t.Errorf("path = %q, want suffix /instance/i-abc/status", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"metrics": {"cpu": {"idle": 1}},
			"state": {
				"ipAddress": "10.0.0.1",
				"lastSeen": "2026-05-27T13:19:37.921Z",
				"runningConfigurationVersion": 1356,
				"serverVersion": "ref: refs/heads/development/9.1\n",
				"capabilities": {"a": 1},
				"latestConfigurationOverlay": {"b": 2}
			}
		}`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	st, err := client.GetInstanceStatus(context.Background(), "i-abc")
	if err != nil {
		t.Fatalf("GetInstanceStatus returned error: %v", err)
	}
	if st.IPAddress != "10.0.0.1" {
		t.Errorf("IPAddress = %q", st.IPAddress)
	}
	if st.RunningConfigurationVersion != 1356 {
		t.Errorf("RunningConfigurationVersion = %d", st.RunningConfigurationVersion)
	}
	if !strings.Contains(st.ServerVersion, "refs/heads/development/9.1") {
		t.Errorf("ServerVersion = %q", st.ServerVersion)
	}
	if st.LastSeen != "2026-05-27T13:19:37.921Z" {
		t.Errorf("LastSeen = %q", st.LastSeen)
	}
}

func TestGetInstanceStatusError(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`internal error`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	_, err := client.GetInstanceStatus(context.Background(), "i-abc")
	if err == nil {
		t.Fatal("expected error for 500")
	}
	if !strings.Contains(err.Error(), "get instance status failed") {
		t.Errorf("error = %q, want 'get instance status failed'", err.Error())
	}
}
