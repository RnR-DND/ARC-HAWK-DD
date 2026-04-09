package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config holds all agent configuration loaded from YAML + environment overrides.
type Config struct {
	ServerURL    string     `yaml:"server_url"`
	AgentID      string     `yaml:"agent_id"`
	ScanSchedule string     `yaml:"scan_schedule"`
	DataDir      string     `yaml:"data_dir"`
	BufferMaxMB  int        `yaml:"buffer_max_mb"`
	Auth         AuthConfig `yaml:"auth"`
	LogLevel     string     `yaml:"log_level"`
}

// AuthConfig holds Keycloak client-credentials settings.
type AuthConfig struct {
	KeycloakURL  string `yaml:"keycloak_url"`
	Realm        string `yaml:"realm"`
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
}

// defaults returns a Config with sensible default values.
func defaults() *Config {
	return &Config{
		ServerURL:    "https://scanner.hawk.internal",
		AgentID:      "agent-001",
		ScanSchedule: "0 0 * * 0",
		DataDir:      "/var/lib/hawk-agent",
		BufferMaxMB:  500,
		Auth: AuthConfig{
			KeycloakURL: "https://auth.hawk.internal",
			Realm:       "hawk",
			ClientID:    "hawk-agent",
		},
		LogLevel: "info",
	}
}

// Load reads config from YAML (if present) and applies environment overrides.
// Config file path: $HAWK_CONFIG_FILE or $HAWK_DATA_DIR/config.yaml or /etc/hawk-agent/config.yaml.
func Load() (*Config, error) {
	cfg := defaults()

	// Determine config file path.
	configPath := os.Getenv("HAWK_CONFIG_FILE")
	if configPath == "" {
		dataDir := os.Getenv("HAWK_DATA_DIR")
		if dataDir != "" {
			configPath = filepath.Join(dataDir, "config.yaml")
		} else {
			configPath = "/etc/hawk-agent/config.yaml"
		}
	}

	// Load YAML if file exists.
	data, err := os.ReadFile(configPath)
	if err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse config %s: %w", configPath, err)
		}
	}
	// Silently ignore missing config file; we fall through to env overrides.

	// Environment overrides take precedence.
	applyEnv(cfg)

	// Validate required fields.
	if cfg.AgentID == "" {
		return nil, fmt.Errorf("agent_id is required (set HAWK_AGENT_ID or configure in YAML)")
	}
	if cfg.ServerURL == "" {
		return nil, fmt.Errorf("server_url is required (set HAWK_SERVER_URL or configure in YAML)")
	}

	return cfg, nil
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("HAWK_SERVER_URL"); v != "" {
		cfg.ServerURL = v
	}
	if v := os.Getenv("HAWK_AGENT_ID"); v != "" {
		cfg.AgentID = v
	}
	if v := os.Getenv("HAWK_SCAN_SCHEDULE"); v != "" {
		cfg.ScanSchedule = v
	}
	if v := os.Getenv("HAWK_DATA_DIR"); v != "" {
		cfg.DataDir = v
	}
	if v := os.Getenv("HAWK_BUFFER_MAX_MB"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.BufferMaxMB = n
		}
	}
	if v := os.Getenv("HAWK_LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}

	// Auth overrides.
	if v := os.Getenv("HAWK_KEYCLOAK_URL"); v != "" {
		cfg.Auth.KeycloakURL = v
	}
	if v := os.Getenv("HAWK_KEYCLOAK_REALM"); v != "" {
		cfg.Auth.Realm = v
	}
	if v := os.Getenv("HAWK_AGENT_CLIENT_ID"); v != "" {
		cfg.Auth.ClientID = v
	}
	if v := os.Getenv("HAWK_AGENT_CLIENT_SECRET"); v != "" {
		cfg.Auth.ClientSecret = v
	}
}

// DBPath returns the full path to the SQLite queue database.
func (c *Config) DBPath() string {
	return filepath.Join(c.DataDir, "agent_queue.db")
}
