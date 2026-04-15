package warehouses

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"
	"github.com/arc-platform/go-scanner/internal/connectors"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// BigQueryConnector scans BigQuery datasets and tables.
// Config keys: project_id, credentials_json (optional), dataset (optional, scans all if empty)
type BigQueryConnector struct {
	client    *bigquery.Client
	projectID string
	dataset   string
}

func (c *BigQueryConnector) SourceType() string { return "bigquery" }

func (c *BigQueryConnector) Connect(ctx context.Context, cfg map[string]any) error {
	c.projectID = fmt.Sprintf("%v", cfg["project_id"])
	if c.projectID == "" || c.projectID == "<nil>" {
		return fmt.Errorf("bigquery: project_id is required")
	}
	c.dataset = fmt.Sprintf("%v", cfg["dataset"])
	if c.dataset == "<nil>" {
		c.dataset = ""
	}

	var opts []option.ClientOption
	credsJSON := fmt.Sprintf("%v", cfg["credentials_json"])
	if credsJSON != "" && credsJSON != "<nil>" {
		opts = append(opts, option.WithCredentialsJSON([]byte(credsJSON)))
	}

	var err error
	c.client, err = bigquery.NewClient(ctx, c.projectID, opts...)
	return err
}

func (c *BigQueryConnector) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

func (c *BigQueryConnector) StreamFields(ctx context.Context) (<-chan connectors.FieldRecord, <-chan error) {
	out := make(chan connectors.FieldRecord, 100)
	errc := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errc)

		datasets, err := c.listDatasets(ctx)
		if err != nil {
			errc <- err
			return
		}
		for _, ds := range datasets {
			if c.dataset != "" && ds != c.dataset {
				continue
			}
			if err := c.scanDataset(ctx, ds, out); err != nil {
				continue
			}
		}
	}()
	return out, errc
}

func (c *BigQueryConnector) listDatasets(ctx context.Context) ([]string, error) {
	var names []string
	it := c.client.Datasets(ctx)
	for {
		ds, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("bigquery: list datasets: %w", err)
		}
		names = append(names, ds.DatasetID)
	}
	return names, nil
}

func (c *BigQueryConnector) scanDataset(ctx context.Context, datasetID string, out chan<- connectors.FieldRecord) error {
	ds := c.client.Dataset(datasetID)
	it := ds.Tables(ctx)
	for {
		tbl, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		if err := c.scanTable(ctx, datasetID, tbl.TableID, out); err != nil {
			continue
		}
	}
	return nil
}

func (c *BigQueryConnector) scanTable(ctx context.Context, datasetID, tableID string, out chan<- connectors.FieldRecord) error {
	q := c.client.Query(fmt.Sprintf("SELECT * FROM `%s.%s.%s` LIMIT 10000", c.projectID, datasetID, tableID))
	it, err := q.Read(ctx)
	if err != nil {
		return err
	}
	for {
		var row []bigquery.Value
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		schema := it.Schema
		for i, val := range row {
			if val == nil || i >= len(schema) {
				continue
			}
			strVal := fmt.Sprintf("%v", val)
			if strVal == "" {
				continue
			}
			select {
			case out <- connectors.FieldRecord{
				Value:        strVal,
				FieldName:    schema[i].Name,
				SourcePath:   fmt.Sprintf("%s.%s.%s", datasetID, tableID, schema[i].Name),
				IsStructured: true,
			}:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return nil
}
