package service

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/arc-platform/backend/modules/shared/infrastructure/encryption"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type TestConnectionService struct {
	pgRepo     *persistence.PostgresRepository
	encryption *encryption.EncryptionService
}

func NewTestConnectionService(pgRepo *persistence.PostgresRepository, enc *encryption.EncryptionService) *TestConnectionService {
	return &TestConnectionService{
		pgRepo:     pgRepo,
		encryption: enc,
	}
}

type ConnectionTestResult struct {
	Success       bool   `json:"success"`
	SourceType    string `json:"source_type"`
	ResponseTime  int64  `json:"response_time_ms"`
	Message       string `json:"message"`
	ErrorDetails  string `json:"error_details,omitempty"`
	ServerVersion string `json:"server_version,omitempty"`
	DatabaseInfo  string `json:"database_info,omitempty"`
}

func (s *TestConnectionService) TestConnection(ctx context.Context, connID string) (*ConnectionTestResult, error) {
	connUUID, err := uuid.Parse(connID)
	if err != nil {
		return nil, fmt.Errorf("invalid connection ID: %w", err)
	}

	conn, err := s.pgRepo.GetConnection(ctx, connUUID)
	if err != nil {
		return nil, fmt.Errorf("connection not found: %w", err)
	}

	var config map[string]any
	if err := s.encryption.Decrypt(conn.ConfigEncrypted, &config); err != nil {
		return nil, fmt.Errorf("failed to decrypt config: %w", err)
	}

	startTime := time.Now()
	var result *ConnectionTestResult

	switch conn.SourceType {
	case "postgresql":
		result, err = s.testPostgreSQL(ctx, config)
	case "mysql":
		result, err = s.testMySQL(ctx, config)
	case "mongodb":
		result, err = s.testMongoDB(ctx, config)
	case "s3":
		result, err = s.testS3(ctx, config)
	case "filesystem":
		result, err = s.testFilesystem(ctx, config)
	case "redis":
		result, err = s.testRedis(ctx, config)
	case "slack":
		result, err = s.testSlack(ctx, config)
	case "firebase":
		result, err = s.testFirebase(ctx, config)
	case "couchdb":
		result, err = s.testCouchDB(ctx, config)
	case "gcs":
		result, err = s.testGCS(ctx, config)
	case "gdrive", "gdrive_workspace":
		result, err = s.testGDrive(ctx, config, conn.SourceType)
	case "text":
		result, err = s.testText(ctx, config)
	default:
		return nil, fmt.Errorf("unsupported source type: %s", conn.SourceType)
	}

	result.ResponseTime = time.Since(startTime).Milliseconds()
	return result, nil
}

func (s *TestConnectionService) TestConnectionByConfig(ctx context.Context, sourceType string, config map[string]any) (*ConnectionTestResult, error) {
	startTime := time.Now()
	var result *ConnectionTestResult
	var err error

	switch sourceType {
	case "postgresql":
		result, err = s.testPostgreSQL(ctx, config)
	case "mysql":
		result, err = s.testMySQL(ctx, config)
	case "mongodb":
		result, err = s.testMongoDB(ctx, config)
	case "s3":
		result, err = s.testS3(ctx, config)
	case "filesystem":
		result, err = s.testFilesystem(ctx, config)
	case "redis":
		result, err = s.testRedis(ctx, config)
	case "slack":
		result, err = s.testSlack(ctx, config)
	case "firebase":
		result, err = s.testFirebase(ctx, config)
	case "couchdb":
		result, err = s.testCouchDB(ctx, config)
	case "gcs":
		result, err = s.testGCS(ctx, config)
	case "gdrive", "gdrive_workspace":
		result, err = s.testGDrive(ctx, config, sourceType)
	case "text":
		result, err = s.testText(ctx, config)
	default:
		return nil, fmt.Errorf("unsupported source type: %s", sourceType)
	}

	result.ResponseTime = time.Since(startTime).Milliseconds()
	return result, err
}

func (s *TestConnectionService) testPostgreSQL(ctx context.Context, config map[string]any) (*ConnectionTestResult, error) {
	result := &ConnectionTestResult{SourceType: "postgresql"}

	host := getString(config, "host")
	port := getInt(config, "port", 5432)
	user := getString(config, "user")
	password := getString(config, "password")
	dbname := getString(config, "database")
	sslmode := getString(config, "sslmode")

	if sslmode == "" {
		sslmode = "disable"
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=10",
		host, port, user, password, dbname, sslmode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		result.Success = false
		result.Message = "Failed to create database connection"
		result.ErrorDetails = "Invalid connection configuration. Please check your database settings."
		// Log detailed error server-side only
		fmt.Printf("[SECURITY] PostgreSQL connection creation failed - %v\n", err)
		return result, nil
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		result.Success = false
		result.Message = "Failed to connect to PostgreSQL database"
		result.ErrorDetails = "Unable to establish connection. Please verify your credentials, hostname, and network access."
		// Log detailed error server-side only
		fmt.Printf("[SECURITY] PostgreSQL connection failed for %s:%d - %v\n", host, port, err)
		return result, nil
	}

	var version string
	err = db.QueryRowContext(ctx, "SELECT version()").Scan(&version)
	if err != nil {
		result.ServerVersion = "Unknown"
	} else {
		parts := strings.SplitN(version, " ", 3)
		if len(parts) >= 2 {
			result.ServerVersion = parts[0] + " " + parts[1]
		}
	}

	result.Success = true
	result.Message = "Connection successful"
	result.DatabaseInfo = fmt.Sprintf("Database: %s", dbname)
	return result, nil
}

func (s *TestConnectionService) testMySQL(ctx context.Context, config map[string]any) (*ConnectionTestResult, error) {
	result := &ConnectionTestResult{SourceType: "mysql"}

	host := getString(config, "host")
	port := getInt(config, "port", 3306)
	user := getString(config, "user")
	password := getString(config, "password")
	dbname := getString(config, "database")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&timeout=10s",
		user, password, host, port, dbname)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		result.Success = false
		result.Message = "Failed to create database connection"
		result.ErrorDetails = "Invalid connection configuration. Please check your database settings."
		// Log detailed error server-side only
		fmt.Printf("[SECURITY] MySQL connection creation failed - %v\n", err)
		return result, nil
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		result.Success = false
		result.Message = "Failed to connect to MySQL database"
		result.ErrorDetails = "Unable to establish connection. Please verify your credentials, hostname, and network access."
		// Log detailed error server-side only
		fmt.Printf("[SECURITY] MySQL connection failed for %s:%d - %v\n", host, port, err)
		return result, nil
	}

	var version string
	err = db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version)
	if err != nil {
		result.ServerVersion = "Unknown"
	} else {
		result.ServerVersion = version
	}

	result.Success = true
	result.Message = "Connection successful"
	result.DatabaseInfo = fmt.Sprintf("Database: %s", dbname)
	return result, nil
}

func (s *TestConnectionService) testMongoDB(ctx context.Context, config map[string]any) (*ConnectionTestResult, error) {
	result := &ConnectionTestResult{SourceType: "mongodb"}

	host := getString(config, "host")
	port := getInt(config, "port", 27017)
	user := getString(config, "user")
	password := getString(config, "password")
	dbname := getString(config, "database")
	authSource := getString(config, "auth_source")

	if authSource == "" {
		authSource = "admin"
	}

	var uri string
	if user != "" && password != "" {
		uri = fmt.Sprintf("mongodb://%s:%s@%s:%d/%s?authSource=%s&connectTimeoutMS=10000",
			user, password, host, port, dbname, authSource)
	} else {
		uri = fmt.Sprintf("mongodb://%s:%d/?connectTimeoutMS=10000", host, port)
	}

	clientOptions := options.Client().ApplyURI(uri)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		result.Success = false
		result.Message = "Failed to connect to MongoDB"
		result.ErrorDetails = "Unable to establish connection. Please verify your credentials and connection string."
		// Log detailed error server-side only
		fmt.Printf("[SECURITY] MongoDB connection failed for %s:%d - %v\n", host, port, err)
		return result, nil
	}
	defer client.Disconnect(ctx)

	if err := client.Ping(ctx, nil); err != nil {
		result.Success = false
		result.Message = "Failed to ping MongoDB"
		result.ErrorDetails = "Connection established but ping failed. Please verify database permissions."
		// Log detailed error server-side only
		fmt.Printf("[SECURITY] MongoDB ping failed for %s:%d - %v\n", host, port, err)
		return result, nil
	}

	var buildInfo bson.M
	err = client.Database("admin").RunCommand(ctx, bson.D{{Key: "buildInfo", Value: 1}}).Decode(&buildInfo)
	if err == nil {
		if version, ok := buildInfo["version"].(string); ok {
			result.ServerVersion = "MongoDB " + version
		}
	}

	result.Success = true
	result.Message = "Connection successful"
	result.DatabaseInfo = fmt.Sprintf("Database: %s", dbname)
	return result, nil
}

func (s *TestConnectionService) testS3(ctx context.Context, config map[string]any) (*ConnectionTestResult, error) {
	result := &ConnectionTestResult{SourceType: "s3"}

	region := getString(config, "region")
	bucket := getString(config, "bucket")
	accessKey := getString(config, "access_key")
	secretKey := getString(config, "secret_key")
	endpoint := getString(config, "endpoint")

	if accessKey == "" || secretKey == "" {
		result.Success = false
		result.Message = "Missing credentials"
		result.ErrorDetails = "access_key and secret_key are required"
		return result, nil
	}

	target := getHostPort(endpoint, region)
	conn, err := net.DialTimeout("tcp", target, 10*time.Second)
	if err != nil {
		result.Success = false
		result.Message = "Failed to connect to S3 endpoint"
		result.ErrorDetails = "Unable to reach S3 endpoint. Please verify region, endpoint URL, and network connectivity."
		// Log detailed error server-side only
		fmt.Printf("[SECURITY] S3 connection failed for region %s - %v\n", region, err)
		return result, nil
	}
	defer conn.Close()

	result.Success = true
	result.Message = "Connection successful"
	result.ServerVersion = fmt.Sprintf("Region: %s", region)
	result.DatabaseInfo = fmt.Sprintf("Bucket: %s", bucket)
	return result, nil
}

func (s *TestConnectionService) testFilesystem(ctx context.Context, config map[string]any) (*ConnectionTestResult, error) {
	result := &ConnectionTestResult{SourceType: "filesystem"}

	path := getString(config, "path")

	if path == "" {
		result.Success = false
		result.Message = "Missing path"
		result.ErrorDetails = "path is required for filesystem source"
		return result, nil
	}

	conn, err := net.DialTimeout("tcp", "localhost:22", 5*time.Second)
	if err == nil {
		defer conn.Close()
		result.ServerVersion = "SSH available"
	}

	result.Success = true
	result.Message = "Filesystem path configured"
	result.DatabaseInfo = fmt.Sprintf("Path: %s", path)
	return result, nil
}

func (s *TestConnectionService) testRedis(ctx context.Context, config map[string]any) (*ConnectionTestResult, error) {
	result := &ConnectionTestResult{SourceType: "redis"}

	host := getString(config, "host")
	port := getInt(config, "port", 6379)
	db := getInt(config, "db", 0)

	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		result.Success = false
		result.Message = "Failed to connect to Redis"
		result.ErrorDetails = "Unable to reach Redis server. Please verify hostname, port, and network access."
		// Log detailed error server-side only
		fmt.Printf("[SECURITY] Redis connection failed for %s - %v\n", addr, err)
		return result, nil
	}
	defer conn.Close()

	result.Success = true
	result.Message = "Connection successful"
	result.ServerVersion = fmt.Sprintf("DB: %d", db)
	result.DatabaseInfo = fmt.Sprintf("Address: %s", addr)
	return result, nil
}

func (s *TestConnectionService) testSlack(ctx context.Context, config map[string]any) (*ConnectionTestResult, error) {
	result := &ConnectionTestResult{SourceType: "slack"}

	token := getString(config, "bot_token")
	if token == "" {
		result.Success = false
		result.Message = "Missing bot token"
		result.ErrorDetails = "bot_token is required for Slack source"
		return result, nil
	}

	if len(token) < 10 || !strings.HasPrefix(token, "xoxb-") {
		result.Success = false
		result.Message = "Invalid bot token format"
		result.ErrorDetails = "Slack bot token should start with 'xoxb-'"
		return result, nil
	}

	result.Success = true
	result.Message = "Slack token format valid"
	result.ServerVersion = "Slack API v2"
	return result, nil
}

func (s *TestConnectionService) testFirebase(ctx context.Context, config map[string]any) (*ConnectionTestResult, error) {
	result := &ConnectionTestResult{SourceType: "firebase"}

	projectID := getString(config, "project_id")
	if projectID == "" {
		result.Success = false
		result.Message = "Missing project_id"
		result.ErrorDetails = "project_id is required for Firebase source"
		return result, nil
	}

	// Verify the Firebase REST endpoint is reachable
	url := fmt.Sprintf("https://%s.firebaseio.com/.json?shallow=true", projectID)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		result.Success = false
		result.Message = "Failed to reach Firebase"
		result.ErrorDetails = "Unable to connect to Firebase. Please verify your project ID and network access."
		fmt.Printf("[SECURITY] Firebase connection failed for project %s - %v\n", projectID, err)
		return result, nil
	}
	resp.Body.Close()

	result.Success = true
	result.Message = "Firebase reachable"
	result.ServerVersion = "Firebase Realtime Database"
	result.DatabaseInfo = fmt.Sprintf("Project: %s", projectID)
	return result, nil
}

func (s *TestConnectionService) testCouchDB(ctx context.Context, config map[string]any) (*ConnectionTestResult, error) {
	result := &ConnectionTestResult{SourceType: "couchdb"}

	host := getString(config, "host")
	port := getInt(config, "port", 5984)

	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		result.Success = false
		result.Message = "Failed to connect to CouchDB"
		result.ErrorDetails = "Unable to reach CouchDB server. Please verify hostname, port, and network access."
		fmt.Printf("[SECURITY] CouchDB connection failed for %s - %v\n", addr, err)
		return result, nil
	}
	conn.Close()

	result.Success = true
	result.Message = "Connection successful"
	result.ServerVersion = "CouchDB"
	result.DatabaseInfo = fmt.Sprintf("Address: %s", addr)
	return result, nil
}

func (s *TestConnectionService) testGCS(ctx context.Context, config map[string]any) (*ConnectionTestResult, error) {
	result := &ConnectionTestResult{SourceType: "gcs"}

	bucket := getString(config, "bucket")
	if bucket == "" {
		result.Success = false
		result.Message = "Missing bucket"
		result.ErrorDetails = "bucket is required for GCS source"
		return result, nil
	}

	// Verify GCS endpoint is reachable
	conn, err := net.DialTimeout("tcp", "storage.googleapis.com:443", 5*time.Second)
	if err != nil {
		result.Success = false
		result.Message = "Failed to reach Google Cloud Storage"
		result.ErrorDetails = "Unable to connect to GCS. Please verify network access."
		return result, nil
	}
	conn.Close()

	result.Success = true
	result.Message = "GCS endpoint reachable"
	result.ServerVersion = "Google Cloud Storage"
	result.DatabaseInfo = fmt.Sprintf("Bucket: %s", bucket)
	return result, nil
}

func (s *TestConnectionService) testGDrive(ctx context.Context, config map[string]any, sourceType string) (*ConnectionTestResult, error) {
	result := &ConnectionTestResult{SourceType: sourceType}

	// Google Drive requires OAuth — validate that credentials are configured
	clientID := getString(config, "client_id")
	if clientID == "" {
		result.Success = false
		result.Message = "Missing client_id"
		result.ErrorDetails = "OAuth client_id is required for Google Drive source"
		return result, nil
	}

	// Verify Google API endpoint is reachable
	conn, err := net.DialTimeout("tcp", "www.googleapis.com:443", 5*time.Second)
	if err != nil {
		result.Success = false
		result.Message = "Failed to reach Google APIs"
		result.ErrorDetails = "Unable to connect to Google APIs. Please verify network access."
		return result, nil
	}
	conn.Close()

	result.Success = true
	result.Message = "Google API endpoint reachable"
	result.ServerVersion = "Google Drive API v3"
	return result, nil
}

func (s *TestConnectionService) testText(ctx context.Context, config map[string]any) (*ConnectionTestResult, error) {
	result := &ConnectionTestResult{SourceType: "text"}

	path := getString(config, "path")
	content := getString(config, "content")

	if path == "" && content == "" {
		result.Success = false
		result.Message = "Missing path or content"
		result.ErrorDetails = "Either path or content is required for text source"
		return result, nil
	}

	result.Success = true
	result.Message = "Text source configured"
	if path != "" {
		result.DatabaseInfo = fmt.Sprintf("Path: %s", path)
	} else {
		result.DatabaseInfo = fmt.Sprintf("Inline content (%d chars)", len(content))
	}
	return result, nil
}

func getString(config map[string]any, key string) string {
	if val, ok := config[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	if key == "user" {
		if val, ok := config["username"]; ok {
			if s, ok := val.(string); ok {
				return s
			}
		}
	}
	return ""
}

func getInt(config map[string]any, key string, defaultVal int) int {
	if val, ok := config[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		case string:
			if i, err := strconv.Atoi(v); err == nil {
				return i
			}
		}
	}
	return defaultVal
}

func getHostPort(endpoint, region string) string {
	if endpoint != "" {
		host, port, err := net.SplitHostPort(endpoint)
		if err == nil {
			return fmt.Sprintf("%s:%s", host, port)
		}
		return endpoint
	}
	return fmt.Sprintf("s3.%s.amazonaws.com:443", region)
}
