package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLifecycleRuleUpdated(t *testing.T) {
	for _, tc := range []struct {
		name     string
		raw      string
		next     LifecycleRule
		contains []string
		absent   []string
	}{
		{
			name:     "And filter keeps the tag",
			raw:      `<ID>r</ID><Status>Enabled</Status><Filter><And><Prefix>logs/</Prefix><Tag><Key>k</Key><Value>v</Value></Tag></And></Filter><Expiration><Days>30</Days></Expiration>`,
			next:     LifecycleRule{ID: "r", Status: "Disabled", Prefix: "new/", ExpirationDays: 60},
			contains: []string{`<And><Prefix>new/</Prefix><Tag><Key>k</Key><Value>v</Value></Tag></And>`, `<Days>60</Days>`, `<Status>Disabled</Status>`},
		},
		{
			name:     "legacy top-level prefix stays top-level",
			raw:      `<ID>r</ID><Status>Enabled</Status><Prefix>old/</Prefix><Expiration><Days>30</Days></Expiration>`,
			next:     LifecycleRule{ID: "r", Status: "Enabled", Prefix: "new/", ExpirationDays: 30},
			contains: []string{`<Prefix>new/</Prefix>`},
			absent:   []string{`<Filter>`},
		},
		{
			name:     "tag-only filter gains an And with the prefix",
			raw:      `<ID>r</ID><Status>Enabled</Status><Filter><Tag><Key>k</Key><Value>v</Value></Tag></Filter><Expiration><Days>30</Days></Expiration>`,
			next:     LifecycleRule{ID: "r", Status: "Enabled", Prefix: "p/", ExpirationDays: 30},
			contains: []string{`<Filter><And><Prefix>p/</Prefix><Tag><Key>k</Key><Value>v</Value></Tag></And></Filter>`},
		},
		{
			name:     "other actions are kept",
			raw:      `<ID>r</ID><Status>Enabled</Status><Filter><Prefix>a/</Prefix></Filter><Expiration><Days>30</Days></Expiration><NoncurrentVersionExpiration><NoncurrentDays>7</NoncurrentDays></NoncurrentVersionExpiration><AbortIncompleteMultipartUpload><DaysAfterInitiation>3</DaysAfterInitiation></AbortIncompleteMultipartUpload>`,
			next:     LifecycleRule{ID: "r", Status: "Enabled", Prefix: "a/", ExpirationDays: 45},
			contains: []string{`<Days>45</Days>`, `<NoncurrentDays>7</NoncurrentDays>`, `<DaysAfterInitiation>3</DaysAfterInitiation>`},
		},
		{
			name:     "a transition update keeps the expiration",
			raw:      `<ID>r</ID><Status>Enabled</Status><Filter><Prefix>a/</Prefix></Filter><Expiration><Days>60</Days></Expiration><Transition><Days>10</Days><StorageClass>old-loc</StorageClass></Transition>`,
			next:     LifecycleRule{ID: "r", Status: "Enabled", Prefix: "a/", TransitionDays: 20, TransitionLocation: "new-loc"},
			contains: []string{`<Expiration><Days>60</Days></Expiration>`, `<Transition><Days>20</Days><StorageClass>new-loc</StorageClass></Transition>`},
		},
		{
			name:     "days replace an expired-delete-marker expiration",
			raw:      `<ID>r</ID><Status>Enabled</Status><Filter><Prefix>b/</Prefix></Filter><Expiration><ExpiredObjectDeleteMarker>true</ExpiredObjectDeleteMarker></Expiration>`,
			next:     LifecycleRule{ID: "r", Status: "Enabled", Prefix: "b/", ExpirationDays: 30},
			contains: []string{`<Expiration><Days>30</Days></Expiration>`},
			absent:   []string{`ExpiredObjectDeleteMarker`},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := LifecycleRule{ID: "r", raw: tc.raw}.Updated(tc.next)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range tc.contains {
				if !strings.Contains(got.raw, want) {
					t.Errorf("rule %s\n  missing %s", got.raw, want)
				}
			}
			for _, unwanted := range tc.absent {
				if strings.Contains(got.raw, unwanted) {
					t.Errorf("rule %s\n  still has %s", got.raw, unwanted)
				}
			}
			if strings.Contains(got.raw, "xmlns") {
				t.Errorf("rule %s has namespace noise", got.raw)
			}
		})
	}
}

func TestLifecycleRuleUpdatedWithoutRaw(t *testing.T) {
	next := LifecycleRule{ID: "r", Status: "Enabled", Prefix: "a/", ExpirationDays: 30}
	got, err := LifecycleRule{ID: "r"}.Updated(next)
	if err != nil || got != next {
		t.Errorf("Updated = %+v, %v; want next unchanged", got, err)
	}
}

func TestGetBucketLifecyclePrefixForms(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<LifecycleConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">` +
			`<Rule><ID>and</ID><Status>Enabled</Status><Filter><And><Prefix>logs/</Prefix><Tag><Key>k</Key><Value>v</Value></Tag></And></Filter><Expiration><Days>30</Days></Expiration></Rule>` +
			`<Rule><ID>legacy</ID><Status>Enabled</Status><Prefix>old/</Prefix><Expiration><Days>30</Days></Expiration></Rule>` +
			`<Rule><ID>filter</ID><Status>Enabled</Status><Filter><Prefix>f/</Prefix></Filter><Expiration><Days>30</Days></Expiration></Rule>` +
			`</LifecycleConfiguration>`))
	}))
	defer srv.Close()
	rules, err := NewS3Client(srv.URL, "us-east-1", false).GetBucketLifecycle(context.Background(), Credentials{AccessKey: "AKID", SecretKey: "secret"}, "bkt")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"and": "logs/", "legacy": "old/", "filter": "f/"}
	for _, r := range rules {
		if r.Prefix != want[r.ID] {
			t.Errorf("rule %s prefix = %q, want %q", r.ID, r.Prefix, want[r.ID])
		}
	}
}
