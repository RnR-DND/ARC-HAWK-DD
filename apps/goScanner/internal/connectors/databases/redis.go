package databases

import (
	"context"
	"fmt"

	"github.com/arc-platform/go-scanner/internal/connectors"
	"github.com/redis/go-redis/v9"
)

// RedisConnector scans Redis string keys.
type RedisConnector struct{ client *redis.Client }

func (c *RedisConnector) SourceType() string { return "redis" }

func (c *RedisConnector) Connect(ctx context.Context, config map[string]any) error {
	addr := fmt.Sprintf("%v:%v", config["host"], config["port"])
	c.client = redis.NewClient(&redis.Options{Addr: addr})
	return c.client.Ping(ctx).Err()
}

func (c *RedisConnector) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

func (c *RedisConnector) StreamFields(ctx context.Context) (<-chan connectors.FieldRecord, <-chan error) {
	out := make(chan connectors.FieldRecord, 100)
	errc := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errc)
		var cursor uint64
		for {
			keys, next, err := c.client.Scan(ctx, cursor, "*", 100).Result()
			if err != nil {
				errc <- err
				return
			}
			for _, key := range keys {
				val, err := c.client.Get(ctx, key).Result()
				if err != nil {
					continue
				}
				select {
				case out <- connectors.FieldRecord{
					Value:        val,
					FieldName:    key,
					SourcePath:   "redis:" + key,
					IsStructured: false,
				}:
				case <-ctx.Done():
					return
				}
			}
			cursor = next
			if cursor == 0 {
				break
			}
		}
	}()
	return out, errc
}
