package repository

import (
	"github.com/google/uuid"
)

// FindingFilters defines filters for finding queries
type FindingFilters struct {
	ScanRunID    *uuid.UUID
	AssetID      *uuid.UUID
	Severity     string
	PatternName  string
	DataSource   string
	Search       string
	AssetName    string // filter by asset display name (ILIKE)
	ReviewStatus string // "Active" | "Suppressed" | "Remediated"
	SortBy       string // "created_at" | "severity" | "pattern_name" | "asset_name" | "confidence"
	SortOrder    string // "asc" | "desc"
}

// RelationshipFilters defines filters for relationship queries
type RelationshipFilters struct {
	RelationshipType string
	SourceAssetID    *uuid.UUID
	TargetAssetID    *uuid.UUID
}
