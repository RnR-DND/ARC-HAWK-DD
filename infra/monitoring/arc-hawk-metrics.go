package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {
	// Get target URL from environment or use default
	targetURL := os.Getenv("TARGET_URL")
	if targetURL == "" {
		targetURL = "http://localhost:8080/metrics" // ARC-Hawk backend metrics
	}

	log.Printf("Starting ARC-Hawk metrics generation for: %s", targetURL)

	// Run forever with metrics collection
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var (
		totalScans    = 0.0
		totalFindings = 0.0
		totalAssets   = 0.0
		scanDuration  = 0.0
		requestCount  = 0.0
		errorCount    = 0.0
	)

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Make HTTP request
		resp, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
		if err != nil {
			errorCount++
			log.Printf("Error creating request: %v", err)
			continue
		}

		client := &http.Client{Timeout: 10 * time.Second}
		response, err := client.Do(resp)
		if err != nil {
			errorCount++
			log.Printf("Error making request: %v", err)
			continue
		}
		defer response.Body.Close()

		requestCount++

		if response.StatusCode == 200 {
			// Read response body
			body, err := ioutil.ReadAll(response.Body)
			if err == nil {
				// Extract numbers from JSON (simplified extraction)
				responseStr := string(body)

				// Simple extraction - look for number patterns
				totalScans += extractNumber(responseStr, "total_scans", 1)
				totalFindings += extractNumber(responseStr, "total_findings", 0)
				totalAssets += extractNumber(responseStr, "total_assets", 0)
				scanDuration += extractNumber(responseStr, "avg_scan_duration", 0)
			}
		} else {
			errorCount++
			log.Printf("HTTP Error: %d", response.StatusCode)
		}

		// Generate time series data points for better Grafana visualization
		currentTime := time.Now().UnixMilli()

		// Print metrics in Prometheus format
		fmt.Printf("# HELP arc_hawk_total_scans Total number of scans performed\n")
		fmt.Printf("# TYPE arc_hawk_total_scans gauge\n")
		fmt.Printf("arc_hawk_total_scans %f %d\n", totalScans, currentTime)

		fmt.Printf("# HELP arc_hawk_total_findings Total number of PII findings discovered\n")
		fmt.Printf("# TYPE arc_hawk_total_findings gauge\n")
		fmt.Printf("arc_hawk_total_findings %f %d\n", totalFindings, currentTime)

		fmt.Printf("# HELP arc_hawk_total_assets Total number of assets scanned\n")
		fmt.Printf("# TYPE arc_hawk_total_assets gauge\n")
		fmt.Printf("arc_hawk_total_assets %f %d\n", totalAssets, currentTime)

		fmt.Printf("# HELP arc_hawk_scan_duration Average scan duration in seconds\n")
		fmt.Printf("# TYPE arc_hawk_scan_duration gauge\n")
		fmt.Printf("arc_hawk_scan_duration %f %d\n", scanDuration, currentTime)

		// Combined health score
		fmt.Printf("# HELP arc_hawk_health_score Overall system health score (0-100)\n")
		fmt.Printf("# TYPE arc_hawk_health_score gauge\n")
		fmt.Printf("arc_hawk_health_score %f %d\n", calculateHealthScore(totalScans, totalFindings, errorCount), currentTime)

		// Log to console for debugging
		log.Printf("Requests: %.0f, Errors: %.0f, TotalScans: %.0f, TotalFindings: %.0f, TotalAssets: %.0f, Duration: %.0f",
			requestCount, errorCount, totalScans, totalFindings, totalAssets, scanDuration)
	}
}

func extractNumber(text, key string, defaultValue float64) float64 {
	// Simple number extraction from JSON response
	// This is a simplified version - in production, you'd parse JSON properly
	start := findSubstring(text, `"total_pii":`)
	end := findSubstring(text, ",")
	if start != -1 && end != -1 {
		numStr := text[start+len(`"total_pii":`) : end]
		if num, err := strconv.ParseFloat(numStr, 64); err == nil {
			return num
		}
	}
	return defaultValue
}

func findSubstring(text, substr string) int {
	for i := 0; i <= len(text)-len(substr); i++ {
		if text[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func calculateHealthScore(totalScans, totalFindings, errorCount float64) float64 {
	// Simple health score calculation
	// Penalize errors heavily
	score := 100.0
	score -= errorCount * 10.0 // Each error costs 10 points

	// Bonus for activity
	if totalScans > 0 {
		score += 5.0
	}
	if totalFindings > 0 {
		score += 5.0
	}

	// Ensure score stays in bounds
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}
