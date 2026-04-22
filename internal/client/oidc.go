package client

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// OIDC constants.
const (
	oidcHTTPTimeout    = 30 * time.Second
	oidcTokenPreExpiry = 30 * time.Second
)

type OIDCTokenSource struct {
	tokenURL   string
	clientID   string
	username   string
	password   string
	httpClient *http.Client

	mu          sync.Mutex
	cachedToken string
	tokenExpiry time.Time
}

type oidcTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

func NewOIDCTokenSource(oidcURL, realm, clientID, username, password string, insecureSkipVerify bool) *OIDCTokenSource {
	tokenURL := fmt.Sprintf("%s/auth/realms/%s/protocol/openid-connect/token", strings.TrimRight(oidcURL, "/"), url.PathEscape(realm))

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecureSkipVerify,
		},
	}

	return &OIDCTokenSource{
		tokenURL: tokenURL,
		clientID: clientID,
		username: username,
		password: password,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   oidcHTTPTimeout,
		},
	}
}

func (s *OIDCTokenSource) Token(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cachedToken != "" && time.Now().Before(s.tokenExpiry.Add(-oidcTokenPreExpiry)) {
		return s.cachedToken, nil
	}

	data := url.Values{
		"grant_type": {"password"},
		"client_id":  {s.clientID},
		"username":   {s.username},
		"password":   {s.password},
		"scope":      {"openid"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("creating OIDC token request: %w", err)
	}
	req.Header.Set("Content-Type", contentTypeForm)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("requesting OIDC token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return "", fmt.Errorf("reading OIDC token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OIDC token request failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp oidcTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("parsing OIDC token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("OIDC token response did not contain an access_token")
	}

	s.cachedToken = tokenResp.AccessToken
	s.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return s.cachedToken, nil
}

// InstanceIDs extracts the instanceIds claim from the cached JWT token.
func (s *OIDCTokenSource) InstanceIDs(ctx context.Context) ([]string, error) {
	token, err := s.Token(ctx)
	if err != nil {
		return nil, err
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT token format")
	}

	payload := parts[1]
	// Add padding if needed
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}

	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("decoding JWT payload: %w", err)
	}

	var claims struct {
		InstanceIDs []string `json:"instanceIds"`
	}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil, fmt.Errorf("parsing JWT claims: %w", err)
	}

	return claims.InstanceIDs, nil
}
