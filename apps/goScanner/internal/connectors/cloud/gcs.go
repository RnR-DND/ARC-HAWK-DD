package cloud

import (
	"context"
	"fmt"
	"io"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/arc-platform/go-scanner/internal/connectors"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// GCSConnector scans objects in a Google Cloud Storage bucket.
// Config keys: bucket, credentials_json (JSON string or path)
type GCSConnector struct {
	client *storage.Client
	bucket string
	prefix string
}

func (c *GCSConnector) SourceType() string { return "gcs" }

func (c *GCSConnector) Connect(ctx context.Context, cfg map[string]any) error {
	c.bucket = fmt.Sprintf("%v", cfg["bucket"])
	c.prefix = fmt.Sprintf("%v", cfg["prefix"])
	if c.prefix == "<nil>" {
		c.prefix = ""
	}

	var opts []option.ClientOption
	credsJSON := fmt.Sprintf("%v", cfg["credentials_json"])
	if credsJSON != "" && credsJSON != "<nil>" {
		opts = append(opts, option.WithCredentialsJSON([]byte(credsJSON)))
	}

	var err error
	c.client, err = storage.NewClient(ctx, opts...)
	if err != nil {
		return fmt.Errorf("gcs: create client: %w", err)
	}
	return nil
}

func (c *GCSConnector) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

func (c *GCSConnector) StreamFields(ctx context.Context) (<-chan connectors.FieldRecord, <-chan error) {
	out := make(chan connectors.FieldRecord, 100)
	errc := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errc)

		bkt := c.client.Bucket(c.bucket)
		query := &storage.Query{Prefix: c.prefix}
		it := bkt.Objects(ctx, query)

		for {
			attrs, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				errc <- fmt.Errorf("gcs: list objects: %w", err)
				return
			}
			if !isTextLike(attrs.Name) {
				continue
			}
			if err := c.streamObject(ctx, bkt, attrs.Name, out); err != nil {
				continue
			}
		}
	}()
	return out, errc
}

func (c *GCSConnector) streamObject(ctx context.Context, bkt *storage.BucketHandle, name string, out chan<- connectors.FieldRecord) error {
	obj := bkt.Object(name)
	r, err := obj.NewReader(ctx)
	if err != nil {
		return err
	}
	defer r.Close()

	data, err := io.ReadAll(io.LimitReader(r, 10*1024*1024))
	if err != nil {
		return err
	}
	select {
	case out <- connectors.FieldRecord{
		Value:        string(data),
		FieldName:    name,
		SourcePath:   fmt.Sprintf("gs://%s/%s", c.bucket, name),
		IsStructured: strings.HasSuffix(strings.ToLower(name), ".csv"),
	}:
	case <-ctx.Done():
	}
	return nil
}
