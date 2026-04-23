package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
)

type contextKey string

const (
	ContextKeyIPAddress contextKey = "ip_address"
	ContextKeyUserAgent contextKey = "user_agent"
)

// IPContextMiddleware extracts client IP and User-Agent into request context.
func IPContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, ContextKeyIPAddress, c.ClientIP())
		ctx = context.WithValue(ctx, ContextKeyUserAgent, c.Request.Header.Get("User-Agent"))
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
