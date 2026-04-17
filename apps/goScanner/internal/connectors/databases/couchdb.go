package databases

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/arc-platform/go-scanner/internal/connectors"
)

// CouchDBConnector scans a CouchDB instance via its HTTP REST API.
type CouchDBConnector struct {
	baseURL string
	client  *http.Client
}

func (c *CouchDBConnector) SourceType() string { return "couchdb" }

func (c *CouchDBConnector) Connect(_ context.Context, config map[string]any) error {
	host := cfgString(config, "host")
	port := cfgString(config, "port")
	if port == "" {
		port = "5984"
	}
	user := cfgString(config, "user", "username")
	pass := cfgString(config, "password")
	proto := "http"
	if tls, ok := config["tls"].(bool); ok && tls {
		proto = "https"
	}
	if user != "" {
		c.baseURL = fmt.Sprintf("%s://%s:%s@%s:%s", proto, user, pass, host, port)
	} else {
		c.baseURL = fmt.Sprintf("%s://%s:%s", proto, host, port)
	}
	c.client = &http.Client{}

	resp, err := c.client.Get(c.baseURL + "/")
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *CouchDBConnector) Close() error { return nil }

func (c *CouchDBConnector) StreamFields(ctx context.Context) (<-chan connectors.FieldRecord, <-chan error) {
	out := make(chan connectors.FieldRecord, 100)
	errc := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errc)

		dbs, err := c.listDatabases(ctx)
		if err != nil {
			errc <- err
			return
		}
		systemDBs := map[string]bool{"_replicator": true, "_users": true, "_global_changes": true}
		for _, db := range dbs {
			if systemDBs[db] {
				continue
			}
			if err := c.scanDatabase(ctx, db, out); err != nil {
				continue
			}
		}
	}()
	return out, errc
}

func (c *CouchDBConnector) listDatabases(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/_all_dbs", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var dbs []string
	if err := json.Unmarshal(body, &dbs); err != nil {
		return nil, fmt.Errorf("couchdb: parse _all_dbs: %w", err)
	}
	return dbs, nil
}

func (c *CouchDBConnector) scanDatabase(ctx context.Context, db string, out chan<- connectors.FieldRecord) error {
	url := fmt.Sprintf("%s/%s/_all_docs?include_docs=true&limit=10000", c.baseURL, db)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Rows []struct {
			Doc map[string]interface{} `json:"doc"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("couchdb: parse %s docs: %w", db, err)
	}

	for _, row := range result.Rows {
		for key, val := range row.Doc {
			if key == "_id" || key == "_rev" {
				continue
			}
			strVal := fmt.Sprintf("%v", val)
			if strVal == "" || strVal == "<nil>" {
				continue
			}
			select {
			case out <- connectors.FieldRecord{
				Value:        strVal,
				FieldName:    key,
				SourcePath:   fmt.Sprintf("%s.%s", db, key),
				IsStructured: true,
			}:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return nil
}
