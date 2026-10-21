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

func TestPutBucketLifecycleEncodesActions(t *testing.T) {
	for _, tc := range []struct {
		name string
		rule LifecycleRule
		want []string
	}{
		{"noncurrent expiration, abort multipart, two tags",
			LifecycleRule{ID: "r", Status: "Enabled", Prefix: "p/", Tags: []BucketTag{{"a", "1"}, {"b", "2"}}, NoncurrentExpirationDays: 30, AbortMultipartDays: 7},
			[]string{`<Filter><And><Prefix>p/</Prefix><Tag><Key>a</Key><Value>1</Value></Tag><Tag><Key>b</Key><Value>2</Value></Tag></And></Filter>`,
				`<NoncurrentVersionExpiration><NoncurrentDays>30</NoncurrentDays></NoncurrentVersionExpiration>`,
				`<AbortIncompleteMultipartUpload><DaysAfterInitiation>7</DaysAfterInitiation></AbortIncompleteMultipartUpload>`}},
		{"single tag without prefix",
			LifecycleRule{ID: "r", Status: "Enabled", Tags: []BucketTag{{"a", "1"}}, ExpirationDays: 5},
			[]string{`<Filter><Tag><Key>a</Key><Value>1</Value></Tag></Filter>`}},
		{"tags without prefix",
			LifecycleRule{ID: "r", Status: "Enabled", Tags: []BucketTag{{"a", "1"}, {"b", "2"}}, ExpirationDays: 5},
			[]string{`<Filter><And><Tag><Key>a</Key><Value>1</Value></Tag><Tag><Key>b</Key><Value>2</Value></Tag></And></Filter>`}},
		{"expiration date",
			LifecycleRule{ID: "r", Status: "Enabled", ExpirationDate: "2027-01-01"},
			[]string{`<Expiration><Date>2027-01-01T00:00:00Z</Date></Expiration>`}},
		{"expired delete markers",
			LifecycleRule{ID: "r", Status: "Enabled", ExpiredObjectDeleteMarker: true},
			[]string{`<Expiration><ExpiredObjectDeleteMarker>true</ExpiredObjectDeleteMarker></Expiration>`}},
		{"transition date",
			LifecycleRule{ID: "r", Status: "Enabled", TransitionDate: "2027-02-01", TransitionLocation: "cold"},
			[]string{`<Transition><Date>2027-02-01T00:00:00Z</Date><StorageClass>cold</StorageClass></Transition>`}},
		{"noncurrent transition",
			LifecycleRule{ID: "r", Status: "Enabled", NoncurrentTransitionDays: 0, NoncurrentTransitionLocation: "cold"},
			[]string{`<NoncurrentVersionTransition><NoncurrentDays>0</NoncurrentDays><StorageClass>cold</StorageClass></NoncurrentVersionTransition>`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var body string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				b, _ := io.ReadAll(r.Body)
				body = string(b)
			}))
			defer server.Close()
			if err := NewS3Client(server.URL, "us-east-1", false).PutBucketLifecycle(context.Background(), testCreds, "bkt", []LifecycleRule{tc.rule}); err != nil {
				t.Fatalf("PutBucketLifecycle returned error: %v", err)
			}
			for _, want := range tc.want {
				if !strings.Contains(body, want) {
					t.Errorf("body = %s\n  missing %s", body, want)
				}
			}
			if strings.Contains(body, "<Transition>") && tc.rule.TransitionLocation == "" {
				t.Errorf("body = %s has a Transition the rule doesn't set", body)
			}
		})
	}
}

func TestGetBucketLifecycleParsesActions(t *testing.T) {
	// Shapes as ARTESCA returns them.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<LifecycleConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">` +
			`<Rule><ID>r1</ID><Status>Enabled</Status><Filter><Tag><Key>k</Key><Value>v</Value></Tag></Filter><Expiration><Date>2027-01-01T00:00:00Z</Date></Expiration><NoncurrentVersionExpiration><NoncurrentDays>30</NoncurrentDays></NoncurrentVersionExpiration></Rule>` +
			`<Rule><ID>r2</ID><Status>Enabled</Status><Filter><And><Prefix>p/</Prefix><Tag><Key>b</Key><Value>2</Value></Tag><Tag><Key>a</Key><Value>1</Value></Tag></And></Filter><NoncurrentVersionTransition><NoncurrentDays>0</NoncurrentDays><StorageClass>cold</StorageClass></NoncurrentVersionTransition></Rule>` +
			`<Rule><ID>r3</ID><Status>Enabled</Status><Filter><Prefix></Prefix></Filter><Expiration><ExpiredObjectDeleteMarker>true</ExpiredObjectDeleteMarker></Expiration><AbortIncompleteMultipartUpload><DaysAfterInitiation>7</DaysAfterInitiation></AbortIncompleteMultipartUpload></Rule>` +
			`<Rule><ID>r4</ID><Status>Enabled</Status><Filter><Prefix></Prefix></Filter><Transition><Date>2027-02-01T00:00:00.000Z</Date><StorageClass>cold</StorageClass></Transition></Rule>` +
			`</LifecycleConfiguration>`))
	}))
	defer srv.Close()
	rules, err := NewS3Client(srv.URL, "us-east-1", false).GetBucketLifecycle(context.Background(), testCreds, "bkt")
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 4 {
		t.Fatalf("got %d rules", len(rules))
	}
	r1, r2, r3, r4 := rules[0], rules[1], rules[2], rules[3]
	if r1.ExpirationDate != "2027-01-01" || r1.NoncurrentExpirationDays != 30 || len(r1.Tags) != 1 || r1.Tags[0] != (BucketTag{"k", "v"}) || r1.Prefix != "" {
		t.Errorf("r1 = %+v", r1)
	}
	if r2.Prefix != "p/" || len(r2.Tags) != 2 || r2.Tags[0].Key != "a" || r2.NoncurrentTransitionLocation != "cold" || r2.NoncurrentTransitionDays != 0 {
		t.Errorf("r2 = %+v (tags should be sorted)", r2)
	}
	if !r3.ExpiredObjectDeleteMarker || r3.AbortMultipartDays != 7 {
		t.Errorf("r3 = %+v", r3)
	}
	if r4.TransitionDate != "2027-02-01" || r4.TransitionLocation != "cold" {
		t.Errorf("r4 = %+v", r4)
	}
}
