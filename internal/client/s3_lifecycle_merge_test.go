package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestLifecycleRuleUpdated(t *testing.T) {
	tag := []BucketTag{{"k", "v"}}
	for _, tc := range []struct {
		name     string
		raw      string
		next     LifecycleRule
		scope    LifecycleScope
		contains []string
		absent   []string
	}{
		{
			name:     "And filter takes the new prefix and keeps the tag",
			raw:      `<ID>r</ID><Status>Enabled</Status><Filter><And><Prefix>logs/</Prefix><Tag><Key>k</Key><Value>v</Value></Tag></And></Filter><Expiration><Days>30</Days></Expiration>`,
			next:     LifecycleRule{ID: "r", Status: "Disabled", Prefix: "new/", Tags: tag, ExpirationDays: 60},
			contains: []string{`<And><Prefix>new/</Prefix><Tag><Key>k</Key><Value>v</Value></Tag></And>`, `<Days>60</Days>`, `<Status>Disabled</Status>`},
		},
		{
			name:     "removing the tags leaves a prefix filter",
			raw:      `<ID>r</ID><Status>Enabled</Status><Filter><And><Prefix>logs/</Prefix><Tag><Key>k</Key><Value>v</Value></Tag></And></Filter><Expiration><Days>30</Days></Expiration>`,
			next:     LifecycleRule{ID: "r", Status: "Enabled", Prefix: "logs/", ExpirationDays: 30},
			contains: []string{`<Filter><Prefix>logs/</Prefix></Filter>`},
			absent:   []string{`<Tag>`, `<And>`},
		},
		{
			name:     "a single tag without prefix is a Tag filter",
			raw:      `<ID>r</ID><Status>Enabled</Status><Filter><Prefix>a/</Prefix></Filter><Expiration><Days>30</Days></Expiration>`,
			next:     LifecycleRule{ID: "r", Status: "Enabled", Tags: tag, ExpirationDays: 30},
			contains: []string{`<Filter><Tag><Key>k</Key><Value>v</Value></Tag></Filter>`},
			absent:   []string{`<And>`, `<Prefix>`},
		},
		{
			name:     "legacy top-level prefix stays top-level",
			raw:      `<ID>r</ID><Status>Enabled</Status><Prefix>old/</Prefix><Expiration><Days>30</Days></Expiration>`,
			next:     LifecycleRule{ID: "r", Status: "Enabled", Prefix: "new/", ExpirationDays: 30},
			contains: []string{`<Prefix>new/</Prefix>`},
			absent:   []string{`<Filter>`},
		},
		{
			name:     "legacy prefix moves into a Filter when tags are added",
			raw:      `<ID>r</ID><Status>Enabled</Status><Prefix>old/</Prefix><Expiration><Days>30</Days></Expiration>`,
			next:     LifecycleRule{ID: "r", Status: "Enabled", Prefix: "old/", Tags: tag, ExpirationDays: 30},
			contains: []string{`<Filter><And><Prefix>old/</Prefix><Tag><Key>k</Key><Value>v</Value></Tag></And></Filter>`},
			absent:   []string{`<Status>Enabled</Status><Prefix>`},
		},
		{
			name:     "filter conditions the provider doesn't model are kept",
			raw:      `<ID>r</ID><Status>Enabled</Status><Filter><And><Prefix>a/</Prefix><ObjectSizeGreaterThan>100</ObjectSizeGreaterThan></And></Filter><Expiration><Days>30</Days></Expiration>`,
			next:     LifecycleRule{ID: "r", Status: "Enabled", Prefix: "b/", ExpirationDays: 30},
			contains: []string{`<And><Prefix>b/</Prefix><ObjectSizeGreaterThan>100</ObjectSizeGreaterThan></And>`},
		},
		{
			name:     "expiration scope sets noncurrent and multipart actions and keeps transitions",
			raw:      `<ID>r</ID><Status>Enabled</Status><Filter><Prefix>a/</Prefix></Filter><Expiration><Days>30</Days></Expiration><Transition><Days>10</Days><StorageClass>cold</StorageClass></Transition><NoncurrentVersionExpiration><NoncurrentDays>7</NoncurrentDays><NewerNoncurrentVersions>2</NewerNoncurrentVersions></NoncurrentVersionExpiration>`,
			next:     LifecycleRule{ID: "r", Status: "Enabled", Prefix: "a/", ExpirationDays: 45, NoncurrentExpirationDays: 14, AbortMultipartDays: 3},
			contains: []string{`<Expiration><Days>45</Days></Expiration>`, `<NoncurrentDays>14</NoncurrentDays><NewerNoncurrentVersions>2</NewerNoncurrentVersions>`, `<AbortIncompleteMultipartUpload><DaysAfterInitiation>3</DaysAfterInitiation></AbortIncompleteMultipartUpload>`, `<Transition><Days>10</Days><StorageClass>cold</StorageClass></Transition>`},
		},
		{
			name:     "expiration scope removes actions the plan no longer sets",
			raw:      `<ID>r</ID><Status>Enabled</Status><Filter><Prefix>a/</Prefix></Filter><Expiration><Days>30</Days></Expiration><NoncurrentVersionExpiration><NoncurrentDays>7</NoncurrentDays></NoncurrentVersionExpiration><AbortIncompleteMultipartUpload><DaysAfterInitiation>3</DaysAfterInitiation></AbortIncompleteMultipartUpload>`,
			next:     LifecycleRule{ID: "r", Status: "Enabled", Prefix: "a/", NoncurrentExpirationDays: 7},
			contains: []string{`<NoncurrentDays>7</NoncurrentDays>`},
			absent:   []string{`<Expiration>`, `AbortIncompleteMultipartUpload`},
		},
		{
			name:     "days replace an expired-delete-marker expiration",
			raw:      `<ID>r</ID><Status>Enabled</Status><Filter><Prefix>b/</Prefix></Filter><Expiration><ExpiredObjectDeleteMarker>true</ExpiredObjectDeleteMarker></Expiration>`,
			next:     LifecycleRule{ID: "r", Status: "Enabled", Prefix: "b/", ExpirationDays: 30},
			contains: []string{`<Expiration><Days>30</Days></Expiration>`},
			absent:   []string{`ExpiredObjectDeleteMarker`},
		},
		{
			name:     "a date replaces days",
			raw:      `<ID>r</ID><Status>Enabled</Status><Filter><Prefix></Prefix></Filter><Expiration><Days>30</Days></Expiration>`,
			next:     LifecycleRule{ID: "r", Status: "Enabled", ExpirationDate: "2027-01-01"},
			contains: []string{`<Expiration><Date>2027-01-01T00:00:00Z</Date></Expiration>`},
			absent:   []string{`<Days>`},
		},
		{
			name:     "transition scope keeps the expiration and the noncurrent transition",
			raw:      `<ID>r</ID><Status>Enabled</Status><Filter><Prefix>a/</Prefix></Filter><Expiration><Days>60</Days></Expiration><Transition><Days>10</Days><StorageClass>old-loc</StorageClass></Transition><NoncurrentVersionTransition><NoncurrentDays>2</NoncurrentDays><StorageClass>nc</StorageClass></NoncurrentVersionTransition>`,
			next:     LifecycleRule{ID: "r", Status: "Enabled", Prefix: "a/", TransitionDays: 20, TransitionLocation: "new-loc"},
			scope:    ScopeTransition,
			contains: []string{`<Expiration><Days>60</Days></Expiration>`, `<Transition><Days>20</Days><StorageClass>new-loc</StorageClass></Transition>`, `<NoncurrentDays>2</NoncurrentDays><StorageClass>nc</StorageClass>`},
		},
		{
			name:     "a transition date replaces days",
			raw:      `<ID>r</ID><Status>Enabled</Status><Filter><Prefix></Prefix></Filter><Transition><Days>10</Days><StorageClass>cold</StorageClass></Transition>`,
			next:     LifecycleRule{ID: "r", Status: "Enabled", TransitionDate: "2027-02-01", TransitionLocation: "cold"},
			scope:    ScopeTransition,
			contains: []string{`<Transition><Date>2027-02-01T00:00:00Z</Date><StorageClass>cold</StorageClass></Transition>`},
			absent:   []string{`<Days>`},
		},
		{
			name:     "noncurrent transition scope keeps the current transition",
			raw:      `<ID>r</ID><Status>Enabled</Status><Filter><Prefix></Prefix></Filter><Transition><Days>10</Days><StorageClass>cold</StorageClass></Transition><NoncurrentVersionTransition><NoncurrentDays>5</NoncurrentDays><StorageClass>old</StorageClass><NewerNoncurrentVersions>1</NewerNoncurrentVersions></NoncurrentVersionTransition>`,
			next:     LifecycleRule{ID: "r", Status: "Enabled", NoncurrentTransitionDays: 0, NoncurrentTransitionLocation: "new"},
			scope:    ScopeNoncurrentTransition,
			contains: []string{`<Transition><Days>10</Days><StorageClass>cold</StorageClass></Transition>`, `<NoncurrentVersionTransition><NoncurrentDays>0</NoncurrentDays><StorageClass>new</StorageClass><NewerNoncurrentVersions>1</NewerNoncurrentVersions></NoncurrentVersionTransition>`},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := LifecycleRule{ID: "r", raw: tc.raw}.Updated(tc.next, tc.scope)
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
	got, err := LifecycleRule{ID: "r"}.Updated(next, ScopeExpiration)
	if err != nil || !reflect.DeepEqual(got, next) {
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
