package saas

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/arc-platform/go-scanner/internal/connectors"
)

// TeamsConnector scans Microsoft Teams messages via the Microsoft Graph API.
// Config keys: tenant_id, client_id, client_secret
type TeamsConnector struct {
	accessToken string
	httpClient  *http.Client
}

func (c *TeamsConnector) SourceType() string { return "ms_teams" }

func (c *TeamsConnector) Connect(ctx context.Context, cfg map[string]any) error {
	tenantID := fmt.Sprintf("%v", cfg["tenant_id"])
	clientID := fmt.Sprintf("%v", cfg["client_id"])
	clientSecret := fmt.Sprintf("%v", cfg["client_secret"])

	if tenantID == "<nil>" || clientID == "<nil>" || clientSecret == "<nil>" {
		return fmt.Errorf("ms_teams: tenant_id, client_id, client_secret are required")
	}

	c.httpClient = &http.Client{Timeout: 30 * time.Second}
	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"scope":         {"https://graph.microsoft.com/.default"},
	}

	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ms_teams: oauth: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.AccessToken == "" {
		return fmt.Errorf("ms_teams: oauth failed: %s", string(body))
	}
	c.accessToken = result.AccessToken
	return nil
}

func (c *TeamsConnector) Close() error { return nil }

func (c *TeamsConnector) StreamFields(ctx context.Context) (<-chan connectors.FieldRecord, <-chan error) {
	out := make(chan connectors.FieldRecord, 100)
	errc := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errc)

		// List teams
		teams, err := c.graphGet(ctx, "https://graph.microsoft.com/v1.0/teams")
		if err != nil {
			errc <- fmt.Errorf("ms_teams: list teams: %w", err)
			return
		}
		teamItems, _ := teams["value"].([]interface{})

		for _, t := range teamItems {
			team, _ := t.(map[string]interface{})
			teamID, _ := team["id"].(string)
			teamName, _ := team["displayName"].(string)
			if teamID == "" {
				continue
			}

			// List channels
			channels, err := c.graphGet(ctx, fmt.Sprintf("https://graph.microsoft.com/v1.0/teams/%s/channels", teamID))
			if err != nil {
				continue
			}
			channelItems, _ := channels["value"].([]interface{})

			for _, ch := range channelItems {
				channel, _ := ch.(map[string]interface{})
				channelID, _ := channel["id"].(string)
				channelName, _ := channel["displayName"].(string)
				if channelID == "" {
					continue
				}

				// List messages
				msgs, err := c.graphGet(ctx, fmt.Sprintf(
					"https://graph.microsoft.com/v1.0/teams/%s/channels/%s/messages?$top=50",
					teamID, channelID))
				if err != nil {
					continue
				}
				msgItems, _ := msgs["value"].([]interface{})

				for _, m := range msgItems {
					msg, _ := m.(map[string]interface{})
					msgID, _ := msg["id"].(string)
					body_, _ := msg["body"].(map[string]interface{})
					content, _ := body_["content"].(string)
					if content == "" {
						continue
					}
					select {
					case out <- connectors.FieldRecord{
						Value:        content,
						FieldName:    "message",
						SourcePath:   fmt.Sprintf("teams://%s/%s/%s", teamName, channelName, msgID),
						IsStructured: false,
					}:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()
	return out, errc
}

func (c *TeamsConnector) graphGet(ctx context.Context, url string) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("graph: parse response: %w", err)
	}
	return result, nil
}
