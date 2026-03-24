package config

import (
	"os"
	"strconv"
)

type Config struct {
	Classification ClassificationConfig
	PIIStorage     PIIStorageConfig
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
	}
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
