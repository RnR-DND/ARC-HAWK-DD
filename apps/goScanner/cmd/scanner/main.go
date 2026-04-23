package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	scannerapi "github.com/arc-platform/go-scanner/api"
	// Blank imports trigger each sub-package's init(), which self-registers connectors.
	_ "github.com/arc-platform/go-scanner/internal/connectors/cloud"
	_ "github.com/arc-platform/go-scanner/internal/connectors/databases"
	_ "github.com/arc-platform/go-scanner/internal/connectors/files"
	_ "github.com/arc-platform/go-scanner/internal/connectors/queues"
	_ "github.com/arc-platform/go-scanner/internal/connectors/saas"
	_ "github.com/arc-platform/go-scanner/internal/connectors/warehouses"
	"github.com/arc-platform/go-scanner/internal/telemetry"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func main() {
	_ = godotenv.Load()

	shutdownTracer, err := telemetry.InitTracer(context.Background(), "arc-hawk-scanner")
	if err != nil {
		log.Printf("WARN: OTel tracer init failed: %v (tracing disabled)", err)
		shutdownTracer = func(context.Context) error { return nil }
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8001"
	}

	r := gin.Default()
	r.Use(otelgin.Middleware("arc-hawk-scanner"))

	// /health and /metrics remain public so Docker and Prometheus can probe.
	r.GET("/health", scannerapi.HealthHandler)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// /scan is privileged — require the shared service token. Backend must
	// send X-Scanner-Token. See apps/goScanner/api/auth_middleware.go.
	authed := r.Group("/", scannerapi.ServiceTokenAuth())
	authed.POST("/scan", scannerapi.ScanHandler)

	if os.Getenv("SCANNER_AUTH_REQUIRED") == "false" {
		log.Printf("WARN: scanner auth disabled (SCANNER_AUTH_REQUIRED=false) — dev mode only")
	} else if os.Getenv("SCANNER_SERVICE_TOKEN") == "" {
		log.Printf("WARN: SCANNER_SERVICE_TOKEN is empty; /scan will reject all requests")
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Graceful shutdown on SIGINT / SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Go scanner starting on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("scanner failed: %v", err)
		}
	}()

	<-quit
	log.Println("Scanner shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Scanner forced shutdown: %v", err)
	}

	if err := shutdownTracer(ctx); err != nil {
		log.Printf("Tracer shutdown error: %v", err)
	}

	log.Println("Scanner stopped.")
}
