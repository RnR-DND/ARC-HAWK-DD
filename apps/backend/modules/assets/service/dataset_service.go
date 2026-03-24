package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/arc-platform/backend/modules/shared/domain/entity"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
)

type DatasetService struct {
	repo *persistence.PostgresRepository
}

func NewDatasetService(repo *persistence.PostgresRepository) *DatasetService {
	return &DatasetService{
		repo: repo,
	}
}

// DatasetEntry represents a single line in the Golden Dataset (JSONL)
type DatasetEntry struct {
	Text           string `json:"text"`
	Label          string `json:"label"`
	IsPIIConfirmed bool   `json:"is_pii_confirmed"`
	FeedbackType   string `json:"feedback_type"`
	Comment        string `json:"comment,omitempty"`
}

// GenerateGoldenDataset retrieves feedback and formats it as JSONL
func (s *DatasetService) GenerateGoldenDataset(ctx context.Context) ([]byte, error) {
	items, err := s.repo.GetFeedbackForDataset(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feedback: %w", err)
	}

	var sb strings.Builder
	encoder := json.NewEncoder(&sb)

	for _, item := range items {
		entry := DatasetEntry{
			Text:           item.Finding.SampleText,
			Label:          item.Finding.PatternName, // e.g. "Credit Card Number"
			IsPIIConfirmed: item.Feedback.FeedbackType == entity.FeedbackTypeConfirmed,
			FeedbackType:   item.Feedback.FeedbackType,
			Comment:        item.Feedback.Comments,
		}

		if err := encoder.Encode(entry); err != nil {
			return nil, fmt.Errorf("failed to encode entry: %w", err)
		}
	}

	return []byte(sb.String()), nil
}
