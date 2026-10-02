package client

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type assumeRoleWithWebIdentityResponse struct {
	XMLName xml.Name `xml:"AssumeRoleWithWebIdentityResponse"`
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
	} `xml:"AssumeRoleWithWebIdentityResult"`
}

// AssumeRoleWithWebIdentity exchanges an OIDC token for temporary credentials
// on roleArn. The request is unsigned; the token authenticates it.
func (c *STSClient) AssumeRoleWithWebIdentity(ctx context.Context, webIdentityToken, roleArn, sessionName string, durationSeconds int64) (*AssumedRoleCredentials, error) {
	params := url.Values{
		"Action":           {"AssumeRoleWithWebIdentity"},
		"Version":          {stsAPIVersion},
		"RoleArn":          {roleArn},
		"RoleSessionName":  {sessionName},
		"WebIdentityToken": {webIdentityToken},
	}
	if durationSeconds > 0 {
		params.Set("DurationSeconds", fmt.Sprintf("%d", durationSeconds))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/", strings.NewReader(params.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", contentTypeForm)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var stsErr stsErrorResponse
		if xmlErr := xml.Unmarshal(body, &stsErr); xmlErr == nil && stsErr.Error.Code != "" {
			return nil, fmt.Errorf("assume role with web identity: %s: %s", stsErr.Error.Code, stsErr.Error.Message)
		}
		return nil, fmt.Errorf("assume role with web identity failed (status %d): %s", resp.StatusCode, string(body))
	}

	var out assumeRoleWithWebIdentityResponse
	if err := xml.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("parsing assume role with web identity response: %w", err)
	}

	exp, err := time.Parse(time.RFC3339, out.Result.Credentials.Expiration)
	if err != nil {
		return nil, fmt.Errorf("parsing assume role with web identity expiration %q: %w", out.Result.Credentials.Expiration, err)
	}

	return &AssumedRoleCredentials{
		AccessKeyID:     out.Result.Credentials.AccessKeyID,
		SecretAccessKey: out.Result.Credentials.SecretAccessKey,
		SessionToken:    out.Result.Credentials.SessionToken,
		Expiration:      exp,
		AssumedRoleID:   out.Result.AssumedRoleUser.AssumedRoleID,
		AssumedRoleArn:  out.Result.AssumedRoleUser.Arn,
	}, nil
}

// DeriveSTSEndpointFromManagement derives the STS endpoint from the management
// endpoint ("management." -> "sts."), for when no S3 endpoint is configured.
func DeriveSTSEndpointFromManagement(managementEndpoint string) (string, error) {
	u, err := url.Parse(managementEndpoint)
	if err != nil {
		return "", fmt.Errorf("parsing management endpoint: %w", err)
	}

	host := u.Hostname()
	port := u.Port()

	if !strings.HasPrefix(host, "management.") {
		return "", fmt.Errorf("cannot derive STS endpoint: management endpoint hostname %q does not start with 'management.'", host)
	}

	stsHost := "sts." + strings.TrimPrefix(host, "management.")
	if port != "" {
		stsHost = stsHost + ":" + port
	}

	u.Host = stsHost
	u.Path = ""
	return u.String(), nil
}
