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
	"sync"
	"time"
)

const (
	s3AWSService  = "s3"
	s3HTTPTimeout = 30 * time.Second
)

type S3Client struct {
	endpoint    string
	region      string
	httpClient  *http.Client
	lifecycleMu sync.Mutex
}

func (c *S3Client) LockLifecycle()   { c.lifecycleMu.Lock() }
func (c *S3Client) UnlockLifecycle() { c.lifecycleMu.Unlock() }

func NewS3Client(endpoint, region string, insecureSkipVerify bool) *S3Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecureSkipVerify,
		},
	}

	return &S3Client{
		endpoint: strings.TrimRight(endpoint, "/"),
		region:   region,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   s3HTTPTimeout,
		},
	}
}

type s3ErrorResponse struct {
	XMLName xml.Name `xml:"Error"`
	Code    string   `xml:"Code"`
	Message string   `xml:"Message"`
}

// isLocationPropagationError returns true when the S3 service has not yet
// learned about a location that was recently created via the management API.
func isLocationPropagationError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "InvalidLocationConstraint") ||
		(strings.Contains(msg, "MalformedXML") && strings.Contains(msg, "StorageClass"))
}

const propagationTimeout = 300 * time.Second

func (c *S3Client) CreateBucket(ctx context.Context, accessKey, secretKey, bucket, locationConstraint string) error {
	var body string
	if locationConstraint != "" {
		body = fmt.Sprintf(
			`<CreateBucketConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><LocationConstraint>%s</LocationConstraint></CreateBucketConfiguration>`,
			locationConstraint,
		)
	}

	deadline := time.Now().Add(propagationTimeout)
	backoff := 5 * time.Second

	for {
		_, status, err := c.doSignedRequest(ctx, http.MethodPut, "/"+bucket, "", body, accessKey, secretKey)
		if err != nil && isLocationPropagationError(err) && time.Now().Before(deadline) {
			time.Sleep(backoff)
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		if err != nil {
			return err
		}
		if status != http.StatusOK && status != http.StatusCreated {
			return fmt.Errorf("create bucket failed (status %d)", status)
		}

		return nil
	}
}

func (c *S3Client) PutBucketVersioning(ctx context.Context, accessKey, secretKey, bucket string, enabled bool) error {
	status := "Suspended"
	if enabled {
		status = "Enabled"
	}
	body := fmt.Sprintf(
		`<VersioningConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Status>%s</Status></VersioningConfiguration>`,
		status,
	)
	_, httpStatus, err := c.doSignedRequest(ctx, http.MethodPut, "/"+bucket, "versioning", body, accessKey, secretKey)
	if err != nil {
		return err
	}
	if httpStatus != http.StatusOK && httpStatus != http.StatusNoContent {
		return fmt.Errorf("put bucket versioning failed (status %d)", httpStatus)
	}

	return nil
}

func (c *S3Client) GetBucketVersioning(ctx context.Context, accessKey, secretKey, bucket string) (bool, error) {
	respBody, status, err := c.doSignedRequest(ctx, http.MethodGet, "/"+bucket, "versioning", "", accessKey, secretKey)
	if err != nil {
		return false, err
	}
	if status != http.StatusOK {
		return false, fmt.Errorf("get bucket versioning failed (status %d)", status)
	}

	var conf struct {
		XMLName xml.Name `xml:"VersioningConfiguration"`
		Status  string   `xml:"Status"`
	}
	if err := xml.Unmarshal(respBody, &conf); err != nil {
		return false, fmt.Errorf("parsing versioning response: %w", err)
	}

	return conf.Status == "Enabled", nil
}

func (c *S3Client) HeadBucket(ctx context.Context, accessKey, secretKey, bucket string) (bool, error) {
	_, status, err := c.doSignedRequest(ctx, http.MethodHead, "/"+bucket, "", "", accessKey, secretKey)
	if err != nil {
		return false, err
	}
	if status == http.StatusOK {
		return true, nil
	}
	if status == http.StatusNotFound || status == http.StatusForbidden {
		return false, nil
	}

	return false, fmt.Errorf("head bucket unexpected status %d", status)
}

func (c *S3Client) GetBucketLocation(ctx context.Context, accessKey, secretKey, bucket string) (string, error) {
	respBody, status, err := c.doSignedRequest(ctx, http.MethodGet, "/"+bucket, "location", "", accessKey, secretKey)
	if err != nil {
		return "", err
	}
	if status == http.StatusNotFound {
		return "", fmt.Errorf("bucket %q not found", bucket)
	}
	if status != http.StatusOK {
		return "", fmt.Errorf("get bucket location failed (status %d)", status)
	}

	var loc struct {
		XMLName  xml.Name `xml:"LocationConstraint"`
		Location string   `xml:",chardata"`
	}
	if err := xml.Unmarshal(respBody, &loc); err != nil {
		return "", fmt.Errorf("parsing location response: %w", err)
	}

	return loc.Location, nil
}

func (c *S3Client) DeleteBucket(ctx context.Context, accessKey, secretKey, bucket string) error {
	_, status, err := c.doSignedRequest(ctx, http.MethodDelete, "/"+bucket, "", "", accessKey, secretKey)
	if status == http.StatusNotFound {
		return nil
	}
	if err != nil {
		return err
	}
	if status != http.StatusNoContent && status != http.StatusOK {
		return fmt.Errorf("delete bucket failed (status %d)", status)
	}

	return nil
}

func (c *S3Client) doSignedRequest(ctx context.Context, method, path, query, body, accessKey, secretKey string) ([]byte, int, error) {
	u, err := url.Parse(c.endpoint)
	if err != nil {
		return nil, 0, fmt.Errorf("parsing endpoint: %w", err)
	}

	host := u.Host
	now := time.Now().UTC()
	datestamp := now.Format("20060102")
	amzdate := now.Format("20060102T150405Z")

	payloadHash := sha256Hex([]byte(body))

	canonicalQueryString := ""
	if query != "" {
		canonicalQueryString = query + "="
	}

	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n", host, payloadHash, amzdate)
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"

	canonicalRequest := strings.Join([]string{
		method,
		path,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", datestamp, c.region, s3AWSService)
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzdate,
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	signingKey := getSignatureKey(secretKey, datestamp, c.region, s3AWSService)
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		accessKey, credentialScope, signedHeaders, signature)

	fullURL := c.endpoint + path
	if query != "" {
		fullURL += "?" + canonicalQueryString
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, strings.NewReader(body))
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Host", host)
	req.Header.Set("X-Amz-Date", amzdate)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	req.Header.Set("Authorization", authHeader)
	if body != "" {
		req.Header.Set("Content-Type", "application/xml")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, 0, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var s3Err s3ErrorResponse
		if xmlErr := xml.Unmarshal(respBody, &s3Err); xmlErr == nil && s3Err.Code != "" {
			return respBody, resp.StatusCode, fmt.Errorf("%s: %s", s3Err.Code, s3Err.Message)
		}
	}

	return respBody, resp.StatusCode, nil
}
