package entity

import (
	"time"

	"github.com/google/uuid"
)

// FeedbackType constants
const (
	FeedbackTypeFalsePositive = "FALSE_POSITIVE"
	FeedbackTypeFalseNegative = "FALSE_NEGATIVE"
	FeedbackTypeConfirmed     = "CONFIRMED"
)

// FindingFeedback represents user feedback on a finding
type FindingFeedback struct {
	ID                     uuid.UUID `json:"id"`
	FindingID              uuid.UUID `json:"finding_id"`
	UserID                 string    `json:"user_id"`
	FeedbackType           string    `json:"feedback_type"`
	OriginalClassification string    `json:"original_classification"`
	ProposedClassification string    `json:"proposed_classification,omitempty"`
	Comments               string    `json:"comments,omitempty"`
	CreatedAt              time.Time `json:"created_at"`
	Processed              bool      `json:"processed"`
}

// FeedbackWithFinding represents feedback joined with finding details for dataset generation
type FeedbackWithFinding struct {
	Feedback FindingFeedback
	Finding  Finding
}
