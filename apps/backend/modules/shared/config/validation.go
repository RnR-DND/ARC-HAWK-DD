package config

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// EnvError collects all environment validation errors so operators can fix
// everything in one pass instead of playing whack-a-mole.
type EnvError struct {
	Missing []string
	Invalid []string
}

func (e *EnvError) HasErrors() bool {
	return len(e.Missing) > 0 || len(e.Invalid) > 0
}

func (e *EnvError) Error() string {
	var parts []string
	if len(e.Missing) > 0 {
		parts = append(parts, fmt.Sprintf("missing: %s", strings.Join(e.Missing, ", ")))
	}
	if len(e.Invalid) > 0 {
		parts = append(parts, fmt.Sprintf("invalid: %s", strings.Join(e.Invalid, ", ")))
	}
	return "environment validation failed — " + strings.Join(parts, "; ")
}

// ValidateRequiredEnvVars checks all environment variables needed for
// the backend to function. Call this at the very start of main(), before
// any database or service connections.
//
// isProduction: when true, also enforces JWT_SECRET and AUTH_REQUIRED.
func ValidateRequiredEnvVars(isProduction bool) error {
	e := &EnvError{}

	// ── Database (always required) ─────────────────────────────────────
	requireNonEmpty(e, "DB_HOST")
	requireNonEmpty(e, "DB_PORT")
	requireNonEmpty(e, "DB_USER")
	requireNonEmpty(e, "DB_PASSWORD")
	requireNonEmpty(e, "DB_NAME")

	// ── Neo4j (always required) ────────────────────────────────────────
	requireNonEmpty(e, "NEO4J_PASSWORD")

	// ── Encryption (always required, must be 32 bytes) ─────────────────
	if key := os.Getenv("ENCRYPTION_KEY"); key == "" {
		e.Missing = append(e.Missing, "ENCRYPTION_KEY")
	} else if len(key) != 32 {
		e.Invalid = append(e.Invalid,
			fmt.Sprintf("ENCRYPTION_KEY (must be exactly 32 bytes, got %d)", len(key)))
	}

	// ── Production-only requirements ───────────────────────────────────
	if isProduction {
		if secret := os.Getenv("JWT_SECRET"); secret == "" {
			e.Missing = append(e.Missing, "JWT_SECRET")
		} else if len(secret) < 32 {
			e.Invalid = append(e.Invalid,
				fmt.Sprintf("JWT_SECRET (must be at least 32 chars in production, got %d)", len(secret)))
		}
	}

	if e.HasErrors() {
		return e
	}
	return nil
}

// MustEnv returns the value of key, or calls log.Fatal with a helpful message.
// Use sparingly — prefer ValidateRequiredEnvVars at startup.
func MustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required environment variable %s is not set", key)
	}
	return v
}

// requireNonEmpty appends key to e.Missing if the variable is unset or empty.
func requireNonEmpty(e *EnvError, key string) {
	if os.Getenv(key) == "" {
		e.Missing = append(e.Missing, key)
	}
}
