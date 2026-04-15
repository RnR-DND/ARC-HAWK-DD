package queues

import "github.com/arc-platform/go-scanner/internal/connectors"

func init() {
	connectors.Register("kafka", func() connectors.Connector { return &KafkaConnector{} })
	connectors.Register("kinesis", func() connectors.Connector { return &KinesisConnector{} })
}
