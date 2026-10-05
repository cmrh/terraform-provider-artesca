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

// ---------------------------------------------------------------------------
// URL escaping verification
// ---------------------------------------------------------------------------

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
