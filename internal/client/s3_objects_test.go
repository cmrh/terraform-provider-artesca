package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestSigV4CanonicalQuery(t *testing.T) {
	for _, tc := range []struct {
		q    url.Values
		want string
	}{
		{nil, ""},
		{url.Values{"versions": {""}}, "versions="},
		{url.Values{"versions": {""}, "key-marker": {"a b"}, "version-id-marker": {"x+y"}}, "key-marker=a%20b&version-id-marker=x%2By&versions="},
		{url.Values{"versionId": {"~a/b"}}, "versionId=~a%2Fb"},
	} {
		if got := sigV4CanonicalQuery(tc.q); got != tc.want {
			t.Errorf("sigV4CanonicalQuery(%v) = %q, want %q", tc.q, got, tc.want)
		}
	}
}

func TestSigV4EscapePath(t *testing.T) {
	for in, want := range map[string]string{
		"/bkt":                 "/bkt",
		"/bkt/dir/a b+c~d.txt": "/bkt/dir/a%20b%2Bc~d.txt",
		"/bkt/q?=&%":           "/bkt/q%3F%3D%26%25",
		"/bkt/é":               "/bkt/%C3%A9",
	} {
		if got := sigV4EscapePath(in); got != want {
			t.Errorf("sigV4EscapePath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestListObjectVersionsFollowsMarkers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if !q.Has("versions") {
			t.Errorf("query %q has no versions", r.URL.RawQuery)
		}
		if q.Get("key-marker") == "" {
			_, _ = w.Write([]byte(`<ListVersionsResult><IsTruncated>true</IsTruncated><NextKeyMarker>a b</NextKeyMarker><NextVersionIdMarker>v1</NextVersionIdMarker><Version><Key>a b</Key><VersionId>v1</VersionId></Version></ListVersionsResult>`))
			return
		}
		if q.Get("key-marker") != "a b" || q.Get("version-id-marker") != "v1" {
			t.Errorf("markers = %q / %q", q.Get("key-marker"), q.Get("version-id-marker"))
		}
		_, _ = w.Write([]byte(`<ListVersionsResult><IsTruncated>false</IsTruncated><DeleteMarker><Key>c</Key><VersionId>v2</VersionId></DeleteMarker></ListVersionsResult>`))
	}))
	defer srv.Close()

	got, err := NewS3Client(srv.URL, "us-east-1", false).ListObjectVersions(context.Background(), Credentials{AccessKey: "AKID", SecretKey: "secret"}, "bkt")
	if err != nil {
		t.Fatalf("ListObjectVersions: %v", err)
	}
	want := []ObjectVersion{{"a b", "v1"}, {"c", "v2"}}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("versions = %v, want %v", got, want)
	}
}

func TestListObjectVersionsRejectsRepeatedMarker(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<ListVersionsResult><IsTruncated>true</IsTruncated><NextKeyMarker>k</NextKeyMarker><NextVersionIdMarker>v</NextVersionIdMarker></ListVersionsResult>`))
	}))
	defer srv.Close()
	if _, err := NewS3Client(srv.URL, "us-east-1", false).ListObjectVersions(context.Background(), Credentials{AccessKey: "AKID", SecretKey: "secret"}, "bkt"); err == nil {
		t.Fatal("expected an error for a repeated marker")
	}
}

func TestDeleteObjectVersion(t *testing.T) {
	var gotPath, gotVersion string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotVersion = r.URL.EscapedPath(), r.URL.Query().Get("versionId")
		if gotVersion == "gone" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`<Error><Code>NoSuchVersion</Code><Message>gone</Message></Error>`))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	c := NewS3Client(srv.URL, "us-east-1", false)
	creds := Credentials{AccessKey: "AKID", SecretKey: "secret"}
	if err := c.DeleteObjectVersion(context.Background(), creds, "bkt", "dir/a b", "v1", false); err != nil {
		t.Fatalf("DeleteObjectVersion: %v", err)
	}
	if gotPath != "/bkt/dir/a%20b" || gotVersion != "v1" {
		t.Errorf("request = %s ?versionId=%s", gotPath, gotVersion)
	}
	if err := c.DeleteObjectVersion(context.Background(), creds, "bkt", "k", "gone", false); err != nil {
		t.Errorf("a version already gone should count as deleted: %v", err)
	}
}

func TestDeleteAccountNotEmpty(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))
	defer apiServer.Close()
	c, cleanup := newTestManagementClient(t, apiServer)
	defer cleanup()
	if err := c.DeleteAccount(context.Background(), "acct"); !errors.Is(err, ErrAccountNotEmpty) {
		t.Fatalf("error = %v, want ErrAccountNotEmpty", err)
	}
}

func TestListGroupMembersAndPolicies(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		switch r.Form.Get("Action") {
		case "GetGroup":
			_, _ = w.Write([]byte(`<GetGroupResponse><GetGroupResult><Users><member><UserName>alice</UserName></member><member><UserName>bob</UserName></member></Users><IsTruncated>false</IsTruncated></GetGroupResult></GetGroupResponse>`))
		case "ListGroupPolicies":
			_, _ = w.Write([]byte(`<ListGroupPoliciesResponse><ListGroupPoliciesResult><PolicyNames><member>p1</member></PolicyNames><IsTruncated>false</IsTruncated></ListGroupPoliciesResult></ListGroupPoliciesResponse>`))
		}
	}))
	defer srv.Close()
	c := NewIAMClient(srv.URL, "us-east-1", false)
	creds := Credentials{AccessKey: "AKID", SecretKey: "secret"}
	members, err := c.ListGroupMembers(context.Background(), creds, "g")
	if err != nil || len(members) != 2 || members[0] != "alice" || members[1] != "bob" {
		t.Errorf("ListGroupMembers = %v, %v", members, err)
	}
	policies, err := c.ListGroupPolicies(context.Background(), creds, "g")
	if err != nil || len(policies) != 1 || policies[0] != "p1" {
		t.Errorf("ListGroupPolicies = %v, %v", policies, err)
	}
}
