package queues

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/arc-platform/go-scanner/internal/connectors"
	"github.com/segmentio/kafka-go"
)

// KafkaConnector consumes messages from Kafka topics for a configurable window.
// Config keys: brokers (comma-separated), topics (comma-separated), group_id,
//              window_seconds (default 60), tls (bool)
type KafkaConnector struct {
	readers []*kafka.Reader
}

func (c *KafkaConnector) SourceType() string { return "kafka" }

func (c *KafkaConnector) Connect(_ context.Context, cfg map[string]any) error {
	brokersStr := fmt.Sprintf("%v", cfg["brokers"])
	if brokersStr == "" || brokersStr == "<nil>" {
		return fmt.Errorf("kafka: brokers is required")
	}
	topicsStr := fmt.Sprintf("%v", cfg["topics"])
	if topicsStr == "" || topicsStr == "<nil>" {
		return fmt.Errorf("kafka: topics is required")
	}
	groupID := fmt.Sprintf("%v", cfg["group_id"])
	if groupID == "" || groupID == "<nil>" {
		groupID = "arc-hawk-scanner"
	}

	brokers := strings.Split(brokersStr, ",")
	topics := strings.Split(topicsStr, ",")

	for _, topic := range topics {
		topic = strings.TrimSpace(topic)
		r := kafka.NewReader(kafka.ReaderConfig{
			Brokers:     brokers,
			Topic:       topic,
			GroupID:     groupID,
			StartOffset: kafka.LastOffset,
			MaxBytes:    10e6,
		})
		c.readers = append(c.readers, r)
	}
	return nil
}

func (c *KafkaConnector) Close() error {
	for _, r := range c.readers {
		r.Close()
	}
	return nil
}

func (c *KafkaConnector) StreamFields(ctx context.Context) (<-chan connectors.FieldRecord, <-chan error) {
	out := make(chan connectors.FieldRecord, 100)
	errc := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errc)

		deadline := time.Now().Add(60 * time.Second)
		scanCtx, cancel := context.WithDeadline(ctx, deadline)
		defer cancel()

		for _, r := range c.readers {
			topic := r.Config().Topic
			for {
				msg, err := r.ReadMessage(scanCtx)
				if err != nil {
					break // deadline or error
				}
				value := string(msg.Value)
				if value == "" {
					continue
				}
				select {
				case out <- connectors.FieldRecord{
					Value:        value,
					FieldName:    "message",
					SourcePath:   fmt.Sprintf("kafka://%s/partition/%d/offset/%d", topic, msg.Partition, msg.Offset),
					IsStructured: false,
				}:
				case <-scanCtx.Done():
					return
				}
			}
		}
	}()
	return out, errc
}
