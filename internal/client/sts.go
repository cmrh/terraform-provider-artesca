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

// STS constants.
const (
	stsAPIVersion  = "2011-06-15"
	stsAWSService  = "sts"
	stsHTTPTimeout = 30 * time.Second
)

// STSClient communicates with the ARTESCA STS API using AWS SigV4 signing.
type STSClient struct {
	endpoint   string
	region     string
	httpClient *http.Client
}

func NewSTSClient(endpoint, region string, insecureSkipVerify bool) *STSClient {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecureSkipVerify,
		},
	}

	return &STSClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		region:   region,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   stsHTTPTimeout,
		},
	}
}

// DeriveSTSEndpoint derives the STS endpoint from the S3 endpoint by replacing
// the leading "s3." hostname segment with "sts.". Mirrors the management→iam
// derivation pattern.
func DeriveSTSEndpoint(s3Endpoint string) (string, error) {
	u, err := url.Parse(s3Endpoint)
	if err != nil {
		return "", fmt.Errorf("parsing S3 endpoint: %w", err)
	}

	host := u.Hostname()
	port := u.Port()

	if !strings.HasPrefix(host, "s3.") {
		return "", fmt.Errorf("cannot derive STS endpoint: S3 endpoint hostname %q does not start with 's3.'", host)
	}

	stsHost := "sts." + strings.TrimPrefix(host, "s3.")
	if port != "" {
		stsHost = stsHost + ":" + port
	}

	u.Host = stsHost
	u.Path = ""
	return u.String(), nil
}

// --- STS Response Types ---

type stsErrorResponse struct {
	XMLName xml.Name `xml:"ErrorResponse"`
	Error   struct {
		Code    string `xml:"Code"`
		Message string `xml:"Message"`
	} `xml:"Error"`
}

type AssumedRoleCredentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Expiration      time.Time
	AssumedRoleID   string
	AssumedRoleArn  string
}

type assumeRoleResponse struct {
	XMLName xml.Name `xml:"AssumeRoleResponse"`
	Result  struct {
		Credentials struct {
			AccessKeyID     string `xml:"AccessKeyId"`
			SecretAccessKey string `xml:"SecretAccessKey"`
			SessionToken    string `xml:"SessionToken"`
			Expiration      string `xml:"Expiration"`
		} `xml:"Credentials"`
		AssumedRoleUser struct {
			AssumedRoleID string `xml:"AssumedRoleId"`
			Arn           string `xml:"Arn"`
		} `xml:"AssumedRoleUser"`
	} `xml:"AssumeRoleResult"`
}

type CallerIdentity struct {
	UserID  string
	Account string
	Arn     string
}

type getCallerIdentityResponse struct {
	XMLName xml.Name `xml:"GetCallerIdentityResponse"`
	Result  struct {
		UserID  string `xml:"UserId"`
		Account string `xml:"Account"`
		Arn     string `xml:"Arn"`
	} `xml:"GetCallerIdentityResult"`
}

// --- STS Operations ---

// AssumeRoleOptions carries optional parameters for AssumeRole. Zero values
// fall back to STS server defaults.
type AssumeRoleOptions struct {
	DurationSeconds int64
	ExternalID      string
	Policy          string
}

// AssumeRole calls sts:AssumeRole and returns temporary credentials.
func (c *STSClient) AssumeRole(ctx context.Context, accessKey, secretKey, roleArn, sessionName string, opts AssumeRoleOptions) (*AssumedRoleCredentials, error) {
	params := url.Values{
		"Action":          {"AssumeRole"},
		"RoleArn":         {roleArn},
		"RoleSessionName": {sessionName},
	}
	if opts.DurationSeconds > 0 {
		params.Set("DurationSeconds", fmt.Sprintf("%d", opts.DurationSeconds))
	}
	if opts.ExternalID != "" {
		params.Set("ExternalId", opts.ExternalID)
	}
	if opts.Policy != "" {
		params.Set("Policy", opts.Policy)
	}

	body, err := c.doSignedRequest(ctx, accessKey, secretKey, "", params)
	if err != nil {
		return nil, fmt.Errorf("assume role: %w", err)
	}

	var resp assumeRoleResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing assume role response: %w", err)
	}

	exp, err := time.Parse(time.RFC3339, resp.Result.Credentials.Expiration)
	if err != nil {
		return nil, fmt.Errorf("parsing assume role expiration %q: %w", resp.Result.Credentials.Expiration, err)
	}

	return &AssumedRoleCredentials{
		AccessKeyID:     resp.Result.Credentials.AccessKeyID,
		SecretAccessKey: resp.Result.Credentials.SecretAccessKey,
		SessionToken:    resp.Result.Credentials.SessionToken,
		Expiration:      exp,
		AssumedRoleID:   resp.Result.AssumedRoleUser.AssumedRoleID,
		AssumedRoleArn:  resp.Result.AssumedRoleUser.Arn,
	}, nil
}

// GetCallerIdentity returns the identity associated with the request
// credentials. Pass a non-empty sessionToken to introspect temporary
// credentials (e.g. those minted by AssumeRole); pass "" for static IAM-user
// or account credentials.
func (c *STSClient) GetCallerIdentity(ctx context.Context, accessKey, secretKey, sessionToken string) (*CallerIdentity, error) {
	params := url.Values{
		"Action": {"GetCallerIdentity"},
	}

	body, err := c.doSignedRequest(ctx, accessKey, secretKey, sessionToken, params)
	if err != nil {
		return nil, fmt.Errorf("get caller identity: %w", err)
	}

	var resp getCallerIdentityResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing get caller identity response: %w", err)
	}

	return &CallerIdentity{
		UserID:  resp.Result.UserID,
		Account: resp.Result.Account,
		Arn:     resp.Result.Arn,
	}, nil
}

// --- SigV4 Signing (mirrors IAMClient.doSignedRequest) ---

// doSignedRequest signs and sends a POST request to the STS endpoint. When
// sessionToken is non-empty (temporary credentials from sts:AssumeRole or
// similar), it is included in the signed X-Amz-Security-Token header.
func (c *STSClient) doSignedRequest(ctx context.Context, accessKey, secretKey, sessionToken string, params url.Values) ([]byte, error) {
	u, err := url.Parse(c.endpoint)
	if err != nil {
		return nil, fmt.Errorf("parsing endpoint: %w", err)
	}

	params.Set("Version", stsAPIVersion)

	host := u.Host
	now := time.Now().UTC()
	datestamp := now.Format("20060102")
	amzdate := now.Format("20060102T150405Z")

	body := params.Encode()

	// SigV4 canonical headers must be alphabetically sorted by header name.
	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-date:%s\n", host, amzdate)
	signedHeaders := "host;x-amz-date"
	if sessionToken != "" {
		canonicalHeaders = fmt.Sprintf("host:%s\nx-amz-date:%s\nx-amz-security-token:%s\n", host, amzdate, sessionToken)
		signedHeaders = "host;x-amz-date;x-amz-security-token"
	}
	payloadHash := sha256Hex([]byte(body))

	canonicalRequest := strings.Join([]string{
		http.MethodPost,
		"/",
		"",
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	region := c.region
	service := stsAWSService
	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", datestamp, region, service)
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzdate,
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	signingKey := getSignatureKey(secretKey, datestamp, region, service)
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		accessKey, credentialScope, signedHeaders, signature)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/", strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", contentTypeForm)
	req.Header.Set("Host", host)
	req.Header.Set("X-Amz-Date", amzdate)
	if sessionToken != "" {
		req.Header.Set("X-Amz-Security-Token", sessionToken)
	}
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
		var stsErr stsErrorResponse
		if xmlErr := xml.Unmarshal(respBody, &stsErr); xmlErr == nil && stsErr.Error.Code != "" {
			return nil, fmt.Errorf("%s: %s", stsErr.Error.Code, stsErr.Error.Message)
		}
		return nil, fmt.Errorf("STS request failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
