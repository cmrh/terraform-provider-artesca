package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

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

	return nil, nil
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
