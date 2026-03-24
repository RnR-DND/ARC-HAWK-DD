package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/arc-platform/backend/modules/shared/domain/entity"
	"github.com/google/uuid"
)

// ============================================================================
// PatternRepository Implementation
// ============================================================================

func (r *PostgresRepository) CreatePattern(ctx context.Context, pattern *entity.Pattern) error {
	query := `
		INSERT INTO patterns (id, name, pattern_type, category, description, pattern_definition, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at`

	return r.db.QueryRowContext(ctx, query,
		pattern.ID, pattern.Name, pattern.PatternType, pattern.Category,
		pattern.Description, pattern.PatternDefinition, pattern.IsActive,
	).Scan(&pattern.CreatedAt, &pattern.UpdatedAt)
}

func (r *PostgresRepository) GetPatternByID(ctx context.Context, id uuid.UUID) (*entity.Pattern, error) {
	query := `
		SELECT id, name, pattern_type, category, description, pattern_definition, is_active, created_at, updated_at
		FROM patterns WHERE id = $1`

	pattern := &entity.Pattern{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&pattern.ID, &pattern.Name, &pattern.PatternType, &pattern.Category,
		&pattern.Description, &pattern.PatternDefinition, &pattern.IsActive,
		&pattern.CreatedAt, &pattern.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("pattern not found")
		}
		return nil, err
	}

	return pattern, nil
}

func (r *PostgresRepository) GetPatternByName(ctx context.Context, name string) (*entity.Pattern, error) {
	query := `
		SELECT id, name, pattern_type, category, description, pattern_definition, is_active, created_at, updated_at
		FROM patterns WHERE name = $1`

	pattern := &entity.Pattern{}
	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&pattern.ID, &pattern.Name, &pattern.PatternType, &pattern.Category,
		&pattern.Description, &pattern.PatternDefinition, &pattern.IsActive,
		&pattern.CreatedAt, &pattern.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return pattern, nil
}

func (r *PostgresRepository) ListPatterns(ctx context.Context) ([]*entity.Pattern, error) {
	query := `
		SELECT id, name, pattern_type, category, description, pattern_definition, is_active, created_at, updated_at
		FROM patterns 
		ORDER BY name`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patterns []*entity.Pattern
	for rows.Next() {
		pattern := &entity.Pattern{}
		err := rows.Scan(
			&pattern.ID, &pattern.Name, &pattern.PatternType, &pattern.Category,
			&pattern.Description, &pattern.PatternDefinition, &pattern.IsActive,
			&pattern.CreatedAt, &pattern.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		patterns = append(patterns, pattern)
	}

	return patterns, rows.Err()
}
