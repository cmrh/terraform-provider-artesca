package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Management API constants.
const (
	managementAPIPath     = "/api/v1"
	managementHTTPTimeout = 60 * time.Second
	contentTypeJSON       = "application/json"
)

// ManagementClient communicates with the ARTESCA management API.
type ManagementClient struct {
	BaseURL     string
	InstanceID  string
	TokenSource *OIDCTokenSource
	HTTPClient  *http.Client
}

func NewManagementClient(baseURL, instanceID string, tokenSource *OIDCTokenSource, insecureSkipVerify bool) *ManagementClient {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecureSkipVerify,
		},
	}

	return &ManagementClient{
		BaseURL:     strings.TrimRight(baseURL, "/"),
		InstanceID:  instanceID,
		TokenSource: tokenSource,
		HTTPClient: &http.Client{
			Transport: transport,
			Timeout:   managementHTTPTimeout,
		},
	}
}

func (c *ManagementClient) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, int, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshaling request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	fullURL := fmt.Sprintf("%s%s%s", c.BaseURL, managementAPIPath, path)
	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}

	token, err := c.TokenSource.Token(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("getting auth token: %w", err)
	}

	req.Header.Set("X-Authentication-Token", token)
	req.Header.Set("Content-Type", contentTypeJSON)
	req.Header.Set("Accept", contentTypeJSON)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("reading response: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

// --- Config Overlay ---

type ConfigOverlay struct {
	InstanceID         string              `json:"instanceId"`
	Locations          map[string]Location `json:"locations"`
	Users              []User              `json:"users"`
	Endpoints          []Endpoint          `json:"endpoints"`
	ReplicationStreams []ReplicationStream `json:"replicationStreams"`
	Version            int64               `json:"version"`
	UpdatedAt          string              `json:"updatedAt"`
}

type Location struct {
	Name              string           `json:"name"`
	LocationType      string           `json:"locationType"`
	ObjectID          string           `json:"objectId,omitempty"`
	IsBuiltin         bool             `json:"isBuiltin,omitempty"`
	IsTransient       bool             `json:"isTransient,omitempty"`
	IsCold            bool             `json:"isCold,omitempty"`
	LegacyAwsBehavior bool             `json:"legacyAwsBehavior,omitempty"`
	SizeLimitGB       int64            `json:"sizeLimitGB,omitempty"`
	Details           *LocationDetails `json:"details,omitempty"`
}

type LocationDetails struct {
	AccessKey            string   `json:"accessKey,omitempty"`
	SecretKey            string   `json:"secretKey,omitempty"`
	BucketName           string   `json:"bucketName,omitempty"`
	BucketMatch          *bool    `json:"bucketMatch,omitempty"`
	Endpoint             string   `json:"endpoint,omitempty"`
	Region               string   `json:"region,omitempty"`
	ServerSideEncryption *bool    `json:"serverSideEncryption,omitempty"`
	StorageClass         string   `json:"storageClass,omitempty"`
	MpuBucketName        string   `json:"mpuBucketName,omitempty"`
	Username             string   `json:"username,omitempty"`
	Password             string   `json:"password,omitempty"`
	TenantName           string   `json:"tenantName,omitempty"`
	SubscriptionID       string   `json:"subscriptionId,omitempty"`
	ResourceGroup        string   `json:"resourceGroup,omitempty"`
	StorageAccountName   string   `json:"storageAccountName,omitempty"`
	StorageContainerName string   `json:"storageContainerName,omitempty"`
	NsID                 string   `json:"nsId,omitempty"`
	RepoID               []string `json:"repoId,omitempty"`
	ProxyPath            string   `json:"proxyPath,omitempty"`
	BootstrapList        []string `json:"bootstrapList,omitempty"`
	ChordCos             *int64   `json:"chordCos,omitempty"`
	CodingParts          *int64   `json:"codingParts,omitempty"`
	DataParts            *int64   `json:"dataParts,omitempty"`
	GcpEndpoint          string   `json:"gcpEndpoint,omitempty"`
	BucketPrefix         string   `json:"bucketPrefix,omitempty"`
}

type User struct {
	AccountName string `json:"accountName"`
	AccessKey   string `json:"accessKey,omitempty"`
	SecretKey   string `json:"secretKey,omitempty"`
	ARN         string `json:"arn,omitempty"`
	CanonicalID string `json:"canonicalId,omitempty"`
	Email       string `json:"email,omitempty"`
	CreateDate  string `json:"createDate,omitempty"`
	UserName    string `json:"userName,omitempty"`
	ID          string `json:"id,omitempty"`
}

type Endpoint struct {
	Hostname     string `json:"hostname"`
	LocationName string `json:"locationName,omitempty"`
	IsBuiltin    bool   `json:"isBuiltin,omitempty"`
}

type ReplicationStream struct {
	StreamID    string             `json:"streamId,omitempty"`
	Name        string             `json:"name"`
	Version     int                `json:"version"`
	Enabled     bool               `json:"enabled"`
	Source      *ReplicationSource `json:"source,omitempty"`
	Destination *ReplicationDest   `json:"destination,omitempty"`
}

type ReplicationSource struct {
	BucketName string `json:"bucketName"`
	Prefix     string `json:"prefix"`
	Location   string `json:"location,omitempty"`
}

type ReplicationDest struct {
	BucketName            string                    `json:"bucketName,omitempty"`
	Location              string                    `json:"location,omitempty"`
	Locations             []ReplicationDestLocation `json:"locations,omitempty"`
	PreferredReadLocation string                    `json:"preferredReadLocation,omitempty"`
	Role                  string                    `json:"role,omitempty"`
}

type ReplicationDestLocation struct {
	Name         string `json:"name"`
	StorageClass string `json:"storageClass,omitempty"`
}

// --- Bucket Workflow Types ---

type WorkflowFilter struct {
	ObjectKeyPrefix string        `json:"objectKeyPrefix,omitempty"`
	ObjectTags      []WorkflowTag `json:"objectTags,omitempty"`
}

type WorkflowTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type BucketWorkflowExpiration struct {
	WorkflowID                                string          `json:"workflowId,omitempty"`
	Name                                      string          `json:"name,omitempty"`
	Enabled                                   bool            `json:"enabled"`
	BucketName                                string          `json:"bucketName"`
	Type                                      string          `json:"type"`
	Filter                                    *WorkflowFilter `json:"filter,omitempty"`
	CurrentVersionTriggerDelayDate            string          `json:"currentVersionTriggerDelayDate,omitempty"`
	CurrentVersionTriggerDelayDays            *int            `json:"currentVersionTriggerDelayDays,omitempty"`
	ExpireDeleteMarkersTrigger                *bool           `json:"expireDeleteMarkersTrigger,omitempty"`
	IncompleteMultipartUploadTriggerDelayDays *int            `json:"incompleteMultipartUploadTriggerDelayDays,omitempty"`
	PreviousVersionTriggerDelayDays           *int            `json:"previousVersionTriggerDelayDays,omitempty"`
}

type BucketWorkflowTransition struct {
	WorkflowID       string          `json:"workflowId,omitempty"`
	Name             string          `json:"name,omitempty"`
	Enabled          bool            `json:"enabled"`
	BucketName       string          `json:"bucketName"`
	Type             string          `json:"type"`
	Filter           *WorkflowFilter `json:"filter,omitempty"`
	LocationName     string          `json:"locationName"`
	ApplyToVersion   string          `json:"applyToVersion"`
	TriggerDelayDate string          `json:"triggerDelayDate,omitempty"`
	TriggerDelayDays *int            `json:"triggerDelayDays,omitempty"`
}

// --- Overlay Operations ---

func (c *ManagementClient) GetOverlay(ctx context.Context) (*ConfigOverlay, error) {
	path := fmt.Sprintf("/config/overlay/view/%s", c.InstanceID)
	body, status, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("get overlay failed (status %d): %s", status, string(body))
	}

	var overlay ConfigOverlay
	if err := json.Unmarshal(body, &overlay); err != nil {
		return nil, fmt.Errorf("parsing overlay response: %w", err)
	}

	return &overlay, nil
}

// --- Location Operations ---

func (c *ManagementClient) GetLocation(ctx context.Context, name string) (*Location, error) {
	overlay, err := c.GetOverlay(ctx)
	if err != nil {
		return nil, err
	}

	loc, ok := overlay.Locations[name]
	if !ok {
		return nil, nil // not found
	}

	return &loc, nil
}

func (c *ManagementClient) CreateLocation(ctx context.Context, loc *Location) (*Location, error) {
	path := fmt.Sprintf("/config/%s/location", c.InstanceID)
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
	path := fmt.Sprintf("/config/%s/location/%s", c.InstanceID, name)
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
	path := fmt.Sprintf("/config/%s/location/%s", c.InstanceID, name)
	body, status, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	if status != http.StatusNoContent && status != http.StatusOK {
		return fmt.Errorf("delete location failed (status %d): %s", status, string(body))
	}

	return nil
}

// --- Account/User Operations ---

func (c *ManagementClient) GetAccount(ctx context.Context, name string) (*User, error) {
	overlay, err := c.GetOverlay(ctx)
	if err != nil {
		return nil, err
	}

	for _, user := range overlay.Users {
		if user.AccountName == name || user.UserName == name {
			return &user, nil
		}
	}

	return nil, nil // not found
}

type createUserRequest struct {
	UserName string `json:"userName"`
	Email    string `json:"email,omitempty"`
}

func (c *ManagementClient) CreateAccount(ctx context.Context, userName, email string) (*User, error) {
	path := fmt.Sprintf("/config/%s/user", c.InstanceID)
	reqBody := createUserRequest{
		UserName: userName,
		Email:    email,
	}
	body, status, err := c.doRequest(ctx, http.MethodPost, path, reqBody)
	if err != nil {
		return nil, err
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return nil, fmt.Errorf("create account failed (status %d): %s", status, string(body))
	}

	var user User
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("parsing create account response: %w", err)
	}

	return &user, nil
}

func (c *ManagementClient) DeleteAccount(ctx context.Context, accountName string) error {
	path := fmt.Sprintf("/config/%s/user?accountName=%s", c.InstanceID, accountName)
	body, status, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	if status != http.StatusNoContent && status != http.StatusOK {
		return fmt.Errorf("delete account failed (status %d): %s", status, string(body))
	}

	return nil
}

func (c *ManagementClient) GenerateAccountKey(ctx context.Context, accountName string) (*User, error) {
	path := fmt.Sprintf("/config/%s/user/%s/key", c.InstanceID, accountName)
	body, status, err := c.doRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return nil, fmt.Errorf("generate account key failed (status %d): %s", status, string(body))
	}

	var user User
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("parsing generate key response: %w", err)
	}

	return &user, nil
}

// --- Endpoint Operations ---

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

// --- Replication Stream Operations ---

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
	path := fmt.Sprintf("/config/%s/replication", c.InstanceID)
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
	path := fmt.Sprintf("/config/%s/replication/%s", c.InstanceID, streamID)
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
	path := fmt.Sprintf("/config/%s/replication/%s", c.InstanceID, streamID)
	body, status, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	if status != http.StatusNoContent && status != http.StatusOK {
		return fmt.Errorf("delete replication stream failed (status %d): %s", status, string(body))
	}

	return nil
}

// --- Bucket Workflow Expiration Operations ---

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

// --- Bucket Workflow Transition Operations ---

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
