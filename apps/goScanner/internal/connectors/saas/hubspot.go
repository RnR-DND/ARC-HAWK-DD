package saas

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/arc-platform/go-scanner/internal/connectors"
)

// HubSpotConnector scans HubSpot CRM contacts, companies, and deals via REST API v3.
// Config keys: api_key (Private App access token, pat-...)
type HubSpotConnector struct {
	apiKey     string
	httpClient *http.Client
}

func (c *HubSpotConnector) SourceType() string { return "hubspot" }

func (c *HubSpotConnector) Connect(_ context.Context, cfg map[string]any) error {
	c.apiKey = fmt.Sprintf("%v", cfg["api_key"])
	if c.apiKey == "" || c.apiKey == "<nil>" {
		return fmt.Errorf("hubspot: api_key is required (Private App access token)")
	}
	c.httpClient = &http.Client{}
	return nil
}

func (c *HubSpotConnector) Close() error { return nil }

func (c *HubSpotConnector) StreamFields(ctx context.Context) (<-chan connectors.FieldRecord, <-chan error) {
	out := make(chan connectors.FieldRecord, 100)
	errc := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errc)

		endpoints := []struct {
			path       string
			objectType string
			fields     []string
		}{
			{"/crm/v3/objects/contacts", "contact", []string{"email", "firstname", "lastname", "phone", "mobilephone", "address"}},
			{"/crm/v3/objects/companies", "company", []string{"name", "phone", "address", "city"}},
		}

		for _, ep := range endpoints {
			c.fetchObjects(ctx, ep.path, ep.objectType, ep.fields, out)
		}
	}()
	return out, errc
}

func (c *HubSpotConnector) fetchObjects(ctx context.Context, path, objectType string, fields []string, out chan<- connectors.FieldRecord) {
	after := ""
	limit := 100
	for {
		url := fmt.Sprintf("https://api.hubapi.com%s?limit=%d&properties=%v",
			path, limit, joinFields(fields))
		if after != "" {
			url += "&after=" + after
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var result struct {
			Results []struct {
				ID         string                 `json:"id"`
				Properties map[string]interface{} `json:"properties"`
			} `json:"results"`
			Paging *struct {
				Next *struct {
					After string `json:"after"`
				} `json:"next"`
			} `json:"paging"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return
		}

		for _, obj := range result.Results {
			for field, val := range obj.Properties {
				if val == nil {
					continue
				}
				strVal := fmt.Sprintf("%v", val)
				if strVal == "" {
					continue
				}
				select {
				case out <- connectors.FieldRecord{
					Value:        strVal,
					FieldName:    field,
					SourcePath:   fmt.Sprintf("hubspot://%s/%s.%s", objectType, obj.ID, field),
					IsStructured: true,
				}:
				case <-ctx.Done():
					return
				}
			}
		}

		if result.Paging == nil || result.Paging.Next == nil {
			break
		}
		after = result.Paging.Next.After
	}
}

func joinFields(fields []string) string {
	result := ""
	for i, f := range fields {
		if i > 0 {
			result += ","
		}
		result += f
	}
	return result
}
