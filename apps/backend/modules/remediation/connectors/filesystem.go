package connectors

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// FilesystemConnector implements remediation for filesystem
type FilesystemConnector struct {
	basePath string
}

// safeJoinPath joins basePath and location, rejecting any path traversal attempts.
func safeJoinPath(basePath, location string) (string, error) {
	joined := filepath.Join(basePath, location)
	rel, err := filepath.Rel(basePath, joined)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path traversal attempt detected: %q", location)
	}
	return joined, nil
}

// encryptAESGCM encrypts plaintext with AES-GCM.
// keyStr must be either a 32-byte raw string or a 64-character hex-encoded string.
func encryptAESGCM(keyStr, plaintext string) (string, error) {
	var key []byte
	if len(keyStr) == 64 {
		var err error
		key, err = hex.DecodeString(keyStr)
		if err != nil {
			return "", fmt.Errorf("invalid hex encryption key: %w", err)
		}
	} else if len(keyStr) == 32 {
		key = []byte(keyStr)
	} else {
		return "", fmt.Errorf("invalid encryption key: must be 32 bytes or 64-char hex string")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
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
	filePath, err := safeJoinPath(c.basePath, location)
	if err != nil {
		return err
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Create backup
	backupPath := filePath + ".backup"
	if err := os.WriteFile(backupPath, content, 0644); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Mask PII based on field name pattern
	maskedContent := c.maskPIIInContent(string(content), fieldName)

	// Write masked content
	if err := os.WriteFile(filePath, []byte(maskedContent), 0644); err != nil {
		return fmt.Errorf("failed to write masked file: %w", err)
	}

	return nil
}

// Delete removes file
func (c *FilesystemConnector) Delete(ctx context.Context, location string, recordID string) error {
	filePath, err := safeJoinPath(c.basePath, location)
	if err != nil {
		return err
	}

	// Create backup before deletion
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file for backup: %w", err)
	}

	backupPath := filePath + ".deleted.backup"
	if err := os.WriteFile(backupPath, content, 0644); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Delete file
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// Encrypt encrypts file content with AES-GCM
func (c *FilesystemConnector) Encrypt(ctx context.Context, location string, fieldName string, recordID string, encryptionKey string) error {
	filePath, err := safeJoinPath(c.basePath, location)
	if err != nil {
		return err
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Create backup
	backupPath := filePath + ".backup"
	if err := os.WriteFile(backupPath, content, 0644); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Encrypt with AES-GCM
	encryptedContent, err := encryptAESGCM(encryptionKey, string(content))
	if err != nil {
		return fmt.Errorf("failed to encrypt file: %w", err)
	}

	// Write encrypted content
	if err := os.WriteFile(filePath, []byte(encryptedContent), 0644); err != nil {
		return fmt.Errorf("failed to write encrypted file: %w", err)
	}

	return nil
}

// GetOriginalValue retrieves original file content
func (c *FilesystemConnector) GetOriginalValue(ctx context.Context, location string, fieldName string, recordID string) (string, error) {
	filePath, err := safeJoinPath(c.basePath, location)
	if err != nil {
		return "", err
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(content), nil
}

// RestoreValue restores original file content
func (c *FilesystemConnector) RestoreValue(ctx context.Context, location string, fieldName string, recordID string, originalValue string) error {
	filePath, err := safeJoinPath(c.basePath, location)
	if err != nil {
		return err
	}

	// Write original content back
	if err := os.WriteFile(filePath, []byte(originalValue), 0644); err != nil {
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
