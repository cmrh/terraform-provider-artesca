package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const webIdentityPage1 = `{"IsTruncated":true,"Marker":"m1","Accounts":[
 {"Name":"team-a","CreationDate":"2026-10-02T16:12:57Z","CanonicalId":"cid-a",
  "Roles":[{"Name":"storage-manager-role","Arn":"arn:aws:iam::111111111111:role/scality-internal/storage-manager-role"}]}]}`

// Page 2 repeats team-a (its second role) before team-b, as the server does
// when MaxItems splits an account's roles across pages.
const webIdentityPage2 = `{"IsTruncated":false,"Accounts":[
 {"Name":"team-a","CreationDate":"2026-10-02T16:12:57Z","CanonicalId":"cid-a",
  "Roles":[{"Name":"storage-usage-consumer-role","Arn":"arn:aws:iam::111111111111:role/scality-internal/storage-usage-consumer-role"}]},
 {"Name":"team-b","CreationDate":"2026-10-01T23:06:31Z","CanonicalId":"cid-b",
  "Roles":[{"Name":"storage-manager-role","Arn":"arn:aws:iam::222222222222:role/scality-internal/storage-manager-role"}]}]}`

func newWebIdentityServer(t *testing.T, handler func(w http.ResponseWriter, form map[string]string)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.Header.Get("Authorization") != "" {
			t.Error("request must not be SigV4-signed")
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("parsing form: %v", err)
			return
		}
		form := map[string]string{}
		for k := range r.PostForm {
			form[k] = r.PostForm.Get(k)
		}
		if form["Action"] != "GetRolesForWebIdentity" {
			t.Errorf("Action = %q, want GetRolesForWebIdentity", form["Action"])
		}
		if form["WebIdentityToken"] != "tok" {
			t.Errorf("WebIdentityToken = %q, want tok", form["WebIdentityToken"])
		}
		handler(w, form)
	}))
}

func TestListAccountsPaginatesAndMerges(t *testing.T) {
	var markers []string
	server := newWebIdentityServer(t, func(w http.ResponseWriter, form map[string]string) {
		markers = append(markers, form["Marker"])
		w.Header().Set("Content-Type", "application/json")
		if form["Marker"] == "" {
			_, _ = w.Write([]byte(webIdentityPage1))
			return
		}
		_, _ = w.Write([]byte(webIdentityPage2))
	})
	defer server.Close()

	client := NewIAMClient(server.URL, "us-east-1", false)
	accounts, err := client.ListAccounts(context.Background(), "tok")
	if err != nil {
		t.Fatalf("ListAccounts returned error: %v", err)
	}

	if len(markers) != 2 || markers[0] != "" || markers[1] != "m1" {
		t.Errorf("markers sent = %q, want [\"\" \"m1\"]", markers)
	}
	if len(accounts) != 2 {
		t.Fatalf("got %d accounts, want 2 (team-a must not be duplicated): %+v", len(accounts), accounts)
	}
	want := []IAMAccount{
		{Name: "team-a", ID: "111111111111", CanonicalID: "cid-a", CreationDate: "2026-10-02T16:12:57Z"},
		{Name: "team-b", ID: "222222222222", CanonicalID: "cid-b", CreationDate: "2026-10-01T23:06:31Z"},
	}
	for i := range want {
		if accounts[i] != want[i] {
			t.Errorf("accounts[%d] = %+v, want %+v", i, accounts[i], want[i])
		}
	}
}

func TestListAccountsEmpty(t *testing.T) {
	server := newWebIdentityServer(t, func(w http.ResponseWriter, _ map[string]string) {
		_, _ = w.Write([]byte(`{"IsTruncated":false,"Accounts":[]}`))
	})
	defer server.Close()

	client := NewIAMClient(server.URL, "us-east-1", false)
	accounts, err := client.ListAccounts(context.Background(), "tok")
	if err != nil {
		t.Fatalf("ListAccounts returned error: %v", err)
	}
	if len(accounts) != 0 {
		t.Errorf("got %d accounts, want 0", len(accounts))
	}
}

func TestListAccountsInvalidToken(t *testing.T) {
	server := newWebIdentityServer(t, func(w http.ResponseWriter, _ map[string]string) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(xmlErrorResponse("InvalidToken", "The provided token is malformed or otherwise invalid.")))
	})
	defer server.Close()

	client := NewIAMClient(server.URL, "us-east-1", false)
	_, err := client.ListAccounts(context.Background(), "tok")
	if err == nil || !strings.Contains(err.Error(), "InvalidToken") {
		t.Fatalf("err = %v, want InvalidToken", err)
	}
}

func TestListAccountsTruncatedWithoutMarker(t *testing.T) {
	server := newWebIdentityServer(t, func(w http.ResponseWriter, _ map[string]string) {
		_, _ = w.Write([]byte(`{"IsTruncated":true,"Accounts":[]}`))
	})
	defer server.Close()

	client := NewIAMClient(server.URL, "us-east-1", false)
	_, err := client.ListAccounts(context.Background(), "tok")
	if err == nil || !strings.Contains(err.Error(), "no Marker") {
		t.Fatalf("err = %v, want truncated-without-marker error", err)
	}
}

func TestListAccountsRepeatedMarker(t *testing.T) {
	calls := 0
	server := newWebIdentityServer(t, func(w http.ResponseWriter, _ map[string]string) {
		calls++
		if calls > 3 {
			t.Error("client kept paginating on a repeated Marker")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"IsTruncated":true,"Marker":"same","Accounts":[]}`))
	})
	defer server.Close()

	client := NewIAMClient(server.URL, "us-east-1", false)
	_, err := client.ListAccounts(context.Background(), "tok")
	if err == nil || !strings.Contains(err.Error(), "repeated Marker") {
		t.Fatalf("err = %v, want repeated-marker error", err)
	}
}

func TestGetAccountByName(t *testing.T) {
	server := newWebIdentityServer(t, func(w http.ResponseWriter, form map[string]string) {
		if form["Marker"] == "" {
			_, _ = w.Write([]byte(webIdentityPage1))
			return
		}
		_, _ = w.Write([]byte(webIdentityPage2))
	})
	defer server.Close()

	client := NewIAMClient(server.URL, "us-east-1", false)

	acct, err := client.GetAccountByName(context.Background(), "tok", "team-b")
	if err != nil {
		t.Fatalf("GetAccountByName returned error: %v", err)
	}
	if acct == nil || acct.ID != "222222222222" || acct.CanonicalID != "cid-b" {
		t.Errorf("got %+v, want team-b with ID 222222222222", acct)
	}

	missing, err := client.GetAccountByName(context.Background(), "tok", "nonexistent")
	if err != nil {
		t.Fatalf("GetAccountByName returned error: %v", err)
	}
	if missing != nil {
		t.Errorf("got %+v, want nil for nonexistent account", missing)
	}
}

func TestAccountARN(t *testing.T) {
	if got := AccountARN("138747885454"); got != "arn:aws:iam::138747885454:root" {
		t.Errorf("AccountARN = %q", got)
	}
}

func TestAccountIDFromARN(t *testing.T) {
	cases := map[string]string{
		"arn:aws:iam::138747885454:role/scality-internal/storage-manager-role": "138747885454",
		"arn:aws:iam::691102982860:/overlay-probe/":                            "691102982860",
		"not-an-arn":  "",
		"arn:aws:iam": "",
	}
	for in, want := range cases {
		if got := accountIDFromARN(in); got != want {
			t.Errorf("accountIDFromARN(%q) = %q, want %q", in, got, want)
		}
	}
}
