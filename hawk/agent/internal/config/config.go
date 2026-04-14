// Package config provides configuration for the hawk agent.
package config

// Config holds all runtime configuration for the hawk agent.
type Config struct {
	AgentID      string
	ServerURL    string
	ScanSchedule string
	DBPath       string
	MaxSizeMB    int
}

// GetScanSchedule returns the cron schedule string.
func (c *Config) GetScanSchedule() string {
	if c.ScanSchedule == "" {
		return "0 * * * *" // hourly default
	}
	return c.ScanSchedule
}
