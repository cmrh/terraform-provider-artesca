package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	managementAPIPath     = "/api/v1"
	managementHTTPTimeout = 180 * time.Second
	contentTypeJSON       = "application/json"
	maxResponseBytes      = 1 << 20 // 1 MB
)

type ManagementClient struct {
	BaseURL     string
	InstanceID  string
	TokenSource *OIDCTokenSource
	HTTPClient  *http.Client
	overlayMu   sync.Mutex
}

func NewManagementClient(baseURL, instanceID string, tokenSource *OIDCTokenSource, insecureSkipVerify bool) *ManagementClient {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecureSkipVerify, // #nosec G402 -- gated on the user-set insecure_skip_verify provider attribute
		},
	}

	return &ManagementClient{
		BaseURL:     strings.TrimRight(baseURL, "/"),
		InstanceID:  instanceID,
		TokenSource: tokenSource,
		HTTPClient: &http.Client{
			Transport: transport,
			Timeout:   managementHTTPTimeout,
		},
	}
}

func (c *ManagementClient) doRequest(ctx context.Context, method, path string, body any) ([]byte, int, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshaling request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	fullURL := fmt.Sprintf("%s%s%s", c.BaseURL, managementAPIPath, path)
	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}

	token, err := c.TokenSource.Token(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("getting auth token: %w", err)
	}

	req.Header.Set("X-Authentication-Token", token)
	req.Header.Set("Content-Type", contentTypeJSON)
	req.Header.Set("Accept", contentTypeJSON)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("reading response: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

func (c *ManagementClient) GetOverlay(ctx context.Context) (*ConfigOverlay, error) {
	path := fmt.Sprintf("/config/overlay/view/%s", url.PathEscape(c.InstanceID))
	body, status, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("get overlay failed (status %d): %s", status, string(body))
	}

	var overlay ConfigOverlay
	if err := json.Unmarshal(body, &overlay); err != nil {
		return nil, fmt.Errorf("parsing overlay response: %w", err)
	}

	return &overlay, nil
}

// overlayLookupDelays controls how the overlay is polled when the caller's
// finder returns false. The management API's overlay view is eventually
// consistent immediately after a Create/Update — a freshly-mutated entity
// may take a second or two to appear. Total budget ~3.5s; on the final miss
// the last overlay is returned so the caller can decide "genuinely gone."
var overlayLookupDelays = []time.Duration{
	0,
	500 * time.Millisecond,
	1 * time.Second,
	2 * time.Second,
}

// LookupInOverlay fetches the overlay and applies find(); if find returns
// false, retries with exponential backoff up to a small budget, then returns
// the last overlay it observed. Callers do their own extraction from the
// returned overlay — LookupInOverlay only decides whether to keep polling.
func (c *ManagementClient) LookupInOverlay(ctx context.Context, find func(*ConfigOverlay) bool) (*ConfigOverlay, error) {
	var lastOverlay *ConfigOverlay
	for _, delay := range overlayLookupDelays {
		if delay > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}
		overlay, err := c.GetOverlay(ctx)
		if err != nil {
			return nil, err
		}
		lastOverlay = overlay
		if find(overlay) {
			return overlay, nil
		}
	}
	return lastOverlay, nil
}
