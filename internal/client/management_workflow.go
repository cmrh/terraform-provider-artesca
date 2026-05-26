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

// SearchedWorkflow is one entry returned by SearchWorkflows. Each entry has
// exactly one of Expiration/Replication/Transition populated.
type SearchedWorkflow struct {
	Expiration  *BucketWorkflowExpiration `json:"expiration,omitempty"`
	Replication *ReplicationStream        `json:"replication,omitempty"`
	Transition  *BucketWorkflowTransition `json:"transition,omitempty"`
}

type searchWorkflowsRequest struct {
	BucketList []string `json:"bucketList,omitempty"`
}

// SearchWorkflows returns the workflows configured under an account. Pass a
// non-empty bucketList to scope the search to specific buckets; pass nil/empty
// to return all workflows in the account.
func (c *ManagementClient) SearchWorkflows(ctx context.Context, instanceID, accountID string, bucketList []string) ([]SearchedWorkflow, error) {
	path := fmt.Sprintf("/instance/%s/account/%s/workflow/search", url.PathEscape(instanceID), url.PathEscape(accountID))
	req := searchWorkflowsRequest{BucketList: bucketList}

	body, status, err := c.doRequest(ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK && status != http.StatusPartialContent {
		return nil, fmt.Errorf("search workflows failed (status %d): %s", status, string(body))
	}

	var results []SearchedWorkflow
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("parsing search workflows response: %w", err)
	}
	return results, nil
}
