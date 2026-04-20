package vault

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	vaultapi "github.com/hashicorp/vault/api"
)

// Client wraps HashiCorp Vault KV v2 operations for credential storage.
// When Vault is disabled (VAULT_ENABLED != "true"), all methods are no-ops
// that return ErrVaultDisabled so callers can fall back to PostgreSQL.
type Client struct {
	client      *vaultapi.Client
	enabled     bool
	secretMount string // KV v2 mount path, e.g. "secret"
	mu          sync.RWMutex
}

// ErrVaultDisabled is returned when Vault integration is not enabled.
var ErrVaultDisabled = fmt.Errorf("vault integration is disabled")

// NewClient creates a Vault client from environment variables.
//
// Environment:
//   - VAULT_ENABLED  — "true" to enable (default "false")
//   - VAULT_ADDR     — Vault server address (default "http://vault:8200")
//   - VAULT_TOKEN    — Authentication token (required when enabled)
//   - VAULT_SECRET_MOUNT — KV v2 mount path (default "secret")
func NewClient() (*Client, error) {
	enabled := strings.EqualFold(os.Getenv("VAULT_ENABLED"), "true")
	if !enabled {
		log.Println("Vault integration disabled (VAULT_ENABLED != true)")
		return &Client{enabled: false}, nil
	}

	cfg := vaultapi.DefaultConfig()
	addr := os.Getenv("VAULT_ADDR")
	if addr != "" {
		cfg.Address = addr
	}

	vc, err := vaultapi.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	token := os.Getenv("VAULT_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("VAULT_TOKEN is required when VAULT_ENABLED=true")
	}
	vc.SetToken(token)

	mount := os.Getenv("VAULT_SECRET_MOUNT")
	if mount == "" {
		mount = "secret"
	}

	log.Printf("Vault integration enabled (addr=%s, mount=%s)", cfg.Address, mount)
	return &Client{
		client:      vc,
		enabled:     true,
		secretMount: mount,
	}, nil
}

// IsEnabled reports whether the Vault backend is active.
func (c *Client) IsEnabled() bool {
	return c.enabled
}

// connectionPath builds the KV v2 data path for a connection.
// Path convention: {mount}/data/connections/{sourceType}/{profileName}
func (c *Client) connectionPath(sourceType, profileName string) string {
	return fmt.Sprintf("%s/data/connections/%s/%s", c.secretMount, sourceType, profileName)
}

// WriteConnectionSecret stores a connection's config in Vault KV v2.
func (c *Client) WriteConnectionSecret(sourceType, profileName string, config map[string]interface{}) error {
	if !c.enabled {
		return ErrVaultDisabled
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	path := c.connectionPath(sourceType, profileName)
	data := map[string]interface{}{
		"data": config,
	}

	_, err := c.client.Logical().Write(path, data)
	if err != nil {
		return fmt.Errorf("vault write failed (%s): %w", path, err)
	}

	log.Printf("Vault: stored secret for %s/%s", sourceType, profileName)
	return nil
}

// ReadConnectionSecret retrieves a connection's config from Vault KV v2.
// Returns nil, nil if the secret does not exist.
func (c *Client) ReadConnectionSecret(sourceType, profileName string) (map[string]interface{}, error) {
	if !c.enabled {
		return nil, ErrVaultDisabled
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	path := c.connectionPath(sourceType, profileName)
	secret, err := c.client.Logical().Read(path)
	if err != nil {
		return nil, fmt.Errorf("vault read failed (%s): %w", path, err)
	}
	if secret == nil || secret.Data == nil {
		return nil, nil
	}

	// KV v2 wraps the actual data under a "data" key
	data, ok := secret.Data["data"]
	if !ok {
		return nil, nil
	}

	// The Vault API returns map[string]interface{} but we need to ensure
	// the types are consistent with what JSON would produce.
	switch v := data.(type) {
	case map[string]interface{}:
		return v, nil
	default:
		// Round-trip through JSON for type consistency
		raw, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("vault data marshal failed: %w", err)
		}
		var result map[string]interface{}
		if err := json.Unmarshal(raw, &result); err != nil {
			return nil, fmt.Errorf("vault data unmarshal failed: %w", err)
		}
		return result, nil
	}
}

// DeleteConnectionSecret removes a connection's secret from Vault.
// Uses metadata delete (permanently removes all versions).
func (c *Client) DeleteConnectionSecret(sourceType, profileName string) error {
	if !c.enabled {
		return ErrVaultDisabled
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// KV v2 metadata path for permanent deletion
	path := fmt.Sprintf("%s/metadata/connections/%s/%s", c.secretMount, sourceType, profileName)
	_, err := c.client.Logical().Delete(path)
	if err != nil {
		return fmt.Errorf("vault delete failed (%s): %w", path, err)
	}

	log.Printf("Vault: deleted secret for %s/%s", sourceType, profileName)
	return nil
}

// HealthCheck verifies connectivity to Vault.
func (c *Client) HealthCheck() error {
	if !c.enabled {
		return ErrVaultDisabled
	}

	health, err := c.client.Sys().Health()
	if err != nil {
		return fmt.Errorf("vault health check failed: %w", err)
	}
	if health.Sealed {
		return fmt.Errorf("vault is sealed")
	}
	return nil
}
