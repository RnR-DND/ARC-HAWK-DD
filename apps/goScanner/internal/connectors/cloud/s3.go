package cloud

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/arc-platform/go-scanner/internal/connectors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Connector scans objects in an AWS S3 bucket.
// Config keys: bucket, region, access_key_id, secret_access_key, prefix (optional)
type S3Connector struct {
	client *s3.Client
	bucket string
	prefix string
}

func (c *S3Connector) SourceType() string { return "s3" }

func (c *S3Connector) Connect(ctx context.Context, cfg map[string]any) error {
	c.bucket = fmt.Sprintf("%v", cfg["bucket"])
	c.prefix = fmt.Sprintf("%v", cfg["prefix"])
	if c.prefix == "<nil>" {
		c.prefix = ""
	}
	region := fmt.Sprintf("%v", cfg["region"])
	if region == "<nil>" || region == "" {
		region = "us-east-1"
	}

	var awsCfg aws.Config
	var err error

	akid := fmt.Sprintf("%v", cfg["access_key_id"])
	secret := fmt.Sprintf("%v", cfg["secret_access_key"])
	if akid != "" && akid != "<nil>" {
		awsCfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(akid, secret, "")),
		)
	} else {
		awsCfg, err = config.LoadDefaultConfig(ctx, config.WithRegion(region))
	}
	if err != nil {
		return fmt.Errorf("s3: load config: %w", err)
	}
	c.client = s3.NewFromConfig(awsCfg)
	return nil
}

func (c *S3Connector) Close() error { return nil }

func (c *S3Connector) StreamFields(ctx context.Context) (<-chan connectors.FieldRecord, <-chan error) {
	out := make(chan connectors.FieldRecord, 100)
	errc := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errc)

		paginator := s3.NewListObjectsV2Paginator(c.client, &s3.ListObjectsV2Input{
			Bucket: aws.String(c.bucket),
			Prefix: aws.String(c.prefix),
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				errc <- fmt.Errorf("s3: list objects: %w", err)
				return
			}
			for _, obj := range page.Contents {
				key := aws.ToString(obj.Key)
				if !isTextLike(key) {
					continue
				}
				if err := c.streamObject(ctx, key, out); err != nil {
					continue
				}
			}
		}
	}()
	return out, errc
}

func (c *S3Connector) streamObject(ctx context.Context, key string, out chan<- connectors.FieldRecord) error {
	resp, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // 10MB limit
	if err != nil {
		return err
	}
	text := string(data)
	select {
	case out <- connectors.FieldRecord{
		Value:        text,
		FieldName:    key,
		SourcePath:   fmt.Sprintf("s3://%s/%s", c.bucket, key),
		IsStructured: strings.HasSuffix(strings.ToLower(key), ".csv"),
	}:
	case <-ctx.Done():
	}
	return nil
}

func isTextLike(key string) bool {
	lower := strings.ToLower(key)
	textExts := []string{".txt", ".csv", ".json", ".xml", ".log", ".md", ".yaml", ".yml", ".html", ".htm", ".tsv"}
	for _, ext := range textExts {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}
