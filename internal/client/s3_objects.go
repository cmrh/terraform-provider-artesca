package client

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// ObjectVersion is one object version or delete marker in a bucket.
type ObjectVersion struct {
	Key       string
	VersionID string
}

type listVersionsResult struct {
	XMLName             xml.Name `xml:"ListVersionsResult"`
	IsTruncated         bool     `xml:"IsTruncated"`
	NextKeyMarker       string   `xml:"NextKeyMarker"`
	NextVersionIDMarker string   `xml:"NextVersionIdMarker"`
	Versions            []struct {
		Key       string `xml:"Key"`
		VersionID string `xml:"VersionId"`
	} `xml:"Version"`
	DeleteMarkers []struct {
		Key       string `xml:"Key"`
		VersionID string `xml:"VersionId"`
	} `xml:"DeleteMarker"`
}

// ListObjectVersions returns every object version and delete marker in the
// bucket, following pagination.
func (c *S3Client) ListObjectVersions(ctx context.Context, creds Credentials, bucket string) ([]ObjectVersion, error) {
	var out []ObjectVersion
	query := url.Values{"versions": {""}}
	if c.listPageSizeOverride > 0 {
		query.Set("max-keys", fmt.Sprint(c.listPageSizeOverride))
	}
	seen := map[string]bool{}
	for {
		body, status, _, err := c.doSignedRequestQuery(ctx, http.MethodGet, "/"+bucket, query, "", creds)
		if err != nil {
			return nil, fmt.Errorf("list object versions: %w", err)
		}
		if status != http.StatusOK {
			return nil, fmt.Errorf("list object versions failed (status %d)", status)
		}
		var page listVersionsResult
		if err := xml.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("parsing list object versions response: %w", err)
		}
		for _, v := range page.Versions {
			out = append(out, ObjectVersion{Key: v.Key, VersionID: v.VersionID})
		}
		for _, m := range page.DeleteMarkers {
			out = append(out, ObjectVersion{Key: m.Key, VersionID: m.VersionID})
		}
		if !page.IsTruncated {
			return out, nil
		}
		marker := page.NextKeyMarker + "\x00" + page.NextVersionIDMarker
		if page.NextKeyMarker == "" || seen[marker] {
			return nil, fmt.Errorf("list object versions: truncated response without a new marker")
		}
		seen[marker] = true
		query.Set("key-marker", page.NextKeyMarker)
		query.Set("version-id-marker", page.NextVersionIDMarker)
	}
}

// DeleteObjectVersion permanently deletes one object version or delete
// marker. A version that is already gone counts as deleted. bypassGovernance
// also deletes a version under GOVERNANCE retention.
func (c *S3Client) DeleteObjectVersion(ctx context.Context, creds Credentials, bucket, key, versionID string, bypassGovernance bool) error {
	var headers map[string]string
	if bypassGovernance {
		headers = map[string]string{"x-amz-bypass-governance-retention": "true"}
	}
	_, status, _, err := c.doSignedRequestHeaders(ctx, http.MethodDelete, "/"+bucket+"/"+key, url.Values{"versionId": {versionID}}, "", headers, creds)
	if err != nil && !IsNoSuchKey(err) {
		return fmt.Errorf("delete object %q version %q: %w", key, versionID, err)
	}
	if err == nil && status != http.StatusNoContent && status != http.StatusOK {
		return fmt.Errorf("delete object %q version %q failed (status %d)", key, versionID, status)
	}
	return nil
}

// EmptyBucket deletes every object version and delete marker in the bucket.
// On an object-lock bucket, objectLock bypasses GOVERNANCE retention; a
// version still under COMPLIANCE retention stops it with an error naming the
// version and its retain-until date.
func (c *S3Client) EmptyBucket(ctx context.Context, creds Credentials, bucket string, objectLock bool) error {
	versions, err := c.ListObjectVersions(ctx, creds, bucket)
	if err != nil {
		return err
	}
	for _, v := range versions {
		err := c.DeleteObjectVersion(ctx, creds, bucket, v.Key, v.VersionID, objectLock)
		if err == nil {
			continue
		}
		if objectLock && strings.Contains(err.Error(), "AccessDenied:") {
			if mode, until, rerr := c.GetObjectRetention(ctx, creds, bucket, v.Key, v.VersionID); rerr == nil && mode == "COMPLIANCE" {
				return fmt.Errorf("object %q version %q is under COMPLIANCE retention until %s and can't be deleted before then", v.Key, v.VersionID, until)
			}
		}
		return err
	}
	return nil
}

// PutObject uploads body as key. Acceptance tests use it to fill buckets.
func (c *S3Client) PutObject(ctx context.Context, creds Credentials, bucket, key, body string) error {
	_, status, _, err := c.doSignedRequestQuery(ctx, http.MethodPut, "/"+bucket+"/"+key, nil, body, creds)
	if err != nil {
		return fmt.Errorf("put object %q: %w", key, err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("put object %q failed (status %d)", key, status)
	}
	return nil
}

// DeleteObject deletes key; in a versioned bucket this adds a delete marker.
func (c *S3Client) DeleteObject(ctx context.Context, creds Credentials, bucket, key string) error {
	_, status, _, err := c.doSignedRequestQuery(ctx, http.MethodDelete, "/"+bucket+"/"+key, nil, "", creds)
	if err != nil {
		return fmt.Errorf("delete object %q: %w", key, err)
	}
	if status != http.StatusNoContent && status != http.StatusOK {
		return fmt.Errorf("delete object %q failed (status %d)", key, status)
	}
	return nil
}
