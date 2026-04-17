package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetReplicationStream(t *testing.T) {
	overlay := ConfigOverlay{
		ReplicationStreams: []ReplicationStream{
			{StreamID: "rs-1", Name: "primary-backup", Enabled: true},
			{StreamID: "rs-2", Name: "secondary", Enabled: false},
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

	rs, err := client.GetReplicationStream(context.Background(), "rs-1")
	if err != nil {
		t.Fatalf("GetReplicationStream returned error: %v", err)
	}
	if rs == nil {
		t.Fatal("GetReplicationStream returned nil")
	}
	if rs.Name != "primary-backup" {
		t.Errorf("Name = %q, want primary-backup", rs.Name)
	}
	if !rs.Enabled {
		t.Error("expected Enabled=true")
	}
}

func TestGetReplicationStreamNotFound(t *testing.T) {
	overlay := ConfigOverlay{
		ReplicationStreams: []ReplicationStream{
			{StreamID: "rs-1", Name: "other"},
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

	rs, err := client.GetReplicationStream(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetReplicationStream returned error: %v", err)
	}
	if rs != nil {
		t.Errorf("expected nil, got: %+v", rs)
	}
}

func TestCreateReplicationStream(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		var stream ReplicationStream
		json.NewDecoder(r.Body).Decode(&stream)
		if stream.Name != "new-stream" {
			t.Errorf("Name = %q, want new-stream", stream.Name)
		}

		stream.StreamID = "rs-new"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(stream)
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	created, err := client.CreateReplicationStream(context.Background(), &ReplicationStream{
		Name:    "new-stream",
		Version: 1,
		Enabled: true,
		Source:  &ReplicationSource{BucketName: "src", Prefix: ""},
	})
	if err != nil {
		t.Fatalf("CreateReplicationStream returned error: %v", err)
	}
	if created.StreamID != "rs-new" {
		t.Errorf("StreamID = %q, want rs-new", created.StreamID)
	}
}

func TestCreateReplicationStreamError(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte(`bad request`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	_, err := client.CreateReplicationStream(context.Background(), &ReplicationStream{Name: "bad"})
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
}

func TestUpdateReplicationStream(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/replication/rs-1") {
			t.Errorf("path = %q, want /replication/rs-1", r.URL.Path)
		}
		var stream ReplicationStream
		json.NewDecoder(r.Body).Decode(&stream)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(stream)
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	updated, err := client.UpdateReplicationStream(context.Background(), "rs-1", &ReplicationStream{
		Name:    "updated",
		Enabled: false,
	})
	if err != nil {
		t.Fatalf("UpdateReplicationStream returned error: %v", err)
	}
	if updated.Name != "updated" {
		t.Errorf("Name = %q, want updated", updated.Name)
	}
}

func TestUpdateReplicationStreamError(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`error`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	_, err := client.UpdateReplicationStream(context.Background(), "rs-1", &ReplicationStream{})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestDeleteReplicationStream(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/replication/rs-1") {
			t.Errorf("path = %q, want /replication/rs-1", r.URL.Path)
		}
		w.WriteHeader(204)
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	err := client.DeleteReplicationStream(context.Background(), "rs-1")
	if err != nil {
		t.Fatalf("DeleteReplicationStream returned error: %v", err)
	}
}

func TestDeleteReplicationStreamError(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(409)
		w.Write([]byte(`in use`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	err := client.DeleteReplicationStream(context.Background(), "rs-1")
	if err == nil {
		t.Fatal("expected error for 409 response")
	}
}
