package persistence

import (
	"context"
	"database/sql"
	"fmt"

	authentity "github.com/arc-platform/backend/modules/auth/entity"
	fplearningentity "github.com/arc-platform/backend/modules/fplearning/entity"
	"github.com/arc-platform/backend/modules/shared/domain/entity"
	"github.com/google/uuid"
)

// PostgresRepository implements all repository interfaces
type PostgresRepository struct {
	db *sql.DB
}

// PostgresTransaction wraps sql.Tx and provides repository methods
type PostgresTransaction struct {
	tx *sql.Tx
	db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// BeginTx starts a new database transaction
func (r *PostgresRepository) BeginTx(ctx context.Context) (*PostgresTransaction, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return &PostgresTransaction{
		tx: tx,
		db: r.db,
	}, nil
}

// Commit commits the transaction
func (t *PostgresTransaction) Commit() error {
	return t.tx.Commit()
}

// Rollback rolls back the transaction
func (t *PostgresTransaction) Rollback() error {
	return t.tx.Rollback()
}

// GetDB returns the underlying database connection (for read-only operations outside transaction)
func (r *PostgresRepository) GetDB() *sql.DB {
	return r.db
}

// MigrateSchema updates the database schema with new columns
func (r *PostgresRepository) MigrateSchema(ctx context.Context) error {
	queries := []string{
		"ALTER TABLE assets ADD COLUMN IF NOT EXISTS environment TEXT DEFAULT ''",
		"ALTER TABLE assets ADD COLUMN IF NOT EXISTS owner TEXT DEFAULT ''",
		"ALTER TABLE assets ADD COLUMN IF NOT EXISTS source_system TEXT DEFAULT ''",
	}
	for _, q := range queries {
		if _, err := r.db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("migration failed: %s: %w", q, err)
		}
	}
	return nil
}

// ===== Connection Repository Methods =====

// CreateConnection stores a new connection with encrypted config
func (r *PostgresRepository) CreateConnection(ctx context.Context, conn *entity.Connection) error {
	query := `
		INSERT INTO connections (id, source_type, profile_name, config_encrypted, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at
	`
	return r.db.QueryRowContext(ctx, query,
		conn.ID, conn.SourceType, conn.ProfileName, conn.ConfigEncrypted, conn.CreatedBy,
	).Scan(&conn.CreatedAt, &conn.UpdatedAt)
}

// GetConnection retrieves a connection by ID
func (r *PostgresRepository) GetConnection(ctx context.Context, id uuid.UUID) (*entity.Connection, error) {
	query := `
		SELECT id, source_type, profile_name, config_encrypted, validation_status,
		       last_validated_at, validation_error, created_by, created_at, updated_at
		FROM connections WHERE id = $1
	`
	conn := &entity.Connection{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&conn.ID, &conn.SourceType, &conn.ProfileName, &conn.ConfigEncrypted,
		&conn.ValidationStatus, &conn.LastValidatedAt, &conn.ValidationError,
		&conn.CreatedBy, &conn.CreatedAt, &conn.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// GetConnectionByProfile retrieves a connection by source type and profile name
func (r *PostgresRepository) GetConnectionByProfile(ctx context.Context, sourceType, profileName string) (*entity.Connection, error) {
	query := `
		SELECT id, source_type, profile_name, config_encrypted, validation_status,
		       last_validated_at, validation_error, created_by, created_at, updated_at
		FROM connections WHERE source_type = $1 AND profile_name = $2
	`
	conn := &entity.Connection{}
	err := r.db.QueryRowContext(ctx, query, sourceType, profileName).Scan(
		&conn.ID, &conn.SourceType, &conn.ProfileName, &conn.ConfigEncrypted,
		&conn.ValidationStatus, &conn.LastValidatedAt, &conn.ValidationError,
		&conn.CreatedBy, &conn.CreatedAt, &conn.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// ListConnections retrieves all connections (without decrypted config)
func (r *PostgresRepository) ListConnections(ctx context.Context) ([]*entity.Connection, error) {
	query := `
		SELECT id, source_type, profile_name, validation_status,
		       last_validated_at, created_by, created_at, updated_at
		FROM connections ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var connections []*entity.Connection
	for rows.Next() {
		conn := &entity.Connection{}
		err := rows.Scan(&conn.ID, &conn.SourceType, &conn.ProfileName,
			&conn.ValidationStatus, &conn.LastValidatedAt, &conn.CreatedBy,
			&conn.CreatedAt, &conn.UpdatedAt)
		if err != nil {
			return nil, err
		}
		connections = append(connections, conn)
	}
	return connections, rows.Err()
}

// UpdateConnectionValidation updates the validation status of a connection
func (r *PostgresRepository) UpdateConnectionValidation(ctx context.Context, id uuid.UUID, status string, validationError *string) error {
	query := `
		UPDATE connections 
		SET validation_status = $1, 
		    last_validated_at = NOW(),
		    validation_error = $2
		WHERE id = $3
	`
	_, err := r.db.ExecContext(ctx, query, status, validationError, id)
	return err
}

// DeleteConnection deletes a connection by ID
func (r *PostgresRepository) DeleteConnection(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM connections WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// ===== User Repository Methods =====

// CreateUser creates a new user
func (r *PostgresRepository) CreateUser(ctx context.Context, user *authentity.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, first_name, last_name, role, tenant_id, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING created_at, updated_at
	`
	return r.db.QueryRowContext(ctx, query,
		user.ID, user.Email, user.PasswordHash, user.FirstName, user.LastName,
		user.Role, user.TenantID, user.IsActive, user.CreatedAt, user.UpdatedAt,
	).Scan(&user.CreatedAt, &user.UpdatedAt)
}

// GetUserByID retrieves a user by ID
func (r *PostgresRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*authentity.User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, role, tenant_id, is_active, last_login_at, created_at, updated_at
		FROM users WHERE id = $1
	`
	user := &authentity.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName,
		&user.Role, &user.TenantID, &user.IsActive, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetUserByEmail retrieves a user by email
func (r *PostgresRepository) GetUserByEmail(ctx context.Context, email string) (*authentity.User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, role, tenant_id, is_active, last_login_at, created_at, updated_at
		FROM users WHERE email = $1
	`
	user := &authentity.User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName,
		&user.Role, &user.TenantID, &user.IsActive, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetUsersByTenant retrieves all users for a tenant
func (r *PostgresRepository) GetUsersByTenant(ctx context.Context, tenantID uuid.UUID) ([]*authentity.User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, role, tenant_id, is_active, last_login_at, created_at, updated_at
		FROM users WHERE tenant_id = $1 ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*authentity.User
	for rows.Next() {
		user := &authentity.User{}
		err := rows.Scan(
			&user.ID, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName,
			&user.Role, &user.TenantID, &user.IsActive, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

// UpdateUser updates a user
func (r *PostgresRepository) UpdateUser(ctx context.Context, user *authentity.User) error {
	query := `
		UPDATE users SET email = $1, first_name = $2, last_name = $3, role = $4,
		is_active = $5, last_login_at = $6, updated_at = NOW()
		WHERE id = $7
	`
	_, err := r.db.ExecContext(ctx, query,
		user.Email, user.FirstName, user.LastName, user.Role,
		user.IsActive, user.LastLoginAt, user.ID,
	)
	return err
}

// ===== Tenant Repository Methods =====

// CreateTenant creates a new tenant
func (r *PostgresRepository) CreateTenant(ctx context.Context, tenant *authentity.Tenant) error {
	query := `
		INSERT INTO tenants (id, name, slug, description, is_active, settings, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at
	`
	return r.db.QueryRowContext(ctx, query,
		tenant.ID, tenant.Name, tenant.Slug, tenant.Description, tenant.IsActive, tenant.Settings,
		tenant.CreatedAt, tenant.UpdatedAt,
	).Scan(&tenant.CreatedAt, &tenant.UpdatedAt)
}

// UpdateTenant updates a tenant
func (r *PostgresRepository) UpdateTenant(ctx context.Context, tenant *authentity.Tenant) error {
	query := `
		UPDATE tenants SET name = $1, slug = $2, description = $3, is_active = $4,
		settings = $5, updated_at = NOW()
		WHERE id = $6
	`
	_, err := r.db.ExecContext(ctx, query,
		tenant.Name, tenant.Slug, tenant.Description, tenant.IsActive, tenant.Settings, tenant.ID,
	)
	return err
}

// GetTenantByID retrieves a tenant by ID
func (r *PostgresRepository) GetTenantByID(ctx context.Context, id uuid.UUID) (*authentity.Tenant, error) {
	query := `
		SELECT id, name, slug, description, is_active, settings, created_at, updated_at
		FROM tenants WHERE id = $1
	`
	tenant := &authentity.Tenant{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&tenant.ID, &tenant.Name, &tenant.Slug, &tenant.Description,
		&tenant.IsActive, &tenant.Settings, &tenant.CreatedAt, &tenant.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return tenant, nil
}

// GetTenantBySlug retrieves a tenant by slug
func (r *PostgresRepository) GetTenantBySlug(ctx context.Context, slug string) (*authentity.Tenant, error) {
	query := `
		SELECT id, name, slug, description, is_active, settings, created_at, updated_at
		FROM tenants WHERE slug = $1
	`
	tenant := &authentity.Tenant{}
	err := r.db.QueryRowContext(ctx, query, slug).Scan(
		&tenant.ID, &tenant.Name, &tenant.Slug, &tenant.Description,
		&tenant.IsActive, &tenant.Settings, &tenant.CreatedAt, &tenant.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return tenant, nil
}

// ===== Audit Log Repository Methods =====

// CreateAuditLog creates an audit log entry
func (r *PostgresRepository) CreateAuditLog(ctx context.Context, log *authentity.AuditLog) error {
	query := `
		INSERT INTO audit_logs (id, tenant_id, user_id, action, resource_type, resource_id, ip_address, user_agent, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING created_at
	`
	return r.db.QueryRowContext(ctx, query,
		log.ID, log.TenantID, log.UserID, log.Action, log.ResourceType,
		log.ResourceID, log.IPAddress, log.UserAgent, log.Metadata, log.CreatedAt,
	).Scan(&log.CreatedAt)
}

// GetAuditLogsByUser retrieves audit logs for a user
func (r *PostgresRepository) GetAuditLogsByUser(ctx context.Context, userID uuid.UUID, limit int) ([]*authentity.AuditLog, error) {
	query := `
		SELECT id, tenant_id, user_id, action, resource_type, resource_id, ip_address, user_agent, metadata, created_at
		FROM audit_logs WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2
	`
	rows, err := r.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*authentity.AuditLog
	for rows.Next() {
		log := &authentity.AuditLog{}
		err := rows.Scan(
			&log.ID, &log.TenantID, &log.UserID, &log.Action, &log.ResourceType,
			&log.ResourceID, &log.IPAddress, &log.UserAgent, &log.Metadata, &log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

// GetAuditLogsByResource retrieves audit logs for a resource
func (r *PostgresRepository) GetAuditLogsByResource(ctx context.Context, resourceType, resourceID string, limit int) ([]*authentity.AuditLog, error) {
	query := `
		SELECT id, tenant_id, user_id, action, resource_type, resource_id, ip_address, user_agent, metadata, created_at
		FROM audit_logs WHERE resource_type = $1 AND resource_id = $2 ORDER BY created_at DESC LIMIT $3
	`
	rows, err := r.db.QueryContext(ctx, query, resourceType, resourceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*authentity.AuditLog
	for rows.Next() {
		log := &authentity.AuditLog{}
		err := rows.Scan(
			&log.ID, &log.TenantID, &log.UserID, &log.Action, &log.ResourceType,
			&log.ResourceID, &log.IPAddress, &log.UserAgent, &log.Metadata, &log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

// ===== FP Learning Repository Methods =====

// CreateFPLearning creates a new FP learning record
func (r *PostgresRepository) CreateFPLearning(ctx context.Context, fp *fplearningentity.FPLearning) error {
	query := `
		INSERT INTO fp_learning (id, tenant_id, user_id, asset_id, pattern_name, pii_type,
			field_name, field_path, matched_value, learning_type, version, justification,
			source_finding_id, scan_run_id, expires_at, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`
	_, err := r.db.ExecContext(ctx, query,
		fp.ID, fp.TenantID, fp.UserID, fp.AssetID, fp.PatternName, fp.PIIType,
		fp.FieldName, fp.FieldPath, fp.MatchedValue, fp.LearningType, fp.Version, fp.Justification,
		fp.SourceFindingID, fp.ScanRunID, fp.ExpiresAt, fp.IsActive, fp.CreatedAt, fp.UpdatedAt,
	)
	return err
}

// GetFPLearningByID retrieves an FP learning by ID
func (r *PostgresRepository) GetFPLearningByID(ctx context.Context, id uuid.UUID) (*fplearningentity.FPLearning, error) {
	query := `
		SELECT id, tenant_id, user_id, asset_id, pattern_name, pii_type, field_name, field_path,
			matched_value, learning_type, version, previous_value, justification, source_finding_id,
			scan_run_id, expires_at, is_active, created_at, updated_at
		FROM fp_learning WHERE id = $1
	`
	fp := &fplearningentity.FPLearning{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&fp.ID, &fp.TenantID, &fp.UserID, &fp.AssetID, &fp.PatternName, &fp.PIIType,
		&fp.FieldName, &fp.FieldPath, &fp.MatchedValue, &fp.LearningType, &fp.Version,
		&fp.PreviousValue, &fp.Justification, &fp.SourceFindingID, &fp.ScanRunID,
		&fp.ExpiresAt, &fp.IsActive, &fp.CreatedAt, &fp.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return fp, nil
}

// GetFPLearningByFilter retrieves FP learning by filter
func (r *PostgresRepository) GetFPLearningByFilter(ctx context.Context, filter fplearningentity.FPLearningFilter) (*fplearningentity.FPLearning, error) {
	query := `
		SELECT id, tenant_id, user_id, asset_id, pattern_name, pii_type, field_name, field_path,
			matched_value, learning_type, version, previous_value, justification, source_finding_id,
			scan_run_id, expires_at, is_active, created_at, updated_at
		FROM fp_learning WHERE tenant_id = $1 AND is_active = true
	`
	args := []interface{}{filter.TenantID}
	argIndex := 2

	if filter.AssetID != nil {
		query += fmt.Sprintf(" AND asset_id = $%d", argIndex)
		args = append(args, *filter.AssetID)
		argIndex++
	}
	if filter.PatternName != "" {
		query += fmt.Sprintf(" AND pattern_name = $%d", argIndex)
		args = append(args, filter.PatternName)
		argIndex++
	}
	if filter.PIIType != "" {
		query += fmt.Sprintf(" AND pii_type = $%d", argIndex)
		args = append(args, filter.PIIType)
		argIndex++
	}
	if filter.LearningType != nil {
		query += fmt.Sprintf(" AND learning_type = $%d", argIndex)
		args = append(args, *filter.LearningType)
		argIndex++
	}
	if filter.IsActive != nil {
		query += fmt.Sprintf(" AND is_active = $%d", argIndex)
		args = append(args, *filter.IsActive)
	}

	query += " LIMIT 1"

	fp := &fplearningentity.FPLearning{}
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&fp.ID, &fp.TenantID, &fp.UserID, &fp.AssetID, &fp.PatternName, &fp.PIIType,
		&fp.FieldName, &fp.FieldPath, &fp.MatchedValue, &fp.LearningType, &fp.Version,
		&fp.PreviousValue, &fp.Justification, &fp.SourceFindingID, &fp.ScanRunID,
		&fp.ExpiresAt, &fp.IsActive, &fp.CreatedAt, &fp.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return fp, nil
}

// GetFPLearnings retrieves FP learnings with pagination
func (r *PostgresRepository) GetFPLearnings(ctx context.Context, filter fplearningentity.FPLearningFilter, page, pageSize int) ([]*fplearningentity.FPLearning, int, error) {
	baseQuery := `FROM fp_learning WHERE tenant_id = $1`
	args := []interface{}{filter.TenantID}
	argIndex := 2

	if filter.AssetID != nil {
		baseQuery += fmt.Sprintf(" AND asset_id = $%d", argIndex)
		args = append(args, *filter.AssetID)
		argIndex++
	}
	if filter.PatternName != "" {
		baseQuery += fmt.Sprintf(" AND pattern_name = $%d", argIndex)
		args = append(args, filter.PatternName)
		argIndex++
	}
	if filter.PIIType != "" {
		baseQuery += fmt.Sprintf(" AND pii_type = $%d", argIndex)
		args = append(args, filter.PIIType)
		argIndex++
	}
	if filter.LearningType != nil {
		baseQuery += fmt.Sprintf(" AND learning_type = $%d", argIndex)
		args = append(args, *filter.LearningType)
		argIndex++
	}
	if filter.IsActive != nil {
		baseQuery += fmt.Sprintf(" AND is_active = $%d", argIndex)
		args = append(args, *filter.IsActive)
	}

	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	query := "SELECT id, tenant_id, user_id, asset_id, pattern_name, pii_type, field_name, field_path, " +
		"matched_value, learning_type, version, previous_value, justification, source_finding_id, " +
		"scan_run_id, expires_at, is_active, created_at, updated_at " + baseQuery +
		fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var fps []*fplearningentity.FPLearning
	for rows.Next() {
		fp := &fplearningentity.FPLearning{}
		err := rows.Scan(
			&fp.ID, &fp.TenantID, &fp.UserID, &fp.AssetID, &fp.PatternName, &fp.PIIType,
			&fp.FieldName, &fp.FieldPath, &fp.MatchedValue, &fp.LearningType, &fp.Version,
			&fp.PreviousValue, &fp.Justification, &fp.SourceFindingID, &fp.ScanRunID,
			&fp.ExpiresAt, &fp.IsActive, &fp.CreatedAt, &fp.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		fps = append(fps, fp)
	}

	return fps, total, rows.Err()
}

// GetAllFPLearnings retrieves all FP learnings for a tenant
func (r *PostgresRepository) GetAllFPLearnings(ctx context.Context, filter fplearningentity.FPLearningFilter) ([]*fplearningentity.FPLearning, error) {
	query := `
		SELECT id, tenant_id, user_id, asset_id, pattern_name, pii_type, field_name, field_path,
			matched_value, learning_type, version, previous_value, justification, source_finding_id,
			scan_run_id, expires_at, is_active, created_at, updated_at
		FROM fp_learning WHERE tenant_id = $1
	`
	args := []interface{}{filter.TenantID}

	if filter.IsActive != nil {
		query += " AND is_active = $2"
		args = append(args, *filter.IsActive)
	}
	if filter.LearningType != nil {
		if len(args) == 1 {
			query += " AND learning_type = $2"
		} else {
			query += " AND learning_type = $3"
		}
		args = append(args, *filter.LearningType)
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fps []*fplearningentity.FPLearning
	for rows.Next() {
		fp := &fplearningentity.FPLearning{}
		err := rows.Scan(
			&fp.ID, &fp.TenantID, &fp.UserID, &fp.AssetID, &fp.PatternName, &fp.PIIType,
			&fp.FieldName, &fp.FieldPath, &fp.MatchedValue, &fp.LearningType, &fp.Version,
			&fp.PreviousValue, &fp.Justification, &fp.SourceFindingID, &fp.ScanRunID,
			&fp.ExpiresAt, &fp.IsActive, &fp.CreatedAt, &fp.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		fps = append(fps, fp)
	}

	return fps, rows.Err()
}

// UpdateFPLearning updates an FP learning record
func (r *PostgresRepository) UpdateFPLearning(ctx context.Context, fp *fplearningentity.FPLearning) error {
	query := `
		UPDATE fp_learning SET learning_type = $1, version = $2, is_active = $3, updated_at = NOW()
		WHERE id = $4
	`
	_, err := r.db.ExecContext(ctx, query, fp.LearningType, fp.Version, fp.IsActive, fp.ID)
	return err
}
