package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPutBucketLifecycleZeroDayTransition(t *testing.T) {
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		body = string(b)
	}))
	defer server.Close()

	rules := []LifecycleRule{{ID: "r1", Status: "Enabled", TransitionDays: 0, TransitionLocation: "cold"}}
	if err := NewS3Client(server.URL, "us-east-1", false).PutBucketLifecycle(context.Background(), testCreds, "bkt", rules); err != nil {
		t.Fatalf("PutBucketLifecycle returned error: %v", err)
	}
	if want := "<Transition><Days>0</Days><StorageClass>cold</StorageClass></Transition>"; !strings.Contains(body, want) {
		t.Errorf("body = %s, want it to contain %s", body, want)
	}
}

// Rules read from the server are written back unchanged, including elements
// the provider doesn't model; new rules are encoded from their fields.
func TestPutBucketLifecycleKeepsUnmodelledElements(t *testing.T) {
	uiRules := []string{
		`<ID>ui-tag</ID><Status>Enabled</Status><Filter><Tag><Key>env</Key><Value>dev</Value></Tag></Filter><Expiration><Days>7</Days></Expiration>`,
		`<ID>ui-and</ID><Status>Enabled</Status><Filter><And><Prefix>logs/</Prefix><Tag><Key>tier</Key><Value>cold</Value></Tag></And></Filter><Expiration><Date>2030-01-01T00:00:00.000Z</Date></Expiration>`,
		`<ID>ui-noncurrent</ID><Status>Enabled</Status><Filter><Prefix></Prefix></Filter><NoncurrentVersionExpiration><NoncurrentDays>10</NoncurrentDays></NoncurrentVersionExpiration><AbortIncompleteMultipartUpload><DaysAfterInitiation>3</DaysAfterInitiation></AbortIncompleteMultipartUpload>`,
	}
	var put string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`<LifecycleConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Rule>` +
				strings.Join(uiRules, "</Rule><Rule>") + `</Rule></LifecycleConfiguration>`))
			return
		}
		b, _ := io.ReadAll(r.Body)
		put = string(b)
	}))
	defer server.Close()

	c := NewS3Client(server.URL, "us-east-1", false)
	rules, err := c.GetBucketLifecycle(context.Background(), testCreds, "bkt")
	if err != nil {
		t.Fatalf("GetBucketLifecycle returned error: %v", err)
	}
	if rules[0].ExpirationDays != 7 || rules[0].Prefix != "" {
		t.Errorf("rule 0 parsed as %+v", rules[0])
	}
	rules = append(rules, LifecycleRule{ID: "tf", Status: "Enabled", Prefix: "tf/", ExpirationDays: 30})
	if err := c.PutBucketLifecycle(context.Background(), testCreds, "bkt", rules); err != nil {
		t.Fatalf("PutBucketLifecycle returned error: %v", err)
	}
	for _, r := range uiRules {
		if !strings.Contains(put, "<Rule>"+r+"</Rule>") {
			t.Errorf("body missing rule %s:\n%s", r, put)
		}
	}
	if want := `<Rule><ID>tf</ID><Status>Enabled</Status><Filter><Prefix>tf/</Prefix></Filter><Expiration><Days>30</Days></Expiration></Rule>`; !strings.Contains(put, want) {
		t.Errorf("body missing %s:\n%s", want, put)
	}
}
