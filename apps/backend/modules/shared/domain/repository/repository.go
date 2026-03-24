package repository

import (
	"github.com/google/uuid"
)

// FindingFilters defines filters for finding queries
type FindingFilters struct {
	ScanRunID   *uuid.UUID
	AssetID     *uuid.UUID
	Severity    string
	PatternName string
	DataSource  string
}

// RelationshipFilters defines filters for relationship queries
type RelationshipFilters struct {
	RelationshipType string
	SourceAssetID    *uuid.UUID
	TargetAssetID    *uuid.UUID
}
