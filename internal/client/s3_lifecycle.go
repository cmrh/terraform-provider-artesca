package client

import (
	"context"
	"encoding/xml"
	"fmt"
	"time"
)

type lifecycleConfiguration struct {
	XMLName xml.Name        `xml:"LifecycleConfiguration"`
	Rules   []lifecycleRule `xml:"Rule"`
}

type lifecycleRule struct {
	ID         string               `xml:"ID"`
	Status     string               `xml:"Status"`
	Filter     *lifecycleFilter     `xml:"Filter,omitempty"`
	Expiration *lifecycleExpiration `xml:"Expiration,omitempty"`
	Transition *lifecycleTransition `xml:"Transition,omitempty"`
	Inner      string               `xml:",innerxml"` // set on read only
}

// MarshalXML writes a rule read from the server back unchanged, keeping
// elements the provider doesn't model (tag filters, dates, noncurrent-version
// actions, ...). Rules built by the provider are encoded from their fields.
func (r lifecycleRule) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if r.Inner != "" {
		return e.EncodeElement(rawXML{Inner: r.Inner}, start)
	}
	type plain lifecycleRule
	return e.EncodeElement(plain(r), start)
}

type lifecycleFilter struct {
	Prefix string `xml:"Prefix"`
}

type lifecycleExpiration struct {
	Days int `xml:"Days,omitempty"`
}

type lifecycleTransition struct {
	Days         int    `xml:"Days"` // 0 is valid; ARTESCA rejects a Transition without Days
	StorageClass string `xml:"StorageClass"`
}

// LifecycleRule is one rule of a bucket's lifecycle configuration. Rules
// returned by GetBucketLifecycle are written back by PutBucketLifecycle exactly
// as read; to change a rule, replace it with a new LifecycleRule.
type LifecycleRule struct {
	ID                 string
	Status             string
	Prefix             string
	ExpirationDays     int
	TransitionDays     int
	TransitionLocation string

	raw string // inner XML as read from the server
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
		if r.Filter != nil {
			rule.Prefix = r.Filter.Prefix
		}
		if r.Expiration != nil {
			rule.ExpirationDays = r.Expiration.Days
		}
		if r.Transition != nil {
			rule.TransitionDays = r.Transition.Days
			rule.TransitionLocation = r.Transition.StorageClass
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
		xmlRule := lifecycleRule{
			ID:     r.ID,
			Status: r.Status,
			Filter: &lifecycleFilter{Prefix: r.Prefix},
		}
		if r.ExpirationDays > 0 {
			xmlRule.Expiration = &lifecycleExpiration{
				Days: r.ExpirationDays,
			}
		}
		if r.TransitionDays > 0 || r.TransitionLocation != "" {
			xmlRule.Transition = &lifecycleTransition{
				Days:         r.TransitionDays,
				StorageClass: r.TransitionLocation,
			}
		}
		config.Rules = append(config.Rules, xmlRule)
	}

	xmlBody, err := xml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshaling lifecycle config: %w", err)
	}

	deadline := time.Now().Add(propagationTimeout)
	backoff := 5 * time.Second

	for {
		respBody, statusCode, err := c.doSignedRequest(ctx, "PUT", "/"+bucket, "lifecycle", string(xmlBody), creds)
		if err != nil && isLocationPropagationError(err) && time.Now().Before(deadline) {
			time.Sleep(backoff)
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
