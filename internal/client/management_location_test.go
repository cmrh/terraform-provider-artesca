package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetLocation(t *testing.T) {
	overlay := ConfigOverlay{
		Locations: map[string]Location{
			"us-east-1": {Name: "us-east-1", LocationType: "location-aws-s3-v1", ObjectID: "obj-1"},
			"eu-west-1": {Name: "eu-west-1", LocationType: "location-azure-v1", ObjectID: "obj-2"},
		},
	}
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_ = json.NewEncoder(w).Encode(overlay)
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	loc, err := client.GetLocation(context.Background(), "us-east-1")
	if err != nil {
		t.Fatalf("GetLocation returned error: %v", err)
	}
	if loc == nil {
		t.Fatal("GetLocation returned nil")
	}
	if loc.LocationType != "location-aws-s3-v1" {
		t.Errorf("LocationType = %q, want location-aws-s3-v1", loc.LocationType)
	}
	if loc.ObjectID != "obj-1" {
		t.Errorf("ObjectID = %q, want obj-1", loc.ObjectID)
	}
}

func TestGetLocationNotFound(t *testing.T) {
	overlay := ConfigOverlay{
		Locations: map[string]Location{
			"us-east-1": {Name: "us-east-1"},
		},
	}
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_ = json.NewEncoder(w).Encode(overlay)
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	loc, err := client.GetLocation(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetLocation returned error: %v", err)
	}
	if loc != nil {
		t.Errorf("expected nil for nonexistent location, got: %+v", loc)
	}
}

func TestCreateLocation(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// CreateLocation posts to /location, then LookupInOverlay GETs the
		// overlay to confirm the location is visible. Route by method.
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/config/overlay") {
			_ = json.NewEncoder(w).Encode(ConfigOverlay{
				Locations: map[string]Location{"new-loc": {Name: "new-loc"}},
			})
			return
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		var loc Location
		if err := json.NewDecoder(r.Body).Decode(&loc); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if loc.Name != "new-loc" {
			t.Errorf("Name = %q, want new-loc", loc.Name)
		}
		if loc.LocationType != "location-aws-s3-v1" {
			t.Errorf("LocationType = %q, want location-aws-s3-v1", loc.LocationType)
		}

		loc.ObjectID = "generated-id"
		w.WriteHeader(201)
		_ = json.NewEncoder(w).Encode(loc)
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	created, err := client.CreateLocation(context.Background(), &Location{
		Name:         "new-loc",
		LocationType: "location-aws-s3-v1",
	})
	if err != nil {
		t.Fatalf("CreateLocation returned error: %v", err)
	}
	if created.ObjectID != "generated-id" {
		t.Errorf("ObjectID = %q, want generated-id", created.ObjectID)
	}
}

func TestCreateLocationError(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"error":"invalid"}`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	_, err := client.CreateLocation(context.Background(), &Location{Name: "bad"})
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("error = %q, want 400", err.Error())
	}
}

func TestUpdateLocation(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/location/my-loc") {
			t.Errorf("path = %q, want /location/my-loc", r.URL.Path)
		}
		var loc Location
		_ = json.NewDecoder(r.Body).Decode(&loc)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_ = json.NewEncoder(w).Encode(loc)
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	updated, err := client.UpdateLocation(context.Background(), "my-loc", &Location{
		Name:         "my-loc",
		LocationType: "location-aws-s3-v1",
		IsTransient:  true,
	})
	if err != nil {
		t.Fatalf("UpdateLocation returned error: %v", err)
	}
	if !updated.IsTransient {
		t.Error("expected IsTransient=true")
	}
}

func TestUpdateLocationError(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`error`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	_, err := client.UpdateLocation(context.Background(), "my-loc", &Location{})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestDeleteLocation(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/location/my-loc") {
			t.Errorf("path = %q, want /location/my-loc", r.URL.Path)
		}
		w.WriteHeader(204)
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	err := client.DeleteLocation(context.Background(), "my-loc")
	if err != nil {
		t.Fatalf("DeleteLocation returned error: %v", err)
	}
}

func TestDeleteLocationNotFound(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`not found`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	err := client.DeleteLocation(context.Background(), "gone-loc")
	if err != nil {
		t.Fatalf("expected 404 to be treated as success, got error: %v", err)
	}
}

func TestDeleteLocationError(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(409)
		_, _ = w.Write([]byte(`{"error":"in use"}`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	err := client.DeleteLocation(context.Background(), "busy-loc")
	if err == nil {
		t.Fatal("expected error for 409 response")
	}
}
