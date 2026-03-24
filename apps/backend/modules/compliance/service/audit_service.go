package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// AuditLogEntry represents an audit log entry
type AuditLogEntry struct {
	ID           string                 `json:"id"`
	UserID       string                 `json:"user_id"`
	Action       string                 `json:"action"`
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id"`
	IPAddress    string                 `json:"ip_address"`
	Result       string                 `json:"result"` // SUCCESS, FAILED
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	EventTime    time.Time              `json:"event_time"`
}

// AuditFilters represents filters for querying audit logs
type AuditFilters struct {
	UserID       string
	Action       string
	ResourceType string
	ResourceID   string
	StartTime    *time.Time
	EndTime      *time.Time
	Limit        int
	Offset       int
}

// UserActivity represents user activity summary
type UserActivity struct {
	UserID      string    `json:"user_id"`
	ActionCount int       `json:"action_count"`
	LastAction  time.Time `json:"last_action"`
	Actions     []string  `json:"actions"`
}

// AuditService handles audit logging operations
type AuditService struct {
	db *sql.DB
}

// NewAuditService creates a new audit service
func NewAuditService(db *sql.DB) *AuditService {
	return &AuditService{db: db}
}

// RecordAuditLog records an audit log entry
func (s *AuditService) RecordAuditLog(ctx context.Context, entry AuditLogEntry) error {
	query := `
		INSERT INTO audit_logs (
			user_id, action, resource_type, resource_id,
			ip_address, result, metadata, event_time
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := s.db.ExecContext(
		ctx, query,
		entry.UserID, entry.Action, entry.ResourceType, entry.ResourceID,
		entry.IPAddress, entry.Result, entry.Metadata, entry.EventTime,
	)

	if err != nil {
		return fmt.Errorf("failed to record audit log: %w", err)
	}

	return nil
}

// ListAuditLogs lists audit logs with optional filters
func (s *AuditService) ListAuditLogs(ctx context.Context, filters AuditFilters) ([]AuditLogEntry, error) {
	query := `
		SELECT 
			id, user_id, action, resource_type, resource_id,
			ip_address, result, metadata, event_time
		FROM audit_logs
		WHERE 1=1
	`

	args := []interface{}{}
	argCount := 1

	if filters.UserID != "" {
		query += fmt.Sprintf(" AND user_id = $%d", argCount)
		args = append(args, filters.UserID)
		argCount++
	}

	if filters.Action != "" {
		query += fmt.Sprintf(" AND action = $%d", argCount)
		args = append(args, filters.Action)
		argCount++
	}

	if filters.ResourceType != "" {
		query += fmt.Sprintf(" AND resource_type = $%d", argCount)
		args = append(args, filters.ResourceType)
		argCount++
	}

	if filters.ResourceID != "" {
		query += fmt.Sprintf(" AND resource_id = $%d", argCount)
		args = append(args, filters.ResourceID)
		argCount++
	}

	if filters.StartTime != nil {
		query += fmt.Sprintf(" AND event_time >= $%d", argCount)
		args = append(args, *filters.StartTime)
		argCount++
	}

	if filters.EndTime != nil {
		query += fmt.Sprintf(" AND event_time <= $%d", argCount)
		args = append(args, *filters.EndTime)
		argCount++
	}

	query += " ORDER BY event_time DESC"

	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filters.Limit)
		argCount++
	}

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, filters.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list audit logs: %w", err)
	}
	defer rows.Close()

	var logs []AuditLogEntry
	for rows.Next() {
		var log AuditLogEntry
		var metadata []byte

		err := rows.Scan(
			&log.ID, &log.UserID, &log.Action, &log.ResourceType, &log.ResourceID,
			&log.IPAddress, &log.Result, &metadata, &log.EventTime,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}

		logs = append(logs, log)
	}

	return logs, nil
}

// GetUserActivity gets activity summary for a user
func (s *AuditService) GetUserActivity(ctx context.Context, userID string, limit int) (*UserActivity, error) {
	query := `
		SELECT 
			user_id,
			COUNT(*) as action_count,
			MAX(event_time) as last_action,
			ARRAY_AGG(DISTINCT action) as actions
		FROM audit_logs
		WHERE user_id = $1
		GROUP BY user_id
	`

	var activity UserActivity
	var actions []string

	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&activity.UserID,
		&activity.ActionCount,
		&activity.LastAction,
		&actions,
	)

	if err == sql.ErrNoRows {
		return &UserActivity{
			UserID:      userID,
			ActionCount: 0,
			Actions:     []string{},
		}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get user activity: %w", err)
	}

	activity.Actions = actions
	return &activity, nil
}

// GetResourceHistory gets audit history for a specific resource
func (s *AuditService) GetResourceHistory(ctx context.Context, resourceType, resourceID string) ([]AuditLogEntry, error) {
	query := `
		SELECT 
			id, user_id, action, resource_type, resource_id,
			ip_address, result, metadata, event_time
		FROM audit_logs
		WHERE resource_type = $1 AND resource_id = $2
		ORDER BY event_time DESC
	`

	rows, err := s.db.QueryContext(ctx, query, resourceType, resourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource history: %w", err)
	}
	defer rows.Close()

	var logs []AuditLogEntry
	for rows.Next() {
		var log AuditLogEntry
		var metadata []byte

		err := rows.Scan(
			&log.ID, &log.UserID, &log.Action, &log.ResourceType, &log.ResourceID,
			&log.IPAddress, &log.Result, &metadata, &log.EventTime,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}

		logs = append(logs, log)
	}

	return logs, nil
}

// GetRecentActivity gets recent activity across all users
func (s *AuditService) GetRecentActivity(ctx context.Context, limit int) ([]AuditLogEntry, error) {
	query := `
		SELECT 
			id, user_id, action, resource_type, resource_id,
			ip_address, result, metadata, event_time
		FROM audit_logs
		ORDER BY event_time DESC
		LIMIT $1
	`

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent activity: %w", err)
	}
	defer rows.Close()

	var logs []AuditLogEntry
	for rows.Next() {
		var log AuditLogEntry
		var metadata []byte

		err := rows.Scan(
			&log.ID, &log.UserID, &log.Action, &log.ResourceType, &log.ResourceID,
			&log.IPAddress, &log.Result, &metadata, &log.EventTime,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}

		logs = append(logs, log)
	}

	return logs, nil
}

// GetActiveUsers gets count of active users in a time period
func (s *AuditService) GetActiveUsers(ctx context.Context, since time.Time) (int, error) {
	query := `
		SELECT COUNT(DISTINCT user_id)
		FROM audit_logs
		WHERE event_time >= $1
	`

	var count int
	err := s.db.QueryRowContext(ctx, query, since).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get active users: %w", err)
	}

	return count, nil
}
