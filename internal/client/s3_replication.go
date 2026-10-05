package client

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"
)

// DefaultReplicationRole is the role ARTESCA sets on bucket replication
// configurations it creates; it is used when a bucket has no configuration yet.
const DefaultReplicationRole = "arn:aws:iam::root:role/s3-replication-role"

// ReplicationRule is one rule of a bucket's S3 replication configuration.
type ReplicationRule struct {
	ID                string
	Status            string // "Enabled" or "Disabled"
	Prefix            string
	DestinationBucket string // bucket name, not ARN
}

// BucketReplication is a bucket's S3 replication configuration.
type BucketReplication struct {
	Role  string
	Rules []ReplicationRule
}

type replicationConfiguration struct {
	XMLName xml.Name           `xml:"ReplicationConfiguration"`
	Xmlns   string             `xml:"xmlns,attr,omitempty"`
	Role    string             `xml:"Role"`
	Rules   []replicationRuleX `xml:"Rule"`
}

type replicationRuleX struct {
	ID          string `xml:"ID"`
	Prefix      string `xml:"Prefix"`
	Status      string `xml:"Status"`
	Destination struct {
		Bucket string `xml:"Bucket"`
	} `xml:"Destination"`
}

const s3BucketARNPrefix = "arn:aws:s3:::"

// GetBucketReplication returns the bucket's replication configuration, or nil
// if it has none.
func (c *S3Client) GetBucketReplication(ctx context.Context, creds Credentials, bucket string) (*BucketReplication, error) {
	body, statusCode, err := c.doSignedRequest(ctx, "GET", "/"+bucket, "replication", "", creds)
	if statusCode == 404 && (err == nil || strings.Contains(err.Error(), "ReplicationConfigurationNotFoundError")) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get bucket replication: %w", err)
	}
	if statusCode != 200 {
		return nil, fmt.Errorf("get bucket replication failed (status %d): %s", statusCode, string(body))
	}

	var config replicationConfiguration
	if err := xml.Unmarshal(body, &config); err != nil {
		return nil, fmt.Errorf("parsing replication response: %w", err)
	}

	out := &BucketReplication{Role: config.Role, Rules: make([]ReplicationRule, 0, len(config.Rules))}
	for _, r := range config.Rules {
		out.Rules = append(out.Rules, ReplicationRule{
			ID:                r.ID,
			Status:            r.Status,
			Prefix:            r.Prefix,
			DestinationBucket: strings.TrimPrefix(r.Destination.Bucket, s3BucketARNPrefix),
		})
	}
	return out, nil
}

// PutBucketReplication replaces the bucket's replication configuration.
func (c *S3Client) PutBucketReplication(ctx context.Context, creds Credentials, bucket string, cfg BucketReplication) error {
	config := replicationConfiguration{Xmlns: "http://s3.amazonaws.com/doc/2006-03-01/", Role: cfg.Role}
	for _, r := range cfg.Rules {
		x := replicationRuleX{ID: r.ID, Prefix: r.Prefix, Status: r.Status}
		x.Destination.Bucket = s3BucketARNPrefix + r.DestinationBucket
		config.Rules = append(config.Rules, x)
	}

	body, err := xml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshaling replication config: %w", err)
	}

	respBody, statusCode, err := c.doSignedRequest(ctx, "PUT", "/"+bucket, "replication", string(body), creds)
	if err != nil {
		return fmt.Errorf("put bucket replication: %w", err)
	}
	if statusCode != 200 {
		return fmt.Errorf("put bucket replication failed (status %d): %s", statusCode, string(respBody))
	}
	return nil
}

// DeleteBucketReplication removes the bucket's replication configuration.
func (c *S3Client) DeleteBucketReplication(ctx context.Context, creds Credentials, bucket string) error {
	body, statusCode, err := c.doSignedRequest(ctx, "DELETE", "/"+bucket, "replication", "", creds)
	if statusCode == 404 {
		return nil
	}
	if err != nil {
		return fmt.Errorf("delete bucket replication: %w", err)
	}
	if statusCode != 204 && statusCode != 200 {
		return fmt.Errorf("delete bucket replication failed (status %d): %s", statusCode, string(body))
	}
	return nil
}
