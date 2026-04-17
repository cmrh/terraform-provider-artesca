package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// --- Expiration ---

func (c *ManagementClient) CreateBucketWorkflowExpiration(ctx context.Context, instanceID, accountID, bucketName string, wf *BucketWorkflowExpiration) (*BucketWorkflowExpiration, error) {
	path := fmt.Sprintf("/instance/%s/account/%s/bucket/%s/workflow/expiration", instanceID, accountID, bucketName)
	body, status, err := c.doRequest(ctx, http.MethodPost, path, wf)
	if err != nil {
		return nil, err
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return nil, fmt.Errorf("create expiration workflow failed (status %d): %s", status, string(body))
	}

	var created BucketWorkflowExpiration
	if err := json.Unmarshal(body, &created); err != nil {
		return nil, fmt.Errorf("parsing create expiration workflow response: %w", err)
	}

	return &created, nil
}

func (c *ManagementClient) UpdateBucketWorkflowExpiration(ctx context.Context, instanceID, accountID, bucketName, workflowID string, wf *BucketWorkflowExpiration) (*BucketWorkflowExpiration, error) {
	path := fmt.Sprintf("/instance/%s/account/%s/bucket/%s/workflow/expiration/%s", instanceID, accountID, bucketName, workflowID)
	body, status, err := c.doRequest(ctx, http.MethodPut, path, wf)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("update expiration workflow failed (status %d): %s", status, string(body))
	}

	var updated BucketWorkflowExpiration
	if err := json.Unmarshal(body, &updated); err != nil {
		return nil, fmt.Errorf("parsing update expiration workflow response: %w", err)
	}

	return &updated, nil
}

func (c *ManagementClient) DeleteBucketWorkflowExpiration(ctx context.Context, instanceID, accountID, bucketName, workflowID string) error {
	path := fmt.Sprintf("/instance/%s/account/%s/bucket/%s/workflow/expiration/%s", instanceID, accountID, bucketName, workflowID)
	body, status, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	if status != http.StatusNoContent && status != http.StatusOK {
		return fmt.Errorf("delete expiration workflow failed (status %d): %s", status, string(body))
	}

	return nil
}

// --- Transition ---

func (c *ManagementClient) CreateBucketWorkflowTransition(ctx context.Context, instanceID, accountID, bucketName string, wf *BucketWorkflowTransition) (*BucketWorkflowTransition, error) {
	path := fmt.Sprintf("/instance/%s/account/%s/bucket/%s/workflow/transition", instanceID, accountID, bucketName)
	body, status, err := c.doRequest(ctx, http.MethodPost, path, wf)
	if err != nil {
		return nil, err
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return nil, fmt.Errorf("create transition workflow failed (status %d): %s", status, string(body))
	}

	var created BucketWorkflowTransition
	if err := json.Unmarshal(body, &created); err != nil {
		return nil, fmt.Errorf("parsing create transition workflow response: %w", err)
	}

	return &created, nil
}

func (c *ManagementClient) UpdateBucketWorkflowTransition(ctx context.Context, instanceID, accountID, bucketName, workflowID string, wf *BucketWorkflowTransition) (*BucketWorkflowTransition, error) {
	path := fmt.Sprintf("/instance/%s/account/%s/bucket/%s/workflow/transition/%s", instanceID, accountID, bucketName, workflowID)
	body, status, err := c.doRequest(ctx, http.MethodPut, path, wf)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("update transition workflow failed (status %d): %s", status, string(body))
	}

	var updated BucketWorkflowTransition
	if err := json.Unmarshal(body, &updated); err != nil {
		return nil, fmt.Errorf("parsing update transition workflow response: %w", err)
	}

	return &updated, nil
}

func (c *ManagementClient) DeleteBucketWorkflowTransition(ctx context.Context, instanceID, accountID, bucketName, workflowID string) error {
	path := fmt.Sprintf("/instance/%s/account/%s/bucket/%s/workflow/transition/%s", instanceID, accountID, bucketName, workflowID)
	body, status, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	if status != http.StatusNoContent && status != http.StatusOK {
		return fmt.Errorf("delete transition workflow failed (status %d): %s", status, string(body))
	}

	return nil
}

// --- Replication ---

func (c *ManagementClient) CreateBucketWorkflowReplication(ctx context.Context, instanceID, accountID, bucketName string, stream *ReplicationStream) (*ReplicationStream, error) {
	path := fmt.Sprintf("/instance/%s/account/%s/bucket/%s/workflow/replication", instanceID, accountID, bucketName)
	body, status, err := c.doRequest(ctx, http.MethodPost, path, stream)
	if err != nil {
		return nil, err
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return nil, fmt.Errorf("create bucket workflow replication failed (status %d): %s", status, string(body))
	}

	var created ReplicationStream
	if err := json.Unmarshal(body, &created); err != nil {
		return nil, fmt.Errorf("parsing create bucket workflow replication response: %w", err)
	}

	return &created, nil
}

func (c *ManagementClient) UpdateBucketWorkflowReplication(ctx context.Context, instanceID, accountID, bucketName, workflowID string, stream *ReplicationStream) (*ReplicationStream, error) {
	path := fmt.Sprintf("/instance/%s/account/%s/bucket/%s/workflow/replication/%s", instanceID, accountID, bucketName, workflowID)
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
	path := fmt.Sprintf("/instance/%s/account/%s/bucket/%s/workflow/replication/%s", instanceID, accountID, bucketName, workflowID)
	body, status, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	if status != http.StatusNoContent && status != http.StatusOK {
		return fmt.Errorf("delete bucket workflow replication failed (status %d): %s", status, string(body))
	}

	return nil
}
