package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter implements a simple in-memory rate limiter using token bucket algorithm
type RateLimiter struct {
	mu           sync.RWMutex
	clients      map[string]*clientState
	requestsRate int           // requests per window
	window       time.Duration // time window
	cleanupTick  time.Duration
	stop         chan struct{}
}

type clientState struct {
	tokens    int
	lastReset time.Time
}

// RateLimiterConfig configures rate limiting parameters
type RateLimiterConfig struct {
	RequestsPerMinute int  // Max requests per minute per IP
	BurstSize         int  // Max burst size (defaults to RequestsPerMinute)
	Enabled           bool // Enable/disable rate limiting
}

// DefaultRateLimiterConfig returns sensible defaults
func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		RequestsPerMinute: 100,
		BurstSize:         100,
		Enabled:           true,
	}
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimiterConfig) *RateLimiter {
	if !config.Enabled {
		return nil
	}

	rl := &RateLimiter{
		clients:      make(map[string]*clientState),
		requestsRate: config.RequestsPerMinute,
		window:       time.Minute,
		cleanupTick:  5 * time.Minute,
		stop:         make(chan struct{}),
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Middleware returns a Gin middleware handler for rate limiting.
//
// Rate limiting key selection (in order):
//  1. tenant_id + user_id (from JWT/API-key auth) — prevents one tenant's
//     noisy user from affecting another tenant. Also immune to X-Forwarded-For
//     spoofing.
//  2. Client IP (Gin's ClientIP, honors trusted proxy config) — for
//     unauthenticated routes (login, register, health).
//
// P1-11 replaces the previous IP-only scheme, which let a single shared IP
// (corporate NAT, mobile carrier) DOS an entire tenant.
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if rl == nil {
			c.Next()
			return
		}

		key := rateLimitKey(c)

		if !rl.allow(key) {
			c.Header("Retry-After", "60")
			c.Header("X-RateLimit-Limit", strconv.Itoa(rl.requestsRate))
			c.Header("X-RateLimit-Remaining", "0")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Too many requests. Please wait and try again.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// rateLimitKey chooses a stable identity for rate-limit bucketing.
func rateLimitKey(c *gin.Context) string {
	if tenantID, ok := c.Get("tenant_id"); ok && tenantID != nil {
		if userID, ok := c.Get("user_id"); ok && userID != nil {
			return "t:" + toStr(tenantID) + "|u:" + toStr(userID)
		}
		return "t:" + toStr(tenantID)
	}
	return "ip:" + c.ClientIP()
}

func toStr(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return stringerOrFmt(v)
}

func stringerOrFmt(v interface{}) string {
	if s, ok := v.(interface{ String() string }); ok {
		return s.String()
	}
	return ""
}

// allow checks if the client is allowed to make a request
func (rl *RateLimiter) allow(clientIP string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	state, exists := rl.clients[clientIP]
	if !exists {
		// New client
		rl.clients[clientIP] = &clientState{
			tokens:    rl.requestsRate - 1,
			lastReset: now,
		}
		return true
	}

	// Check if window has passed and reset tokens
	if now.Sub(state.lastReset) >= rl.window {
		state.tokens = rl.requestsRate - 1
		state.lastReset = now
		return true
	}

	// Check if we have tokens left
	if state.tokens > 0 {
		state.tokens--
		return true
	}

	return false
}

// cleanup periodically removes stale entries
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupTick)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for ip, state := range rl.clients {
				if now.Sub(state.lastReset) > 2*rl.window {
					delete(rl.clients, ip)
				}
			}
			rl.mu.Unlock()
		case <-rl.stop:
			return
		}
	}
}

// Stop shuts down the cleanup goroutine.
func (rl *RateLimiter) Stop() {
	close(rl.stop)
}

// StrictRateLimiter returns a stricter rate limiter for sensitive endpoints
func StrictRateLimiter() *RateLimiter {
	return NewRateLimiter(RateLimiterConfig{
		RequestsPerMinute: 10,
		Enabled:           true,
	})
}

// APIRateLimiter returns a rate limiter for general API endpoints
func APIRateLimiter() *RateLimiter {
	return NewRateLimiter(DefaultRateLimiterConfig())
}
