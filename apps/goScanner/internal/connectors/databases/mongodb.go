package databases

import (
	"context"
	"fmt"

	"github.com/arc-platform/go-scanner/internal/connectors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDBConnector scans a MongoDB instance across all databases and collections.
type MongoDBConnector struct {
	client     *mongo.Client
	sampleSize int
}

func (c *MongoDBConnector) SourceType() string { return "mongodb" }

func (c *MongoDBConnector) Connect(ctx context.Context, config map[string]any) error {
	uri := cfgString(config, "uri", "connection_string")
	if uri == "" {
		host := cfgString(config, "host")
		port := cfgString(config, "port")
		if port == "" {
			port = "27017"
		}
		user := cfgString(config, "user", "username")
		pass := cfgString(config, "password")
		if user != "" {
			uri = fmt.Sprintf("mongodb://%s:%s@%s:%s", user, pass, host, port)
		} else {
			uri = fmt.Sprintf("mongodb://%s:%s", host, port)
		}
	}
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return err
	}
	c.client = client
	c.sampleSize = cfgInt(config, "sample_size", 1000, 50000)
	return client.Ping(ctx, nil)
}

func (c *MongoDBConnector) Close() error {
	if c.client != nil {
		return c.client.Disconnect(context.Background())
	}
	return nil
}

func (c *MongoDBConnector) StreamFields(ctx context.Context) (<-chan connectors.FieldRecord, <-chan error) {
	out := make(chan connectors.FieldRecord, 100)
	errc := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errc)

		dbNames, err := c.client.ListDatabaseNames(ctx, bson.M{})
		if err != nil {
			errc <- err
			return
		}
		systemDBs := map[string]bool{"admin": true, "local": true, "config": true}
		for _, dbName := range dbNames {
			if systemDBs[dbName] {
				continue
			}
			db := c.client.Database(dbName)
			colls, err := db.ListCollectionNames(ctx, bson.M{})
			if err != nil {
				continue
			}
			for _, collName := range colls {
				coll := db.Collection(collName)
				cursor, err := coll.Find(ctx, bson.M{}, options.Find().SetLimit(int64(c.sampleSize)))
				if err != nil {
					continue
				}
				for cursor.Next(ctx) {
					var doc bson.M
					if err := cursor.Decode(&doc); err != nil {
						continue
					}
					flattenBSON(doc, fmt.Sprintf("%s.%s", dbName, collName), out, ctx)
				}
				cursor.Close(ctx)
			}
		}
	}()
	return out, errc
}

func flattenBSON(doc bson.M, prefix string, out chan<- connectors.FieldRecord, ctx context.Context) {
	for key, val := range doc {
		if key == "_id" {
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
			SourcePath:   fmt.Sprintf("%s.%s", prefix, key),
			IsStructured: true,
		}:
		case <-ctx.Done():
			return
		}
	}
}
