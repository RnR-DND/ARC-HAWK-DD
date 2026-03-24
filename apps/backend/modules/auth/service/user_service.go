package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/arc-platform/backend/modules/auth/entity"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrInvalidPassword   = errors.New("invalid password")
	ErrUserInactive      = errors.New("user account is inactive")
	ErrEmailExists       = errors.New("email already registered")
)

type UserService struct {
	repo       *persistence.PostgresRepository
	jwtService *JWTService
}

func NewUserService(repo *persistence.PostgresRepository) *UserService {
	return &UserService{
		repo:       repo,
		jwtService: NewJWTService(),
	}
}

func (s *UserService) CreateUser(ctx context.Context, tenantID uuid.UUID, email, password, firstName, lastName string, role entity.UserRole) (*entity.User, error) {
	existing, err := s.repo.GetUserByEmail(ctx, email)
	if err == nil && existing != nil {
		return nil, ErrEmailExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &entity.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hashedPassword),
		FirstName:    firstName,
		LastName:     lastName,
		Role:         role,
		TenantID:     tenantID,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

func (s *UserService) Authenticate(ctx context.Context, email, password, tenantIDStr string) (*entity.User, string, string, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, "", "", ErrUserNotFound
	}

	if !user.IsActive {
		return nil, "", "", ErrUserInactive
	}

	if user.TenantID.String() != tenantIDStr {
		return nil, "", "", ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, "", "", ErrInvalidPassword
	}

	sessionID := uuid.New()
	token, refreshToken, err := s.jwtService.GenerateToken(user, sessionID)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to generate token: %w", err)
	}

	now := time.Now()
	user.LastLoginAt = &now
	s.repo.UpdateUser(ctx, user)

	return user, token, refreshToken, nil
}

func (s *UserService) GetUserByID(ctx context.Context, userID uuid.UUID) (*entity.User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *UserService) UpdateUser(ctx context.Context, user *entity.User) error {
	user.UpdatedAt = time.Now()
	return s.repo.UpdateUser(ctx, user)
}

func (s *UserService) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return ErrInvalidPassword
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user.PasswordHash = string(hashedPassword)
	return s.UpdateUser(ctx, user)
}

func (s *UserService) ResetPassword(ctx context.Context, userID uuid.UUID, newPassword string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	user.PasswordHash = string(hashedPassword)
	return s.UpdateUser(ctx, user)
}

func (s *UserService) DeactivateUser(ctx context.Context, userID uuid.UUID) error {
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	user.IsActive = false
	return s.UpdateUser(ctx, user)
}

func (s *UserService) GetUsersByTenant(ctx context.Context, tenantID uuid.UUID) ([]*entity.User, error) {
	return s.repo.GetUsersByTenant(ctx, tenantID)
}

func (s *UserService) HasPermission(user *entity.User, permission entity.Permission) bool {
	permissions, ok := entity.RolePermissions[user.Role]
	if !ok {
		return false
	}

	for _, p := range permissions {
		if p == permission {
			return true
		}
	}

	return false
}

func (s *UserService) HasAnyPermission(user *entity.User, permissions ...entity.Permission) bool {
	for _, p := range permissions {
		if s.HasPermission(user, p) {
			return true
		}
	}
	return false
}

func (s *UserService) HasAllPermissions(user *entity.User, permissions ...entity.Permission) bool {
	for _, p := range permissions {
		if !s.HasPermission(user, p) {
			return false
		}
	}
	return true
}

func GetEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func (s *UserService) CreateDefaultAdmin(ctx context.Context, tenantID uuid.UUID) (*entity.User, error) {
	email := GetEnv("ADMIN_EMAIL", "admin@arc-hawk.local")
	password := GetEnv("ADMIN_PASSWORD", "ArcHawkAdmin2024!")

	return s.CreateUser(ctx, tenantID, email, password, "Admin", "User", entity.RoleAdmin)
}
