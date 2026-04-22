package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Neo4jSyncQueue struct {
	ID          uuid.UUID       `db:"id" json:"id"`
	Operation   string          `db:"operation" json:"operation"`
	Payload     json.RawMessage `db:"payload" json:"payload"`
	Status      string          `db:"status" json:"status"`
	Attempts    int             `db:"attempts" json:"attempts"`
	LastError   *string         `db:"last_error" json:"last_error,omitempty"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
	ProcessedAt *time.Time      `db:"processed_at" json:"processed_at,omitempty"`
}
