package main

import (
	"log"
	"net/http"
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

	serviceToken := os.Getenv("SCANNER_SERVICE_TOKEN")
	if serviceToken != "" {
		r.Use(func(c *gin.Context) {
			if c.GetHeader("X-Service-Token") != serviceToken {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
				return
			}
			c.Next()
		})
	}

	r.POST("/scan", scannerapi.ScanHandler)
	r.GET("/health", scannerapi.HealthHandler)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	log.Printf("Go scanner starting on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("scanner failed: %v", err)
	}
}
