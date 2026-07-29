package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func (c *ManagementClient) GetLocation(ctx context.Context, name string) (*Location, error) {
	overlay, err := c.LookupInOverlay(ctx, func(o *ConfigOverlay) bool {
		_, ok := o.Locations[name]
		return ok
	})
	if err != nil {
		return nil, err
	}

	loc, ok := overlay.Locations[name]
	if !ok {
		return nil, nil
	}

	return &loc, nil
}

func (c *ManagementClient) CreateLocation(ctx context.Context, loc *Location) (*Location, error) {
	c.overlayMu.Lock()
	defer c.overlayMu.Unlock()

	path := fmt.Sprintf("/config/%s/location", url.PathEscape(c.InstanceID))
	body, status, err := c.doRequest(ctx, http.MethodPost, path, loc)
	if err != nil {
		return nil, err
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return nil, fmt.Errorf("create location failed (status %d): %s", status, string(body))
	}

	var created Location
	if err := json.Unmarshal(body, &created); err != nil {
		return nil, fmt.Errorf("parsing create location response: %w", err)
	}

	return &created, nil
}

func (c *ManagementClient) UpdateLocation(ctx context.Context, name string, loc *Location) (*Location, error) {
	c.overlayMu.Lock()
	defer c.overlayMu.Unlock()

	path := fmt.Sprintf("/config/%s/location/%s", url.PathEscape(c.InstanceID), url.PathEscape(name))
	body, status, err := c.doRequest(ctx, http.MethodPut, path, loc)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("update location failed (status %d): %s", status, string(body))
	}

	var updated Location
	if err := json.Unmarshal(body, &updated); err != nil {
		return nil, fmt.Errorf("parsing update location response: %w", err)
	}

	return &updated, nil
}

func (c *ManagementClient) DeleteLocation(ctx context.Context, name string) error {
	c.overlayMu.Lock()
	defer c.overlayMu.Unlock()

	path := fmt.Sprintf("/config/%s/location/%s", url.PathEscape(c.InstanceID), url.PathEscape(name))
	body, status, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	if status == http.StatusNotFound {
		return nil
	}
	if status != http.StatusNoContent && status != http.StatusOK {
		return fmt.Errorf("delete location failed (status %d): %s", status, string(body))
	}

	return nil
}
