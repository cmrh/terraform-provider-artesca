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
			InsecureSkipVerify: insecureSkipVerify,
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
