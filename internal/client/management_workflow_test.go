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
// Replication workflows
// ---------------------------------------------------------------------------

func TestCreateBucketWorkflowReplication(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/workflow/replication") {
			t.Errorf("path = %q, want /workflow/replication", r.URL.Path)
		}
		var stream ReplicationStream
		_ = json.NewDecoder(r.Body).Decode(&stream)
		if stream.Name != "replicate" {
			t.Errorf("Name = %q, want replicate", stream.Name)
		}

		stream.StreamID = "wf-rep-1"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		_ = json.NewEncoder(w).Encode(stream)
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	created, err := client.CreateBucketWorkflowReplication(context.Background(), "i", "a", "b", &ReplicationStream{
		Name:    "replicate",
		Version: 1,
		Enabled: true,
		Source:  &ReplicationSource{BucketName: "src", Prefix: ""},
	})
	if err != nil {
		t.Fatalf("CreateBucketWorkflowReplication returned error: %v", err)
	}
	if created.StreamID != "wf-rep-1" {
		t.Errorf("StreamID = %q, want wf-rep-1", created.StreamID)
	}
}

func TestCreateBucketWorkflowReplicationError(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`bad`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	_, err := client.CreateBucketWorkflowReplication(context.Background(), "i", "a", "b", &ReplicationStream{})
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
}

func TestUpdateBucketWorkflowReplication(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/workflow/replication/wf-1") {
			t.Errorf("path = %q", r.URL.Path)
		}
		var stream ReplicationStream
		_ = json.NewDecoder(r.Body).Decode(&stream)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_ = json.NewEncoder(w).Encode(stream)
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	_, err := client.UpdateBucketWorkflowReplication(context.Background(), "i", "a", "b", "wf-1", &ReplicationStream{
		Enabled: false,
	})
	if err != nil {
		t.Fatalf("UpdateBucketWorkflowReplication returned error: %v", err)
	}
}

func TestUpdateBucketWorkflowReplicationError(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`error`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	_, err := client.UpdateBucketWorkflowReplication(context.Background(), "i", "a", "b", "wf-1", &ReplicationStream{})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "update bucket workflow replication failed") {
		t.Errorf("error = %q, want 'update bucket workflow replication failed'", err.Error())
	}
}

func TestDeleteBucketWorkflowReplication(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		w.WriteHeader(204)
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	err := client.DeleteBucketWorkflowReplication(context.Background(), "i", "a", "b", "wf-1")
	if err != nil {
		t.Fatalf("DeleteBucketWorkflowReplication returned error: %v", err)
	}
}

func TestDeleteBucketWorkflowReplicationNotFound(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`not found`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	err := client.DeleteBucketWorkflowReplication(context.Background(), "i", "a", "b", "wf-1")
	if err != nil {
		t.Fatalf("expected 404 to be treated as success, got error: %v", err)
	}
}

func TestDeleteBucketWorkflowReplicationError(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`error`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	err := client.DeleteBucketWorkflowReplication(context.Background(), "i", "a", "b", "wf-1")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "delete bucket workflow replication failed") {
		t.Errorf("error = %q, want 'delete bucket workflow replication failed'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// URL escaping verification
// ---------------------------------------------------------------------------

func TestWorkflowPathEscapesSpecialChars(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.RawPath
		if path == "" {
			path = r.URL.Path
		}
		if strings.Contains(path, "my bucket") {
			t.Error("bucket name not escaped in URL path")
		}
		if strings.Contains(path, "acct/id") {
			t.Error("account ID not escaped in URL path")
		}
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"streamId":"wf-1","name":"test","enabled":true}`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	_, err := client.CreateBucketWorkflowReplication(context.Background(), "inst/1", "acct/id", "my bucket", &ReplicationStream{
		Name:    "test",
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Workflow search
// ---------------------------------------------------------------------------

func TestSearchWorkflows(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/workflow/search") {
			t.Errorf("path = %q, want /workflow/search", r.URL.Path)
		}
		var req searchWorkflowsRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if len(req.BucketList) != 1 || req.BucketList[0] != "b1" {
			t.Errorf("bucketList = %v, want [b1]", req.BucketList)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`[
			{"replication": {"name": "r1", "version": 1, "enabled": true, "streamId": "s1"}},
			{"expiration": {"workflowId": "e1", "name": "e1", "bucketName": "b1", "type": "bucket-workflow-v1", "enabled": true}},
			{"transition": {"workflowId": "t1", "name": "t1", "bucketName": "b1", "type": "bucket-workflow-v1", "enabled": true, "locationName": "loc-cold", "applyToVersion": "current"}}
		]`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	results, err := client.SearchWorkflows(context.Background(), "i", "a", []string{"b1"})
	if err != nil {
		t.Fatalf("SearchWorkflows returned error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("len(results) = %d, want 3", len(results))
	}
	if results[0].Replication == nil || results[0].Replication.Name != "r1" {
		t.Errorf("results[0] replication mismatch: %+v", results[0])
	}
	if results[1].Expiration == nil || results[1].Expiration.WorkflowID != "e1" {
		t.Errorf("results[1] expiration mismatch: %+v", results[1])
	}
	if results[2].Transition == nil || results[2].Transition.LocationName != "loc-cold" {
		t.Errorf("results[2] transition mismatch: %+v", results[2])
	}
}

func TestSearchWorkflowsEmpty(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	results, err := client.SearchWorkflows(context.Background(), "i", "a", nil)
	if err != nil {
		t.Fatalf("SearchWorkflows returned error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("len(results) = %d, want 0", len(results))
	}
}

func TestSearchWorkflowsError(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`server error`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	_, err := client.SearchWorkflows(context.Background(), "i", "a", nil)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "search workflows failed") {
		t.Errorf("error = %q, want 'search workflows failed'", err.Error())
	}
}
