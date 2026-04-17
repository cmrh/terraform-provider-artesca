package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

func (c *ManagementClient) GetEndpoint(ctx context.Context, hostname string) (*Endpoint, error) {
	overlay, err := c.GetOverlay(ctx)
	if err != nil {
		return nil, err
	}

	for _, ep := range overlay.Endpoints {
		if ep.Hostname == hostname {
			return &ep, nil
		}
	}

	return nil, nil
}

func (c *ManagementClient) CreateEndpoint(ctx context.Context, ep *Endpoint) (*Endpoint, error) {
	path := fmt.Sprintf("/config/%s/endpoint", c.InstanceID)
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
	path := fmt.Sprintf("/config/%s/endpoint/%s", c.InstanceID, hostname)
	body, status, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	if status != http.StatusNoContent && status != http.StatusOK {
		return fmt.Errorf("delete endpoint failed (status %d): %s", status, string(body))
	}

	return nil
}
