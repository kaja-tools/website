package users

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client provides access to the users service
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// GetUserRequest is the request to get a user
type GetUserRequest struct {
	ID string `json:"id"`
}

// User represents a user from the users service
type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GetUserResponse is the response from GetUser
type GetUserResponse struct {
	User *User `json:"user"`
}

// TwirpError represents an error response from Twirp
type TwirpError struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
}

// NewClient creates a new users service client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// UserExists checks if a user exists in the users service
func (c *Client) UserExists(ctx context.Context, userID string) (bool, error) {
	reqBody := GetUserRequest{ID: userID}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return false, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + "/twirp/Users/GetUser"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to call users service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	// Parse error response
	var twirpErr TwirpError
	if err := json.NewDecoder(resp.Body).Decode(&twirpErr); err != nil {
		return false, fmt.Errorf("users service returned status %d", resp.StatusCode)
	}

	if twirpErr.Code == "not_found" {
		return false, nil
	}

	return false, fmt.Errorf("users service error: %s - %s", twirpErr.Code, twirpErr.Msg)
}
