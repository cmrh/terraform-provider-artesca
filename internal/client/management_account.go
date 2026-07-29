package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func (c *ManagementClient) GetAccount(ctx context.Context, name string) (*User, error) {
	overlay, err := c.LookupInOverlay(ctx, func(o *ConfigOverlay) bool {
		for i := range o.Users {
			if o.Users[i].AccountName == name || o.Users[i].UserName == name {
				return true
			}
		}
		return false
	})
	if err != nil {
		return nil, err
	}

	for i := range overlay.Users {
		if overlay.Users[i].AccountName == name || overlay.Users[i].UserName == name {
			return &overlay.Users[i], nil
		}
	}

	return nil, nil
}

type createUserRequest struct {
	UserName string `json:"userName"`
	Email    string `json:"email,omitempty"`
}

func (c *ManagementClient) CreateAccount(ctx context.Context, userName, email string) (*User, error) {
	c.overlayMu.Lock()
	defer c.overlayMu.Unlock()

	path := fmt.Sprintf("/config/%s/user", url.PathEscape(c.InstanceID))
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
	c.overlayMu.Lock()
	defer c.overlayMu.Unlock()

	path := fmt.Sprintf("/config/%s/user?accountName=%s&roleName=%s",
		url.PathEscape(c.InstanceID),
		url.QueryEscape(accountName),
		url.QueryEscape("scality-internal/storage-manager-role"))
	body, status, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	if status == http.StatusNotFound {
		return nil
	}
	if status != http.StatusNoContent && status != http.StatusOK {
		return fmt.Errorf("delete account failed (status %d): %s", status, string(body))
	}

	return nil
}

func (c *ManagementClient) GenerateAccountKey(ctx context.Context, accountName string) (*User, error) {
	c.overlayMu.Lock()
	defer c.overlayMu.Unlock()

	path := fmt.Sprintf("/config/%s/user/%s/key", url.PathEscape(c.InstanceID), url.PathEscape(accountName))
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
