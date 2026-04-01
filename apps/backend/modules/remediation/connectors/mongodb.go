package connectors

import (
	"context"
	"fmt"
	"net/url"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDBConnector implements remediation for MongoDB databases
type MongoDBConnector struct {
	client *mongo.Client
	config map[string]interface{}
}

// Connect establishes connection to MongoDB
func (c *MongoDBConnector) Connect(ctx context.Context, config map[string]interface{}) error {
	// Build connection URI
	uri := fmt.Sprintf("mongodb://%s:%s@%s:%v/%s",
		getString(config, "username", ""),
		getString(config, "password", ""),
		getString(config, "host", "localhost"),
		getInt(config, "port", 27017),
		getString(config, "database", "admin"),
	)

	// Add authentication database if specified
	if authDB := getString(config, "auth_database", ""); authDB != "" {
		uri += "?authSource=" + url.QueryEscape(authDB)
	}

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		client.Disconnect(ctx)
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	c.client = client
	c.config = config
	return nil
}

// Close closes the MongoDB connection
func (c *MongoDBConnector) Close() error {
	if c.client != nil {
		ctx := context.Background()
		return c.client.Disconnect(ctx)
	}
	return nil
}

// Mask redacts PII in MongoDB document
func (c *MongoDBConnector) Mask(ctx context.Context, location string, fieldName string, recordID string) error {
	if c.client == nil {
		return fmt.Errorf("MongoDB client not connected")
	}

	db := c.client.Database(getString(c.config, "database", "admin"))
	collection := db.Collection(location)

	// Create filter - try ObjectID first, fallback to string
	filter := bson.M{"_id": recordID}

	// Update with masked value
	update := bson.M{"$set": bson.M{fieldName: "***REDACTED***"}}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to mask field in MongoDB: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("no document found with id: %s", recordID)
	}

	return nil
}

// Delete removes MongoDB document
func (c *MongoDBConnector) Delete(ctx context.Context, location string, recordID string) error {
	if c.client == nil {
		return fmt.Errorf("MongoDB client not connected")
	}

	db := c.client.Database(getString(c.config, "database", "admin"))
	collection := db.Collection(location)

	filter := bson.M{"_id": recordID}

	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete document from MongoDB: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("no document found with id: %s", recordID)
	}

	return nil
}

// Encrypt encrypts PII in MongoDB document
func (c *MongoDBConnector) Encrypt(ctx context.Context, location string, fieldName string, recordID string, encryptionKey string) error {
	if c.client == nil {
		return fmt.Errorf("MongoDB client not connected")
	}

	// Get original value
	originalValue, err := c.GetOriginalValue(ctx, location, fieldName, recordID)
	if err != nil {
		return err
	}

	// Encrypt with AES-GCM
	encryptedValue, err := encryptAESGCM(encryptionKey, originalValue)
	if err != nil {
		return fmt.Errorf("failed to encrypt value: %w", err)
	}

	db := c.client.Database(getString(c.config, "database", "admin"))
	collection := db.Collection(location)

	filter := bson.M{"_id": recordID}
	update := bson.M{"$set": bson.M{fieldName: encryptedValue}}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to encrypt field in MongoDB: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("no document found with id: %s", recordID)
	}

	return nil
}

// GetOriginalValue retrieves original MongoDB document value
func (c *MongoDBConnector) GetOriginalValue(ctx context.Context, location string, fieldName string, recordID string) (string, error) {
	if c.client == nil {
		return "", fmt.Errorf("MongoDB client not connected")
	}

	db := c.client.Database(getString(c.config, "database", "admin"))
	collection := db.Collection(location)

	filter := bson.M{"_id": recordID}
	projection := bson.M{fieldName: 1}

	var result bson.M
	err := collection.FindOne(ctx, filter, options.FindOne().SetProjection(projection)).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "", fmt.Errorf("no document found with id: %s", recordID)
		}
		return "", fmt.Errorf("failed to get original value from MongoDB: %w", err)
	}

	// Extract field value
	if value, ok := result[fieldName]; ok {
		return fmt.Sprintf("%v", value), nil
	}

	return "", fmt.Errorf("field %s not found in document", fieldName)
}

// RestoreValue restores original MongoDB document value
func (c *MongoDBConnector) RestoreValue(ctx context.Context, location string, fieldName string, recordID string, originalValue string) error {
	if c.client == nil {
		return fmt.Errorf("MongoDB client not connected")
	}

	db := c.client.Database(getString(c.config, "database", "admin"))
	collection := db.Collection(location)

	filter := bson.M{"_id": recordID}
	update := bson.M{"$set": bson.M{fieldName: originalValue}}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to restore value in MongoDB: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("no document found with id: %s", recordID)
	}

	return nil
}

// Helper functions
func getString(config map[string]interface{}, key string, defaultValue string) string {
	if val, ok := config[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

func getInt(config map[string]interface{}, key string, defaultValue int) int {
	if val, ok := config[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case string:
			// Try to parse string to int
			var i int
			fmt.Sscanf(v, "%d", &i)
			return i
		}
	}
	return defaultValue
}
