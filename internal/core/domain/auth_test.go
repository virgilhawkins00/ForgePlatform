package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewUser(t *testing.T) {
	user, err := NewUser("testuser", "test@example.com", "password123", RoleOperator)
	if err != nil {
		t.Fatalf("NewUser() error = %v", err)
	}

	if user.Username != "testuser" {
		t.Errorf("Username = %v, want testuser", user.Username)
	}
	if user.Email != "test@example.com" {
		t.Errorf("Email = %v, want test@example.com", user.Email)
	}
	if user.Role != RoleOperator {
		t.Errorf("Role = %v, want operator", user.Role)
	}
	if user.Status != UserStatusActive {
		t.Errorf("Status = %v, want active", user.Status)
	}
	if user.PasswordHash == "" {
		t.Error("PasswordHash is empty")
	}
}

func TestUser_CheckPassword(t *testing.T) {
	user, _ := NewUser("testuser", "test@example.com", "password123", RoleAdmin)

	if !user.CheckPassword("password123") {
		t.Error("CheckPassword() = false for correct password")
	}
	if user.CheckPassword("wrongpassword") {
		t.Error("CheckPassword() = true for wrong password")
	}
}

func TestUser_SetPassword(t *testing.T) {
	user, _ := NewUser("testuser", "test@example.com", "oldpassword", RoleAdmin)
	oldHash := user.PasswordHash

	err := user.SetPassword("newpassword")
	if err != nil {
		t.Fatalf("SetPassword() error = %v", err)
	}

	if user.PasswordHash == oldHash {
		t.Error("PasswordHash not changed")
	}
	if !user.CheckPassword("newpassword") {
		t.Error("CheckPassword() = false for new password")
	}
}

func TestUser_IsLocked(t *testing.T) {
	user, _ := NewUser("testuser", "test@example.com", "password", RoleAdmin)

	if user.IsLocked() {
		t.Error("IsLocked() = true for new user")
	}

	// Lock the user
	user.Status = UserStatusLocked
	if !user.IsLocked() {
		t.Error("IsLocked() = false for locked user")
	}

	// Lock with expiration in the past
	past := time.Now().Add(-time.Hour)
	user.LockedUntil = &past
	if user.IsLocked() {
		t.Error("IsLocked() = true for expired lock")
	}
}

func TestUser_RecordFailedLogin(t *testing.T) {
	user, _ := NewUser("testuser", "test@example.com", "password", RoleAdmin)

	user.RecordFailedLogin(3, time.Hour)
	if user.FailedLogins != 1 {
		t.Errorf("FailedLogins = %d, want 1", user.FailedLogins)
	}
	if user.Status == UserStatusLocked {
		t.Error("User locked after 1 failed login")
	}

	user.RecordFailedLogin(3, time.Hour)
	user.RecordFailedLogin(3, time.Hour)
	if user.Status != UserStatusLocked {
		t.Error("User not locked after 3 failed logins")
	}
}

func TestUser_ResetFailedLogins(t *testing.T) {
	user, _ := NewUser("testuser", "test@example.com", "password", RoleAdmin)
	user.FailedLogins = 5
	user.Status = UserStatusLocked

	user.ResetFailedLogins()

	if user.FailedLogins != 0 {
		t.Errorf("FailedLogins = %d, want 0", user.FailedLogins)
	}
	if user.Status != UserStatusActive {
		t.Errorf("Status = %v, want active", user.Status)
	}
	if user.LastLoginAt == nil {
		t.Error("LastLoginAt not set")
	}
}

func TestGenerateAPIKey(t *testing.T) {
	userID := uuid.Must(uuid.NewV7())
	apiKey, key, err := GenerateAPIKey(userID, "test-key", []string{"read", "write"}, nil)

	if err != nil {
		t.Fatalf("GenerateAPIKey() error = %v", err)
	}
	if apiKey == nil {
		t.Fatal("GenerateAPIKey() returned nil apiKey")
	}
	if key == "" {
		t.Error("GenerateAPIKey() returned empty key")
	}
	if len(key) != 64 { // 32 bytes = 64 hex chars
		t.Errorf("key length = %d, want 64", len(key))
	}
	if apiKey.KeyPrefix != key[:8] {
		t.Errorf("KeyPrefix = %v, want %v", apiKey.KeyPrefix, key[:8])
	}
}

func TestAPIKey_ValidateKey(t *testing.T) {
	userID := uuid.Must(uuid.NewV7())
	apiKey, key, _ := GenerateAPIKey(userID, "test-key", []string{"read"}, nil)

	if !apiKey.ValidateKey(key) {
		t.Error("ValidateKey() = false for correct key")
	}
	if apiKey.ValidateKey("wrongkey") {
		t.Error("ValidateKey() = true for wrong key")
	}
}

func TestAPIKey_IsValid(t *testing.T) {
	userID := uuid.Must(uuid.NewV7())
	apiKey, _, _ := GenerateAPIKey(userID, "test-key", []string{"read"}, nil)

	if !apiKey.IsValid() {
		t.Error("IsValid() = false for new key")
	}

	// Test revoked key
	apiKey.Revoke()
	if apiKey.IsValid() {
		t.Error("IsValid() = true for revoked key")
	}

	// Test expired key
	apiKey2, _, _ := GenerateAPIKey(userID, "test-key2", []string{"read"}, nil)
	past := time.Now().Add(-time.Hour)
	apiKey2.ExpiresAt = &past
	if apiKey2.IsValid() {
		t.Error("IsValid() = true for expired key")
	}
}

func TestAPIKey_Revoke(t *testing.T) {
	userID := uuid.Must(uuid.NewV7())
	apiKey, _, _ := GenerateAPIKey(userID, "test-key", []string{"read"}, nil)

	if apiKey.RevokedAt != nil {
		t.Error("RevokedAt not nil for new key")
	}

	apiKey.Revoke()
	if apiKey.RevokedAt == nil {
		t.Error("RevokedAt is nil after Revoke()")
	}
}

func TestAPIKey_HasPermission(t *testing.T) {
	userID := uuid.Must(uuid.NewV7())
	apiKey, _, _ := GenerateAPIKey(userID, "test-key", []string{"read", "write"}, nil)

	if !apiKey.HasPermission("read") {
		t.Error("HasPermission() = false for 'read'")
	}
	if !apiKey.HasPermission("write") {
		t.Error("HasPermission() = false for 'write'")
	}
	if apiKey.HasPermission("delete") {
		t.Error("HasPermission() = true for 'delete'")
	}

	// Test wildcard permission
	apiKey2, _, _ := GenerateAPIKey(userID, "admin-key", []string{"*"}, nil)
	if !apiKey2.HasPermission("anything") {
		t.Error("HasPermission() = false for wildcard")
	}
}

func TestAPIKey_RecordUsage(t *testing.T) {
	userID := uuid.Must(uuid.NewV7())
	apiKey, _, _ := GenerateAPIKey(userID, "test-key", []string{"read"}, nil)

	if apiKey.LastUsedAt != nil {
		t.Error("LastUsedAt not nil for new key")
	}

	apiKey.RecordUsage()
	if apiKey.LastUsedAt == nil {
		t.Error("LastUsedAt is nil after RecordUsage()")
	}
}

func TestGenerateSession(t *testing.T) {
	userID := uuid.Must(uuid.NewV7())
	session, token, err := GenerateSession(userID, "192.168.1.1", "Mozilla/5.0", 24*time.Hour)

	if err != nil {
		t.Fatalf("GenerateSession() error = %v", err)
	}
	if session == nil {
		t.Fatal("GenerateSession() returned nil session")
	}
	if token == "" {
		t.Error("GenerateSession() returned empty token")
	}
	if len(token) != 64 {
		t.Errorf("token length = %d, want 64", len(token))
	}
	if session.IPAddress != "192.168.1.1" {
		t.Errorf("IPAddress = %v, want 192.168.1.1", session.IPAddress)
	}
	if session.UserAgent != "Mozilla/5.0" {
		t.Errorf("UserAgent = %v, want Mozilla/5.0", session.UserAgent)
	}
}

func TestSession_ValidateToken(t *testing.T) {
	userID := uuid.Must(uuid.NewV7())
	session, token, _ := GenerateSession(userID, "", "", time.Hour)

	if !session.ValidateToken(token) {
		t.Error("ValidateToken() = false for correct token")
	}
	if session.ValidateToken("wrongtoken") {
		t.Error("ValidateToken() = true for wrong token")
	}
}

func TestSession_IsValid(t *testing.T) {
	userID := uuid.Must(uuid.NewV7())
	session, _, _ := GenerateSession(userID, "", "", time.Hour)

	if !session.IsValid() {
		t.Error("IsValid() = false for new session")
	}

	// Test revoked session
	session.Revoke()
	if session.IsValid() {
		t.Error("IsValid() = true for revoked session")
	}

	// Test expired session
	session2, _, _ := GenerateSession(userID, "", "", time.Hour)
	session2.ExpiresAt = time.Now().Add(-time.Hour)
	if session2.IsValid() {
		t.Error("IsValid() = true for expired session")
	}
}

func TestSession_Revoke(t *testing.T) {
	userID := uuid.Must(uuid.NewV7())
	session, _, _ := GenerateSession(userID, "", "", time.Hour)

	if session.RevokedAt != nil {
		t.Error("RevokedAt not nil for new session")
	}

	session.Revoke()
	if session.RevokedAt == nil {
		t.Error("RevokedAt is nil after Revoke()")
	}
}

func TestSession_Touch(t *testing.T) {
	userID := uuid.Must(uuid.NewV7())
	session, _, _ := GenerateSession(userID, "", "", time.Hour)
	oldLastActive := session.LastActiveAt

	time.Sleep(10 * time.Millisecond)
	session.Touch()

	if !session.LastActiveAt.After(oldLastActive) {
		t.Error("LastActiveAt not updated after Touch()")
	}
}

func TestSession_Extend(t *testing.T) {
	userID := uuid.Must(uuid.NewV7())
	session, _, _ := GenerateSession(userID, "", "", time.Hour)
	oldExpires := session.ExpiresAt

	time.Sleep(10 * time.Millisecond)
	session.Extend(2 * time.Hour)

	if !session.ExpiresAt.After(oldExpires) {
		t.Error("ExpiresAt not extended after Extend()")
	}
}

func TestNewAuditLog(t *testing.T) {
	userID := uuid.Must(uuid.NewV7())
	audit := NewAuditLog(&userID, "create", "user", "user-123")

	if audit.Action != "create" {
		t.Errorf("Action = %v, want create", audit.Action)
	}
	if audit.Resource != "user" {
		t.Errorf("Resource = %v, want user", audit.Resource)
	}
	if audit.ResourceID != "user-123" {
		t.Errorf("ResourceID = %v, want user-123", audit.ResourceID)
	}
	if !audit.Success {
		t.Error("Success = false for new audit log")
	}
}

func TestAuditLog_WithDetails(t *testing.T) {
	audit := NewAuditLog(nil, "update", "config", "")
	audit.WithDetails(map[string]string{"key": "value", "foo": "bar"})

	if audit.Details["key"] != "value" {
		t.Errorf("Details[key] = %v, want value", audit.Details["key"])
	}
	if audit.Details["foo"] != "bar" {
		t.Errorf("Details[foo] = %v, want bar", audit.Details["foo"])
	}
}

func TestAuditLog_WithError(t *testing.T) {
	audit := NewAuditLog(nil, "delete", "file", "file-123")
	audit.WithError(errors.New("permission denied"))

	if audit.Success {
		t.Error("Success = true after WithError()")
	}
	if audit.Error != "permission denied" {
		t.Errorf("Error = %v, want 'permission denied'", audit.Error)
	}
}

func TestAuditLog_WithContext(t *testing.T) {
	audit := NewAuditLog(nil, "login", "session", "")
	audit.WithContext("10.0.0.1", "curl/7.68.0")

	if audit.IPAddress != "10.0.0.1" {
		t.Errorf("IPAddress = %v, want 10.0.0.1", audit.IPAddress)
	}
	if audit.UserAgent != "curl/7.68.0" {
		t.Errorf("UserAgent = %v, want curl/7.68.0", audit.UserAgent)
	}
}

func TestHasRolePermission(t *testing.T) {
	tests := []struct {
		role       UserRole
		resource   ResourceType
		permission Permission
		want       bool
	}{
		{RoleAdmin, ResourceUsers, PermissionWrite, true},
		{RoleAdmin, ResourceUsers, PermissionAdmin, true},
		{RoleOperator, ResourceUsers, PermissionRead, true},
		{RoleOperator, ResourceUsers, PermissionWrite, false},
		{RoleViewer, ResourceMetrics, PermissionRead, true},
		{RoleViewer, ResourceMetrics, PermissionWrite, false},
		{RoleViewer, ResourceUsers, PermissionRead, false},
	}

	for _, tt := range tests {
		got := HasRolePermission(tt.role, tt.resource, tt.permission)
		if got != tt.want {
			t.Errorf("HasRolePermission(%v, %v, %v) = %v, want %v",
				tt.role, tt.resource, tt.permission, got, tt.want)
		}
	}
}

func TestUser_CanAccess(t *testing.T) {
	admin, _ := NewUser("admin", "admin@test.com", "pass", RoleAdmin)
	viewer, _ := NewUser("viewer", "viewer@test.com", "pass", RoleViewer)

	if !admin.CanAccess(ResourceUsers, PermissionAdmin) {
		t.Error("Admin should access users with admin permission")
	}
	if viewer.CanAccess(ResourceUsers, PermissionWrite) {
		t.Error("Viewer should not write to users")
	}

	// Test inactive user
	admin.Status = UserStatusInactive
	if admin.CanAccess(ResourceUsers, PermissionRead) {
		t.Error("Inactive user should not access anything")
	}
}

