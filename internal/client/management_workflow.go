package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func (c *ManagementClient) CreateBucketWorkflowReplication(ctx context.Context, instanceID, accountID, bucketName string, stream *ReplicationStream) (*ReplicationStream, error) {
	path := fmt.Sprintf("/instance/%s/account/%s/bucket/%s/workflow/replication", url.PathEscape(instanceID), url.PathEscape(accountID), url.PathEscape(bucketName))

	body, status, err := c.doRequest(ctx, http.MethodPost, path, stream)

	if err != nil {
		return nil, err
	}
	if status != http.StatusCreated && status != http.StatusOK {
		if status == http.StatusBadRequest && len(bytes.TrimSpace(body)) == 0 {
			return nil, fmt.Errorf("create bucket workflow replication failed (status 400): bucket %q may not have versioning enabled, which is required for replication workflows", bucketName)
		}
		return nil, fmt.Errorf("create bucket workflow replication failed (status %d): %s", status, string(body))
	}

	var created ReplicationStream
	if err := json.Unmarshal(body, &created); err != nil {
		return nil, fmt.Errorf("parsing create bucket workflow replication response: %w", err)
	}

	return &created, nil
}

func (c *ManagementClient) UpdateBucketWorkflowReplication(ctx context.Context, instanceID, accountID, bucketName, workflowID string, stream *ReplicationStream) (*ReplicationStream, error) {
	path := fmt.Sprintf("/instance/%s/account/%s/bucket/%s/workflow/replication/%s", url.PathEscape(instanceID), url.PathEscape(accountID), url.PathEscape(bucketName), url.PathEscape(workflowID))
	body, status, err := c.doRequest(ctx, http.MethodPut, path, stream)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("update bucket workflow replication failed (status %d): %s", status, string(body))
	}

	var updated ReplicationStream
	if err := json.Unmarshal(body, &updated); err != nil {
		return nil, fmt.Errorf("parsing update bucket workflow replication response: %w", err)
	}

	return &updated, nil
}

func (c *ManagementClient) DeleteBucketWorkflowReplication(ctx context.Context, instanceID, accountID, bucketName, workflowID string) error {
	path := fmt.Sprintf("/instance/%s/account/%s/bucket/%s/workflow/replication/%s", url.PathEscape(instanceID), url.PathEscape(accountID), url.PathEscape(bucketName), url.PathEscape(workflowID))
	body, status, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	if status == http.StatusNotFound {
		return nil
	}
	if status != http.StatusNoContent && status != http.StatusOK {
		return fmt.Errorf("delete bucket workflow replication failed (status %d): %s", status, string(body))
	}

	return nil
}
