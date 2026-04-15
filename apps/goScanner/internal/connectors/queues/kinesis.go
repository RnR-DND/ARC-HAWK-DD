package queues

import (
	"context"
	"fmt"
	"time"

	"github.com/arc-platform/go-scanner/internal/connectors"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
)

// KinesisConnector consumes records from AWS Kinesis streams for a configurable window.
// Config keys: stream_name, region, access_key_id, secret_access_key, window_seconds (default 60)
type KinesisConnector struct {
	client     *kinesis.Client
	streamName string
}

func (c *KinesisConnector) SourceType() string { return "kinesis" }

func (c *KinesisConnector) Connect(ctx context.Context, cfg map[string]any) error {
	c.streamName = fmt.Sprintf("%v", cfg["stream_name"])
	if c.streamName == "" || c.streamName == "<nil>" {
		return fmt.Errorf("kinesis: stream_name is required")
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
		awsCfg, err = awsconfig.LoadDefaultConfig(ctx,
			awsconfig.WithRegion(region),
			awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(akid, secret, "")),
		)
	} else {
		awsCfg, err = awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	}
	if err != nil {
		return fmt.Errorf("kinesis: load config: %w", err)
	}
	c.client = kinesis.NewFromConfig(awsCfg)
	return nil
}

func (c *KinesisConnector) Close() error { return nil }

func (c *KinesisConnector) StreamFields(ctx context.Context) (<-chan connectors.FieldRecord, <-chan error) {
	out := make(chan connectors.FieldRecord, 100)
	errc := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errc)

		deadline := time.Now().Add(60 * time.Second)
		scanCtx, cancel := context.WithDeadline(ctx, deadline)
		defer cancel()

		shards, err := c.client.ListShards(scanCtx, &kinesis.ListShardsInput{
			StreamName: aws.String(c.streamName),
		})
		if err != nil {
			errc <- fmt.Errorf("kinesis: list shards: %w", err)
			return
		}

		for _, shard := range shards.Shards {
			iterOut, err := c.client.GetShardIterator(scanCtx, &kinesis.GetShardIteratorInput{
				StreamName:        aws.String(c.streamName),
				ShardId:           shard.ShardId,
				ShardIteratorType: types.ShardIteratorTypeLatest,
			})
			if err != nil {
				continue
			}
			iter := iterOut.ShardIterator
			for iter != nil {
				records, err := c.client.GetRecords(scanCtx, &kinesis.GetRecordsInput{
					ShardIterator: iter,
					Limit:         aws.Int32(100),
				})
				if err != nil {
					break
				}
				for _, rec := range records.Records {
					value := string(rec.Data)
					if value == "" {
						continue
					}
					select {
					case out <- connectors.FieldRecord{
						Value:        value,
						FieldName:    "record",
						SourcePath:   fmt.Sprintf("kinesis://%s/%s", c.streamName, aws.ToString(shard.ShardId)),
						IsStructured: false,
					}:
					case <-scanCtx.Done():
						return
					}
				}
				iter = records.NextShardIterator
				if len(records.Records) == 0 {
					break
				}
			}
		}
	}()
	return out, errc
}
