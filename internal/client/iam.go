package client

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// IAM / AWS SigV4 constants.
const (
	iamAPIVersion   = "2010-05-08"
	iamAWSService   = "iam"
	iamHTTPTimeout  = 30 * time.Second
	contentTypeForm = "application/x-www-form-urlencoded"
)

// IAMClient communicates with the ARTESCA IAM (Vault) API using AWS SigV4 signing.
type IAMClient struct {
	endpoint   string
	region     string
	httpClient *http.Client
}

func NewIAMClient(endpoint, region string, insecureSkipVerify bool) *IAMClient {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecureSkipVerify, // #nosec G402 -- gated on the user-set insecure_skip_verify provider attribute
		},
	}

	return &IAMClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		region:   region,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   iamHTTPTimeout,
		},
	}
}

// DeriveIAMEndpoint derives the IAM endpoint from the management endpoint
// by replacing "management" with "iam" in the hostname.
func DeriveIAMEndpoint(managementEndpoint string) (string, error) {
	u, err := url.Parse(managementEndpoint)
	if err != nil {
		return "", fmt.Errorf("parsing management endpoint: %w", err)
	}

	host := u.Hostname()
	port := u.Port()

	if !strings.HasPrefix(host, "management.") {
		return "", fmt.Errorf("cannot derive IAM endpoint: management endpoint hostname %q does not start with 'management.'", host)
	}

	iamHost := "iam." + strings.TrimPrefix(host, "management.")
	if port != "" {
		iamHost = iamHost + ":" + port
	}

	u.Host = iamHost
	u.Path = ""
	return u.String(), nil
}

// --- IAM Response Types ---

type iamErrorResponse struct {
	XMLName xml.Name `xml:"ErrorResponse"`
	Error   struct {
		Code    string `xml:"Code"`
		Message string `xml:"Message"`
	} `xml:"Error"`
}

type createUserResponse struct {
	XMLName xml.Name `xml:"CreateUserResponse"`
	Result  struct {
		User iamUser `xml:"User"`
	} `xml:"CreateUserResult"`
}

type getUserResponse struct {
	XMLName xml.Name `xml:"GetUserResponse"`
	Result  struct {
		User iamUser `xml:"User"`
	} `xml:"GetUserResult"`
}

type iamUser struct {
	UserName string `xml:"UserName"`
	UserId   string `xml:"UserId"`
	Arn      string `xml:"Arn"`
	Path     string `xml:"Path"`
}

type getUserPolicyResponse struct {
	XMLName xml.Name `xml:"GetUserPolicyResponse"`
	Result  struct {
		UserName       string `xml:"UserName"`
		PolicyName     string `xml:"PolicyName"`
		PolicyDocument string `xml:"PolicyDocument"`
	} `xml:"GetUserPolicyResult"`
}

// --- IAM Operations ---

func (c *IAMClient) CreateUser(ctx context.Context, accessKey, secretKey, userName string) (*iamUser, error) {
	params := url.Values{
		"Action":   {"CreateUser"},
		"UserName": {userName},
	}

	body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	var resp createUserResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing create user response: %w", err)
	}

	return &resp.Result.User, nil
}

func (c *IAMClient) GetUser(ctx context.Context, accessKey, secretKey, userName string) (*iamUser, error) {
	params := url.Values{
		"Action":   {"GetUser"},
		"UserName": {userName},
	}

	body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchEntity") {
			return nil, nil
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	var resp getUserResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing get user response: %w", err)
	}

	return &resp.Result.User, nil
}

func (c *IAMClient) DeleteUser(ctx context.Context, accessKey, secretKey, userName string) error {
	params := url.Values{
		"Action":   {"DeleteUser"},
		"UserName": {userName},
	}

	_, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchEntity") {
			return nil
		}
		return fmt.Errorf("delete user: %w", err)
	}

	return nil
}

func (c *IAMClient) PutUserPolicy(ctx context.Context, accessKey, secretKey, userName, policyName, policyDocument string) error {
	params := url.Values{
		"Action":         {"PutUserPolicy"},
		"UserName":       {userName},
		"PolicyName":     {policyName},
		"PolicyDocument": {policyDocument},
		"Version":        {"2010-05-08"},
	}

	_, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		return fmt.Errorf("put user policy: %w", err)
	}

	return nil
}

func (c *IAMClient) GetUserPolicy(ctx context.Context, accessKey, secretKey, userName, policyName string) (string, error) {
	params := url.Values{
		"Action":     {"GetUserPolicy"},
		"UserName":   {userName},
		"PolicyName": {policyName},
		"Version":    {"2010-05-08"},
	}

	body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchEntity") {
			return "", nil
		}
		return "", fmt.Errorf("get user policy: %w", err)
	}

	var resp getUserPolicyResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("parsing get user policy response: %w", err)
	}

	// PolicyDocument is URL-encoded in the response
	decoded, err := url.QueryUnescape(resp.Result.PolicyDocument)
	if err != nil {
		return resp.Result.PolicyDocument, nil
	}

	return decoded, nil
}

func (c *IAMClient) DeleteUserPolicy(ctx context.Context, accessKey, secretKey, userName, policyName string) error {
	params := url.Values{
		"Action":     {"DeleteUserPolicy"},
		"UserName":   {userName},
		"PolicyName": {policyName},
		"Version":    {"2010-05-08"},
	}

	_, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchEntity") {
			return nil
		}
		return fmt.Errorf("delete user policy: %w", err)
	}

	return nil
}

// --- Access Key Operations ---

type createAccessKeyResponse struct {
	XMLName xml.Name `xml:"CreateAccessKeyResponse"`
	Result  struct {
		AccessKey iamAccessKey `xml:"AccessKey"`
	} `xml:"CreateAccessKeyResult"`
}

type listAccessKeysResponse struct {
	XMLName xml.Name `xml:"ListAccessKeysResponse"`
	Result  struct {
		AccessKeyMetadata []iamAccessKeyMetadata `xml:"AccessKeyMetadata>member"`
	} `xml:"ListAccessKeysResult"`
}

type iamAccessKey struct {
	UserName        string `xml:"UserName"`
	AccessKeyId     string `xml:"AccessKeyId"`
	SecretAccessKey string `xml:"SecretAccessKey"`
	Status          string `xml:"Status"`
}

type iamAccessKeyMetadata struct {
	UserName    string `xml:"UserName"`
	AccessKeyId string `xml:"AccessKeyId"`
	Status      string `xml:"Status"`
}

func (c *IAMClient) CreateAccessKey(ctx context.Context, accessKey, secretKey, userName string) (*iamAccessKey, error) {
	params := url.Values{
		"Action":   {"CreateAccessKey"},
		"UserName": {userName},
	}

	body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		return nil, fmt.Errorf("create access key: %w", err)
	}

	var resp createAccessKeyResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing create access key response: %w", err)
	}

	return &resp.Result.AccessKey, nil
}

func (c *IAMClient) ListAccessKeys(ctx context.Context, accessKey, secretKey, userName string) ([]iamAccessKeyMetadata, error) {
	params := url.Values{
		"Action":   {"ListAccessKeys"},
		"UserName": {userName},
	}

	body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		return nil, fmt.Errorf("list access keys: %w", err)
	}

	var resp listAccessKeysResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing list access keys response: %w", err)
	}

	return resp.Result.AccessKeyMetadata, nil
}

func (c *IAMClient) DeleteAccessKey(ctx context.Context, accountAccessKey, accountSecretKey, userName, accessKeyId string) error {
	params := url.Values{
		"Action":      {"DeleteAccessKey"},
		"UserName":    {userName},
		"AccessKeyId": {accessKeyId},
		"Version":     {"2010-05-08"},
	}

	_, err := c.doSignedRequest(ctx, accountAccessKey, accountSecretKey, params)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchEntity") {
			return nil
		}
		return fmt.Errorf("delete access key: %w", err)
	}

	return nil
}

// --- Group Operations ---

type iamGroup struct {
	GroupName string `xml:"GroupName"`
	GroupId   string `xml:"GroupId"`
	Arn       string `xml:"Arn"`
	Path      string `xml:"Path"`
}

type createGroupResponse struct {
	XMLName xml.Name `xml:"CreateGroupResponse"`
	Result  struct {
		Group iamGroup `xml:"Group"`
	} `xml:"CreateGroupResult"`
}

type getGroupResponse struct {
	XMLName xml.Name `xml:"GetGroupResponse"`
	Result  struct {
		Group iamGroup  `xml:"Group"`
		Users []iamUser `xml:"Users>member"`
	} `xml:"GetGroupResult"`
}

type listGroupsForUserResponse struct {
	XMLName xml.Name `xml:"ListGroupsForUserResponse"`
	Result  struct {
		Groups []iamGroup `xml:"Groups>member"`
	} `xml:"ListGroupsForUserResult"`
}

func (c *IAMClient) CreateGroup(ctx context.Context, accessKey, secretKey, groupName string) (*iamGroup, error) {
	params := url.Values{
		"Action":    {"CreateGroup"},
		"GroupName": {groupName},
	}
	body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		return nil, fmt.Errorf("create group: %w", err)
	}
	var resp createGroupResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing create group response: %w", err)
	}
	return &resp.Result.Group, nil
}

func (c *IAMClient) GetGroup(ctx context.Context, accessKey, secretKey, groupName string) (*iamGroup, error) {
	params := url.Values{
		"Action":    {"GetGroup"},
		"GroupName": {groupName},
	}
	body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchEntity") {
			return nil, nil
		}
		return nil, fmt.Errorf("get group: %w", err)
	}
	var resp getGroupResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing get group response: %w", err)
	}
	return &resp.Result.Group, nil
}

func (c *IAMClient) DeleteGroup(ctx context.Context, accessKey, secretKey, groupName string) error {
	params := url.Values{
		"Action":    {"DeleteGroup"},
		"GroupName": {groupName},
	}
	_, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchEntity") {
			return nil
		}
		return fmt.Errorf("delete group: %w", err)
	}
	return nil
}

func (c *IAMClient) AddUserToGroup(ctx context.Context, accessKey, secretKey, groupName, userName string) error {
	params := url.Values{
		"Action":    {"AddUserToGroup"},
		"GroupName": {groupName},
		"UserName":  {userName},
	}
	_, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		return fmt.Errorf("add user to group: %w", err)
	}
	return nil
}

func (c *IAMClient) RemoveUserFromGroup(ctx context.Context, accessKey, secretKey, groupName, userName string) error {
	params := url.Values{
		"Action":    {"RemoveUserFromGroup"},
		"GroupName": {groupName},
		"UserName":  {userName},
	}
	_, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchEntity") {
			return nil
		}
		return fmt.Errorf("remove user from group: %w", err)
	}
	return nil
}

// ListGroupsForUser returns the names of groups the user belongs to. Returns an
// empty slice if the user has no groups or does not exist.
func (c *IAMClient) ListGroupsForUser(ctx context.Context, accessKey, secretKey, userName string) ([]string, error) {
	params := url.Values{
		"Action":   {"ListGroupsForUser"},
		"UserName": {userName},
	}
	body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchEntity") {
			return nil, nil
		}
		return nil, fmt.Errorf("list groups for user: %w", err)
	}
	var resp listGroupsForUserResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing list groups for user response: %w", err)
	}
	names := make([]string, 0, len(resp.Result.Groups))
	for _, g := range resp.Result.Groups {
		names = append(names, g.GroupName)
	}
	return names, nil
}

// --- Group Inline Policy Operations ---

type getGroupPolicyResponse struct {
	XMLName xml.Name `xml:"GetGroupPolicyResponse"`
	Result  struct {
		GroupName      string `xml:"GroupName"`
		PolicyName     string `xml:"PolicyName"`
		PolicyDocument string `xml:"PolicyDocument"`
	} `xml:"GetGroupPolicyResult"`
}

func (c *IAMClient) PutGroupPolicy(ctx context.Context, accessKey, secretKey, groupName, policyName, policyDocument string) error {
	params := url.Values{
		"Action":         {"PutGroupPolicy"},
		"GroupName":      {groupName},
		"PolicyName":     {policyName},
		"PolicyDocument": {policyDocument},
	}
	_, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		return fmt.Errorf("put group policy: %w", err)
	}
	return nil
}

func (c *IAMClient) GetGroupPolicy(ctx context.Context, accessKey, secretKey, groupName, policyName string) (string, error) {
	params := url.Values{
		"Action":     {"GetGroupPolicy"},
		"GroupName":  {groupName},
		"PolicyName": {policyName},
	}
	body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchEntity") {
			return "", nil
		}
		return "", fmt.Errorf("get group policy: %w", err)
	}
	var resp getGroupPolicyResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("parsing get group policy response: %w", err)
	}
	decoded, err := url.QueryUnescape(resp.Result.PolicyDocument)
	if err != nil {
		return resp.Result.PolicyDocument, nil
	}
	return decoded, nil
}

func (c *IAMClient) DeleteGroupPolicy(ctx context.Context, accessKey, secretKey, groupName, policyName string) error {
	params := url.Values{
		"Action":     {"DeleteGroupPolicy"},
		"GroupName":  {groupName},
		"PolicyName": {policyName},
	}
	_, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchEntity") {
			return nil
		}
		return fmt.Errorf("delete group policy: %w", err)
	}
	return nil
}

// --- Role Operations ---

type iamRole struct {
	RoleName                 string `xml:"RoleName"`
	RoleId                   string `xml:"RoleId"`
	Arn                      string `xml:"Arn"`
	Path                     string `xml:"Path"`
	AssumeRolePolicyDocument string `xml:"AssumeRolePolicyDocument"`
	Description              string `xml:"Description"`
}

type createRoleResponse struct {
	XMLName xml.Name `xml:"CreateRoleResponse"`
	Result  struct {
		Role iamRole `xml:"Role"`
	} `xml:"CreateRoleResult"`
}

type getRoleResponse struct {
	XMLName xml.Name `xml:"GetRoleResponse"`
	Result  struct {
		Role iamRole `xml:"Role"`
	} `xml:"GetRoleResult"`
}

// CreateRole creates an IAM role. trustPolicy is the assume-role policy document
// (JSON). description may be empty. Note: ARTESCA does NOT implement
// UpdateAssumeRolePolicy, so callers must treat the trust policy as immutable.
func (c *IAMClient) CreateRole(ctx context.Context, accessKey, secretKey, roleName, trustPolicy, description string) (*iamRole, error) {
	params := url.Values{
		"Action":                   {"CreateRole"},
		"RoleName":                 {roleName},
		"AssumeRolePolicyDocument": {trustPolicy},
	}
	if description != "" {
		params.Set("Description", description)
	}
	body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		return nil, fmt.Errorf("create role: %w", err)
	}
	var resp createRoleResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing create role response: %w", err)
	}
	// AssumeRolePolicyDocument comes back URL-encoded.
	if decoded, derr := url.QueryUnescape(resp.Result.Role.AssumeRolePolicyDocument); derr == nil {
		resp.Result.Role.AssumeRolePolicyDocument = decoded
	}
	return &resp.Result.Role, nil
}

func (c *IAMClient) GetRole(ctx context.Context, accessKey, secretKey, roleName string) (*iamRole, error) {
	params := url.Values{
		"Action":   {"GetRole"},
		"RoleName": {roleName},
	}
	body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchEntity") {
			return nil, nil
		}
		return nil, fmt.Errorf("get role: %w", err)
	}
	var resp getRoleResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing get role response: %w", err)
	}
	if decoded, derr := url.QueryUnescape(resp.Result.Role.AssumeRolePolicyDocument); derr == nil {
		resp.Result.Role.AssumeRolePolicyDocument = decoded
	}
	return &resp.Result.Role, nil
}

func (c *IAMClient) DeleteRole(ctx context.Context, accessKey, secretKey, roleName string) error {
	params := url.Values{
		"Action":   {"DeleteRole"},
		"RoleName": {roleName},
	}
	_, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchEntity") {
			return nil
		}
		return fmt.Errorf("delete role: %w", err)
	}
	return nil
}

// --- Managed Policy Operations ---

type iamManagedPolicy struct {
	PolicyName       string `xml:"PolicyName"`
	PolicyId         string `xml:"PolicyId"`
	Arn              string `xml:"Arn"`
	Path             string `xml:"Path"`
	DefaultVersionId string `xml:"DefaultVersionId"`
	Description      string `xml:"Description"`
}

type createPolicyResponse struct {
	XMLName xml.Name `xml:"CreatePolicyResponse"`
	Result  struct {
		Policy iamManagedPolicy `xml:"Policy"`
	} `xml:"CreatePolicyResult"`
}

type getPolicyResponse struct {
	XMLName xml.Name `xml:"GetPolicyResponse"`
	Result  struct {
		Policy iamManagedPolicy `xml:"Policy"`
	} `xml:"GetPolicyResult"`
}

type getPolicyVersionResponse struct {
	XMLName xml.Name `xml:"GetPolicyVersionResponse"`
	Result  struct {
		PolicyVersion struct {
			Document         string `xml:"Document"`
			VersionId        string `xml:"VersionId"`
			IsDefaultVersion bool   `xml:"IsDefaultVersion"`
		} `xml:"PolicyVersion"`
	} `xml:"GetPolicyVersionResult"`
}

func (c *IAMClient) CreatePolicy(ctx context.Context, accessKey, secretKey, policyName, policyDocument, description string) (*iamManagedPolicy, error) {
	params := url.Values{
		"Action":         {"CreatePolicy"},
		"PolicyName":     {policyName},
		"PolicyDocument": {policyDocument},
	}
	if description != "" {
		params.Set("Description", description)
	}
	body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		return nil, fmt.Errorf("create policy: %w", err)
	}
	var resp createPolicyResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing create policy response: %w", err)
	}
	return &resp.Result.Policy, nil
}

func (c *IAMClient) GetPolicy(ctx context.Context, accessKey, secretKey, policyArn string) (*iamManagedPolicy, error) {
	params := url.Values{
		"Action":    {"GetPolicy"},
		"PolicyArn": {policyArn},
	}
	body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchEntity") {
			return nil, nil
		}
		return nil, fmt.Errorf("get policy: %w", err)
	}
	var resp getPolicyResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing get policy response: %w", err)
	}
	return &resp.Result.Policy, nil
}

// GetPolicyDocument fetches the active policy version's JSON document.
func (c *IAMClient) GetPolicyDocument(ctx context.Context, accessKey, secretKey, policyArn, versionId string) (string, error) {
	params := url.Values{
		"Action":    {"GetPolicyVersion"},
		"PolicyArn": {policyArn},
		"VersionId": {versionId},
	}
	body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		return "", fmt.Errorf("get policy version: %w", err)
	}
	var resp getPolicyVersionResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("parsing get policy version response: %w", err)
	}
	decoded, err := url.QueryUnescape(resp.Result.PolicyVersion.Document)
	if err != nil {
		return resp.Result.PolicyVersion.Document, nil
	}
	return decoded, nil
}

func (c *IAMClient) DeletePolicy(ctx context.Context, accessKey, secretKey, policyArn string) error {
	params := url.Values{
		"Action":    {"DeletePolicy"},
		"PolicyArn": {policyArn},
	}
	_, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchEntity") {
			return nil
		}
		return fmt.Errorf("delete policy: %w", err)
	}
	return nil
}

// --- Managed Policy Attachment Operations ---

type attachedPolicy struct {
	PolicyName string `xml:"PolicyName"`
	PolicyArn  string `xml:"PolicyArn"`
}

type listAttachedUserPoliciesResponse struct {
	XMLName xml.Name `xml:"ListAttachedUserPoliciesResponse"`
	Result  struct {
		AttachedPolicies []attachedPolicy `xml:"AttachedPolicies>member"`
	} `xml:"ListAttachedUserPoliciesResult"`
}

type listAttachedGroupPoliciesResponse struct {
	XMLName xml.Name `xml:"ListAttachedGroupPoliciesResponse"`
	Result  struct {
		AttachedPolicies []attachedPolicy `xml:"AttachedPolicies>member"`
	} `xml:"ListAttachedGroupPoliciesResult"`
}

type listAttachedRolePoliciesResponse struct {
	XMLName xml.Name `xml:"ListAttachedRolePoliciesResponse"`
	Result  struct {
		AttachedPolicies []attachedPolicy `xml:"AttachedPolicies>member"`
	} `xml:"ListAttachedRolePoliciesResult"`
}

func (c *IAMClient) AttachUserPolicy(ctx context.Context, accessKey, secretKey, userName, policyArn string) error {
	params := url.Values{
		"Action":    {"AttachUserPolicy"},
		"UserName":  {userName},
		"PolicyArn": {policyArn},
	}
	_, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		return fmt.Errorf("attach user policy: %w", err)
	}
	return nil
}

func (c *IAMClient) DetachUserPolicy(ctx context.Context, accessKey, secretKey, userName, policyArn string) error {
	params := url.Values{
		"Action":    {"DetachUserPolicy"},
		"UserName":  {userName},
		"PolicyArn": {policyArn},
	}
	_, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchEntity") {
			return nil
		}
		return fmt.Errorf("detach user policy: %w", err)
	}
	return nil
}

func (c *IAMClient) ListAttachedUserPolicies(ctx context.Context, accessKey, secretKey, userName string) ([]string, error) {
	params := url.Values{
		"Action":   {"ListAttachedUserPolicies"},
		"UserName": {userName},
	}
	body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchEntity") {
			return nil, nil
		}
		return nil, fmt.Errorf("list attached user policies: %w", err)
	}
	var resp listAttachedUserPoliciesResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing list attached user policies response: %w", err)
	}
	arns := make([]string, 0, len(resp.Result.AttachedPolicies))
	for _, p := range resp.Result.AttachedPolicies {
		arns = append(arns, p.PolicyArn)
	}
	return arns, nil
}

func (c *IAMClient) AttachGroupPolicy(ctx context.Context, accessKey, secretKey, groupName, policyArn string) error {
	params := url.Values{
		"Action":    {"AttachGroupPolicy"},
		"GroupName": {groupName},
		"PolicyArn": {policyArn},
	}
	_, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		return fmt.Errorf("attach group policy: %w", err)
	}
	return nil
}

func (c *IAMClient) DetachGroupPolicy(ctx context.Context, accessKey, secretKey, groupName, policyArn string) error {
	params := url.Values{
		"Action":    {"DetachGroupPolicy"},
		"GroupName": {groupName},
		"PolicyArn": {policyArn},
	}
	_, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchEntity") {
			return nil
		}
		return fmt.Errorf("detach group policy: %w", err)
	}
	return nil
}

func (c *IAMClient) ListAttachedGroupPolicies(ctx context.Context, accessKey, secretKey, groupName string) ([]string, error) {
	params := url.Values{
		"Action":    {"ListAttachedGroupPolicies"},
		"GroupName": {groupName},
	}
	body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchEntity") {
			return nil, nil
		}
		return nil, fmt.Errorf("list attached group policies: %w", err)
	}
	var resp listAttachedGroupPoliciesResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing list attached group policies response: %w", err)
	}
	arns := make([]string, 0, len(resp.Result.AttachedPolicies))
	for _, p := range resp.Result.AttachedPolicies {
		arns = append(arns, p.PolicyArn)
	}
	return arns, nil
}

func (c *IAMClient) AttachRolePolicy(ctx context.Context, accessKey, secretKey, roleName, policyArn string) error {
	params := url.Values{
		"Action":    {"AttachRolePolicy"},
		"RoleName":  {roleName},
		"PolicyArn": {policyArn},
	}
	_, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		return fmt.Errorf("attach role policy: %w", err)
	}
	return nil
}

func (c *IAMClient) DetachRolePolicy(ctx context.Context, accessKey, secretKey, roleName, policyArn string) error {
	params := url.Values{
		"Action":    {"DetachRolePolicy"},
		"RoleName":  {roleName},
		"PolicyArn": {policyArn},
	}
	_, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchEntity") {
			return nil
		}
		return fmt.Errorf("detach role policy: %w", err)
	}
	return nil
}

func (c *IAMClient) ListAttachedRolePolicies(ctx context.Context, accessKey, secretKey, roleName string) ([]string, error) {
	params := url.Values{
		"Action":   {"ListAttachedRolePolicies"},
		"RoleName": {roleName},
	}
	body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchEntity") {
			return nil, nil
		}
		return nil, fmt.Errorf("list attached role policies: %w", err)
	}
	var resp listAttachedRolePoliciesResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing list attached role policies response: %w", err)
	}
	arns := make([]string, 0, len(resp.Result.AttachedPolicies))
	for _, p := range resp.Result.AttachedPolicies {
		arns = append(arns, p.PolicyArn)
	}
	return arns, nil
}

// --- SigV4 Signing ---

func (c *IAMClient) doSignedRequest(ctx context.Context, accessKey, secretKey string, params url.Values) ([]byte, error) {
	u, err := url.Parse(c.endpoint)
	if err != nil {
		return nil, fmt.Errorf("parsing endpoint: %w", err)
	}

	// Set API version once here so callers don't repeat it.
	params.Set("Version", iamAPIVersion)

	host := u.Host
	now := time.Now().UTC()
	datestamp := now.Format("20060102")
	amzdate := now.Format("20060102T150405Z")

	// Build canonical query string
	body := params.Encode()

	// Create the canonical request for POST with form-encoded body
	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-date:%s\n", host, amzdate)
	signedHeaders := "host;x-amz-date"
	payloadHash := sha256Hex([]byte(body))

	canonicalRequest := strings.Join([]string{
		http.MethodPost,
		"/",
		"",
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	// Create string to sign
	region := c.region
	service := iamAWSService
	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", datestamp, region, service)
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzdate,
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	// Calculate signature
	signingKey := getSignatureKey(secretKey, datestamp, region, service)
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	// Build authorization header
	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		accessKey, credentialScope, signedHeaders, signature)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/", strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", contentTypeForm)
	req.Header.Set("Host", host)
	req.Header.Set("X-Amz-Date", amzdate)
	req.Header.Set("Authorization", authHeader)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var iamErr iamErrorResponse
		if xmlErr := xml.Unmarshal(respBody, &iamErr); xmlErr == nil && iamErr.Error.Code != "" {
			return nil, fmt.Errorf("%s: %s", iamErr.Error.Code, iamErr.Error.Message)
		}
		return nil, fmt.Errorf("IAM request failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
