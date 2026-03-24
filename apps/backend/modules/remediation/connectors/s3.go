package connectors

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// S3Connector implements remediation for S3 storage
type S3Connector struct {
	client *s3.S3
	bucket string
}

// Connect establishes connection to S3
func (c *S3Connector) Connect(ctx context.Context, config map[string]interface{}) error {
	region, ok := config["region"].(string)
	if !ok {
		return fmt.Errorf("region not found in config")
	}

	bucket, ok := config["bucket"].(string)
	if !ok {
		return fmt.Errorf("bucket not found in config")
	}

	accessKey, ok := config["access_key"].(string)
	if !ok {
		return fmt.Errorf("access_key not found in config")
	}

	secretKey, ok := config["secret_key"].(string)
	if !ok {
		return fmt.Errorf("secret_key not found in config")
	}

	// Create AWS session
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
	})
	if err != nil {
		return fmt.Errorf("failed to create AWS session: %w", err)
	}

	c.client = s3.New(sess)
	c.bucket = bucket
	return nil
}

// Close closes the S3 connection
func (c *S3Connector) Close() error {
	return nil
}

// Mask redacts PII in S3 object
// location: S3 object key
// fieldName: pattern to match
// recordID: not used for S3
func (c *S3Connector) Mask(ctx context.Context, location string, fieldName string, recordID string) error {
	// Get original object
	result, err := c.client.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(location),
	})
	if err != nil {
		return fmt.Errorf("failed to get S3 object: %w", err)
	}
	defer result.Body.Close()

	// Read content
	content, err := io.ReadAll(result.Body)
	if err != nil {
		return fmt.Errorf("failed to read S3 object: %w", err)
	}

	// Create backup (versioning should be enabled on bucket)
	backupKey := location + ".backup"
	_, err = c.client.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(backupKey),
		Body:   bytes.NewReader(content),
	})
	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Mask PII
	maskedContent := c.maskPIIInContent(string(content), fieldName)

	// Upload masked content
	_, err = c.client.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(location),
		Body:   bytes.NewReader([]byte(maskedContent)),
	})
	if err != nil {
		return fmt.Errorf("failed to upload masked object: %w", err)
	}

	return nil
}

// Delete removes S3 object
func (c *S3Connector) Delete(ctx context.Context, location string, recordID string) error {
	// Get original object for backup
	result, err := c.client.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(location),
	})
	if err != nil {
		return fmt.Errorf("failed to get S3 object: %w", err)
	}
	defer result.Body.Close()

	content, err := io.ReadAll(result.Body)
	if err != nil {
		return fmt.Errorf("failed to read S3 object: %w", err)
	}

	// Create backup
	backupKey := location + ".deleted.backup"
	_, err = c.client.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(backupKey),
		Body:   bytes.NewReader(content),
	})
	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Delete object
	_, err = c.client.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(location),
	})
	if err != nil {
		return fmt.Errorf("failed to delete S3 object: %w", err)
	}

	return nil
}

// Encrypt encrypts S3 object
func (c *S3Connector) Encrypt(ctx context.Context, location string, fieldName string, recordID string, encryptionKey string) error {
	// Get original object
	result, err := c.client.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(location),
	})
	if err != nil {
		return fmt.Errorf("failed to get S3 object: %w", err)
	}
	defer result.Body.Close()

	content, err := io.ReadAll(result.Body)
	if err != nil {
		return fmt.Errorf("failed to read S3 object: %w", err)
	}

	// Create backup
	backupKey := location + ".backup"
	_, err = c.client.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(backupKey),
		Body:   bytes.NewReader(content),
	})
	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Upload with server-side encryption
	_, err = c.client.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket:               aws.String(c.bucket),
		Key:                  aws.String(location),
		Body:                 bytes.NewReader(content),
		ServerSideEncryption: aws.String("AES256"),
	})
	if err != nil {
		return fmt.Errorf("failed to upload encrypted object: %w", err)
	}

	return nil
}

// GetOriginalValue retrieves original S3 object content
func (c *S3Connector) GetOriginalValue(ctx context.Context, location string, fieldName string, recordID string) (string, error) {
	result, err := c.client.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(location),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get S3 object: %w", err)
	}
	defer result.Body.Close()

	content, err := io.ReadAll(result.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read S3 object: %w", err)
	}

	return string(content), nil
}

// RestoreValue restores original S3 object content
func (c *S3Connector) RestoreValue(ctx context.Context, location string, fieldName string, recordID string, originalValue string) error {
	// Upload original content
	_, err := c.client.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(location),
		Body:   bytes.NewReader([]byte(originalValue)),
	})
	if err != nil {
		return fmt.Errorf("failed to restore S3 object: %w", err)
	}

	// Remove backup
	backupKey := location + ".backup"
	c.client.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(backupKey),
	})

	return nil
}

// Helper function to mask PII in content
func (c *S3Connector) maskPIIInContent(content string, fieldName string) string {
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
