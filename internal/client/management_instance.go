package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// Instance is the response from GET /instance/{uuid}. Only the fields the
// server actually returns on ARTESCA today are modeled here; the swagger
// advertises additional fields (friendlyName, organizationID, etc.) that the
// server does not populate.
type Instance struct {
	InstanceID string `json:"instanceId"`
	CreatedAt  string `json:"createdAt,omitempty"`
	State      string `json:"state,omitempty"`
	PublicKey  string `json:"publicKey,omitempty"`
}

// InstanceStatus is the response from GET /instance/{uuid}/status. The full
// response also includes a `metrics` object and nested `capabilities` /
// `latestConfigurationOverlay` blocks that we do not surface today.
type InstanceStatus struct {
	IPAddress                   string `json:"ipAddress,omitempty"`
	LastSeen                    string `json:"lastSeen,omitempty"`
	RunningConfigurationVersion int64  `json:"runningConfigurationVersion,omitempty"`
	ServerVersion               string `json:"serverVersion,omitempty"`
}

type instanceStatusResponse struct {
	State InstanceStatus `json:"state"`
}

// GetInstance returns instance metadata (id, state, public key, creation time).
func (c *ManagementClient) GetInstance(ctx context.Context, instanceID string) (*Instance, error) {
	path := fmt.Sprintf("/instance/%s", url.PathEscape(instanceID))
	body, status, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("get instance failed (status %d): %s", status, string(body))
	}

	var inst Instance
	if err := json.Unmarshal(body, &inst); err != nil {
		return nil, fmt.Errorf("parsing instance response: %w", err)
	}
	return &inst, nil
}

// GetInstanceStatus returns the latest instance status snapshot. The
// response's top-level `metrics` and the `state.capabilities` /
// `state.latestConfigurationOverlay` blocks are intentionally ignored.
func (c *ManagementClient) GetInstanceStatus(ctx context.Context, instanceID string) (*InstanceStatus, error) {
	path := fmt.Sprintf("/instance/%s/status", url.PathEscape(instanceID))
	body, status, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("get instance status failed (status %d): %s", status, string(body))
	}

	var resp instanceStatusResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing instance status response: %w", err)
	}
	return &resp.State, nil
}
