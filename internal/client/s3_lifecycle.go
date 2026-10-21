package client

import (
	"context"
	"encoding/xml"
	"fmt"
	"sort"
	"time"
)

type lifecycleConfiguration struct {
	XMLName xml.Name        `xml:"LifecycleConfiguration"`
	Rules   []lifecycleRule `xml:"Rule"`
}

type lifecycleRule struct {
	ID                             string                         `xml:"ID"`
	Status                         string                         `xml:"Status"`
	Prefix                         *string                        `xml:"Prefix,omitempty"` // legacy form, read only
	Filter                         *lifecycleFilter               `xml:"Filter,omitempty"`
	Expiration                     *lifecycleExpiration           `xml:"Expiration,omitempty"`
	NoncurrentVersionExpiration    *lifecycleNoncurrentExpiration `xml:"NoncurrentVersionExpiration,omitempty"`
	AbortIncompleteMultipartUpload *lifecycleAbortMultipart       `xml:"AbortIncompleteMultipartUpload,omitempty"`
	Transition                     *lifecycleTransition           `xml:"Transition,omitempty"`
	NoncurrentVersionTransition    *lifecycleNoncurrentTransition `xml:"NoncurrentVersionTransition,omitempty"`
	Inner                          string                         `xml:",innerxml"` // set on read only
}

// MarshalXML writes a rule read from the server back unchanged, keeping
// elements the provider doesn't model. Rules built by the provider are
// encoded from their fields.
func (r lifecycleRule) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if r.Inner != "" {
		return e.EncodeElement(rawXML{Inner: r.Inner}, start)
	}
	type plain lifecycleRule
	return e.EncodeElement(plain(r), start)
}

type lifecycleTag struct {
	Key   string `xml:"Key"`
	Value string `xml:"Value"`
}

type lifecycleFilter struct {
	Prefix *string       `xml:"Prefix,omitempty"`
	Tag    *lifecycleTag `xml:"Tag,omitempty"`
	And    *struct {
		Prefix *string        `xml:"Prefix,omitempty"`
		Tags   []lifecycleTag `xml:"Tag"`
	} `xml:"And,omitempty"`
}

type lifecycleExpiration struct {
	Days                      int    `xml:"Days,omitempty"`
	Date                      string `xml:"Date,omitempty"`
	ExpiredObjectDeleteMarker bool   `xml:"ExpiredObjectDeleteMarker,omitempty"`
}

type lifecycleNoncurrentExpiration struct {
	NoncurrentDays int `xml:"NoncurrentDays"`
}

type lifecycleAbortMultipart struct {
	DaysAfterInitiation int `xml:"DaysAfterInitiation"`
}

type lifecycleTransition struct {
	Days         *int   `xml:"Days,omitempty"` // 0 is valid; ARTESCA rejects a Transition without Days or Date
	Date         string `xml:"Date,omitempty"`
	StorageClass string `xml:"StorageClass"`
}

type lifecycleNoncurrentTransition struct {
	NoncurrentDays int    `xml:"NoncurrentDays"` // 0 is valid
	StorageClass   string `xml:"StorageClass"`
}

// LifecycleRule is one rule of a bucket's lifecycle configuration. Rules
// returned by GetBucketLifecycle are written back by PutBucketLifecycle exactly
// as read; to change a rule, merge the new values in with Updated.
type LifecycleRule struct {
	ID     string
	Status string
	Prefix string
	Tags   []BucketTag // filter tags, sorted by key

	// Expiration actions.
	ExpirationDays            int
	ExpirationDate            string // YYYY-MM-DD
	ExpiredObjectDeleteMarker bool
	NoncurrentExpirationDays  int
	AbortMultipartDays        int

	// Transition actions. A transition is set when its location is.
	TransitionDays               int
	TransitionDate               string // YYYY-MM-DD; replaces TransitionDays
	TransitionLocation           string
	NoncurrentTransitionDays     int
	NoncurrentTransitionLocation string

	raw string // inner XML as read from the server
}

// lifecycleDate is the form ARTESCA requires for lifecycle dates: ISO 8601 at
// midnight UTC.
func lifecycleDate(day string) string { return day + "T00:00:00Z" }

// lifecycleDay returns the YYYY-MM-DD part of a lifecycle date.
func lifecycleDay(date string) string {
	if len(date) >= 10 {
		return date[:10]
	}
	return date
}

func sortedTags(tags []lifecycleTag) []BucketTag {
	out := make([]BucketTag, 0, len(tags))
	for _, t := range tags {
		out = append(out, BucketTag(t))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// buildFilter encodes a prefix and tags the way ARTESCA accepts them: a lone
// prefix as Filter/Prefix, a lone tag as Filter/Tag, anything else as an And.
func buildFilter(prefix string, tags []BucketTag) *lifecycleFilter {
	f := &lifecycleFilter{}
	switch {
	case len(tags) == 0:
		f.Prefix = &prefix
	case len(tags) == 1 && prefix == "":
		tag := lifecycleTag(tags[0])
		f.Tag = &tag
	default:
		f.And = &struct {
			Prefix *string        `xml:"Prefix,omitempty"`
			Tags   []lifecycleTag `xml:"Tag"`
		}{}
		if prefix != "" {
			f.And.Prefix = &prefix
		}
		for _, t := range tags {
			f.And.Tags = append(f.And.Tags, lifecycleTag(t))
		}
	}
	return f
}

func (c *S3Client) GetBucketLifecycle(ctx context.Context, creds Credentials, bucket string) ([]LifecycleRule, error) {
	body, statusCode, err := c.doSignedRequest(ctx, "GET", "/"+bucket, "lifecycle", "", creds)
	if err != nil && statusCode == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get bucket lifecycle: %w", err)
	}
	if statusCode == 404 {
		return nil, nil
	}
	if statusCode != 200 {
		return nil, fmt.Errorf("get bucket lifecycle failed (status %d): %s", statusCode, string(body))
	}

	var config lifecycleConfiguration
	if err := xml.Unmarshal(body, &config); err != nil {
		return nil, fmt.Errorf("parsing lifecycle response: %w", err)
	}

	rules := make([]LifecycleRule, 0, len(config.Rules))
	for _, r := range config.Rules {
		rule := LifecycleRule{
			ID:     r.ID,
			Status: r.Status,
			raw:    r.Inner,
		}
		switch {
		case r.Filter != nil && r.Filter.And != nil:
			if r.Filter.And.Prefix != nil {
				rule.Prefix = *r.Filter.And.Prefix
			}
			rule.Tags = sortedTags(r.Filter.And.Tags)
		case r.Filter != nil && r.Filter.Tag != nil:
			rule.Tags = sortedTags([]lifecycleTag{*r.Filter.Tag})
		case r.Filter != nil && r.Filter.Prefix != nil:
			rule.Prefix = *r.Filter.Prefix
		case r.Prefix != nil:
			rule.Prefix = *r.Prefix
		}
		if e := r.Expiration; e != nil {
			rule.ExpirationDays = e.Days
			rule.ExpirationDate = lifecycleDay(e.Date)
			rule.ExpiredObjectDeleteMarker = e.ExpiredObjectDeleteMarker
		}
		if e := r.NoncurrentVersionExpiration; e != nil {
			rule.NoncurrentExpirationDays = e.NoncurrentDays
		}
		if a := r.AbortIncompleteMultipartUpload; a != nil {
			rule.AbortMultipartDays = a.DaysAfterInitiation
		}
		if t := r.Transition; t != nil {
			if t.Days != nil {
				rule.TransitionDays = *t.Days
			}
			rule.TransitionDate = lifecycleDay(t.Date)
			rule.TransitionLocation = t.StorageClass
		}
		if t := r.NoncurrentVersionTransition; t != nil {
			rule.NoncurrentTransitionDays = t.NoncurrentDays
			rule.NoncurrentTransitionLocation = t.StorageClass
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

func (c *S3Client) PutBucketLifecycle(ctx context.Context, creds Credentials, bucket string, rules []LifecycleRule) error {
	config := lifecycleConfiguration{}

	for _, r := range rules {
		if r.raw != "" {
			config.Rules = append(config.Rules, lifecycleRule{Inner: r.raw})
			continue
		}
		config.Rules = append(config.Rules, encodeLifecycleRule(r))
	}

	xmlBody, err := xml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshaling lifecycle config: %w", err)
	}

	deadline := time.Now().Add(propagationTimeout)
	backoff := c.propagationBackoff()

	for {
		respBody, statusCode, err := c.doSignedRequest(ctx, "PUT", "/"+bucket, "lifecycle", string(xmlBody), creds)
		if err != nil && isLocationPropagationError(err) && time.Now().Before(deadline) {
			if err := sleepCtx(ctx, backoff); err != nil {
				return fmt.Errorf("put bucket lifecycle: %w", err)
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		if err != nil {
			return fmt.Errorf("put bucket lifecycle: %w", err)
		}
		if statusCode != 200 {
			return fmt.Errorf("put bucket lifecycle failed (status %d): %s", statusCode, string(respBody))
		}

		return nil
	}
}

// encodeLifecycleRule builds the XML form of a rule the provider created.
func encodeLifecycleRule(r LifecycleRule) lifecycleRule {
	x := lifecycleRule{
		ID:     r.ID,
		Status: r.Status,
		Filter: buildFilter(r.Prefix, r.Tags),
	}
	switch {
	case r.ExpirationDays > 0:
		x.Expiration = &lifecycleExpiration{Days: r.ExpirationDays}
	case r.ExpirationDate != "":
		x.Expiration = &lifecycleExpiration{Date: lifecycleDate(r.ExpirationDate)}
	case r.ExpiredObjectDeleteMarker:
		x.Expiration = &lifecycleExpiration{ExpiredObjectDeleteMarker: true}
	}
	if r.NoncurrentExpirationDays > 0 {
		x.NoncurrentVersionExpiration = &lifecycleNoncurrentExpiration{NoncurrentDays: r.NoncurrentExpirationDays}
	}
	if r.AbortMultipartDays > 0 {
		x.AbortIncompleteMultipartUpload = &lifecycleAbortMultipart{DaysAfterInitiation: r.AbortMultipartDays}
	}
	if r.TransitionLocation != "" {
		t := &lifecycleTransition{StorageClass: r.TransitionLocation}
		if r.TransitionDate != "" {
			t.Date = lifecycleDate(r.TransitionDate)
		} else {
			days := r.TransitionDays
			t.Days = &days
		}
		x.Transition = t
	}
	if r.NoncurrentTransitionLocation != "" {
		x.NoncurrentVersionTransition = &lifecycleNoncurrentTransition{
			NoncurrentDays: r.NoncurrentTransitionDays,
			StorageClass:   r.NoncurrentTransitionLocation,
		}
	}
	return x
}

func (c *S3Client) DeleteBucketLifecycle(ctx context.Context, creds Credentials, bucket string) error {
	body, statusCode, err := c.doSignedRequest(ctx, "DELETE", "/"+bucket, "lifecycle", "", creds)
	if err != nil && statusCode != 404 {
		return fmt.Errorf("delete bucket lifecycle: %w", err)
	}
	if statusCode != 204 && statusCode != 404 && statusCode != 200 {
		return fmt.Errorf("delete bucket lifecycle failed (status %d): %s", statusCode, string(body))
	}

	return nil
}
