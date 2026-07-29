package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (c *ManagementClient) GetReplicationStream(ctx context.Context, streamID string) (*ReplicationStream, error) {
	overlay, err := c.LookupInOverlay(ctx, func(o *ConfigOverlay) bool {
		for i := range o.ReplicationStreams {
			if o.ReplicationStreams[i].StreamID == streamID {
				return true
			}
		}
		return false
	})
	if err != nil {
		return nil, err
	}

	for i := range overlay.ReplicationStreams {
		if overlay.ReplicationStreams[i].StreamID == streamID {
			return &overlay.ReplicationStreams[i], nil
		}
	}

	return nil, nil
}

func (c *ManagementClient) findReplicationStreamByName(ctx context.Context, name string) (*ReplicationStream, error) {
	overlay, err := c.GetOverlay(ctx)
	if err != nil {
		return nil, err
	}
	for _, rs := range overlay.ReplicationStreams {
		if rs.Name == name {
			return &rs, nil
		}
	}
	return nil, nil
}

func (c *ManagementClient) CreateReplicationStream(ctx context.Context, stream *ReplicationStream) (*ReplicationStream, error) {
	c.overlayMu.Lock()
	defer c.overlayMu.Unlock()

	apiPath := fmt.Sprintf("/config/%s/replication", url.PathEscape(c.InstanceID))

	deadline := time.Now().Add(propagationTimeout)
	backoff := 5 * time.Second

	for {
		body, status, err := c.doRequest(ctx, http.MethodPost, apiPath, stream)
		if err != nil {
			return nil, err
		}
		if status == http.StatusCreated || status == http.StatusOK {
			var created ReplicationStream
			if err := json.Unmarshal(body, &created); err != nil {
				return nil, fmt.Errorf("parsing create replication stream response: %w", err)
			}
			return &created, nil
		}

		// The management API may create the stream but return 400 due to
		// propagation delay (e.g. "could not find source bucket"). Check
		// whether the stream was actually persisted despite the error.
		if status == http.StatusBadRequest {
			msg := string(body)
			isPropagation := strings.Contains(msg, "could not find source bucket") ||
				strings.Contains(msg, "could not find destination bucket")
			isOverlap := strings.Contains(msg, "overlapping prefix")

			if isPropagation || isOverlap {
				existing, lookupErr := c.findReplicationStreamByName(ctx, stream.Name)
				if lookupErr == nil && existing != nil {
					return existing, nil
				}
			}
			if isPropagation && time.Now().Before(deadline) {
				time.Sleep(backoff)
				if backoff < 30*time.Second {
					backoff *= 2
				}
				continue
			}
		}

		return nil, fmt.Errorf("create replication stream failed (status %d): %s", status, string(body))
	}
}

func (c *ManagementClient) UpdateReplicationStream(ctx context.Context, streamID string, stream *ReplicationStream) (*ReplicationStream, error) {
	c.overlayMu.Lock()
	defer c.overlayMu.Unlock()

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
	c.overlayMu.Lock()
	defer c.overlayMu.Unlock()

	path := fmt.Sprintf("/config/%s/replication/%s", url.PathEscape(c.InstanceID), url.PathEscape(streamID))
	body, status, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	if status == http.StatusNotFound {
		return nil
	}
	if status != http.StatusNoContent && status != http.StatusOK {
		return fmt.Errorf("delete replication stream failed (status %d): %s", status, string(body))
	}

	return nil
}
