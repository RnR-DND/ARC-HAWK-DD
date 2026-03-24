package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"os"
)

// EncryptionService provides AES-256-GCM encryption for sensitive data
type EncryptionService struct {
	key []byte
}

// NewEncryptionService creates a new encryption service using the ENCRYPTION_KEY environment variable
// The key must be exactly 32 bytes (256 bits) for AES-256
func NewEncryptionService() (*EncryptionService, error) {
	key := os.Getenv("ENCRYPTION_KEY")
	if key == "" {
		return nil, errors.New("ENCRYPTION_KEY environment variable not set")
	}
	if len(key) != 32 {
		return nil, errors.New("ENCRYPTION_KEY must be exactly 32 bytes for AES-256")
	}
	return &EncryptionService{key: []byte(key)}, nil
}

// Encrypt encrypts data using AES-256-GCM
// The data is first marshaled to JSON, then encrypted
// Returns the ciphertext with the nonce prepended
func (s *EncryptionService) Encrypt(data interface{}) ([]byte, error) {
	// Marshal data to JSON
	plaintext, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	// Create AES cipher
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Encrypt and prepend nonce to ciphertext
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts data using AES-256-GCM
// The ciphertext must have the nonce prepended (as returned by Encrypt)
// The decrypted data is unmarshaled into the dest parameter
func (s *EncryptionService) Decrypt(ciphertext []byte, dest interface{}) error {
	// Create AES cipher
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return err
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	// Extract nonce from ciphertext
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return err
	}

	// Unmarshal JSON into dest
	return json.Unmarshal(plaintext, dest)
}
