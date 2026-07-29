package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func (c *ManagementClient) GetEndpoint(ctx context.Context, hostname string) (*Endpoint, error) {
	overlay, err := c.LookupInOverlay(ctx, func(o *ConfigOverlay) bool {
		for i := range o.Endpoints {
			if o.Endpoints[i].Hostname == hostname {
				return true
			}
		}
		return false
	})
	if err != nil {
		return nil, err
	}

	for i := range overlay.Endpoints {
		if overlay.Endpoints[i].Hostname == hostname {
			return &overlay.Endpoints[i], nil
		}
	}

	return nil, nil
}

func (c *ManagementClient) CreateEndpoint(ctx context.Context, ep *Endpoint) (*Endpoint, error) {
	c.overlayMu.Lock()
	defer c.overlayMu.Unlock()

	path := fmt.Sprintf("/config/%s/endpoint", url.PathEscape(c.InstanceID))
	body, status, err := c.doRequest(ctx, http.MethodPost, path, ep)
	if err != nil {
		return nil, err
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return nil, fmt.Errorf("create endpoint failed (status %d): %s", status, string(body))
	}

	var created Endpoint
	if err := json.Unmarshal(body, &created); err != nil {
		return nil, fmt.Errorf("parsing create endpoint response: %w", err)
	}

	return &created, nil
}

func (c *ManagementClient) DeleteEndpoint(ctx context.Context, hostname string) error {
	c.overlayMu.Lock()
	defer c.overlayMu.Unlock()

	path := fmt.Sprintf("/config/%s/endpoint/%s", url.PathEscape(c.InstanceID), url.PathEscape(hostname))
	body, status, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	if status == http.StatusNotFound {
		return nil
	}
	if status != http.StatusNoContent && status != http.StatusOK {
		return fmt.Errorf("delete endpoint failed (status %d): %s", status, string(body))
	}

	return nil
}
