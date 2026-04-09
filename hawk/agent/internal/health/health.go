package health

import (
	"net/http"
	"time"

	"github.com/arc/hawk/agent/internal/buffer"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Server serves the local health endpoint on :9090/health.
type Server struct {
	agentID   string
	queue     *buffer.LocalQueue
	syncLoop  *buffer.SyncLoop
	logger    *zap.Logger
	startedAt time.Time
}

// HealthResponse is returned by the /health endpoint.
type HealthResponse struct {
	AgentID            string  `json:"agent_id"`
	Uptime             string  `json:"uptime"`
	UptimeSeconds      float64 `json:"uptime_seconds"`
	LastScanAt         string  `json:"last_scan_at,omitempty"`
	QueueDepth         int     `json:"queue_depth"`
	ConnectivityStatus string  `json:"connectivity_status"`
	BufferSizeMB       float64 `json:"buffer_size_mb"`
	Status             string  `json:"status"`
}

// NewServer creates a new health server.
func NewServer(agentID string, queue *buffer.LocalQueue, syncLoop *buffer.SyncLoop, logger *zap.Logger) *Server {
	return &Server{
		agentID:   agentID,
		queue:     queue,
		syncLoop:  syncLoop,
		logger:    logger,
		startedAt: time.Now(),
	}
}

// Router returns the configured gin Engine for the health endpoint.
func (s *Server) Router() http.Handler {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/health", s.handleHealth)

	return r
}

// handleHealth responds with the agent health status.
func (s *Server) handleHealth(c *gin.Context) {
	uptime := time.Since(s.startedAt)

	queueDepth, err := s.queue.PendingCount()
	if err != nil {
		s.logger.Error("failed to get queue depth", zap.Error(err))
		queueDepth = -1
	}

	bufferSizeMB := s.queue.BufferSizeMB()
	connectivity := s.syncLoop.ConnectivityStatus()

	resp := HealthResponse{
		AgentID:            s.agentID,
		Uptime:             uptime.Round(time.Second).String(),
		UptimeSeconds:      uptime.Seconds(),
		QueueDepth:         queueDepth,
		ConnectivityStatus: connectivity,
		BufferSizeMB:       bufferSizeMB,
		Status:             "ok",
	}

	if s.queue.IsPaused() {
		resp.Status = "degraded_buffer_full"
	}

	c.JSON(http.StatusOK, resp)
}
