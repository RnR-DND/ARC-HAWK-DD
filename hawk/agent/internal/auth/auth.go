// Package auth provides authentication primitives for the hawk agent.
// This is a stub implementation — wire in a real OIDC/JWT client when deploying.
package auth

import "fmt"

// Client manages authentication tokens for API calls.
type Client struct {
	token string
}

// NewClient creates a new auth client.
func NewClient(token string) *Client {
	return &Client{token: token}
}

// Token returns the current bearer token.
func (c *Client) Token() (string, error) {
	if c.token == "" {
		return "", fmt.Errorf("no auth token configured")
	}
	return c.token, nil
}
