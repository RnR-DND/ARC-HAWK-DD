package main

import (
	"log"
	"os"

	scannerapi "github.com/arc-platform/go-scanner/api"
	// Blank imports trigger each sub-package's init(), which self-registers connectors.
	_ "github.com/arc-platform/go-scanner/internal/connectors/cloud"
	_ "github.com/arc-platform/go-scanner/internal/connectors/databases"
	_ "github.com/arc-platform/go-scanner/internal/connectors/files"
	_ "github.com/arc-platform/go-scanner/internal/connectors/queues"
	_ "github.com/arc-platform/go-scanner/internal/connectors/saas"
	_ "github.com/arc-platform/go-scanner/internal/connectors/warehouses"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	_ = godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8001"
	}

	r := gin.Default()
	r.POST("/scan", scannerapi.ScanHandler)
	r.GET("/health", scannerapi.HealthHandler)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	log.Printf("Go scanner starting on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("scanner failed: %v", err)
	}
}
