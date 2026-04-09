package middleware

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UserProfile represents a user's profile loaded from the DB for policy evaluation.
type UserProfile struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Role   string    `json:"role"`
}

// UserPolicies holds the policy rules resolved for a user's profile.
type UserPolicies struct {
	AllowedSources       []string `json:"allowed_sources,omitempty"`
	AllowedRiskTiers     []string `json:"allowed_risk_tiers,omitempty"`
	SuppressedConnectors []string `json:"suppressed_connectors,omitempty"`
}

// PolicyFilter is a set of SQL WHERE clause fragments that handlers can AND into
// their queries to enforce policy-based row filtering.
type PolicyFilter struct {
	// SourceFilter is a SQL fragment like: "source_name IN ('s3','postgres')"
	// Empty string means "no restriction".
	SourceFilter string `json:"source_filter,omitempty"`

	// RiskTierFilter is a SQL fragment like: "tier IN ('Critical','High')"
	// Empty string means "no restriction".
	RiskTierFilter string `json:"risk_tier_filter,omitempty"`

	// ConnectorFilter is a SQL fragment like: "connector NOT IN ('slack','jira')"
	// Empty string means "no restriction".
	ConnectorFilter string `json:"connector_filter,omitempty"`
}

// PolicyMiddleware resolves JWT-authenticated user_id -> profile -> policies and
// injects them into the Gin context for downstream handlers to consume.
//
// Context keys set:
//   - "profile"       → *UserProfile
//   - "policies"      → *UserPolicies
//   - "policy_filter" → *PolicyFilter
//
// If the user is not authenticated (user_id not in context), the middleware
// calls c.Next() and lets the auth middleware handle the rejection.
func PolicyMiddleware(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDRaw, exists := c.Get("user_id")
		if !exists {
			// Unauthenticated path — let auth middleware handle.
			c.Next()
			return
		}

		userIDStr, ok := userIDRaw.(string)
		if !ok {
			c.Next()
			return
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			c.Next()
			return
		}

		profile, policies := loadProfilePolicies(db, userID)

		c.Set("profile", profile)
		c.Set("policies", policies)
		c.Set("policy_filter", buildSQLFilter(policies))

		c.Next()
	}
}

// loadProfilePolicies reads the user's profile and policy rules from the database.
// Policies are stored in the user_policies table as a JSONB column. If no row
// exists, empty (unrestricted) policies are returned.
func loadProfilePolicies(db *sql.DB, userID uuid.UUID) (*UserProfile, *UserPolicies) {
	profile := &UserProfile{UserID: userID}
	policies := &UserPolicies{}

	// Load basic profile.
	err := db.QueryRow(`
		SELECT email, role FROM users WHERE id = $1
	`, userID).Scan(&profile.Email, &profile.Role)
	if err != nil {
		log.Printf("WARN: policy middleware: user profile lookup failed for %s: %v", userID, err)
		return profile, policies
	}

	// Load policies from user_policies table.
	// This table is optional — if it doesn't exist or has no row, policies are empty (unrestricted).
	var policiesJSON []byte
	err = db.QueryRow(`
		SELECT policies FROM user_policies WHERE user_id = $1
	`, userID).Scan(&policiesJSON)
	if err != nil {
		// sql.ErrNoRows or table-not-found — both mean "no restrictions".
		return profile, policies
	}

	if len(policiesJSON) > 0 {
		if err := json.Unmarshal(policiesJSON, policies); err != nil {
			log.Printf("WARN: policy middleware: failed to parse policies for %s: %v", userID, err)
		}
	}

	return profile, policies
}

// buildSQLFilter converts UserPolicies into PolicyFilter with safe SQL fragments.
// All values are quoted to prevent injection — the fragments use single-quoted
// string literals with no parameterized placeholders, designed to be ANDed into
// existing WHERE clauses.
func buildSQLFilter(policies *UserPolicies) *PolicyFilter {
	if policies == nil {
		return &PolicyFilter{}
	}

	filter := &PolicyFilter{}

	if len(policies.AllowedSources) > 0 {
		filter.SourceFilter = fmt.Sprintf("source_name IN (%s)", quoteList(policies.AllowedSources))
	}

	if len(policies.AllowedRiskTiers) > 0 {
		filter.RiskTierFilter = fmt.Sprintf("tier IN (%s)", quoteList(policies.AllowedRiskTiers))
	}

	if len(policies.SuppressedConnectors) > 0 {
		filter.ConnectorFilter = fmt.Sprintf("connector NOT IN (%s)", quoteList(policies.SuppressedConnectors))
	}

	return filter
}

// quoteList produces a comma-separated list of single-quoted, SQL-safe strings.
// Each value has single quotes escaped ('' in SQL) to prevent injection.
func quoteList(values []string) string {
	quoted := make([]string, len(values))
	for i, v := range values {
		// Escape single quotes within the value for SQL safety.
		safe := strings.ReplaceAll(v, "'", "''")
		quoted[i] = fmt.Sprintf("'%s'", safe)
	}
	return strings.Join(quoted, ",")
}

// GetPolicyFilter extracts the PolicyFilter from the Gin context.
// Returns nil if no filter was set (unauthenticated or unrestricted user).
func GetPolicyFilter(c *gin.Context) *PolicyFilter {
	v, exists := c.Get("policy_filter")
	if !exists {
		return nil
	}
	pf, ok := v.(*PolicyFilter)
	if !ok {
		return nil
	}
	return pf
}

// GetUserProfile extracts the UserProfile from the Gin context.
func GetUserProfile(c *gin.Context) *UserProfile {
	v, exists := c.Get("profile")
	if !exists {
		return nil
	}
	p, ok := v.(*UserProfile)
	if !ok {
		return nil
	}
	return p
}
