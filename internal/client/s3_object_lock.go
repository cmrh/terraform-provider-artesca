package client

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// ObjectLockRetention is a bucket's default retention. Exactly one of Days
// and Years is set.
type ObjectLockRetention struct {
	Mode  string // GOVERNANCE or COMPLIANCE
	Days  int64
	Years int64
}

type objectLockConfiguration struct {
	XMLName           xml.Name        `xml:"ObjectLockConfiguration"`
	Xmlns             string          `xml:"xmlns,attr,omitempty"`
	ObjectLockEnabled string          `xml:"ObjectLockEnabled"`
	Rule              *objectLockRule `xml:"Rule,omitempty"`
}

type objectLockRule struct {
	DefaultRetention struct {
		Mode  string `xml:"Mode"`
		Days  int64  `xml:"Days,omitempty"`
		Years int64  `xml:"Years,omitempty"`
	} `xml:"DefaultRetention"`
}

// IsObjectLockNotFound reports whether err means the bucket has no object
// lock configuration, i.e. was created without object lock.
func IsObjectLockNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "ObjectLockConfigurationNotFoundError:")
}

// GetObjectLockConfiguration reports whether object lock is enabled on the
// bucket and its default retention (nil when none is set).
func (c *S3Client) GetObjectLockConfiguration(ctx context.Context, creds Credentials, bucket string) (bool, *ObjectLockRetention, error) {
	body, status, err := c.doSignedRequest(ctx, http.MethodGet, "/"+bucket, "object-lock", "", creds)
	if IsObjectLockNotFound(err) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, fmt.Errorf("get object lock configuration: %w", err)
	}
	if status != http.StatusOK {
		return false, nil, fmt.Errorf("get object lock configuration failed (status %d): %s", status, string(body))
	}
	var cfg objectLockConfiguration
	if err := xml.Unmarshal(body, &cfg); err != nil {
		return false, nil, fmt.Errorf("parsing object lock configuration: %w", err)
	}
	if cfg.ObjectLockEnabled != "Enabled" {
		return false, nil, nil
	}
	if cfg.Rule == nil || cfg.Rule.DefaultRetention.Mode == "" {
		return true, nil, nil
	}
	r := cfg.Rule.DefaultRetention
	return true, &ObjectLockRetention{Mode: r.Mode, Days: r.Days, Years: r.Years}, nil
}

// PutObjectLockConfiguration sets the bucket's default retention; nil
// removes it. Object lock stays enabled either way.
func (c *S3Client) PutObjectLockConfiguration(ctx context.Context, creds Credentials, bucket string, retention *ObjectLockRetention) error {
	cfg := objectLockConfiguration{
		Xmlns:             "http://s3.amazonaws.com/doc/2006-03-01/",
		ObjectLockEnabled: "Enabled",
	}
	if retention != nil {
		cfg.Rule = &objectLockRule{}
		cfg.Rule.DefaultRetention.Mode = retention.Mode
		cfg.Rule.DefaultRetention.Days = retention.Days
		cfg.Rule.DefaultRetention.Years = retention.Years
	}
	body, err := xml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encoding object lock configuration: %w", err)
	}
	respBody, status, err := c.doSignedRequest(ctx, http.MethodPut, "/"+bucket, "object-lock", string(body), creds)
	if err != nil {
		return fmt.Errorf("put object lock configuration: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("put object lock configuration failed (status %d): %s", status, string(respBody))
	}
	return nil
}

type objectRetention struct {
	XMLName         xml.Name `xml:"Retention"`
	Mode            string   `xml:"Mode"`
	RetainUntilDate string   `xml:"RetainUntilDate"`
}

// GetObjectRetention returns an object version's retention mode and
// retain-until date, or empty strings when it has none.
func (c *S3Client) GetObjectRetention(ctx context.Context, creds Credentials, bucket, key, versionID string) (mode, retainUntil string, err error) {
	query := url.Values{"retention": {""}, "versionId": {versionID}}
	body, status, _, err := c.doSignedRequestQuery(ctx, http.MethodGet, "/"+bucket+"/"+key, query, "", creds)
	if err != nil && strings.Contains(err.Error(), "NoSuchObjectLockConfiguration:") {
		return "", "", nil
	}
	if err != nil {
		return "", "", fmt.Errorf("get object %q version %q retention: %w", key, versionID, err)
	}
	if status != http.StatusOK {
		return "", "", fmt.Errorf("get object %q version %q retention failed (status %d)", key, versionID, status)
	}
	var r objectRetention
	if err := xml.Unmarshal(body, &r); err != nil {
		return "", "", fmt.Errorf("parsing object retention: %w", err)
	}
	return r.Mode, r.RetainUntilDate, nil
}
