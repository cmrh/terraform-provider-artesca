package client

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
)

// BucketEncryptionConfig captures the SSE configuration for a bucket.
// ARTESCA today supports SSE-S3 (SSEAlgorithm = "AES256"). BucketKeyEnabled is
// returned by the server even when not set on PUT.
type BucketEncryptionConfig struct {
	SSEAlgorithm     string
	BucketKeyEnabled bool
}

type bucketEncryptionRequest struct {
	XMLName xml.Name                      `xml:"ServerSideEncryptionConfiguration"`
	XMLNS   string                        `xml:"xmlns,attr"`
	Rules   []bucketEncryptionRequestRule `xml:"Rule"`
}

type bucketEncryptionRequestRule struct {
	Apply            bucketEncryptionRequestApply `xml:"ApplyServerSideEncryptionByDefault"`
	BucketKeyEnabled bool                         `xml:"BucketKeyEnabled"`
}

type bucketEncryptionRequestApply struct {
	SSEAlgorithm string `xml:"SSEAlgorithm"`
}

type bucketEncryptionResponse struct {
	XMLName xml.Name                       `xml:"ServerSideEncryptionConfiguration"`
	Rules   []bucketEncryptionResponseRule `xml:"Rule"`
}

type bucketEncryptionResponseRule struct {
	Apply            bucketEncryptionResponseApply `xml:"ApplyServerSideEncryptionByDefault"`
	BucketKeyEnabled bool                          `xml:"BucketKeyEnabled"`
}

type bucketEncryptionResponseApply struct {
	SSEAlgorithm string `xml:"SSEAlgorithm"`
}

// PutBucketEncryption sets the bucket's server-side encryption configuration.
// Replaces any existing configuration.
func (c *S3Client) PutBucketEncryption(ctx context.Context, accessKey, secretKey, bucket string, cfg BucketEncryptionConfig) error {
	req := bucketEncryptionRequest{
		XMLNS: "http://s3.amazonaws.com/doc/2006-03-01/",
		Rules: []bucketEncryptionRequestRule{{
			Apply:            bucketEncryptionRequestApply{SSEAlgorithm: cfg.SSEAlgorithm},
			BucketKeyEnabled: cfg.BucketKeyEnabled,
		}},
	}
	body, err := xml.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshaling bucket encryption body: %w", err)
	}
	_, status, err := c.doSignedRequest(ctx, http.MethodPut, "/"+bucket, "encryption", string(body), accessKey, secretKey)
	if err != nil {
		return err
	}
	if status != http.StatusOK && status != http.StatusNoContent {
		return fmt.Errorf("put bucket encryption failed (status %d)", status)
	}
	return nil
}

// GetBucketEncryption returns the bucket's encryption configuration. If no
// configuration is set, returns (nil, nil) so callers can treat absence as a
// state-removed signal.
func (c *S3Client) GetBucketEncryption(ctx context.Context, accessKey, secretKey, bucket string) (*BucketEncryptionConfig, error) {
	respBody, status, err := c.doSignedRequest(ctx, http.MethodGet, "/"+bucket, "encryption", "", accessKey, secretKey)
	if err != nil {
		if strings.Contains(err.Error(), "ServerSideEncryptionConfigurationNotFoundError") {
			return nil, nil
		}
		return nil, err
	}
	if status == http.StatusNotFound {
		return nil, nil
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("get bucket encryption failed (status %d)", status)
	}

	var resp bucketEncryptionResponse
	if err := xml.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("parsing bucket encryption response: %w", err)
	}
	if len(resp.Rules) == 0 {
		return nil, nil
	}
	rule := resp.Rules[0]
	return &BucketEncryptionConfig{
		SSEAlgorithm:     rule.Apply.SSEAlgorithm,
		BucketKeyEnabled: rule.BucketKeyEnabled,
	}, nil
}

// DeleteBucketEncryption removes any server-side encryption configuration from
// the bucket. Treats "not found" as success.
func (c *S3Client) DeleteBucketEncryption(ctx context.Context, accessKey, secretKey, bucket string) error {
	_, status, err := c.doSignedRequest(ctx, http.MethodDelete, "/"+bucket, "encryption", "", accessKey, secretKey)
	if status == http.StatusNotFound {
		return nil
	}
	if err != nil {
		if strings.Contains(err.Error(), "ServerSideEncryptionConfigurationNotFoundError") {
			return nil
		}
		return err
	}
	if status != http.StatusNoContent && status != http.StatusOK {
		return fmt.Errorf("delete bucket encryption failed (status %d)", status)
	}
	return nil
}
