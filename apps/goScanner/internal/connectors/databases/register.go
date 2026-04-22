package databases

import "github.com/arc-platform/go-scanner/internal/connectors"

func init() {
	connectors.Register("postgresql", func() connectors.Connector { return &PostgresConnector{} })
	connectors.Register("mysql", func() connectors.Connector { return &MySQLConnector{} })
	connectors.Register("mongodb", func() connectors.Connector { return &MongoDBConnector{} })
	connectors.Register("redis", func() connectors.Connector { return &RedisConnector{} })
	connectors.Register("sqlite", func() connectors.Connector { return &SQLiteConnector{} })
	connectors.Register("mssql", func() connectors.Connector { return &MSSQLConnector{} })
	connectors.Register("firebase", func() connectors.Connector { return &FirebaseConnector{} })
	connectors.Register("couchdb", func() connectors.Connector { return &CouchDBConnector{} })
}
