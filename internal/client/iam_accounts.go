package client

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// IAMAccount is an account as listed by IAM GetRolesForWebIdentity.
type IAMAccount struct {
	Name         string
	ID           string
	CanonicalID  string
	CreationDate string
}

// AccountARN returns the account's root ARN (arn:aws:iam::<id>:root).
func AccountARN(accountID string) string {
	return "arn:aws:iam::" + accountID + ":root"
}

type getRolesForWebIdentityResponse struct {
	IsTruncated bool   `json:"IsTruncated"`
	Marker      string `json:"Marker"`
	Accounts    []struct {
		Name         string `json:"Name"`
		CreationDate string `json:"CreationDate"`
		CanonicalId  string `json:"CanonicalId"`
		Roles        []struct {
			Name string `json:"Name"`
			Arn  string `json:"Arn"`
		} `json:"Roles"`
	} `json:"Accounts"`
}

// ListAccounts returns every account the web identity can assume a role in,
// via IAM GetRolesForWebIdentity. Pages are keyed by role, so one account can
// span several pages; results are merged by account name.
func (c *IAMClient) ListAccounts(ctx context.Context, webIdentityToken string) ([]IAMAccount, error) {
	var accounts []IAMAccount
	index := map[string]int{}
	seenMarkers := map[string]bool{}
	marker := ""

	for {
		params := url.Values{}
		params.Set("Action", "GetRolesForWebIdentity")
		params.Set("WebIdentityToken", webIdentityToken)
		if marker != "" {
			params.Set("Marker", marker)
		}

		body, err := c.doWebIdentityRequest(ctx, params)
		if err != nil {
			return nil, err
		}

		var page getRolesForWebIdentityResponse
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("parsing GetRolesForWebIdentity response: %w", err)
		}

		for _, a := range page.Accounts {
			i, ok := index[a.Name]
			if !ok {
				i = len(accounts)
				index[a.Name] = i
				accounts = append(accounts, IAMAccount{
					Name:         a.Name,
					CanonicalID:  a.CanonicalId,
					CreationDate: a.CreationDate,
				})
			}
			if accounts[i].ID == "" {
				for _, role := range a.Roles {
					if id := accountIDFromARN(role.Arn); id != "" {
						accounts[i].ID = id
						break
					}
				}
			}
		}

		if !page.IsTruncated {
			return accounts, nil
		}
		if page.Marker == "" {
			return nil, fmt.Errorf("GetRolesForWebIdentity response is truncated but has no Marker")
		}
		if seenMarkers[page.Marker] {
			return nil, fmt.Errorf("GetRolesForWebIdentity returned a repeated Marker")
		}
		seenMarkers[page.Marker] = true
		marker = page.Marker
	}
}

// GetAccountByName returns the named account, or nil if it does not exist.
func (c *IAMClient) GetAccountByName(ctx context.Context, webIdentityToken, name string) (*IAMAccount, error) {
	accounts, err := c.ListAccounts(ctx, webIdentityToken)
	if err != nil {
		return nil, err
	}
	for i := range accounts {
		if accounts[i].Name == name {
			return &accounts[i], nil
		}
	}
	return nil, nil
}

// accountIDFromARN extracts the account ID field from an ARN
// (arn:partition:service:region:account-id:resource).
func accountIDFromARN(arn string) string {
	parts := strings.SplitN(arn, ":", 6)
	if len(parts) < 6 || parts[0] != "arn" {
		return ""
	}
	return parts[4]
}

// doWebIdentityRequest sends an unsigned form POST to the IAM endpoint. The
// caller authenticates via the WebIdentityToken parameter. Success bodies are
// JSON; errors are the standard IAM XML error document.
func (c *IAMClient) doWebIdentityRequest(ctx context.Context, params url.Values) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/", strings.NewReader(params.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", contentTypeForm)
	req.Header.Set("Accept", contentTypeJSON)

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
