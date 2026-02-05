// Package domain contains the core business entities for the Forge platform.
package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// UserRole represents a user's role in the system.
type UserRole string

const (
	RoleAdmin    UserRole = "admin"
	RoleOperator UserRole = "operator"
	RoleViewer   UserRole = "viewer"
)

// UserStatus represents the status of a user account.
type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
	UserStatusLocked   UserStatus = "locked"
)

// User represents a user account in the system.
type User struct {
	ID           uuid.UUID         `json:"id"`
	Username     string            `json:"username"`
	Email        string            `json:"email"`
	PasswordHash string            `json:"-"` // Never serialize
	Role         UserRole          `json:"role"`
	Status       UserStatus        `json:"status"`
	DisplayName  string            `json:"display_name,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	LastLoginAt  *time.Time        `json:"last_login_at,omitempty"`
	FailedLogins int               `json:"failed_logins"`
	LockedUntil  *time.Time        `json:"locked_until,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// APIKey represents an API key for programmatic access.
type APIKey struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID  `json:"user_id"`
	Name        string     `json:"name"`
	KeyHash     string     `json:"-"` // Never serialize the hash
	KeyPrefix   string     `json:"key_prefix"` // First 8 chars for identification
	Permissions []string   `json:"permissions"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
}

// Session represents an active user session.
type Session struct {
	ID           uuid.UUID  `json:"id"`
	UserID       uuid.UUID  `json:"user_id"`
	TokenHash    string     `json:"-"` // Never serialize
	IPAddress    string     `json:"ip_address,omitempty"`
	UserAgent    string     `json:"user_agent,omitempty"`
	ExpiresAt    time.Time  `json:"expires_at"`
	CreatedAt    time.Time  `json:"created_at"`
	LastActiveAt time.Time  `json:"last_active_at"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
}

// AuditLog represents an audit log entry.
type AuditLog struct {
	ID         uuid.UUID         `json:"id"`
	UserID     *uuid.UUID        `json:"user_id,omitempty"`
	Action     string            `json:"action"`
	Resource   string            `json:"resource"`
	ResourceID string            `json:"resource_id,omitempty"`
	Details    map[string]string `json:"details,omitempty"`
	IPAddress  string            `json:"ip_address,omitempty"`
	UserAgent  string            `json:"user_agent,omitempty"`
	Success    bool              `json:"success"`
	Error      string            `json:"error,omitempty"`
	Timestamp  time.Time         `json:"timestamp"`
}

// NewUser creates a new user with the given credentials.
func NewUser(username, email, password string, role UserRole) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	return &User{
		ID:           uuid.Must(uuid.NewV7()),
		Username:     username,
		Email:        email,
		PasswordHash: string(hash),
		Role:         role,
		Status:       UserStatusActive,
		Metadata:     make(map[string]string),
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// CheckPassword verifies the password against the stored hash.
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// SetPassword updates the user's password.
func (u *User) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	u.UpdatedAt = time.Now()
	return nil
}

// IsLocked checks if the user account is locked.
func (u *User) IsLocked() bool {
	if u.Status == UserStatusLocked {
		if u.LockedUntil != nil && time.Now().After(*u.LockedUntil) {
			return false // Lock expired
		}
		return true
	}
	return false
}

// RecordFailedLogin increments failed login count and locks if threshold reached.
func (u *User) RecordFailedLogin(maxAttempts int, lockDuration time.Duration) {
	u.FailedLogins++
	if u.FailedLogins >= maxAttempts {
		u.Status = UserStatusLocked
		lockUntil := time.Now().Add(lockDuration)
		u.LockedUntil = &lockUntil
	}
	u.UpdatedAt = time.Now()
}

// ResetFailedLogins resets the failed login counter after successful login.
func (u *User) ResetFailedLogins() {
	u.FailedLogins = 0
	u.LockedUntil = nil
	if u.Status == UserStatusLocked {
		u.Status = UserStatusActive
	}
	now := time.Now()
	u.LastLoginAt = &now
	u.UpdatedAt = now
}

// GenerateAPIKey creates a new API key and returns both the key and the APIKey struct.
// The returned key should be shown to the user once and never stored in plain text.
func GenerateAPIKey(userID uuid.UUID, name string, permissions []string, expiresAt *time.Time) (*APIKey, string, error) {
	// Generate 32 random bytes for the key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, "", err
	}

	key := hex.EncodeToString(keyBytes)
	keyHash := sha256.Sum256([]byte(key))

	now := time.Now()
	apiKey := &APIKey{
		ID:          uuid.Must(uuid.NewV7()),
		UserID:      userID,
		Name:        name,
		KeyHash:     hex.EncodeToString(keyHash[:]),
		KeyPrefix:   key[:8],
		Permissions: permissions,
		ExpiresAt:   expiresAt,
		CreatedAt:   now,
	}

	return apiKey, key, nil
}

// ValidateKey checks if the provided key matches this API key.
func (k *APIKey) ValidateKey(key string) bool {
	keyHash := sha256.Sum256([]byte(key))
	return k.KeyHash == hex.EncodeToString(keyHash[:])
}

// IsValid checks if the API key is valid (not expired or revoked).
func (k *APIKey) IsValid() bool {
	if k.RevokedAt != nil {
		return false
	}
	if k.ExpiresAt != nil && time.Now().After(*k.ExpiresAt) {
		return false
	}
	return true
}

// Revoke marks the API key as revoked.
func (k *APIKey) Revoke() {
	now := time.Now()
	k.RevokedAt = &now
}

// HasPermission checks if the API key has a specific permission.
func (k *APIKey) HasPermission(permission string) bool {
	for _, p := range k.Permissions {
		if p == permission || p == "*" {
			return true
		}
	}
	return false
}

// RecordUsage updates the last used timestamp.
func (k *APIKey) RecordUsage() {
	now := time.Now()
	k.LastUsedAt = &now
}

// GenerateSession creates a new session and returns both the token and the Session struct.
func GenerateSession(userID uuid.UUID, ipAddress, userAgent string, duration time.Duration) (*Session, string, error) {
	// Generate 32 random bytes for the token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, "", err
	}

	token := hex.EncodeToString(tokenBytes)
	tokenHash := sha256.Sum256([]byte(token))

	now := time.Now()
	session := &Session{
		ID:           uuid.Must(uuid.NewV7()),
		UserID:       userID,
		TokenHash:    hex.EncodeToString(tokenHash[:]),
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		ExpiresAt:    now.Add(duration),
		CreatedAt:    now,
		LastActiveAt: now,
	}

	return session, token, nil
}

// ValidateToken checks if the provided token matches this session.
func (s *Session) ValidateToken(token string) bool {
	tokenHash := sha256.Sum256([]byte(token))
	return s.TokenHash == hex.EncodeToString(tokenHash[:])
}

// IsValid checks if the session is valid (not expired or revoked).
func (s *Session) IsValid() bool {
	if s.RevokedAt != nil {
		return false
	}
	return time.Now().Before(s.ExpiresAt)
}

// Revoke marks the session as revoked.
func (s *Session) Revoke() {
	now := time.Now()
	s.RevokedAt = &now
}

// Touch updates the last active timestamp.
func (s *Session) Touch() {
	s.LastActiveAt = time.Now()
}

// Extend extends the session expiration.
func (s *Session) Extend(duration time.Duration) {
	s.ExpiresAt = time.Now().Add(duration)
	s.Touch()
}

// NewAuditLog creates a new audit log entry.
func NewAuditLog(userID *uuid.UUID, action, resource, resourceID string) *AuditLog {
	return &AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		UserID:     userID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Details:    make(map[string]string),
		Success:    true,
		Timestamp:  time.Now(),
	}
}

// WithDetails adds details to the audit log.
func (a *AuditLog) WithDetails(details map[string]string) *AuditLog {
	for k, v := range details {
		a.Details[k] = v
	}
	return a
}

// WithError marks the audit log as failed with an error.
func (a *AuditLog) WithError(err error) *AuditLog {
	a.Success = false
	a.Error = err.Error()
	return a
}

// WithContext adds request context (IP, user agent) to the audit log.
func (a *AuditLog) WithContext(ipAddress, userAgent string) *AuditLog {
	a.IPAddress = ipAddress
	a.UserAgent = userAgent
	return a
}

// ============================================================================
// RBAC (Role-Based Access Control)
// ============================================================================

// Permission represents an action that can be performed on a resource.
type Permission string

// Common permissions
const (
	PermissionRead   Permission = "read"
	PermissionWrite  Permission = "write"
	PermissionDelete Permission = "delete"
	PermissionAdmin  Permission = "admin"
)

// ResourceType represents a type of resource in the system.
type ResourceType string

// Resource types
const (
	ResourceUsers     ResourceType = "users"
	ResourceAPIKeys   ResourceType = "apikeys"
	ResourceTasks     ResourceType = "tasks"
	ResourceMetrics   ResourceType = "metrics"
	ResourceWorkflows ResourceType = "workflows"
	ResourceAlerts    ResourceType = "alerts"
	ResourcePlugins   ResourceType = "plugins"
	ResourceTraces    ResourceType = "traces"
	ResourceLogs      ResourceType = "logs"
	ResourceProfiles  ResourceType = "profiles"
	ResourceAudit     ResourceType = "audit"
	ResourceSystem    ResourceType = "system"
)

// RolePermissions defines the default permissions for each role.
var RolePermissions = map[UserRole]map[ResourceType][]Permission{
	RoleAdmin: {
		ResourceUsers:     {PermissionRead, PermissionWrite, PermissionDelete, PermissionAdmin},
		ResourceAPIKeys:   {PermissionRead, PermissionWrite, PermissionDelete, PermissionAdmin},
		ResourceTasks:     {PermissionRead, PermissionWrite, PermissionDelete, PermissionAdmin},
		ResourceMetrics:   {PermissionRead, PermissionWrite, PermissionDelete, PermissionAdmin},
		ResourceWorkflows: {PermissionRead, PermissionWrite, PermissionDelete, PermissionAdmin},
		ResourceAlerts:    {PermissionRead, PermissionWrite, PermissionDelete, PermissionAdmin},
		ResourcePlugins:   {PermissionRead, PermissionWrite, PermissionDelete, PermissionAdmin},
		ResourceTraces:    {PermissionRead, PermissionWrite, PermissionDelete, PermissionAdmin},
		ResourceLogs:      {PermissionRead, PermissionWrite, PermissionDelete, PermissionAdmin},
		ResourceProfiles:  {PermissionRead, PermissionWrite, PermissionDelete, PermissionAdmin},
		ResourceAudit:     {PermissionRead, PermissionAdmin},
		ResourceSystem:    {PermissionRead, PermissionWrite, PermissionAdmin},
	},
	RoleOperator: {
		ResourceUsers:     {PermissionRead},
		ResourceAPIKeys:   {PermissionRead, PermissionWrite},
		ResourceTasks:     {PermissionRead, PermissionWrite, PermissionDelete},
		ResourceMetrics:   {PermissionRead, PermissionWrite},
		ResourceWorkflows: {PermissionRead, PermissionWrite, PermissionDelete},
		ResourceAlerts:    {PermissionRead, PermissionWrite, PermissionDelete},
		ResourcePlugins:   {PermissionRead, PermissionWrite},
		ResourceTraces:    {PermissionRead, PermissionWrite},
		ResourceLogs:      {PermissionRead, PermissionWrite},
		ResourceProfiles:  {PermissionRead, PermissionWrite, PermissionDelete},
		ResourceAudit:     {PermissionRead},
		ResourceSystem:    {PermissionRead},
	},
	RoleViewer: {
		ResourceUsers:     {},
		ResourceAPIKeys:   {PermissionRead},
		ResourceTasks:     {PermissionRead},
		ResourceMetrics:   {PermissionRead},
		ResourceWorkflows: {PermissionRead},
		ResourceAlerts:    {PermissionRead},
		ResourcePlugins:   {PermissionRead},
		ResourceTraces:    {PermissionRead},
		ResourceLogs:      {PermissionRead},
		ResourceProfiles:  {PermissionRead},
		ResourceAudit:     {},
		ResourceSystem:    {PermissionRead},
	},
}

// HasPermission checks if a role has a specific permission on a resource.
func HasRolePermission(role UserRole, resource ResourceType, permission Permission) bool {
	rolePerms, ok := RolePermissions[role]
	if !ok {
		return false
	}
	resourcePerms, ok := rolePerms[resource]
	if !ok {
		return false
	}
	for _, p := range resourcePerms {
		if p == permission || p == PermissionAdmin {
			return true
		}
	}
	return false
}

// CanAccess checks if a user can perform an action on a resource.
func (u *User) CanAccess(resource ResourceType, permission Permission) bool {
	if u.Status != UserStatusActive {
		return false
	}
	return HasRolePermission(u.Role, resource, permission)
}
