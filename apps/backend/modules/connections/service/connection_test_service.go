package service

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/arc-platform/backend/modules/shared/infrastructure/encryption"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/arc-platform/backend/modules/shared/infrastructure/vault"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type TestConnectionService struct {
	pgRepo     *persistence.PostgresRepository
	encryption *encryption.EncryptionService
	vault      *vault.Client
}

// NewTestConnectionService constructs the service.
// vaultClient may be nil when Vault is not configured.
func NewTestConnectionService(pgRepo *persistence.PostgresRepository, enc *encryption.EncryptionService, vaultClient *vault.Client) *TestConnectionService {
	return &TestConnectionService{
		pgRepo:     pgRepo,
		encryption: enc,
		vault:      vaultClient,
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
	if s.vault != nil && s.vault.IsEnabled() {
		vaultCfg, vErr := s.vault.ReadConnectionSecret(conn.SourceType, conn.ProfileName)
		if vErr != nil {
			return nil, fmt.Errorf("vault read failed for %s/%s: %w", conn.SourceType, conn.ProfileName, vErr)
		}
		if vaultCfg == nil {
			return nil, fmt.Errorf("credentials not found in Vault for %s/%s", conn.SourceType, conn.ProfileName)
		}
		config = vaultCfg
	} else {
		if err := s.encryption.Decrypt(conn.ConfigEncrypted, &config); err != nil {
			return nil, fmt.Errorf("failed to decrypt config: %w", err)
		}
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
	case "sqlite":
		result, err = s.testSQLite(ctx, config)
	case "oracle":
		result, err = s.testOracle(ctx, config)
	case "mssql":
		result, err = s.testMSSQL(ctx, config)
	case "azure_blob":
		result, err = s.testAzureBlob(ctx, config)
	case "bigquery":
		result, err = s.testBigQuery(ctx, config)
	case "snowflake":
		result, err = s.testSnowflake(ctx, config)
	case "redshift":
		result, err = s.testRedshift(ctx, config)
	case "kafka":
		result, err = s.testKafka(ctx, config)
	case "kinesis":
		result, err = s.testKinesis(ctx, config)
	case "csv_excel":
		result, err = s.testFileSource(ctx, config, "csv_excel", "CSV/Excel")
	case "pdf":
		result, err = s.testFileSource(ctx, config, "pdf", "PDF")
	case "docx":
		result, err = s.testFileSource(ctx, config, "docx", "Word Document")
	case "pptx":
		result, err = s.testFileSource(ctx, config, "pptx", "PowerPoint")
	case "html_files":
		result, err = s.testFileSource(ctx, config, "html_files", "HTML Files")
	case "email_files":
		result, err = s.testFileSource(ctx, config, "email_files", "Email Files")
	case "parquet":
		result, err = s.testFileSource(ctx, config, "parquet", "Parquet")
	case "orc":
		result, err = s.testFileSource(ctx, config, "orc", "ORC")
	case "avro":
		result, err = s.testFileSource(ctx, config, "avro", "Avro")
	case "scanned_images":
		result, err = s.testFileSource(ctx, config, "scanned_images", "Scanned Images")
	case "salesforce":
		result, err = s.testSalesforce(ctx, config)
	case "hubspot":
		result, err = s.testHubSpot(ctx, config)
	case "jira":
		result, err = s.testJira(ctx, config)
	case "ms_teams":
		result, err = s.testMSTeams(ctx, config)
	default:
		return nil, fmt.Errorf("unsupported source type: %s", conn.SourceType)
	}

	result.ResponseTime = time.Since(startTime).Milliseconds()

	// Persist the validation result so the UI reflects the latest test outcome.
	validationStatus := "failed"
	if result.Success {
		validationStatus = "valid"
	}
	var validationErr *string
	if result.ErrorDetails != "" {
		e := result.ErrorDetails
		validationErr = &e
	}
	// Best-effort — don't fail the test response if the DB update fails.
	_ = s.pgRepo.UpdateConnectionValidation(ctx, connUUID, validationStatus, validationErr)

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
	case "sqlite":
		result, err = s.testSQLite(ctx, config)
	case "oracle":
		result, err = s.testOracle(ctx, config)
	case "mssql":
		result, err = s.testMSSQL(ctx, config)
	case "azure_blob":
		result, err = s.testAzureBlob(ctx, config)
	case "bigquery":
		result, err = s.testBigQuery(ctx, config)
	case "snowflake":
		result, err = s.testSnowflake(ctx, config)
	case "redshift":
		result, err = s.testRedshift(ctx, config)
	case "kafka":
		result, err = s.testKafka(ctx, config)
	case "kinesis":
		result, err = s.testKinesis(ctx, config)
	case "csv_excel":
		result, err = s.testFileSource(ctx, config, "csv_excel", "CSV/Excel")
	case "pdf":
		result, err = s.testFileSource(ctx, config, "pdf", "PDF")
	case "docx":
		result, err = s.testFileSource(ctx, config, "docx", "Word Document")
	case "pptx":
		result, err = s.testFileSource(ctx, config, "pptx", "PowerPoint")
	case "html_files":
		result, err = s.testFileSource(ctx, config, "html_files", "HTML Files")
	case "email_files":
		result, err = s.testFileSource(ctx, config, "email_files", "Email Files")
	case "parquet":
		result, err = s.testFileSource(ctx, config, "parquet", "Parquet")
	case "orc":
		result, err = s.testFileSource(ctx, config, "orc", "ORC")
	case "avro":
		result, err = s.testFileSource(ctx, config, "avro", "Avro")
	case "scanned_images":
		result, err = s.testFileSource(ctx, config, "scanned_images", "Scanned Images")
	case "salesforce":
		result, err = s.testSalesforce(ctx, config)
	case "hubspot":
		result, err = s.testHubSpot(ctx, config)
	case "jira":
		result, err = s.testJira(ctx, config)
	case "ms_teams":
		result, err = s.testMSTeams(ctx, config)
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

	if isPrivateOrReservedHost(host) {
		result.Success = false
		result.Message = "Invalid host"
		result.ErrorDetails = "Host resolves to a private or reserved address"
		return result, nil
	}
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
	if isPrivateOrReservedHost(host) {
		result.Success = false
		result.Message = "Invalid host"
		result.ErrorDetails = "Host resolves to a private or reserved address"
		return result, nil
	}
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

// isPrivateOrReservedHost resolves host and returns true if any resolved IP
// falls in a private, loopback, or link-local range — blocking SSRF attacks
// where an attacker supplies an internal address to probe the network.
func isPrivateOrReservedHost(host string) bool {
	if host == "" {
		return true
	}
	// Strip port if present
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	ips, err := net.LookupHost(host)
	if err != nil {
		// Unresolvable host: block as a precaution
		return true
	}
	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return true
		}
		// Block 169.254.0.0/16 (AWS IMDS) explicitly in case IsLinkLocalUnicast misses it
		if ip4 := ip.To4(); ip4 != nil && ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
	}
	return false
}


func (s *TestConnectionService) testSQLite(_ context.Context, config map[string]any) (*ConnectionTestResult, error) {
	result := &ConnectionTestResult{SourceType: "sqlite"}
	path := getString(config, "path")
	if path == "" {
		result.Success = false
		result.Message = "Missing database path"
		result.ErrorDetails = "path is required for SQLite source"
		return result, nil
	}
	if _, err := os.Stat(path); err != nil {
		result.Success = false
		result.Message = "SQLite database file not accessible"
		result.ErrorDetails = "Cannot access the specified file. Please verify the path exists and is readable."
		fmt.Printf("[SECURITY] SQLite file check failed - %v\n", err)
		return result, nil
	}
	result.Success = true
	result.Message = "SQLite database file accessible"
	result.DatabaseInfo = fmt.Sprintf("Path: %s", path)
	return result, nil
}

func (s *TestConnectionService) testOracle(_ context.Context, config map[string]any) (*ConnectionTestResult, error) {
	result := &ConnectionTestResult{SourceType: "oracle"}
	host := getString(config, "host")
	port := getInt(config, "port", 1521)
	serviceName := getString(config, "service_name")
	if host == "" {
		result.Success = false
		result.Message = "Missing host"
		result.ErrorDetails = "host is required for Oracle source"
		return result, nil
	}
	if isPrivateOrReservedHost(host) {
		result.Success = false
		result.Message = "Invalid host"
		result.ErrorDetails = "Host resolves to a private or reserved address"
		return result, nil
	}
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		result.Success = false
		result.Message = "Failed to connect to Oracle database"
		result.ErrorDetails = "Unable to reach Oracle listener. Please verify hostname, port, and network access."
		fmt.Printf("[SECURITY] Oracle connection failed for %s - %v\n", addr, err)
		return result, nil
	}
	conn.Close()
	result.Success = true
	result.Message = "Connection successful"
	result.ServerVersion = "Oracle Database"
	if serviceName != "" {
		result.DatabaseInfo = fmt.Sprintf("Service: %s", serviceName)
	}
	return result, nil
}

func (s *TestConnectionService) testMSSQL(_ context.Context, config map[string]any) (*ConnectionTestResult, error) {
	result := &ConnectionTestResult{SourceType: "mssql"}
	host := getString(config, "host")
	port := getInt(config, "port", 1433)
	dbname := getString(config, "database")
	if host == "" {
		result.Success = false
		result.Message = "Missing host"
		result.ErrorDetails = "host is required for SQL Server source"
		return result, nil
	}
	if isPrivateOrReservedHost(host) {
		result.Success = false
		result.Message = "Invalid host"
		result.ErrorDetails = "Host resolves to a private or reserved address"
		return result, nil
	}
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		result.Success = false
		result.Message = "Failed to connect to SQL Server"
		result.ErrorDetails = "Unable to reach SQL Server. Please verify hostname, port, and network access."
		fmt.Printf("[SECURITY] MSSQL connection failed for %s - %v\n", addr, err)
		return result, nil
	}
	conn.Close()
	result.Success = true
	result.Message = "Connection successful"
	result.ServerVersion = "Microsoft SQL Server"
	if dbname != "" {
		result.DatabaseInfo = fmt.Sprintf("Database: %s", dbname)
	}
	return result, nil
}

func (s *TestConnectionService) testAzureBlob(_ context.Context, config map[string]any) (*ConnectionTestResult, error) {
	result := &ConnectionTestResult{SourceType: "azure_blob"}
	accountName := getString(config, "account_name")
	containerName := getString(config, "container_name")
	if accountName == "" {
		result.Success = false
		result.Message = "Missing account_name"
		result.ErrorDetails = "account_name is required for Azure Blob Storage source"
		return result, nil
	}
	endpoint := fmt.Sprintf("%s.blob.core.windows.net:443", accountName)
	conn, err := net.DialTimeout("tcp", endpoint, 10*time.Second)
	if err != nil {
		result.Success = false
		result.Message = "Failed to reach Azure Blob Storage"
		result.ErrorDetails = "Unable to connect to Azure Blob Storage. Please verify account name and network access."
		fmt.Printf("[SECURITY] Azure Blob connection failed for %s - %v\n", endpoint, err)
		return result, nil
	}
	conn.Close()
	result.Success = true
	result.Message = "Azure Blob Storage endpoint reachable"
	result.ServerVersion = "Azure Blob Storage"
	result.DatabaseInfo = fmt.Sprintf("Account: %s, Container: %s", accountName, containerName)
	return result, nil
}

func (s *TestConnectionService) testBigQuery(_ context.Context, config map[string]any) (*ConnectionTestResult, error) {
	result := &ConnectionTestResult{SourceType: "bigquery"}
	projectID := getString(config, "project_id")
	dataset := getString(config, "dataset")
	if projectID == "" {
		result.Success = false
		result.Message = "Missing project_id"
		result.ErrorDetails = "project_id is required for BigQuery source"
		return result, nil
	}
	conn, err := net.DialTimeout("tcp", "bigquery.googleapis.com:443", 10*time.Second)
	if err != nil {
		result.Success = false
		result.Message = "Failed to reach BigQuery"
		result.ErrorDetails = "Unable to connect to BigQuery. Please verify network access and credentials."
		return result, nil
	}
	conn.Close()
	result.Success = true
	result.Message = "BigQuery endpoint reachable"
	result.ServerVersion = "Google BigQuery"
	info := fmt.Sprintf("Project: %s", projectID)
	if dataset != "" {
		info += fmt.Sprintf(", Dataset: %s", dataset)
	}
	result.DatabaseInfo = info
	return result, nil
}

func (s *TestConnectionService) testSnowflake(_ context.Context, config map[string]any) (*ConnectionTestResult, error) {
	result := &ConnectionTestResult{SourceType: "snowflake"}
	account := getString(config, "account")
	warehouse := getString(config, "warehouse")
	dbname := getString(config, "database")
	if account == "" {
		result.Success = false
		result.Message = "Missing account"
		result.ErrorDetails = "account is required for Snowflake source (e.g. myorg-myaccount)"
		return result, nil
	}
	endpoint := fmt.Sprintf("%s.snowflakecomputing.com:443", account)
	conn, err := net.DialTimeout("tcp", endpoint, 10*time.Second)
	if err != nil {
		result.Success = false
		result.Message = "Failed to reach Snowflake"
		result.ErrorDetails = "Unable to connect to Snowflake. Please verify account identifier and network access."
		fmt.Printf("[SECURITY] Snowflake connection failed for %s - %v\n", endpoint, err)
		return result, nil
	}
	conn.Close()
	result.Success = true
	result.Message = "Snowflake endpoint reachable"
	result.ServerVersion = "Snowflake"
	info := fmt.Sprintf("Account: %s", account)
	if warehouse != "" {
		info += fmt.Sprintf(", Warehouse: %s", warehouse)
	}
	if dbname != "" {
		info += fmt.Sprintf(", Database: %s", dbname)
	}
	result.DatabaseInfo = info
	return result, nil
}

func (s *TestConnectionService) testRedshift(_ context.Context, config map[string]any) (*ConnectionTestResult, error) {
	result := &ConnectionTestResult{SourceType: "redshift"}
	host := getString(config, "host")
	port := getInt(config, "port", 5439)
	dbname := getString(config, "database")
	if host == "" {
		result.Success = false
		result.Message = "Missing host"
		result.ErrorDetails = "host is required for Redshift source"
		return result, nil
	}
	if isPrivateOrReservedHost(host) {
		result.Success = false
		result.Message = "Invalid host"
		result.ErrorDetails = "Host resolves to a private or reserved address"
		return result, nil
	}
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		result.Success = false
		result.Message = "Failed to connect to Redshift"
		result.ErrorDetails = "Unable to reach Redshift cluster. Please verify endpoint, port, and network access."
		fmt.Printf("[SECURITY] Redshift connection failed for %s - %v\n", addr, err)
		return result, nil
	}
	conn.Close()
	result.Success = true
	result.Message = "Connection successful"
	result.ServerVersion = "Amazon Redshift"
	result.DatabaseInfo = fmt.Sprintf("Host: %s, Database: %s", host, dbname)
	return result, nil
}

func (s *TestConnectionService) testKafka(_ context.Context, config map[string]any) (*ConnectionTestResult, error) {
	result := &ConnectionTestResult{SourceType: "kafka"}
	brokers := getString(config, "brokers")
	if brokers == "" {
		result.Success = false
		result.Message = "Missing brokers"
		result.ErrorDetails = "brokers is required for Kafka source (comma-separated host:port list)"
		return result, nil
	}
	broker := strings.TrimSpace(strings.SplitN(brokers, ",", 2)[0])
	brokerHost, _, _ := net.SplitHostPort(broker)
	if brokerHost == "" {
		brokerHost = broker
	}
	if isPrivateOrReservedHost(brokerHost) {
		result.Success = false
		result.Message = "Invalid broker address"
		result.ErrorDetails = "Broker host resolves to a private or reserved address"
		return result, nil
	}
	conn, err := net.DialTimeout("tcp", broker, 10*time.Second)
	if err != nil {
		result.Success = false
		result.Message = "Failed to connect to Kafka broker"
		result.ErrorDetails = "Unable to reach Kafka broker. Please verify broker addresses and network access."
		fmt.Printf("[SECURITY] Kafka connection failed for %s - %v\n", broker, err)
		return result, nil
	}
	conn.Close()
	result.Success = true
	result.Message = "Kafka broker reachable"
	result.ServerVersion = "Apache Kafka"
	result.DatabaseInfo = fmt.Sprintf("Broker: %s", broker)
	return result, nil
}

func (s *TestConnectionService) testKinesis(_ context.Context, config map[string]any) (*ConnectionTestResult, error) {
	result := &ConnectionTestResult{SourceType: "kinesis"}
	region := getString(config, "region")
	streamName := getString(config, "stream_name")
	if region == "" {
		result.Success = false
		result.Message = "Missing region"
		result.ErrorDetails = "region is required for Kinesis source"
		return result, nil
	}
	endpoint := fmt.Sprintf("kinesis.%s.amazonaws.com:443", region)
	conn, err := net.DialTimeout("tcp", endpoint, 10*time.Second)
	if err != nil {
		result.Success = false
		result.Message = "Failed to reach AWS Kinesis"
		result.ErrorDetails = "Unable to connect to Kinesis endpoint. Please verify region and network access."
		fmt.Printf("[SECURITY] Kinesis connection failed for %s - %v\n", endpoint, err)
		return result, nil
	}
	conn.Close()
	result.Success = true
	result.Message = "Kinesis endpoint reachable"
	result.ServerVersion = "AWS Kinesis"
	info := fmt.Sprintf("Region: %s", region)
	if streamName != "" {
		info += fmt.Sprintf(", Stream: %s", streamName)
	}
	result.DatabaseInfo = info
	return result, nil
}

func (s *TestConnectionService) testFileSource(_ context.Context, config map[string]any, sourceType, displayName string) (*ConnectionTestResult, error) {
	result := &ConnectionTestResult{SourceType: sourceType}
	path := getString(config, "path")
	if path == "" {
		result.Success = false
		result.Message = "Missing path"
		result.ErrorDetails = fmt.Sprintf("path is required for %s source", displayName)
		return result, nil
	}
	info, err := os.Stat(path)
	if err != nil {
		// Remote or network path — treat as configured but unverified locally
		result.Success = true
		result.Message = fmt.Sprintf("%s path configured", displayName)
		result.DatabaseInfo = fmt.Sprintf("Path: %s", path)
		return result, nil
	}
	if info.IsDir() {
		result.Message = fmt.Sprintf("%s directory accessible", displayName)
		result.DatabaseInfo = fmt.Sprintf("Directory: %s", path)
	} else {
		result.Message = fmt.Sprintf("%s file accessible", displayName)
		result.DatabaseInfo = fmt.Sprintf("File: %s (%d bytes)", path, info.Size())
	}
	result.Success = true
	return result, nil
}

func (s *TestConnectionService) testSalesforce(_ context.Context, config map[string]any) (*ConnectionTestResult, error) {
	result := &ConnectionTestResult{SourceType: "salesforce"}
	instanceURL := getString(config, "instance_url")
	token := getString(config, "access_token")
	if token == "" && getString(config, "client_id") == "" {
		result.Success = false
		result.Message = "Missing credentials"
		result.ErrorDetails = "access_token or client_id/client_secret are required for Salesforce source"
		return result, nil
	}
	host := "login.salesforce.com"
	if instanceURL != "" {
		trimmed := strings.TrimPrefix(strings.TrimPrefix(instanceURL, "https://"), "http://")
		host = strings.Split(trimmed, "/")[0]
	}
	if isPrivateOrReservedHost(host) {
		result.Success = false
		result.Message = "Invalid instance URL"
		result.ErrorDetails = "Instance host resolves to a private or reserved address"
		return result, nil
	}
	conn, err := net.DialTimeout("tcp", host+":443", 10*time.Second)
	if err != nil {
		result.Success = false
		result.Message = "Failed to reach Salesforce"
		result.ErrorDetails = "Unable to connect to Salesforce. Please verify instance URL and network access."
		fmt.Printf("[SECURITY] Salesforce connection failed for %s - %v\n", host, err)
		return result, nil
	}
	conn.Close()
	result.Success = true
	result.Message = "Salesforce endpoint reachable"
	result.ServerVersion = "Salesforce API"
	result.DatabaseInfo = fmt.Sprintf("Instance: %s", host)
	return result, nil
}

func (s *TestConnectionService) testHubSpot(_ context.Context, config map[string]any) (*ConnectionTestResult, error) {
	result := &ConnectionTestResult{SourceType: "hubspot"}
	apiKey := getString(config, "api_key")
	accessToken := getString(config, "access_token")
	if apiKey == "" && accessToken == "" {
		result.Success = false
		result.Message = "Missing credentials"
		result.ErrorDetails = "api_key or access_token is required for HubSpot source"
		return result, nil
	}
	conn, err := net.DialTimeout("tcp", "api.hubapi.com:443", 10*time.Second)
	if err != nil {
		result.Success = false
		result.Message = "Failed to reach HubSpot API"
		result.ErrorDetails = "Unable to connect to HubSpot. Please verify network access."
		return result, nil
	}
	conn.Close()
	result.Success = true
	result.Message = "HubSpot API endpoint reachable"
	result.ServerVersion = "HubSpot API v3"
	return result, nil
}

func (s *TestConnectionService) testJira(_ context.Context, config map[string]any) (*ConnectionTestResult, error) {
	result := &ConnectionTestResult{SourceType: "jira"}
	jiraURL := getString(config, "url")
	token := getString(config, "api_token")
	if jiraURL == "" {
		result.Success = false
		result.Message = "Missing Jira URL"
		result.ErrorDetails = "url is required for Jira source (e.g. https://yourorg.atlassian.net)"
		return result, nil
	}
	if token == "" {
		result.Success = false
		result.Message = "Missing API token"
		result.ErrorDetails = "api_token is required for Jira source"
		return result, nil
	}
	trimmed := strings.TrimPrefix(strings.TrimPrefix(jiraURL, "https://"), "http://")
	host := strings.Split(trimmed, "/")[0]
	if isPrivateOrReservedHost(host) {
		result.Success = false
		result.Message = "Invalid Jira URL"
		result.ErrorDetails = "Jira host resolves to a private or reserved address"
		return result, nil
	}
	conn, err := net.DialTimeout("tcp", host+":443", 10*time.Second)
	if err != nil {
		result.Success = false
		result.Message = "Failed to reach Jira"
		result.ErrorDetails = "Unable to connect to Jira. Please verify your URL and network access."
		fmt.Printf("[SECURITY] Jira connection failed for %s - %v\n", host, err)
		return result, nil
	}
	conn.Close()
	result.Success = true
	result.Message = "Jira endpoint reachable"
	result.ServerVersion = "Jira REST API"
	result.DatabaseInfo = fmt.Sprintf("Host: %s", host)
	return result, nil
}

func (s *TestConnectionService) testMSTeams(_ context.Context, config map[string]any) (*ConnectionTestResult, error) {
	result := &ConnectionTestResult{SourceType: "ms_teams"}
	webhookURL := getString(config, "webhook_url")
	tenantID := getString(config, "tenant_id")
	clientID := getString(config, "client_id")
	if webhookURL == "" && (tenantID == "" || clientID == "") {
		result.Success = false
		result.Message = "Missing credentials"
		result.ErrorDetails = "webhook_url or tenant_id + client_id are required for Microsoft Teams source"
		return result, nil
	}
	conn, err := net.DialTimeout("tcp", "graph.microsoft.com:443", 10*time.Second)
	if err != nil {
		result.Success = false
		result.Message = "Failed to reach Microsoft Graph API"
		result.ErrorDetails = "Unable to connect to Microsoft services. Please verify network access."
		return result, nil
	}
	conn.Close()
	result.Success = true
	result.Message = "Microsoft Teams endpoint reachable"
	result.ServerVersion = "Microsoft Graph API"
	if tenantID != "" {
		result.DatabaseInfo = fmt.Sprintf("Tenant: %s", tenantID)
	}
	return result, nil
}
