package saas

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/arc-platform/go-scanner/internal/connectors"
)

// SalesforceConnector scans Salesforce objects via the REST API.
// Config keys: instance_url, client_id, client_secret, username, password
type SalesforceConnector struct {
	instanceURL string
	accessToken string
	httpClient  *http.Client
}

func (c *SalesforceConnector) SourceType() string { return "salesforce" }

func (c *SalesforceConnector) Connect(ctx context.Context, cfg map[string]any) error {
	c.instanceURL = fmt.Sprintf("%v", cfg["instance_url"])
	if c.instanceURL == "" || c.instanceURL == "<nil>" {
		c.instanceURL = "https://login.salesforce.com"
	}
	clientID := fmt.Sprintf("%v", cfg["client_id"])
	clientSecret := fmt.Sprintf("%v", cfg["client_secret"])
	username := fmt.Sprintf("%v", cfg["username"])
	password := fmt.Sprintf("%v", cfg["password"])

	// Username-password OAuth flow
	form := url.Values{
		"grant_type":    {"password"},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"username":      {username},
		"password":      {password},
	}
	c.httpClient = &http.Client{}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.instanceURL+"/services/oauth2/token",
		strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("salesforce: oauth: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result struct {
		AccessToken string `json:"access_token"`
		InstanceURL string `json:"instance_url"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.AccessToken == "" {
		return fmt.Errorf("salesforce: oauth failed: %s", string(body))
	}
	c.accessToken = result.AccessToken
	c.instanceURL = result.InstanceURL
	return nil
}

func (c *SalesforceConnector) Close() error { return nil }

func (c *SalesforceConnector) StreamFields(ctx context.Context) (<-chan connectors.FieldRecord, <-chan error) {
	out := make(chan connectors.FieldRecord, 100)
	errc := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errc)

		// Scan standard PII-rich objects
		objects := []struct {
			name   string
			fields []string
		}{
			{"Contact", []string{"Name", "Email", "Phone", "MobilePhone", "MailingStreet", "MailingCity"}},
			{"Lead", []string{"Name", "Email", "Phone", "MobilePhone", "Street", "City"}},
			{"Account", []string{"Name", "Phone", "BillingStreet", "BillingCity"}},
		}

		for _, obj := range objects {
			query := fmt.Sprintf("SELECT %s FROM %s LIMIT 10000", strings.Join(obj.fields, ","), obj.name)
			c.querySalesforce(ctx, query, obj.name, obj.fields, out)
		}
	}()
	return out, errc
}

func (c *SalesforceConnector) querySalesforce(ctx context.Context, query, objName string, fields []string, out chan<- connectors.FieldRecord) {
	url_ := fmt.Sprintf("%s/services/data/v58.0/query?q=%s", c.instanceURL, url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url_, nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Records []map[string]interface{} `json:"records"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return
	}
	for i, rec := range result.Records {
		for _, field := range fields {
			val, ok := rec[field]
			if !ok || val == nil {
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
				SourcePath:   fmt.Sprintf("salesforce://%s:row_%d.%s", objName, i, field),
				IsStructured: true,
			}:
			case <-ctx.Done():
				return
			}
		}
	}
}
