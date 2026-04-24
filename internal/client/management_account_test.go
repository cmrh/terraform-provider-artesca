package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetAccount(t *testing.T) {
	overlay := ConfigOverlay{
		Users: []User{
			{AccountName: "team-a", UserName: "team-a", ARN: "arn:aws:iam::111:root", CanonicalID: "cid-111"},
			{AccountName: "team-b", UserName: "team-b", ARN: "arn:aws:iam::222:root", CanonicalID: "cid-222"},
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

	user, err := client.GetAccount(context.Background(), "team-a")
	if err != nil {
		t.Fatalf("GetAccount returned error: %v", err)
	}
	if user == nil {
		t.Fatal("GetAccount returned nil")
	}
	if user.AccountName != "team-a" {
		t.Errorf("AccountName = %q, want team-a", user.AccountName)
	}
	if user.CanonicalID != "cid-111" {
		t.Errorf("CanonicalID = %q, want cid-111", user.CanonicalID)
	}
}

func TestGetAccountNotFound(t *testing.T) {
	overlay := ConfigOverlay{
		Users: []User{
			{AccountName: "other", UserName: "other"},
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

	user, err := client.GetAccount(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetAccount returned error: %v", err)
	}
	if user != nil {
		t.Errorf("expected nil for nonexistent account, got: %+v", user)
	}
}

func TestGetAccountMatchesByUserName(t *testing.T) {
	overlay := ConfigOverlay{
		Users: []User{
			{AccountName: "different", UserName: "lookup-name", ARN: "arn:test"},
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

	user, err := client.GetAccount(context.Background(), "lookup-name")
	if err != nil {
		t.Fatalf("GetAccount returned error: %v", err)
	}
	if user == nil {
		t.Fatal("expected match by UserName")
	}
	if user.ARN != "arn:test" {
		t.Errorf("ARN = %q, want arn:test", user.ARN)
	}
}

func TestCreateAccount(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/user") {
			t.Errorf("path = %q, want suffix /user", r.URL.Path)
		}
		var req createUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.UserName != "new-account" {
			t.Errorf("userName = %q, want new-account", req.UserName)
		}
		if req.Email != "new@example.com" {
			t.Errorf("email = %q, want new@example.com", req.Email)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(User{
			AccountName: "new-account",
			AccessKey:   "AKIANEW",
			SecretKey:   "secret-new",
			ARN:         "arn:aws:iam::333:root",
			CanonicalID: "cid-333",
		})
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	user, err := client.CreateAccount(context.Background(), "new-account", "new@example.com")
	if err != nil {
		t.Fatalf("CreateAccount returned error: %v", err)
	}
	if user.AccountName != "new-account" {
		t.Errorf("AccountName = %q, want new-account", user.AccountName)
	}
	if user.AccessKey != "AKIANEW" {
		t.Errorf("AccessKey = %q, want AKIANEW", user.AccessKey)
	}
	if user.SecretKey != "secret-new" {
		t.Errorf("SecretKey = %q, want secret-new", user.SecretKey)
	}
}

func TestCreateAccountError(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(409)
		_, _ = w.Write([]byte(`{"error":"conflict"}`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	_, err := client.CreateAccount(context.Background(), "dup", "dup@example.com")
	if err == nil {
		t.Fatal("expected error for 409 response")
	}
	if !strings.Contains(err.Error(), "409") {
		t.Errorf("error = %q, want 409", err.Error())
	}
}

func TestDeleteAccount(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if got := r.URL.Query().Get("accountName"); got != "team-a" {
			t.Errorf("accountName = %q, want team-a", got)
		}
		w.WriteHeader(204)
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	err := client.DeleteAccount(context.Background(), "team-a")
	if err != nil {
		t.Fatalf("DeleteAccount returned error: %v", err)
	}
}

func TestDeleteAccountError(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`server error`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	err := client.DeleteAccount(context.Background(), "team-a")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestDeleteAccountSpecialChars(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("accountName"); got != "team a&b" {
			t.Errorf("accountName = %q, want 'team a&b'", got)
		}
		w.WriteHeader(204)
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	err := client.DeleteAccount(context.Background(), "team a&b")
	if err != nil {
		t.Fatalf("DeleteAccount returned error: %v", err)
	}
}

func TestGenerateAccountKey(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/user/team-a/key") {
			t.Errorf("path = %q, want /user/team-a/key", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(User{
			AccountName: "team-a",
			AccessKey:   "AKIANEWKEY",
			SecretKey:   "new-secret",
		})
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	user, err := client.GenerateAccountKey(context.Background(), "team-a")
	if err != nil {
		t.Fatalf("GenerateAccountKey returned error: %v", err)
	}
	if user.AccessKey != "AKIANEWKEY" {
		t.Errorf("AccessKey = %q, want AKIANEWKEY", user.AccessKey)
	}
	if user.SecretKey != "new-secret" {
		t.Errorf("SecretKey = %q, want new-secret", user.SecretKey)
	}
}

func TestGenerateAccountKeyError(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`error`))
	}))
	defer apiServer.Close()

	client, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()

	_, err := client.GenerateAccountKey(context.Background(), "team-a")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}
