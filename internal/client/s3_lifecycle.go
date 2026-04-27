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
}

type lifecycleFilter struct {
	Prefix string `xml:"Prefix"`
}

type lifecycleExpiration struct {
	Days int `xml:"Days,omitempty"`
}

type lifecycleTransition struct {
	Days         int    `xml:"Days,omitempty"`
	StorageClass string `xml:"StorageClass"`
}

type LifecycleRule struct {
	ID                 string
	Status             string
	Prefix             string
	ExpirationDays     int
	TransitionDays     int
	TransitionLocation string
}

func (c *S3Client) GetBucketLifecycle(ctx context.Context, accessKey, secretKey, bucket string) ([]LifecycleRule, error) {
	body, statusCode, err := c.doSignedRequest(ctx, "GET", "/"+bucket, "lifecycle", "", accessKey, secretKey)
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

func (c *S3Client) PutBucketLifecycle(ctx context.Context, accessKey, secretKey, bucket string, rules []LifecycleRule) error {
	config := lifecycleConfiguration{}

	for _, r := range rules {
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
		respBody, statusCode, err := c.doSignedRequest(ctx, "PUT", "/"+bucket, "lifecycle", string(xmlBody), accessKey, secretKey)
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

func (c *S3Client) DeleteBucketLifecycle(ctx context.Context, accessKey, secretKey, bucket string) error {
	body, statusCode, err := c.doSignedRequest(ctx, "DELETE", "/"+bucket, "lifecycle", "", accessKey, secretKey)
	if err != nil && statusCode != 404 {
		return fmt.Errorf("delete bucket lifecycle: %w", err)
	}
	if statusCode != 204 && statusCode != 404 && statusCode != 200 {
		return fmt.Errorf("delete bucket lifecycle failed (status %d): %s", statusCode, string(body))
	}

	return nil
}
