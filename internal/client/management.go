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
	"time"
)

const (
	managementAPIPath     = "/api/v1"
	managementHTTPTimeout = 60 * time.Second
	contentTypeJSON       = "application/json"
)

type ManagementClient struct {
	BaseURL     string
	InstanceID  string
	TokenSource *OIDCTokenSource
	HTTPClient  *http.Client
}

func NewManagementClient(baseURL, instanceID string, tokenSource *OIDCTokenSource, insecureSkipVerify bool) *ManagementClient {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecureSkipVerify,
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
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
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
