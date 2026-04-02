package websocket

import (
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// MessageType represents different types of WebSocket messages
type MessageType string

const (
	MessageTypeScanProgress MessageType = "scan_progress"
	MessageTypeNewFinding   MessageType = "new_finding"
	MessageTypeScanComplete MessageType = "scan_complete"
	MessageTypeSystemStatus MessageType = "system_status"
	MessageTypeScanStarted  MessageType = "scan_started"
)

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	Type      MessageType `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// Client represents a WebSocket client connection
type Client struct {
	ID   string
	Conn *websocket.Conn
	Send chan WebSocketMessage
	Hub  *Hub
}

// Hub maintains the set of active clients and broadcasts messages to clients
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan WebSocketMessage
	register   chan *Client
	unregister chan *Client
	mutex      sync.RWMutex
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan WebSocketMessage),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub and handles client connections
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()
			log.Printf("WebSocket client connected: %s", client.ID)

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mutex.Unlock()
			log.Printf("WebSocket client disconnected: %s", client.ID)

		case message := <-h.broadcast:
			h.mutex.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mutex.RUnlock()
		}
	}
}

// Broadcast sends a message to all connected clients
func (h *Hub) Broadcast(message WebSocketMessage) {
	select {
	case h.broadcast <- message:
	default:
		log.Println("WebSocket broadcast channel full, dropping message")
	}
}

// GetClientCount returns the number of connected clients
func (h *Hub) GetClientCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.clients)
}

// WebSocketService manages WebSocket connections and message broadcasting
type WebSocketService struct {
	hub      *Hub
	upgrader websocket.Upgrader
}

// NewWebSocketService creates a new WebSocket service
func NewWebSocketService() *WebSocketService {
	hub := NewHub()
	go hub.Run()

	return &WebSocketService{
		hub: hub,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// Allow connections from allowed origins
				origin := r.Header.Get("Origin")
				allowedOrigins := []string{
					"http://localhost:3000",
					"http://localhost:3001",
					"https://localhost:3000",
					"https://localhost:3001",
				}
				for _, allowed := range allowedOrigins {
					if origin == allowed {
						return true
					}
				}
				return false
			},
		},
	}
}

// HandleWebSocket upgrades HTTP connection to WebSocket and manages the client
func (ws *WebSocketService) HandleWebSocket(c *gin.Context) {
	// H-3: Reject unauthenticated connections when auth is required
	userID := c.GetString("user_id")
	if os.Getenv("AUTH_REQUIRED") != "false" && userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required for WebSocket connection"})
		return
	}

	conn, err := ws.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	client := &Client{
		ID:   userID, // Get from auth middleware
		Conn: conn,
		Send: make(chan WebSocketMessage, 256),
		Hub:  ws.hub,
	}

	if client.ID == "" {
		client.ID = "anonymous-" + time.Now().Format("20060102150405")
	}

	client.Hub.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// writePump pumps messages from the hub to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteJSON(message); err != nil {
				log.Printf("Error writing WebSocket message: %v", err)
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump pumps messages from the websocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
		// For now, we don't handle incoming messages from clients
		// This can be extended for bidirectional communication
	}
}

// BroadcastScanProgress broadcasts scan progress updates
func (ws *WebSocketService) BroadcastScanProgress(scanID string, progress int, status string, message string) {
	ws.hub.Broadcast(WebSocketMessage{
		Type: MessageTypeScanProgress,
		Data: map[string]interface{}{
			"scan_id":  scanID,
			"progress": progress,
			"status":   status,
			"message":  message,
		},
		Timestamp: time.Now(),
	})
}

// BroadcastScanStarted broadcasts scan start notifications
func (ws *WebSocketService) BroadcastScanStarted(scanID string, source string, totalFiles int) {
	ws.hub.Broadcast(WebSocketMessage{
		Type: MessageTypeScanStarted,
		Data: map[string]interface{}{
			"scan_id":     scanID,
			"source":      source,
			"total_files": totalFiles,
		},
		Timestamp: time.Now(),
	})
}

// BroadcastNewFinding broadcasts new finding notifications
func (ws *WebSocketService) BroadcastNewFinding(finding map[string]interface{}) {
	ws.hub.Broadcast(WebSocketMessage{
		Type:      MessageTypeNewFinding,
		Data:      finding,
		Timestamp: time.Now(),
	})
}

// BroadcastScanComplete broadcasts scan completion notifications
func (ws *WebSocketService) BroadcastScanComplete(scanID string, totalFindings int, duration time.Duration) {
	ws.hub.Broadcast(WebSocketMessage{
		Type: MessageTypeScanComplete,
		Data: map[string]interface{}{
			"scan_id":        scanID,
			"total_findings": totalFindings,
			"duration_ms":    duration.Milliseconds(),
		},
		Timestamp: time.Now(),
	})
}

// BroadcastSystemStatus broadcasts system status updates
func (ws *WebSocketService) BroadcastSystemStatus(status map[string]interface{}) {
	ws.hub.Broadcast(WebSocketMessage{
		Type:      MessageTypeSystemStatus,
		Data:      status,
		Timestamp: time.Now(),
	})
}

// GetHub returns the WebSocket hub for external access
func (ws *WebSocketService) GetHub() *Hub {
	return ws.hub
}
