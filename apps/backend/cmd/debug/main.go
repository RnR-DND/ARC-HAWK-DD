package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Findings missing required fields
	fmt.Println("=== FINDINGS MISSING REQUIRED FIELDS ===")
	rows, err := db.QueryContext(ctx, `
		SELECT id, scan_run_id, asset_id, pattern_name, severity
		FROM findings f
		WHERE f.scan_run_id IS NULL
		   OR f.asset_id IS NULL
		   OR f.pattern_name IS NULL
		   OR f.pattern_name = ''
		   OR f.severity IS NULL
		   OR f.severity = ''
	`)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, scanID, assetID, pattern, severity interface{}
		rows.Scan(&id, &scanID, &assetID, &pattern, &severity)
		fmt.Printf("ID: %v, ScanID: %v, AssetID: %v, Pattern: %v, Severity: %v\n", id, scanID, assetID, pattern, severity)
	}

	// Findings with invalid PII types (first 10)
	fmt.Println("\n=== FINDINGS WITH INVALID PII TYPES (first 10) ===")
	rows2, err := db.QueryContext(ctx, `
		SELECT f.id, f.pattern_name
		FROM findings f
		LEFT JOIN patterns p ON f.pattern_name = p.name
		WHERE p.name IS NULL
		LIMIT 10
	`)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	defer rows2.Close()

	for rows2.Next() {
		var id, pattern interface{}
		rows2.Scan(&id, &pattern)
		fmt.Printf("ID: %v, Pattern: %v\n", id, pattern)
	}

	// Findings without location (first 10)
	fmt.Println("\n=== FINDINGS WITHOUT LOCATION DATA (first 10) ===")
	rows3, err := db.QueryContext(ctx, `
		SELECT f.id, a.id, a.path
		FROM findings f
		JOIN assets a ON f.asset_id = a.id
		WHERE a.path IS NULL OR a.path = ''
		LIMIT 10
	`)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	defer rows3.Close()

	for rows3.Next() {
		var fid, aid, path interface{}
		rows3.Scan(&fid, &aid, &path)
		fmt.Printf("FindingID: %v, AssetID: %v, Path: %v\n", fid, aid, path)
	}
}
