package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func (c *ManagementClient) GetReplicationStream(ctx context.Context, streamID string) (*ReplicationStream, error) {
	overlay, err := c.GetOverlay(ctx)
	if err != nil {
		return nil, err
	}

	for _, rs := range overlay.ReplicationStreams {
		if rs.StreamID == streamID {
			return &rs, nil
		}
	}

	return nil, nil
}

func (c *ManagementClient) CreateReplicationStream(ctx context.Context, stream *ReplicationStream) (*ReplicationStream, error) {
	path := fmt.Sprintf("/config/%s/replication", url.PathEscape(c.InstanceID))
	body, status, err := c.doRequest(ctx, http.MethodPost, path, stream)
	if err != nil {
		return nil, err
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return nil, fmt.Errorf("create replication stream failed (status %d): %s", status, string(body))
	}

	var created ReplicationStream
	if err := json.Unmarshal(body, &created); err != nil {
		return nil, fmt.Errorf("parsing create replication stream response: %w", err)
	}

	return &created, nil
}

func (c *ManagementClient) UpdateReplicationStream(ctx context.Context, streamID string, stream *ReplicationStream) (*ReplicationStream, error) {
	path := fmt.Sprintf("/config/%s/replication/%s", url.PathEscape(c.InstanceID), url.PathEscape(streamID))
	body, status, err := c.doRequest(ctx, http.MethodPut, path, stream)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("update replication stream failed (status %d): %s", status, string(body))
	}

	var updated ReplicationStream
	if err := json.Unmarshal(body, &updated); err != nil {
		return nil, fmt.Errorf("parsing update replication stream response: %w", err)
	}

	return &updated, nil
}

func (c *ManagementClient) DeleteReplicationStream(ctx context.Context, streamID string) error {
	path := fmt.Sprintf("/config/%s/replication/%s", url.PathEscape(c.InstanceID), url.PathEscape(streamID))
	body, status, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	if status != http.StatusNoContent && status != http.StatusOK {
		return fmt.Errorf("delete replication stream failed (status %d): %s", status, string(body))
	}

	return nil
}
