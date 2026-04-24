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
// Expiration workflows
// ---------------------------------------------------------------------------

func TestCreateBucketWorkflowExpiration(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/instance/inst-1/account/acct-1/bucket/my-bucket/workflow/expiration") {
			t.Errorf("path = %q", r.URL.Path)
		}
		var wf BucketWorkflowExpiration
		json.NewDecoder(r.Body).Decode(&wf)
		if !wf.Enabled {
			t.Error("expected Enabled=true")
		}

		wf.WorkflowID = "wf-exp-1"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(wf)
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	days := int64(30)
	created, err := client.CreateBucketWorkflowExpiration(context.Background(), "inst-1", "acct-1", "my-bucket", &BucketWorkflowExpiration{
		Name:                           "expire-old",
		Enabled:                        true,
		BucketName:                     "my-bucket",
		Type:                           "bucket-workflow-expiration-v1",
		CurrentVersionTriggerDelayDays: &days,
	})
	if err != nil {
		t.Fatalf("CreateBucketWorkflowExpiration returned error: %v", err)
	}
	if created.WorkflowID != "wf-exp-1" {
		t.Errorf("WorkflowID = %q, want wf-exp-1", created.WorkflowID)
	}
}

func TestCreateBucketWorkflowExpirationError(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`bad request`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	_, err := client.CreateBucketWorkflowExpiration(context.Background(), "i", "a", "b", &BucketWorkflowExpiration{})
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
}

func TestUpdateBucketWorkflowExpiration(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/workflow/expiration/wf-1") {
			t.Errorf("path = %q, want /workflow/expiration/wf-1", r.URL.Path)
		}
		var wf BucketWorkflowExpiration
		json.NewDecoder(r.Body).Decode(&wf)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(wf)
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	updated, err := client.UpdateBucketWorkflowExpiration(context.Background(), "i", "a", "b", "wf-1", &BucketWorkflowExpiration{
		Enabled: false,
	})
	if err != nil {
		t.Fatalf("UpdateBucketWorkflowExpiration returned error: %v", err)
	}
	if updated.Enabled {
		t.Error("expected Enabled=false")
	}
}

func TestUpdateBucketWorkflowExpirationError(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`error`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	_, err := client.UpdateBucketWorkflowExpiration(context.Background(), "i", "a", "b", "wf-1", &BucketWorkflowExpiration{})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestDeleteBucketWorkflowExpiration(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/workflow/expiration/wf-1") {
			t.Errorf("path = %q, want /workflow/expiration/wf-1", r.URL.Path)
		}
		w.WriteHeader(204)
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	err := client.DeleteBucketWorkflowExpiration(context.Background(), "i", "a", "b", "wf-1")
	if err != nil {
		t.Fatalf("DeleteBucketWorkflowExpiration returned error: %v", err)
	}
}

func TestDeleteBucketWorkflowExpirationError(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`error`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	err := client.DeleteBucketWorkflowExpiration(context.Background(), "i", "a", "b", "wf-1")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

// ---------------------------------------------------------------------------
// Transition workflows
// ---------------------------------------------------------------------------

func TestCreateBucketWorkflowTransition(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/workflow/transition") {
			t.Errorf("path = %q, want /workflow/transition", r.URL.Path)
		}
		var wf BucketWorkflowTransition
		json.NewDecoder(r.Body).Decode(&wf)
		if wf.LocationName != "cold-storage" {
			t.Errorf("LocationName = %q, want cold-storage", wf.LocationName)
		}

		wf.WorkflowID = "wf-trans-1"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(wf)
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	created, err := client.CreateBucketWorkflowTransition(context.Background(), "i", "a", "b", &BucketWorkflowTransition{
		Enabled:        true,
		LocationName:   "cold-storage",
		ApplyToVersion: "current",
	})
	if err != nil {
		t.Fatalf("CreateBucketWorkflowTransition returned error: %v", err)
	}
	if created.WorkflowID != "wf-trans-1" {
		t.Errorf("WorkflowID = %q, want wf-trans-1", created.WorkflowID)
	}
}

func TestCreateBucketWorkflowTransitionError(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`bad`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	_, err := client.CreateBucketWorkflowTransition(context.Background(), "i", "a", "b", &BucketWorkflowTransition{})
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
}

func TestUpdateBucketWorkflowTransition(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/workflow/transition/wf-1") {
			t.Errorf("path = %q", r.URL.Path)
		}
		var wf BucketWorkflowTransition
		json.NewDecoder(r.Body).Decode(&wf)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(wf)
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	_, err := client.UpdateBucketWorkflowTransition(context.Background(), "i", "a", "b", "wf-1", &BucketWorkflowTransition{
		Enabled: false,
	})
	if err != nil {
		t.Fatalf("UpdateBucketWorkflowTransition returned error: %v", err)
	}
}

func TestUpdateBucketWorkflowTransitionError(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`error`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	_, err := client.UpdateBucketWorkflowTransition(context.Background(), "i", "a", "b", "wf-1", &BucketWorkflowTransition{})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "update transition workflow failed") {
		t.Errorf("error = %q, want 'update transition workflow failed'", err.Error())
	}
}

func TestDeleteBucketWorkflowTransition(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		w.WriteHeader(204)
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	err := client.DeleteBucketWorkflowTransition(context.Background(), "i", "a", "b", "wf-1")
	if err != nil {
		t.Fatalf("DeleteBucketWorkflowTransition returned error: %v", err)
	}
}

func TestDeleteBucketWorkflowTransitionError(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`error`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	err := client.DeleteBucketWorkflowTransition(context.Background(), "i", "a", "b", "wf-1")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "delete transition workflow failed") {
		t.Errorf("error = %q, want 'delete transition workflow failed'", err.Error())
	}
}

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
		json.NewDecoder(r.Body).Decode(&stream)
		if stream.Name != "replicate" {
			t.Errorf("Name = %q, want replicate", stream.Name)
		}

		stream.StreamID = "wf-rep-1"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(stream)
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
		json.NewDecoder(r.Body).Decode(&stream)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(stream)
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
		_, _ = w.Write([]byte(`{"workflowId":"wf-1","enabled":true,"bucketName":"my bucket","type":"bucket-workflow-expiration-v1"}`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	_, err := client.CreateBucketWorkflowExpiration(context.Background(), "inst/1", "acct/id", "my bucket", &BucketWorkflowExpiration{
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
