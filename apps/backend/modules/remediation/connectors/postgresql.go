package connectors

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"

	_ "github.com/lib/pq"
)

// validIdentifier matches safe SQL identifiers (letters, digits, underscores)
var validIdentifier = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// sanitizePgIdentifier validates and quotes a PostgreSQL identifier to prevent SQL injection.
// Table and column names cannot be parameterized with $1, so we must validate them.
func sanitizePgIdentifier(name string) (string, error) {
	if !validIdentifier.MatchString(name) {
		return "", fmt.Errorf("invalid SQL identifier: %q", name)
	}
	// Double-quote the identifier for safety (handles reserved words)
	return fmt.Sprintf(`"%s"`, name), nil
}

// PostgreSQLConnector implements remediation for PostgreSQL databases
type PostgreSQLConnector struct {
	db *sql.DB
}

// Connect establishes connection to PostgreSQL
func (c *PostgreSQLConnector) Connect(ctx context.Context, config map[string]interface{}) error {
	host := config["host"].(string)
	port := config["port"].(int)
	user := config["user"].(string)
	password := config["password"].(string)
	database := config["database"].(string)

	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, database)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	c.db = db
	return nil
}

// Close closes the PostgreSQL connection
func (c *PostgreSQLConnector) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// Mask redacts PII in place
func (c *PostgreSQLConnector) Mask(ctx context.Context, location string, fieldName string, recordID string) error {
	safeTable, err := sanitizePgIdentifier(location)
	if err != nil {
		return fmt.Errorf("invalid table name: %w", err)
	}
	safeField, err := sanitizePgIdentifier(fieldName)
	if err != nil {
		return fmt.Errorf("invalid field name: %w", err)
	}
	query := fmt.Sprintf("UPDATE %s SET %s = 'REDACTED' WHERE id = $1", safeTable, safeField)
	_, err = c.db.ExecContext(ctx, query, recordID)
	if err != nil {
		return fmt.Errorf("failed to mask PII: %w", err)
	}
	return nil
}

// Delete removes the entire record
func (c *PostgreSQLConnector) Delete(ctx context.Context, location string, recordID string) error {
	safeTable, err := sanitizePgIdentifier(location)
	if err != nil {
		return fmt.Errorf("invalid table name: %w", err)
	}
	query := fmt.Sprintf("DELETE FROM %s WHERE id = $1", safeTable)
	_, err = c.db.ExecContext(ctx, query, recordID)
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}
	return nil
}

// Encrypt encrypts PII value
func (c *PostgreSQLConnector) Encrypt(ctx context.Context, location string, fieldName string, recordID string, encryptionKey string) error {
	// Get original value
	originalValue, err := c.GetOriginalValue(ctx, location, fieldName, recordID)
	if err != nil {
		return err
	}

	// Encrypt value with AES-GCM
	encryptedValue, err := encryptAESGCM(encryptionKey, originalValue)
	if err != nil {
		return fmt.Errorf("failed to encrypt value: %w", err)
	}

	safeTable, err := sanitizePgIdentifier(location)
	if err != nil {
		return fmt.Errorf("invalid table name: %w", err)
	}
	safeField, err := sanitizePgIdentifier(fieldName)
	if err != nil {
		return fmt.Errorf("invalid field name: %w", err)
	}
	query := fmt.Sprintf("UPDATE %s SET %s = $1 WHERE id = $2", safeTable, safeField)
	_, err = c.db.ExecContext(ctx, query, encryptedValue, recordID)
	if err != nil {
		return fmt.Errorf("failed to encrypt PII: %w", err)
	}
	return nil
}

// GetOriginalValue retrieves original value before remediation
func (c *PostgreSQLConnector) GetOriginalValue(ctx context.Context, location string, fieldName string, recordID string) (string, error) {
	safeField, err := sanitizePgIdentifier(fieldName)
	if err != nil {
		return "", fmt.Errorf("invalid field name: %w", err)
	}
	safeTable, err := sanitizePgIdentifier(location)
	if err != nil {
		return "", fmt.Errorf("invalid table name: %w", err)
	}
	query := fmt.Sprintf("SELECT %s FROM %s WHERE id = $1", safeField, safeTable)

	var value string
	err = c.db.QueryRowContext(ctx, query, recordID).Scan(&value)
	if err != nil {
		return "", fmt.Errorf("failed to get original value: %w", err)
	}

	return value, nil
}

// RestoreValue restores original value (rollback)
func (c *PostgreSQLConnector) RestoreValue(ctx context.Context, location string, fieldName string, recordID string, originalValue string) error {
	safeTable, err := sanitizePgIdentifier(location)
	if err != nil {
		return fmt.Errorf("invalid table name: %w", err)
	}
	safeField, err := sanitizePgIdentifier(fieldName)
	if err != nil {
		return fmt.Errorf("invalid field name: %w", err)
	}
	query := fmt.Sprintf("UPDATE %s SET %s = $1 WHERE id = $2", safeTable, safeField)
	_, err = c.db.ExecContext(ctx, query, originalValue, recordID)
	if err != nil {
		return fmt.Errorf("failed to restore value: %w", err)
	}
	return nil
}
