package connectors

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"

	_ "github.com/go-sql-driver/mysql"
)

// validIdentifier matches safe SQL identifiers (letters, digits, underscores)
var validIdentifierMySQL = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// sanitizeMySQLIdentifier validates and quotes a MySQL identifier to prevent SQL injection.
func sanitizeMySQLIdentifier(name string) (string, error) {
	if !validIdentifierMySQL.MatchString(name) {
		return "", fmt.Errorf("invalid SQL identifier: %q", name)
	}
	// Backtick-quote the identifier for MySQL
	return fmt.Sprintf("`%s`", name), nil
}

// MySQLConnector implements remediation for MySQL databases
type MySQLConnector struct {
	db *sql.DB
}

// Connect establishes connection to MySQL
func (c *MySQLConnector) Connect(ctx context.Context, config map[string]interface{}) error {
	host := config["host"].(string)
	port := config["port"].(int)
	user := config["user"].(string)
	password := config["password"].(string)
	database := config["database"].(string)

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", user, password, host, port, database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping MySQL: %w", err)
	}

	c.db = db
	return nil
}

// Close closes the MySQL connection
func (c *MySQLConnector) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// Mask redacts PII in place
func (c *MySQLConnector) Mask(ctx context.Context, location string, fieldName string, recordID string) error {
	safeTable, err := sanitizeMySQLIdentifier(location)
	if err != nil {
		return fmt.Errorf("invalid table name: %w", err)
	}
	safeField, err := sanitizeMySQLIdentifier(fieldName)
	if err != nil {
		return fmt.Errorf("invalid field name: %w", err)
	}
	query := fmt.Sprintf("UPDATE %s SET %s = 'REDACTED' WHERE id = ?", safeTable, safeField)
	_, err = c.db.ExecContext(ctx, query, recordID)
	if err != nil {
		return fmt.Errorf("failed to mask PII: %w", err)
	}
	return nil
}

// Delete removes the entire record
func (c *MySQLConnector) Delete(ctx context.Context, location string, recordID string) error {
	safeTable, err := sanitizeMySQLIdentifier(location)
	if err != nil {
		return fmt.Errorf("invalid table name: %w", err)
	}
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", safeTable)
	_, err = c.db.ExecContext(ctx, query, recordID)
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}
	return nil
}

// Encrypt encrypts PII value
func (c *MySQLConnector) Encrypt(ctx context.Context, location string, fieldName string, recordID string, encryptionKey string) error {
	originalValue, err := c.GetOriginalValue(ctx, location, fieldName, recordID)
	if err != nil {
		return err
	}

	encryptedValue := fmt.Sprintf("ENC:%s", originalValue) // Placeholder

	safeTable, err := sanitizeMySQLIdentifier(location)
	if err != nil {
		return fmt.Errorf("invalid table name: %w", err)
	}
	safeField, err := sanitizeMySQLIdentifier(fieldName)
	if err != nil {
		return fmt.Errorf("invalid field name: %w", err)
	}
	query := fmt.Sprintf("UPDATE %s SET %s = ? WHERE id = ?", safeTable, safeField)
	_, err = c.db.ExecContext(ctx, query, encryptedValue, recordID)
	if err != nil {
		return fmt.Errorf("failed to encrypt PII: %w", err)
	}
	return nil
}

// GetOriginalValue retrieves original value before remediation
func (c *MySQLConnector) GetOriginalValue(ctx context.Context, location string, fieldName string, recordID string) (string, error) {
	safeField, err := sanitizeMySQLIdentifier(fieldName)
	if err != nil {
		return "", fmt.Errorf("invalid field name: %w", err)
	}
	safeTable, err := sanitizeMySQLIdentifier(location)
	if err != nil {
		return "", fmt.Errorf("invalid table name: %w", err)
	}
	query := fmt.Sprintf("SELECT %s FROM %s WHERE id = ?", safeField, safeTable)

	var value string
	err = c.db.QueryRowContext(ctx, query, recordID).Scan(&value)
	if err != nil {
		return "", fmt.Errorf("failed to get original value: %w", err)
	}

	return value, nil
}

// RestoreValue restores original value (rollback)
func (c *MySQLConnector) RestoreValue(ctx context.Context, location string, fieldName string, recordID string, originalValue string) error {
	safeTable, err := sanitizeMySQLIdentifier(location)
	if err != nil {
		return fmt.Errorf("invalid table name: %w", err)
	}
	safeField, err := sanitizeMySQLIdentifier(fieldName)
	if err != nil {
		return fmt.Errorf("invalid field name: %w", err)
	}
	query := fmt.Sprintf("UPDATE %s SET %s = ? WHERE id = ?", safeTable, safeField)
	_, err = c.db.ExecContext(ctx, query, originalValue, recordID)
	if err != nil {
		return fmt.Errorf("failed to restore value: %w", err)
	}
	return nil
}
