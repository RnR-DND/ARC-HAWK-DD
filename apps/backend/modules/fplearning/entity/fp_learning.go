package entity

import (
	"time"

	"github.com/google/uuid"
)

type FPLearningType string

const (
	FPLearningTypeFalsePositive FPLearningType = "false_positive"
	FPLearningTypeTruePositive  FPLearningType = "true_positive"
	FPLearningTypeConfirmed     FPLearningType = "confirmed"
)

type FPLearning struct {
	ID              uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	TenantID        uuid.UUID      `json:"tenant_id" gorm:"type:uuid;index"`
	UserID          uuid.UUID      `json:"user_id" gorm:"type:uuid;index"`
	AssetID         uuid.UUID      `json:"asset_id" gorm:"type:uuid;index"`
	PatternName     string         `json:"pattern_name" gorm:"size:100;index"`
	PIIType         string         `json:"pii_type" gorm:"size:50;index"`
	FieldName       string         `json:"field_name" gorm:"size:255"`
	FieldPath       string         `json:"field_path" gorm:"size:500"`
	MatchedValue    string         `json:"matched_value" gorm:"size:500"`
	LearningType    FPLearningType `json:"learning_type" gorm:"size:50"`
	Version         int            `json:"version" gorm:"default:1"`
	PreviousValue   string         `json:"previous_value" gorm:"size:500"`
	Justification   string         `json:"justification" gorm:"type:text"`
	SourceFindingID *uuid.UUID     `json:"source_finding_id,omitempty" gorm:"type:uuid"`
	ScanRunID       *uuid.UUID     `json:"scan_run_id,omitempty" gorm:"type:uuid;index"`
	ExpiresAt       *time.Time     `json:"expires_at,omitempty" gorm:"index"`
	IsActive        bool           `json:"is_active" gorm:"default:true;index"`
	CreatedAt       time.Time      `json:"created_at" gorm:"index"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type FPLearningFilter struct {
	TenantID      uuid.UUID
	AssetID       *uuid.UUID
	PatternName   string
	PIIType       string
	FieldPath     string
	LearningType  *FPLearningType
	IsActive      *bool
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
}

type FPLearningStats struct {
	TotalPatterns    int            `json:"total_patterns"`
	FalsePositives   int            `json:"false_positives"`
	Confirmed        int            `json:"confirmed"`
	ByPIIType        map[string]int `json:"by_pii_type"`
	ByAsset          map[string]int `json:"by_asset"`
	LatestLearningAt *time.Time     `json:"latest_learning_at"`
}

type FPMatchRequest struct {
	AssetID       uuid.UUID `json:"asset_id" binding:"required"`
	PatternName   string    `json:"pattern_name" binding:"required"`
	PIIType       string    `json:"pii_type" binding:"required"`
	FieldName     string    `json:"field_name"`
	FieldPath     string    `json:"field_path"`
	MatchedValue  string    `json:"matched_value" binding:"required"`
	Justification string    `json:"justification"`
}

type FPMatchResponse struct {
	IsFalsePositive bool   `json:"is_false_positive"`
	LearningID      string `json:"learning_id,omitempty"`
	Confidence      int    `json:"confidence"`
}
