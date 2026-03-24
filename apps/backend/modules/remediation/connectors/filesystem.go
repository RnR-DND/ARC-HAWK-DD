package connectors

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// FilesystemConnector implements remediation for filesystem
type FilesystemConnector struct {
	basePath string
}

// Connect establishes connection to filesystem
func (c *FilesystemConnector) Connect(ctx context.Context, config map[string]interface{}) error {
	basePath, ok := config["base_path"].(string)
	if !ok {
		return fmt.Errorf("base_path not found in config")
	}

	// Verify base path exists
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		return fmt.Errorf("base path does not exist: %s", basePath)
	}

	c.basePath = basePath
	return nil
}

// Close closes the filesystem connection
func (c *FilesystemConnector) Close() error {
	return nil
}

// Mask redacts PII in file
// location: relative file path from base_path
// fieldName: pattern to match (e.g., "email", "phone")
// recordID: line number or unique identifier
func (c *FilesystemConnector) Mask(ctx context.Context, location string, fieldName string, recordID string) error {
	filePath := filepath.Join(c.basePath, location)

	// Read file content
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Create backup
	backupPath := filePath + ".backup"
	if err := ioutil.WriteFile(backupPath, content, 0644); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Mask PII based on field name pattern
	maskedContent := c.maskPIIInContent(string(content), fieldName)

	// Write masked content
	if err := ioutil.WriteFile(filePath, []byte(maskedContent), 0644); err != nil {
		return fmt.Errorf("failed to write masked file: %w", err)
	}

	return nil
}

// Delete removes file
func (c *FilesystemConnector) Delete(ctx context.Context, location string, recordID string) error {
	filePath := filepath.Join(c.basePath, location)

	// Create backup before deletion
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file for backup: %w", err)
	}

	backupPath := filePath + ".deleted.backup"
	if err := ioutil.WriteFile(backupPath, content, 0644); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Delete file
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// Encrypt encrypts file
func (c *FilesystemConnector) Encrypt(ctx context.Context, location string, fieldName string, recordID string, encryptionKey string) error {
	filePath := filepath.Join(c.basePath, location)

	// Read file content
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Create backup
	backupPath := filePath + ".backup"
	if err := ioutil.WriteFile(backupPath, content, 0644); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Simple encryption (placeholder - use proper encryption in production)
	encryptedContent := fmt.Sprintf("ENCRYPTED:%s", string(content))

	// Write encrypted content
	if err := ioutil.WriteFile(filePath, []byte(encryptedContent), 0644); err != nil {
		return fmt.Errorf("failed to write encrypted file: %w", err)
	}

	return nil
}

// GetOriginalValue retrieves original file content
func (c *FilesystemConnector) GetOriginalValue(ctx context.Context, location string, fieldName string, recordID string) (string, error) {
	filePath := filepath.Join(c.basePath, location)

	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(content), nil
}

// RestoreValue restores original file content
func (c *FilesystemConnector) RestoreValue(ctx context.Context, location string, fieldName string, recordID string, originalValue string) error {
	filePath := filepath.Join(c.basePath, location)

	// Write original content back
	if err := ioutil.WriteFile(filePath, []byte(originalValue), 0644); err != nil {
		return fmt.Errorf("failed to restore file: %w", err)
	}

	// Remove backup if exists
	backupPath := filePath + ".backup"
	os.Remove(backupPath)

	return nil
}

// Helper function to mask PII in content
func (c *FilesystemConnector) maskPIIInContent(content string, fieldName string) string {
	// Define PII patterns
	patterns := map[string]string{
		"email":       `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`,
		"phone":       `\b\d{10}\b`,
		"aadhaar":     `\b\d{4}\s\d{4}\s\d{4}\b`,
		"pan":         `\b[A-Z]{5}\d{4}[A-Z]\b`,
		"credit_card": `\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`,
	}

	pattern, ok := patterns[strings.ToLower(fieldName)]
	if !ok {
		// Default: mask anything that looks like sensitive data
		return strings.ReplaceAll(content, fieldName, "***REDACTED***")
	}

	re := regexp.MustCompile(pattern)
	return re.ReplaceAllString(content, "***REDACTED***")
}
