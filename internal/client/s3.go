package client

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	s3AWSService  = "s3"
	s3HTTPTimeout = 30 * time.Second

	transientGatewayMaxAttempts = 4
	transientGatewayBackoff     = 500 * time.Millisecond
	transientGatewayBackoffMax  = 8 * time.Second
)

type S3Client struct {
	endpoint                 string
	region                   string
	httpClient               *http.Client
	bucketLocksMu            sync.Mutex
	bucketLocks              map[string]*sync.Mutex
	transientBackoffOverride time.Duration
	// propagationBackoffOverride shortens the wait between retries while a new
	// location propagates; tests only.
	propagationBackoffOverride time.Duration
	// listPageSizeOverride sets max-keys on list requests; tests only.
	listPageSizeOverride int
}

// LockBucket serializes bucket-config writes (tagging, encryption, policy,
// versioning, lifecycle, replication) to one bucket and returns the unlock
// function. ARTESCA can silently drop some of several config writes made
// concurrently to the same bucket, and lifecycle/replication writes are
// read-merge-write, so every such write must hold this lock.
func (c *S3Client) LockBucket(bucket string) (unlock func()) {
	c.bucketLocksMu.Lock()
	if c.bucketLocks == nil {
		c.bucketLocks = map[string]*sync.Mutex{}
	}
	mu, ok := c.bucketLocks[bucket]
	if !ok {
		mu = &sync.Mutex{}
		c.bucketLocks[bucket] = mu
	}
	c.bucketLocksMu.Unlock()

	mu.Lock()
	return mu.Unlock
}

// propagationBackoff is the first wait between retries while S3 doesn't know a
// newly created location yet; it doubles up to 30 s.
func (c *S3Client) propagationBackoff() time.Duration {
	if c.propagationBackoffOverride > 0 {
		return c.propagationBackoffOverride
	}
	return 5 * time.Second
}

func (c *S3Client) transientBackoff() time.Duration {
	if c.transientBackoffOverride > 0 {
		return c.transientBackoffOverride
	}
	return transientGatewayBackoff
}

func NewS3Client(endpoint, region string, insecureSkipVerify bool) *S3Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecureSkipVerify, // #nosec G402 -- gated on the user-set insecure_skip_verify provider attribute
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

// NewRuleID returns a random UUID-formatted ID for a lifecycle or
// replication rule.
func NewRuleID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// rawXML encodes an element's inner XML verbatim.
type rawXML struct {
	Inner string `xml:",innerxml"`
}

// isTransientGatewayStatus returns true for upstream/proxy statuses that
// commonly clear on retry (502/503/504). 500 is excluded because it usually
// signals a real server-side error rather than transient unavailability.
func isTransientGatewayStatus(status int) bool {
	return status == http.StatusBadGateway ||
		status == http.StatusServiceUnavailable ||
		status == http.StatusGatewayTimeout
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

// IsNoSuchKey reports whether err is S3's NoSuchKey or NoSuchVersion error.
func IsNoSuchKey(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "NoSuchKey:") || strings.Contains(err.Error(), "NoSuchVersion:"))
}

// IsNoSuchBucket reports whether err is S3's NoSuchBucket error (not
// NoSuchBucketPolicy and similar).
func IsNoSuchBucket(err error) bool {
	return err != nil && strings.Contains(err.Error(), "NoSuchBucket:")
}

const propagationTimeout = 300 * time.Second

// sleepCtx waits for d, or returns ctx.Err() if ctx ends first.
func sleepCtx(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// ErrBucketAlreadyOwned is returned by CreateBucket when the account already
// owns a bucket with that name.
var ErrBucketAlreadyOwned = errors.New("bucket already exists and is owned by this account")

func (c *S3Client) CreateBucket(ctx context.Context, creds Credentials, bucket, locationConstraint string) error {
	var body string
	if locationConstraint != "" {
		body = fmt.Sprintf(
			`<CreateBucketConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><LocationConstraint>%s</LocationConstraint></CreateBucketConfiguration>`,
			locationConstraint,
		)
	}

	deadline := time.Now().Add(propagationTimeout)
	backoff := c.propagationBackoff()

	for {
		_, status, retried, err := c.doSignedRequestRetried(ctx, http.MethodPut, "/"+bucket, "", body, creds)
		if err != nil && strings.Contains(err.Error(), "BucketAlreadyOwnedByYou") {
			// After a gateway retry, the first attempt may be what created it.
			if retried {
				return nil
			}
			return ErrBucketAlreadyOwned
		}
		if err != nil && isLocationPropagationError(err) && time.Now().Before(deadline) {
			if err := sleepCtx(ctx, backoff); err != nil {
				return err
			}
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

func (c *S3Client) PutBucketVersioning(ctx context.Context, creds Credentials, bucket string, enabled bool) error {
	status := "Suspended"
	if enabled {
		status = "Enabled"
	}
	body := fmt.Sprintf(
		`<VersioningConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Status>%s</Status></VersioningConfiguration>`,
		status,
	)
	_, httpStatus, err := c.doSignedRequest(ctx, http.MethodPut, "/"+bucket, "versioning", body, creds)
	if err != nil {
		return err
	}
	if httpStatus != http.StatusOK && httpStatus != http.StatusNoContent {
		return fmt.Errorf("put bucket versioning failed (status %d)", httpStatus)
	}

	return nil
}

func (c *S3Client) GetBucketVersioning(ctx context.Context, creds Credentials, bucket string) (bool, error) {
	respBody, status, err := c.doSignedRequest(ctx, http.MethodGet, "/"+bucket, "versioning", "", creds)
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

func (c *S3Client) HeadBucket(ctx context.Context, creds Credentials, bucket string) (bool, error) {
	_, status, err := c.doSignedRequest(ctx, http.MethodHead, "/"+bucket, "", "", creds)
	if err != nil {
		return false, err
	}
	if status == http.StatusOK {
		return true, nil
	}
	// Only 404 means absent. 403 is returned for a bucket owned by another
	// account, and for expired credentials or a deny policy.
	if status == http.StatusNotFound {
		return false, nil
	}
	if status == http.StatusForbidden {
		return false, fmt.Errorf("head bucket %q: access denied (status 403)", bucket)
	}

	return false, fmt.Errorf("head bucket unexpected status %d", status)
}

func (c *S3Client) GetBucketLocation(ctx context.Context, creds Credentials, bucket string) (string, error) {
	respBody, status, err := c.doSignedRequest(ctx, http.MethodGet, "/"+bucket, "location", "", creds)
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

// PutBucketPolicy attaches a JSON policy document to the bucket. ARTESCA
// validates the policy server-side (Resource ARNs must match the bucket;
// MalformedPolicy is returned otherwise) -- no client-side validation needed.
func (c *S3Client) PutBucketPolicy(ctx context.Context, creds Credentials, bucket, policy string) error {
	_, status, err := c.doSignedRequest(ctx, http.MethodPut, "/"+bucket, "policy", policy, creds)
	if err != nil {
		return err
	}
	if status != http.StatusOK && status != http.StatusNoContent {
		return fmt.Errorf("put bucket policy failed (status %d)", status)
	}
	return nil
}

// GetBucketPolicy returns the policy document attached to the bucket. If the
// bucket has no policy, it returns ("", nil) so callers can treat it as a
// state-removed signal.
func (c *S3Client) GetBucketPolicy(ctx context.Context, creds Credentials, bucket string) (string, error) {
	body, status, err := c.doSignedRequest(ctx, http.MethodGet, "/"+bucket, "policy", "", creds)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchBucketPolicy") {
			return "", nil
		}
		return "", err
	}
	if status == http.StatusNotFound {
		return "", nil
	}
	if status != http.StatusOK {
		return "", fmt.Errorf("get bucket policy failed (status %d)", status)
	}
	return string(body), nil
}

func (c *S3Client) DeleteBucketPolicy(ctx context.Context, creds Credentials, bucket string) error {
	_, status, err := c.doSignedRequest(ctx, http.MethodDelete, "/"+bucket, "policy", "", creds)
	if status == http.StatusNotFound {
		return nil
	}
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchBucketPolicy") {
			return nil
		}
		return err
	}
	if status != http.StatusNoContent && status != http.StatusOK {
		return fmt.Errorf("delete bucket policy failed (status %d)", status)
	}
	return nil
}

// BucketTag is a single key/value pair in a bucket tag set.
type BucketTag struct {
	Key   string
	Value string
}

type taggingRequest struct {
	XMLName xml.Name             `xml:"Tagging"`
	XMLNS   string               `xml:"xmlns,attr"`
	TagSet  taggingRequestTagSet `xml:"TagSet"`
}

type taggingRequestTagSet struct {
	Tags []taggingRequestTag `xml:"Tag"`
}

type taggingRequestTag struct {
	Key   string `xml:"Key"`
	Value string `xml:"Value"`
}

type taggingResponse struct {
	XMLName xml.Name              `xml:"Tagging"`
	TagSet  taggingResponseTagSet `xml:"TagSet"`
}

type taggingResponseTagSet struct {
	Tags []taggingRequestTag `xml:"Tag"`
}

// PutBucketTagging replaces the bucket's tag set with the given tags.
func (c *S3Client) PutBucketTagging(ctx context.Context, creds Credentials, bucket string, tags []BucketTag) error {
	req := taggingRequest{
		XMLNS: "http://s3.amazonaws.com/doc/2006-03-01/",
	}
	for _, t := range tags {
		req.TagSet.Tags = append(req.TagSet.Tags, taggingRequestTag(t))
	}
	body, err := xml.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshaling tagging body: %w", err)
	}
	_, status, err := c.doSignedRequest(ctx, http.MethodPut, "/"+bucket, "tagging", string(body), creds)
	if err != nil {
		return err
	}
	if status != http.StatusOK && status != http.StatusNoContent {
		return fmt.Errorf("put bucket tagging failed (status %d)", status)
	}
	return nil
}

// GetBucketTagging returns the bucket's tag set. If the bucket has no tags,
// returns (nil, nil) so callers can treat absence as a state-removed signal.
func (c *S3Client) GetBucketTagging(ctx context.Context, creds Credentials, bucket string) ([]BucketTag, error) {
	respBody, status, err := c.doSignedRequest(ctx, http.MethodGet, "/"+bucket, "tagging", "", creds)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchTagSet") {
			return nil, nil
		}
		return nil, err
	}
	if status == http.StatusNotFound {
		return nil, nil
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("get bucket tagging failed (status %d)", status)
	}

	var resp taggingResponse
	if err := xml.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("parsing tagging response: %w", err)
	}
	tags := make([]BucketTag, 0, len(resp.TagSet.Tags))
	for _, t := range resp.TagSet.Tags {
		tags = append(tags, BucketTag(t))
	}
	return tags, nil
}

func (c *S3Client) DeleteBucketTagging(ctx context.Context, creds Credentials, bucket string) error {
	_, status, err := c.doSignedRequest(ctx, http.MethodDelete, "/"+bucket, "tagging", "", creds)
	if status == http.StatusNotFound {
		return nil
	}
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchTagSet") {
			return nil
		}
		return err
	}
	if status != http.StatusNoContent && status != http.StatusOK {
		return fmt.Errorf("delete bucket tagging failed (status %d)", status)
	}
	return nil
}

func (c *S3Client) DeleteBucket(ctx context.Context, creds Credentials, bucket string) error {
	_, status, err := c.doSignedRequest(ctx, http.MethodDelete, "/"+bucket, "", "", creds)
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

// PutBucketSubresourceXML replaces a bucket configuration subresource (for
// example "lifecycle" or "replication") with body as given. Acceptance tests
// use it to create configurations the provider doesn't model.
func (c *S3Client) PutBucketSubresourceXML(ctx context.Context, creds Credentials, bucket, subresource, body string) error {
	respBody, statusCode, err := c.doSignedRequest(ctx, "PUT", "/"+bucket, subresource, body, creds)
	if err != nil {
		return fmt.Errorf("put bucket %s: %w", subresource, err)
	}
	if statusCode != 200 {
		return fmt.Errorf("put bucket %s failed (status %d): %s", subresource, statusCode, string(respBody))
	}
	return nil
}

// GetBucketSubresourceXML returns a bucket configuration subresource as raw XML.
func (c *S3Client) GetBucketSubresourceXML(ctx context.Context, creds Credentials, bucket, subresource string) (string, error) {
	body, statusCode, err := c.doSignedRequest(ctx, "GET", "/"+bucket, subresource, "", creds)
	if err != nil {
		return "", fmt.Errorf("get bucket %s: %w", subresource, err)
	}
	if statusCode != 200 {
		return "", fmt.Errorf("get bucket %s failed (status %d): %s", subresource, statusCode, string(body))
	}
	return string(body), nil
}

func (c *S3Client) doSignedRequest(ctx context.Context, method, path, query, body string, creds Credentials) ([]byte, int, error) {
	respBody, statusCode, _, err := c.doSignedRequestRetried(ctx, method, path, query, body, creds)
	return respBody, statusCode, err
}

// doSignedRequestRetried is doSignedRequest that also reports whether the
// request was resent after a transient gateway status, so an earlier attempt
// may have been applied.
func (c *S3Client) doSignedRequestRetried(ctx context.Context, method, path, query, body string, creds Credentials) ([]byte, int, bool, error) {
	var values url.Values
	if query != "" {
		values = url.Values{query: {""}}
	}
	return c.doSignedRequestQuery(ctx, method, path, values, body, creds)
}

// doSignedRequestQuery is doSignedRequestRetried with any query parameters.
// path is unescaped ("/bucket/key"); each segment is URI-encoded.
func (c *S3Client) doSignedRequestQuery(ctx context.Context, method, path string, query url.Values, body string, creds Credentials) ([]byte, int, bool, error) {
	u, err := url.Parse(c.endpoint)
	if err != nil {
		return nil, 0, false, fmt.Errorf("parsing endpoint: %w", err)
	}

	host := u.Host
	payloadHash := sha256Hex([]byte(body))

	path = sigV4EscapePath(path)
	canonicalQueryString := sigV4CanonicalQuery(query)
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"
	if creds.SessionToken != "" {
		signedHeaders += ";x-amz-security-token"
	}

	fullURL := c.endpoint + path
	if canonicalQueryString != "" {
		fullURL += "?" + canonicalQueryString
	}

	backoff := c.transientBackoff()

	var (
		respBody   []byte
		statusCode int
		retried    bool
	)

	for attempt := range transientGatewayMaxAttempts {
		now := time.Now().UTC()
		datestamp := now.Format("20060102")
		amzdate := now.Format("20060102T150405Z")

		// SigV4 canonical headers must be alphabetically sorted by header name.
		canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n", host, payloadHash, amzdate)
		if creds.SessionToken != "" {
			canonicalHeaders += fmt.Sprintf("x-amz-security-token:%s\n", creds.SessionToken)
		}

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

		signingKey := getSignatureKey(creds.SecretKey, datestamp, c.region, s3AWSService)
		signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

		authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
			creds.AccessKey, credentialScope, signedHeaders, signature)

		req, err := http.NewRequestWithContext(ctx, method, fullURL, strings.NewReader(body))
		if err != nil {
			return nil, 0, retried, fmt.Errorf("creating request: %w", err)
		}

		req.Header.Set("Host", host)
		req.Header.Set("X-Amz-Date", amzdate)
		req.Header.Set("X-Amz-Content-Sha256", payloadHash)
		if creds.SessionToken != "" {
			req.Header.Set("X-Amz-Security-Token", creds.SessionToken)
		}
		req.Header.Set("Authorization", authHeader)
		if body != "" {
			req.Header.Set("Content-Type", "application/xml")
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, 0, retried, fmt.Errorf("executing request: %w", err)
		}

		respBody, err = io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
		_ = resp.Body.Close()
		if err != nil {
			return nil, 0, retried, fmt.Errorf("reading response: %w", err)
		}
		statusCode = resp.StatusCode

		if isTransientGatewayStatus(statusCode) && attempt < transientGatewayMaxAttempts-1 {
			if err := sleepCtx(ctx, backoff); err != nil {
				return respBody, statusCode, retried, err
			}
			if backoff < transientGatewayBackoffMax {
				backoff *= 2
			}
			retried = true
			continue
		}
		break
	}

	if statusCode >= 400 {
		var s3Err s3ErrorResponse
		if xmlErr := xml.Unmarshal(respBody, &s3Err); xmlErr == nil && s3Err.Code != "" {
			return respBody, statusCode, retried, fmt.Errorf("%s: %s", s3Err.Code, s3Err.Message)
		}
	}

	return respBody, statusCode, retried, nil
}

// sigV4Escape URI-encodes s as SigV4 requires: everything but unreserved
// characters (A-Z a-z 0-9 - _ . ~) is percent-encoded, space as %20.
func sigV4Escape(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ('A' <= ch && ch <= 'Z') || ('a' <= ch && ch <= 'z') || ('0' <= ch && ch <= '9') || ch == '-' || ch == '_' || ch == '.' || ch == '~' {
			b.WriteByte(ch)
		} else {
			fmt.Fprintf(&b, "%%%02X", ch)
		}
	}
	return b.String()
}

// sigV4EscapePath encodes each segment of an S3 path, keeping the slashes.
func sigV4EscapePath(path string) string {
	segments := strings.Split(path, "/")
	for i, seg := range segments {
		segments[i] = sigV4Escape(seg)
	}
	return strings.Join(segments, "/")
}

// sigV4CanonicalQuery builds the canonical query string: parameters sorted by
// name (then value), names and values encoded, a bare name as "name=".
func sigV4CanonicalQuery(query url.Values) string {
	keys := make([]string, 0, len(query))
	for k := range query {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		vals := append([]string(nil), query[k]...)
		sort.Strings(vals)
		for _, v := range vals {
			parts = append(parts, sigV4Escape(k)+"="+sigV4Escape(v))
		}
	}
	return strings.Join(parts, "&")
}
