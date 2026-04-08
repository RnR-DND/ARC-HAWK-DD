package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Classification ClassificationConfig
	PIIStorage     PIIStorageConfig
	Discovery      DiscoveryConfig

	// DiscoverySnapshotInterval is exposed at top level for convenience by the
	// discovery module's snapshot worker. Mirrors Discovery.SnapshotInterval.
	DiscoverySnapshotInterval time.Duration
}

// DiscoveryConfig holds tunables for the data discovery module.
type DiscoveryConfig struct {
	SnapshotInterval     time.Duration // default 24h, env DISCOVERY_SNAPSHOT_INTERVAL
	SnapshotTimeout      time.Duration // default 5m per source, env DISCOVERY_SNAPSHOT_TIMEOUT
	ReportMaxRows        int           // default 10000, env DISCOVERY_REPORT_MAX_ROWS
	RiskWeightVolume     float64       // default 1.0
	RiskWeightSensitivity float64      // default 2.0
	RiskWeightExposure   float64       // default 1.5
	FactRetentionDays    int           // default 90 (v1.5: archive)
}

type ClassificationConfig struct {
	WeightRules   float64
	WeightContext float64
	WeightEntropy float64
	Threshold     float64
}

type PIIStringMode string

const (
	PIIModeFull PIIStringMode = "full"
	PIIModeMask PIIStringMode = "mask"
	PIIModeNone PIIStringMode = "none"
)

type PIIStorageConfig struct {
	Mode PIIStringMode
}

func LoadConfig() *Config {
	discovery := DiscoveryConfig{
		SnapshotInterval:      getEnvDuration("DISCOVERY_SNAPSHOT_INTERVAL", 24*time.Hour),
		SnapshotTimeout:       getEnvDuration("DISCOVERY_SNAPSHOT_TIMEOUT", 5*time.Minute),
		ReportMaxRows:         getEnvInt("DISCOVERY_REPORT_MAX_ROWS", 10000),
		RiskWeightVolume:      getEnvFloat("DISCOVERY_RISK_WEIGHT_VOLUME", 1.0),
		RiskWeightSensitivity: getEnvFloat("DISCOVERY_RISK_WEIGHT_SENSITIVITY", 2.0),
		RiskWeightExposure:    getEnvFloat("DISCOVERY_RISK_WEIGHT_EXPOSURE", 1.5),
		FactRetentionDays:     getEnvInt("DISCOVERY_FACT_RETENTION_DAYS", 90),
	}

	return &Config{
		Classification: ClassificationConfig{
			WeightRules:   getEnvFloat("CLASSIFICATION_WEIGHT_RULES", 0.40),
			WeightContext: getEnvFloat("CLASSIFICATION_WEIGHT_CONTEXT", 0.30),
			WeightEntropy: getEnvFloat("CLASSIFICATION_WEIGHT_ENTROPY", 0.10),
			Threshold:     getEnvFloat("CLASSIFICATION_THRESHOLD", 0.60),
		},
		PIIStorage: PIIStorageConfig{
			Mode: getPIIMode(),
		},
		Discovery:                 discovery,
		DiscoverySnapshotInterval: discovery.SnapshotInterval,
	}
}

func getEnvInt(key string, defaultVal int) int {
	if val, exists := os.LookupEnv(key); exists {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val, exists := os.LookupEnv(key); exists {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultVal
}

func getEnvFloat(key string, defaultVal float64) float64 {
	if val, exists := os.LookupEnv(key); exists {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return defaultVal
}

func getPIIMode() PIIStringMode {
	mode := os.Getenv("PII_STORE_MODE")
	switch PIIStringMode(mode) {
	case PIIModeFull, PIIModeMask, PIIModeNone:
		return PIIStringMode(mode)
	default:
		return PIIModeFull
	}
}

func (m PIIStringMode) ShouldStorePII() bool {
	return m != PIIModeNone
}

func (m PIIStringMode) ShouldMaskPII() bool {
	return m == PIIModeMask
}
