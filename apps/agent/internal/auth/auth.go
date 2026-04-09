package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/arc/hawk-agent/internal/config"
	"go.uber.org/zap"
)

// Client manages Keycloak client-credentials token lifecycle.
type Client struct {
	cfg        *config.Config
	logger     *zap.Logger
	httpClient *http.Client

	mu           sync.RWMutex
	accessToken  string
	expiresAt    time.Time
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

// NewClient returns a new Keycloak auth client.
func NewClient(cfg *config.Config, logger *zap.Logger) *Client {
	return &Client{
		cfg:    cfg,
		logger: logger,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Token returns a valid access token, refreshing if expired or nearly expired.
func (c *Client) Token() (string, error) {
	c.mu.RLock()
	if c.accessToken != "" && time.Now().Before(c.expiresAt.Add(-30*time.Second)) {
		tok := c.accessToken
		c.mu.RUnlock()
		return tok, nil
	}
	c.mu.RUnlock()

	return c.refresh()
}

func (c *Client) refresh() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock.
	if c.accessToken != "" && time.Now().Before(c.expiresAt.Add(-30*time.Second)) {
		return c.accessToken, nil
	}

	tokenURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token",
		strings.TrimRight(c.cfg.Auth.KeycloakURL, "/"),
		c.cfg.Auth.Realm,
	)

	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {c.cfg.Auth.ClientID},
		"client_secret": {c.cfg.Auth.ClientSecret},
	}

	resp, err := c.httpClient.PostForm(tokenURL, form)
	if err != nil {
		return "", fmt.Errorf("keycloak token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read keycloak response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("keycloak returned %d: %s", resp.StatusCode, string(body))
	}

	var tok tokenResponse
	if err := json.Unmarshal(body, &tok); err != nil {
		return "", fmt.Errorf("parse keycloak token: %w", err)
	}

	c.accessToken = tok.AccessToken
	c.expiresAt = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)

	c.logger.Debug("keycloak token refreshed",
		zap.Int("expires_in_sec", tok.ExpiresIn),
	)

	return c.accessToken, nil
}
