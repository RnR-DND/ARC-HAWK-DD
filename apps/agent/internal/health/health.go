package health

import (
	"context"
	"net/http"
	"time"

	"github.com/arc/hawk-agent/internal/buffer"
	"github.com/arc/hawk-agent/internal/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Server exposes the local health endpoint at :9090/health.
type Server struct {
	cfg       *config.Config
	queue     *buffer.LocalQueue
	syncLoop  *buffer.SyncLoop
	logger    *zap.Logger
	server    *http.Server
	startedAt time.Time
}

// HealthResponse is the JSON body returned by the health endpoint.
type HealthResponse struct {
	AgentID       string  `json:"agent_id"`
	Uptime        string  `json:"uptime"`
	UptimeSeconds float64 `json:"uptime_seconds"`
	LastScan      string  `json:"last_scan"`
	QueueDepth    int     `json:"queue_depth"`
	Connectivity  string  `json:"connectivity"`
	BufferSizeMB  float64 `json:"buffer_size_mb"`
	Status        string  `json:"status"`
}

// NewServer creates a health server.
func NewServer(cfg *config.Config, queue *buffer.LocalQueue, syncLoop *buffer.SyncLoop, logger *zap.Logger) *Server {
	return &Server{
		cfg:      cfg,
		queue:    queue,
		syncLoop: syncLoop,
		logger:   logger,
	}
}

// Start launches the health HTTP server on :9090 in a goroutine.
func (s *Server) Start() {
	s.startedAt = time.Now()

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/health", s.handleHealth)

	s.server = &http.Server{
		Addr:         ":9090",
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	go func() {
		s.logger.Info("health server starting", zap.String("addr", ":9090"))
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("health server error", zap.Error(err))
		}
	}()
}

// Stop gracefully shuts down the health server.
func (s *Server) Stop(ctx context.Context) {
	if s.server != nil {
		if err := s.server.Shutdown(ctx); err != nil {
			s.logger.Error("health server shutdown error", zap.Error(err))
		}
	}
}

func (s *Server) handleHealth(c *gin.Context) {
	uptime := time.Since(s.startedAt)

	queueDepth, err := s.queue.QueueDepth()
	if err != nil {
		s.logger.Error("queue depth query failed", zap.Error(err))
	}

	bufferSize, err := s.queue.BufferSizeMB()
	if err != nil {
		s.logger.Error("buffer size query failed", zap.Error(err))
	}

	connectivity := "offline"
	if s.syncLoop.IsOnline() {
		connectivity = "online"
	}

	lastScan := "never"
	lastOnline := s.syncLoop.LastOnlineAt()
	if !lastOnline.IsZero() {
		lastScan = lastOnline.Format(time.RFC3339)
	}

	resp := HealthResponse{
		AgentID:       s.cfg.AgentID,
		Uptime:        uptime.Round(time.Second).String(),
		UptimeSeconds: uptime.Seconds(),
		LastScan:      lastScan,
		QueueDepth:    queueDepth,
		Connectivity:  connectivity,
		BufferSizeMB:  bufferSize,
		Status:        "running",
	}

	c.JSON(http.StatusOK, resp)
}
