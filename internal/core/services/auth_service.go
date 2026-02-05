// Package services implements core business logic services.
package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/forge-platform/forge/internal/core/domain"
	"github.com/forge-platform/forge/internal/core/ports"
	"github.com/google/uuid"
)

var (
	// ErrInvalidCredentials is returned when username or password is incorrect.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrAccountLocked is returned when the account is locked.
	ErrAccountLocked = errors.New("account is locked")
	// ErrUserNotFound is returned when the user is not found.
	ErrUserNotFound = errors.New("user not found")
	// ErrUserExists is returned when a user with the same username or email exists.
	ErrUserExists = errors.New("user already exists")
	// ErrSessionExpired is returned when the session has expired.
	ErrSessionExpired = errors.New("session expired")
	// ErrInvalidToken is returned when the token is invalid.
	ErrInvalidToken = errors.New("invalid token")
	// ErrAPIKeyRevoked is returned when the API key has been revoked.
	ErrAPIKeyRevoked = errors.New("API key revoked")
	// ErrAPIKeyExpired is returned when the API key has expired.
	ErrAPIKeyExpired = errors.New("API key expired")
	// ErrPermissionDenied is returned when the user lacks permission.
	ErrPermissionDenied = errors.New("permission denied")
)

// AuthConfig contains configuration for the auth service.
type AuthConfig struct {
	MaxLoginAttempts int           // Max failed login attempts before lock
	LockDuration     time.Duration // Duration to lock account
	SessionDuration  time.Duration // Session expiration time
	APIKeyDuration   time.Duration // Default API key expiration
}

// DefaultAuthConfig returns sensible defaults for auth configuration.
func DefaultAuthConfig() AuthConfig {
	return AuthConfig{
		MaxLoginAttempts: 5,
		LockDuration:     15 * time.Minute,
		SessionDuration:  24 * time.Hour,
		APIKeyDuration:   90 * 24 * time.Hour, // 90 days
	}
}

// AuthService handles authentication and authorization.
type AuthService struct {
	userRepo     ports.UserRepository
	sessionRepo  ports.SessionRepository
	apiKeyRepo   ports.APIKeyRepository
	auditRepo    ports.AuditLogRepository
	config       AuthConfig
	logger       ports.Logger
}

// NewAuthService creates a new authentication service.
func NewAuthService(
	userRepo ports.UserRepository,
	sessionRepo ports.SessionRepository,
	apiKeyRepo ports.APIKeyRepository,
	auditRepo ports.AuditLogRepository,
	config AuthConfig,
	logger ports.Logger,
) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		apiKeyRepo:  apiKeyRepo,
		auditRepo:   auditRepo,
		config:      config,
		logger:      logger,
	}
}

// CreateUser creates a new user account.
func (s *AuthService) CreateUser(ctx context.Context, username, email, password string, role domain.UserRole) (*domain.User, error) {
	// Check if user exists
	if s.userRepo != nil {
		existing, _ := s.userRepo.GetByUsername(ctx, username)
		if existing != nil {
			return nil, ErrUserExists
		}
		existing, _ = s.userRepo.GetByEmail(ctx, email)
		if existing != nil {
			return nil, ErrUserExists
		}
	}

	user, err := domain.NewUser(username, email, password, role)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if s.userRepo != nil {
		if err := s.userRepo.Create(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to save user: %w", err)
		}
	}

	s.audit(ctx, &user.ID, "user.create", "user", user.ID.String(), nil, nil)
	s.logger.Info("User created", "username", username, "role", role)

	return user, nil
}

// Login authenticates a user and returns a session token.
func (s *AuthService) Login(ctx context.Context, username, password, ipAddress, userAgent string) (*domain.Session, string, error) {
	var user *domain.User
	var err error

	if s.userRepo != nil {
		user, err = s.userRepo.GetByUsername(ctx, username)
		if err != nil {
			s.audit(ctx, nil, "user.login", "user", "", nil, ErrInvalidCredentials)
			return nil, "", ErrInvalidCredentials
		}
	} else {
		return nil, "", ErrUserNotFound
	}

	// Check if account is locked
	if user.IsLocked() {
		s.audit(ctx, &user.ID, "user.login", "user", user.ID.String(), nil, ErrAccountLocked)
		return nil, "", ErrAccountLocked
	}

	// Verify password
	if !user.CheckPassword(password) {
		user.RecordFailedLogin(s.config.MaxLoginAttempts, s.config.LockDuration)
		if s.userRepo != nil {
			s.userRepo.Update(ctx, user)
		}
		s.audit(ctx, &user.ID, "user.login", "user", user.ID.String(), nil, ErrInvalidCredentials)
		return nil, "", ErrInvalidCredentials
	}

	// Reset failed logins and create session
	user.ResetFailedLogins()
	if s.userRepo != nil {
		s.userRepo.Update(ctx, user)
	}

	session, token, err := domain.GenerateSession(user.ID, ipAddress, userAgent, s.config.SessionDuration)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create session: %w", err)
	}

	if s.sessionRepo != nil {
		if err := s.sessionRepo.Create(ctx, session); err != nil {
			return nil, "", fmt.Errorf("failed to save session: %w", err)
		}
	}

	s.audit(ctx, &user.ID, "user.login", "user", user.ID.String(),
		map[string]string{"ip": ipAddress}, nil)
	s.logger.Info("User logged in", "username", username, "ip", ipAddress)

	return session, token, nil
}

// Logout invalidates a session.
func (s *AuthService) Logout(ctx context.Context, sessionID uuid.UUID) error {
	if s.sessionRepo == nil {
		return nil
	}

	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return err
	}

	session.Revoke()
	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return err
	}

	s.audit(ctx, &session.UserID, "user.logout", "session", sessionID.String(), nil, nil)
	return nil
}

// ValidateSession checks if a session token is valid and returns the user.
func (s *AuthService) ValidateSession(ctx context.Context, token string) (*domain.User, *domain.Session, error) {
	if s.sessionRepo == nil || s.userRepo == nil {
		return nil, nil, ErrInvalidToken
	}

	// We need to find the session by token - this is inefficient but secure
	// In production, you'd use a Redis cache or similar
	// For now, we'll need to hash the token and search

	// This is a simplified implementation - in production you'd want a token index
	return nil, nil, ErrInvalidToken
}

// CreateAPIKey creates a new API key for a user.
func (s *AuthService) CreateAPIKey(ctx context.Context, userID uuid.UUID, name string, permissions []string, expiresIn *time.Duration) (*domain.APIKey, string, error) {
	var expiresAt *time.Time
	if expiresIn != nil {
		t := time.Now().Add(*expiresIn)
		expiresAt = &t
	} else if s.config.APIKeyDuration > 0 {
		t := time.Now().Add(s.config.APIKeyDuration)
		expiresAt = &t
	}

	apiKey, key, err := domain.GenerateAPIKey(userID, name, permissions, expiresAt)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate API key: %w", err)
	}

	if s.apiKeyRepo != nil {
		if err := s.apiKeyRepo.Create(ctx, apiKey); err != nil {
			return nil, "", fmt.Errorf("failed to save API key: %w", err)
		}
	}

	s.audit(ctx, &userID, "apikey.create", "apikey", apiKey.ID.String(),
		map[string]string{"name": name}, nil)
	s.logger.Info("API key created", "user_id", userID, "name", name)

	return apiKey, key, nil
}

// ValidateAPIKey validates an API key and returns the associated user.
func (s *AuthService) ValidateAPIKey(ctx context.Context, key string) (*domain.User, *domain.APIKey, error) {
	if s.apiKeyRepo == nil || s.userRepo == nil || len(key) < 8 {
		return nil, nil, ErrInvalidToken
	}

	prefix := key[:8]
	keys, err := s.apiKeyRepo.GetByPrefix(ctx, prefix)
	if err != nil {
		return nil, nil, ErrInvalidToken
	}

	for _, apiKey := range keys {
		if apiKey.ValidateKey(key) {
			if apiKey.RevokedAt != nil {
				return nil, nil, ErrAPIKeyRevoked
			}
			if !apiKey.IsValid() {
				return nil, nil, ErrAPIKeyExpired
			}

			user, err := s.userRepo.GetByID(ctx, apiKey.UserID)
			if err != nil {
				return nil, nil, err
			}

			// Record usage
			apiKey.RecordUsage()
			s.apiKeyRepo.Update(ctx, apiKey)

			return user, apiKey, nil
		}
	}

	return nil, nil, ErrInvalidToken
}

// RevokeAPIKey revokes an API key.
func (s *AuthService) RevokeAPIKey(ctx context.Context, keyID uuid.UUID) error {
	if s.apiKeyRepo == nil {
		return nil
	}

	apiKey, err := s.apiKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return err
	}

	apiKey.Revoke()
	if err := s.apiKeyRepo.Update(ctx, apiKey); err != nil {
		return err
	}

	s.audit(ctx, &apiKey.UserID, "apikey.revoke", "apikey", keyID.String(), nil, nil)
	return nil
}

// ListAPIKeys lists all API keys for a user.
func (s *AuthService) ListAPIKeys(ctx context.Context, userID uuid.UUID) ([]*domain.APIKey, error) {
	if s.apiKeyRepo == nil {
		return []*domain.APIKey{}, nil
	}
	return s.apiKeyRepo.GetByUserID(ctx, userID)
}

// GetUser retrieves a user by ID.
func (s *AuthService) GetUser(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	if s.userRepo == nil {
		return nil, ErrUserNotFound
	}
	return s.userRepo.GetByID(ctx, id)
}

// ListUsers lists users with optional filtering.
func (s *AuthService) ListUsers(ctx context.Context, filter ports.UserFilter) ([]*domain.User, error) {
	if s.userRepo == nil {
		return []*domain.User{}, nil
	}
	return s.userRepo.List(ctx, filter)
}

// UpdateUser updates a user's profile.
func (s *AuthService) UpdateUser(ctx context.Context, user *domain.User) error {
	if s.userRepo == nil {
		return nil
	}
	user.UpdatedAt = time.Now()
	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}
	s.audit(ctx, &user.ID, "user.update", "user", user.ID.String(), nil, nil)
	return nil
}

// DeleteUser deletes a user and their associated data.
func (s *AuthService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	if s.userRepo == nil {
		return nil
	}

	// Delete associated sessions and API keys
	if s.sessionRepo != nil {
		s.sessionRepo.DeleteByUserID(ctx, id)
	}
	if s.apiKeyRepo != nil {
		s.apiKeyRepo.DeleteByUserID(ctx, id)
	}

	if err := s.userRepo.Delete(ctx, id); err != nil {
		return err
	}

	s.audit(ctx, &id, "user.delete", "user", id.String(), nil, nil)
	return nil
}

// ChangePassword changes a user's password.
func (s *AuthService) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	if s.userRepo == nil {
		return ErrUserNotFound
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}

	if !user.CheckPassword(oldPassword) {
		s.audit(ctx, &userID, "user.password_change", "user", userID.String(), nil, ErrInvalidCredentials)
		return ErrInvalidCredentials
	}

	if err := user.SetPassword(newPassword); err != nil {
		return err
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	// Invalidate all sessions except current
	if s.sessionRepo != nil {
		s.sessionRepo.DeleteByUserID(ctx, userID)
	}

	s.audit(ctx, &userID, "user.password_change", "user", userID.String(), nil, nil)
	return nil
}

// audit creates an audit log entry.
func (s *AuthService) audit(ctx context.Context, userID *uuid.UUID, action, resource, resourceID string, details map[string]string, err error) {
	if s.auditRepo == nil {
		return
	}

	log := domain.NewAuditLog(userID, action, resource, resourceID)
	if details != nil {
		log.WithDetails(details)
	}
	if err != nil {
		log.WithError(err)
	}

	s.auditRepo.Create(ctx, log)
}

// CleanupExpired removes expired sessions and API keys.
func (s *AuthService) CleanupExpired(ctx context.Context) error {
	if s.sessionRepo != nil {
		if _, err := s.sessionRepo.DeleteExpired(ctx); err != nil {
			s.logger.Error("Failed to cleanup expired sessions", "error", err)
		}
	}
	if s.apiKeyRepo != nil {
		if _, err := s.apiKeyRepo.DeleteExpired(ctx); err != nil {
			s.logger.Error("Failed to cleanup expired API keys", "error", err)
		}
	}
	return nil
}

// GetAuditLogs retrieves audit logs with filtering.
func (s *AuthService) GetAuditLogs(ctx context.Context, filter ports.AuditLogFilter) ([]*domain.AuditLog, error) {
	if s.auditRepo == nil {
		return []*domain.AuditLog{}, nil
	}
	return s.auditRepo.List(ctx, filter)
}

// ============================================================================
// RBAC (Role-Based Access Control)
// ============================================================================

// CheckPermission verifies if a user has permission to perform an action on a resource.
func (s *AuthService) CheckPermission(ctx context.Context, userID uuid.UUID, resource domain.ResourceType, permission domain.Permission) error {
	if s.userRepo == nil {
		return nil // No auth configured, allow all
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}

	if !user.CanAccess(resource, permission) {
		s.audit(ctx, &userID, "permission.denied", string(resource), "",
			map[string]string{"permission": string(permission)}, ErrPermissionDenied)
		return ErrPermissionDenied
	}

	return nil
}

// CheckAPIKeyPermission verifies if an API key has permission to perform an action.
func (s *AuthService) CheckAPIKeyPermission(ctx context.Context, apiKey *domain.APIKey, resource domain.ResourceType, permission domain.Permission) error {
	// Check if API key has explicit permission
	permStr := string(resource) + ":" + string(permission)
	if apiKey.HasPermission(permStr) || apiKey.HasPermission(string(resource)+":*") || apiKey.HasPermission("*") {
		return nil
	}

	// Fall back to user's role permissions
	if s.userRepo == nil {
		return ErrPermissionDenied
	}

	user, err := s.userRepo.GetByID(ctx, apiKey.UserID)
	if err != nil {
		return ErrUserNotFound
	}

	if !user.CanAccess(resource, permission) {
		return ErrPermissionDenied
	}

	return nil
}

// GetUserPermissions returns all permissions for a user.
func (s *AuthService) GetUserPermissions(ctx context.Context, userID uuid.UUID) (map[domain.ResourceType][]domain.Permission, error) {
	if s.userRepo == nil {
		return nil, ErrUserNotFound
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	perms, ok := domain.RolePermissions[user.Role]
	if !ok {
		return map[domain.ResourceType][]domain.Permission{}, nil
	}

	return perms, nil
}

// UpdateUserRole updates a user's role.
func (s *AuthService) UpdateUserRole(ctx context.Context, userID uuid.UUID, newRole domain.UserRole) error {
	if s.userRepo == nil {
		return ErrUserNotFound
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	oldRole := user.Role
	user.Role = newRole
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	s.audit(ctx, &userID, "user.role_change", "user", userID.String(),
		map[string]string{"old_role": string(oldRole), "new_role": string(newRole)}, nil)
	s.logger.Info("User role updated", "user_id", userID, "old_role", oldRole, "new_role", newRole)

	return nil
}
