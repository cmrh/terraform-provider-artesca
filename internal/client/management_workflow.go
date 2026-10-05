package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

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
