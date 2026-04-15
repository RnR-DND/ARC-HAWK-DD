package warehouses

import "github.com/arc-platform/go-scanner/internal/connectors"

func init() {
	connectors.Register("bigquery", func() connectors.Connector { return &BigQueryConnector{} })
	connectors.Register("snowflake", func() connectors.Connector { return &SnowflakeConnector{} })
	connectors.Register("redshift", func() connectors.Connector { return &RedshiftConnector{} })
}
