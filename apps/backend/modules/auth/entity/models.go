package entity

import (
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	RoleAdmin    UserRole = "admin"
	RoleAuditor  UserRole = "auditor"
	RoleOperator UserRole = "operator"
	RoleViewer   UserRole = "viewer"
)

type Permission string

const (
	PermissionScan             Permission = "scan:create"
	PermissionScanRead         Permission = "scan:read"
	PermissionScanDelete       Permission = "scan:delete"
	PermissionRemediate        Permission = "remediation:execute"
	PermissionRemediateApprove Permission = "remediation:approve"
	PermissionSourceManage     Permission = "source:manage"
	PermissionSourceRead       Permission = "source:read"
	PermissionReport           Permission = "report:view"
	PermissionSettings         Permission = "settings:manage"
	PermissionUserManage       Permission = "user:manage"
)

var RolePermissions = map[UserRole][]Permission{
	RoleAdmin: {
		PermissionScan, PermissionScanRead, PermissionScanDelete,
		PermissionRemediate, PermissionRemediateApprove,
		PermissionSourceManage, PermissionSourceRead,
		PermissionReport, PermissionSettings, PermissionUserManage,
	},
	RoleAuditor: {
		PermissionScanRead, PermissionSourceRead, PermissionReport,
	},
	RoleOperator: {
		PermissionScan, PermissionScanRead,
		PermissionSourceManage, PermissionSourceRead,
		PermissionReport,
	},
	RoleViewer: {
		PermissionScanRead, PermissionSourceRead, PermissionReport,
	},
}

type User struct {
	ID           uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
	Email        string     `json:"email" gorm:"uniqueIndex;size:255"`
	PasswordHash string     `json:"-" gorm:"size:255"`
	FirstName    string     `json:"first_name" gorm:"size:100"`
	LastName     string     `json:"last_name" gorm:"size:100"`
	Role         UserRole   `json:"role" gorm:"size:50;default:viewer"`
	TenantID     uuid.UUID  `json:"tenant_id" gorm:"type:uuid;index"`
	IsActive     bool       `json:"is_active" gorm:"default:true"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type Tenant struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	Name        string    `json:"name" gorm:"size:255;uniqueIndex"`
	Slug        string    `json:"slug" gorm:"size:100;uniqueIndex"`
	Description string    `json:"description" gorm:"type:text"`
	IsActive    bool      `json:"is_active" gorm:"default:true"`
	Settings    string    `json:"settings" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type AuditLog struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	EventType    string    `json:"event_type"`
	TenantID     uuid.UUID `json:"tenant_id" gorm:"type:uuid;index"`
	UserID       uuid.UUID `json:"user_id" gorm:"type:uuid;index"`
	Action       string    `json:"action" gorm:"size:100;index"`
	ResourceType string    `json:"resource_type" gorm:"size:100;index"`
	ResourceID   string    `json:"resource_id" gorm:"size:255"`
	IPAddress    string    `json:"ip_address" gorm:"size:45"`
	UserAgent    string    `json:"user_agent" gorm:"size:500"`
	Metadata     string    `json:"metadata" gorm:"type:text"`
	CreatedAt    time.Time `json:"created_at" gorm:"index"`
	// Chain integrity fields added in migration 000029.
	PreviousHash string `json:"previous_hash,omitempty" gorm:"type:text"`
	EntryHash    string `json:"entry_hash,omitempty" gorm:"type:text;index"`
}

type LoginSession struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;index"`
	TenantID  uuid.UUID `json:"tenant_id" gorm:"type:uuid;index"`
	TokenHash string    `json:"-" gorm:"size:64;uniqueIndex"`
	ExpiresAt time.Time `json:"expires_at" gorm:"index"`
	IPAddress string    `json:"ip_address" gorm:"size:45"`
	UserAgent string    `json:"user_agent" gorm:"size:500"`
	CreatedAt time.Time `json:"created_at"`
}
