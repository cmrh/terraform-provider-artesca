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
	"os"
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
// consistent after a Create — a freshly-mutated entity can take up to ~15s
// to appear. Total budget ~31s; on the final miss the last overlay is
// returned so the caller can decide "genuinely gone."
var overlayLookupDelays = []time.Duration{
	0,
	500 * time.Millisecond,
	1 * time.Second,
	2 * time.Second,
	4 * time.Second,
	8 * time.Second,
	16 * time.Second,
}

// LookupInOverlay fetches the overlay and applies find(); if find returns
// false, retries with exponential backoff up to a small budget, then returns
// the last overlay it observed. Callers do their own extraction from the
// returned overlay — LookupInOverlay only decides whether to keep polling.
//
// On final miss, a diagnostic summary of the overlay contents is emitted to
// stderr so acceptance tests can capture what the server actually returned
// (temporary; remove once the miss root cause is understood).
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
	c.dumpOverlayForDiagnosis(lastOverlay)
	return lastOverlay, nil
}

// dumpOverlayForDiagnosis writes a compact summary of every entity in the
// overlay to stderr. Called once from LookupInOverlay when the caller's
// finder never returned true — helps determine whether the overlay is truly
// empty of the target, or contains it under an unexpected shape.
func (c *ManagementClient) dumpOverlayForDiagnosis(o *ConfigOverlay) {
	if o == nil {
		fmt.Fprintf(os.Stderr, "[artesca-diag] LookupInOverlay miss: overlay was nil\n")
		return
	}
	fmt.Fprintf(os.Stderr, "[artesca-diag] LookupInOverlay miss: instanceID=%q updatedAt=%q version=%d users=%d locations=%d endpoints=%d replicationStreams=%d\n",
		o.InstanceID, o.UpdatedAt, o.Version,
		len(o.Users), len(o.Locations), len(o.Endpoints), len(o.ReplicationStreams))
	for i := range o.Users {
		u := &o.Users[i]
		fmt.Fprintf(os.Stderr, "[artesca-diag]   user[%d]: accountName=%q userName=%q id=%q arn=%q\n",
			i, u.AccountName, u.UserName, u.ID, u.ARN)
	}
	for k, l := range o.Locations {
		fmt.Fprintf(os.Stderr, "[artesca-diag]   location[%q]: name=%q type=%q\n", k, l.Name, l.LocationType)
	}
	for i := range o.Endpoints {
		e := &o.Endpoints[i]
		fmt.Fprintf(os.Stderr, "[artesca-diag]   endpoint[%d]: hostname=%q location=%q\n", i, e.Hostname, e.LocationName)
	}
	for i := range o.ReplicationStreams {
		rs := &o.ReplicationStreams[i]
		fmt.Fprintf(os.Stderr, "[artesca-diag]   replication[%d]: streamID=%q name=%q\n", i, rs.StreamID, rs.Name)
	}
}
