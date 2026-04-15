package databases

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/arc-platform/go-scanner/internal/connectors"
)

// FirebaseConnector scans a Firebase Realtime Database via the REST API.
// Config keys: project_id, database_url (optional, defaults to project_id.firebaseio.com),
// access_token (a Google OAuth2 token or Firebase Admin SDK token).
type FirebaseConnector struct {
	databaseURL string
	token       string
}

func (c *FirebaseConnector) SourceType() string { return "firebase" }

func (c *FirebaseConnector) Connect(_ context.Context, config map[string]any) error {
	dbURL := fmt.Sprintf("%v", config["database_url"])
	if dbURL == "" || dbURL == "<nil>" {
		proj := fmt.Sprintf("%v", config["project_id"])
		if proj == "" || proj == "<nil>" {
			return fmt.Errorf("firebase: database_url or project_id is required")
		}
		dbURL = fmt.Sprintf("https://%s-default-rtdb.firebaseio.com", proj)
	}
	token := fmt.Sprintf("%v", config["access_token"])
	if token == "" || token == "<nil>" {
		return fmt.Errorf("firebase: access_token is required")
	}
	c.databaseURL = dbURL
	c.token = token
	return nil
}

func (c *FirebaseConnector) Close() error { return nil }

func (c *FirebaseConnector) StreamFields(ctx context.Context) (<-chan connectors.FieldRecord, <-chan error) {
	out := make(chan connectors.FieldRecord, 100)
	errc := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errc)

		url := fmt.Sprintf("%s/.json?auth=%s&shallow=true", c.databaseURL, c.token)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			errc <- err
			return
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			errc <- err
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var keys map[string]interface{}
		if err := json.Unmarshal(body, &keys); err != nil {
			errc <- fmt.Errorf("firebase: failed to parse root: %w", err)
			return
		}

		for key := range keys {
			c.fetchNode(ctx, key, "/"+key, out)
		}
	}()
	return out, errc
}

func (c *FirebaseConnector) fetchNode(ctx context.Context, key, path string, out chan<- connectors.FieldRecord) {
	url := fmt.Sprintf("%s%s.json?auth=%s", c.databaseURL, path, c.token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var node interface{}
	if err := json.Unmarshal(body, &node); err != nil {
		return
	}
	flattenFirebase(node, key, path, out, ctx)
}

func flattenFirebase(node interface{}, fieldName, sourcePath string, out chan<- connectors.FieldRecord, ctx context.Context) {
	switch v := node.(type) {
	case map[string]interface{}:
		for k, val := range v {
			flattenFirebase(val, k, sourcePath+"/"+k, out, ctx)
		}
	case string:
		if v == "" {
			return
		}
		select {
		case out <- connectors.FieldRecord{
			Value:        v,
			FieldName:    fieldName,
			SourcePath:   sourcePath,
			IsStructured: true,
		}:
		case <-ctx.Done():
		}
	default:
		strVal := fmt.Sprintf("%v", v)
		if strVal == "" || strVal == "<nil>" {
			return
		}
		select {
		case out <- connectors.FieldRecord{
			Value:        strVal,
			FieldName:    fieldName,
			SourcePath:   sourcePath,
			IsStructured: true,
		}:
		case <-ctx.Done():
		}
	}
}
