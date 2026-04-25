// Package websocket pushes real-time scan-progress and alert events to connected browser clients.
package websocket

import (
	"github.com/arc-platform/backend/modules/shared/interfaces"
	"github.com/gin-gonic/gin"
)

// WebSocketModule provides real-time WebSocket communication
type WebSocketModule struct {
	service *WebSocketService
}

// NewWebSocketModule creates a new WebSocket module
func NewWebSocketModule() *WebSocketModule {
	return &WebSocketModule{
		service: NewWebSocketService(),
	}
}

// Name returns the module name
func (m *WebSocketModule) Name() string {
	return "websocket"
}

// Initialize initializes the WebSocket module
func (m *WebSocketModule) Initialize(deps *interfaces.ModuleDependencies) error {
	// WebSocket module doesn't need database dependencies
	return nil
}

// RegisterRoutes registers WebSocket routes
func (m *WebSocketModule) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/ws", m.service.HandleWebSocket)
}

// Shutdown shuts down the WebSocket module
func (m *WebSocketModule) Shutdown() error {
	// Close all WebSocket connections
	return nil
}

// GetWebSocketService returns the WebSocket service
func (m *WebSocketModule) GetWebSocketService() *WebSocketService {
	return m.service
}
