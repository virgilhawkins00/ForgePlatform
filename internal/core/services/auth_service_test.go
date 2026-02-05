package services

import (
	"context"
	"testing"
	"time"

	"github.com/forge-platform/forge/internal/core/domain"
	"github.com/forge-platform/forge/internal/core/ports"
	"github.com/google/uuid"
)

// Mock implementations for auth
type mockUserRepository struct {
	users map[uuid.UUID]*domain.User
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{users: make(map[uuid.UUID]*domain.User)}
}

func (m *mockUserRepository) Create(_ context.Context, user *domain.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepository) GetByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	user, ok := m.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (m *mockUserRepository) GetByUsername(_ context.Context, username string) (*domain.User, error) {
	for _, user := range m.users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, ErrUserNotFound
}

func (m *mockUserRepository) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, ErrUserNotFound
}

func (m *mockUserRepository) Update(_ context.Context, user *domain.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepository) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.users, id)
	return nil
}

func (m *mockUserRepository) List(_ context.Context, _ ports.UserFilter) ([]*domain.User, error) {
	result := make([]*domain.User, 0, len(m.users))
	for _, user := range m.users {
		result = append(result, user)
	}
	return result, nil
}

func (m *mockUserRepository) Count(_ context.Context) (int64, error) {
	return int64(len(m.users)), nil
}

type mockSessionRepository struct {
	sessions map[uuid.UUID]*domain.Session
}

func newMockSessionRepository() *mockSessionRepository {
	return &mockSessionRepository{sessions: make(map[uuid.UUID]*domain.Session)}
}

func (m *mockSessionRepository) Create(_ context.Context, s *domain.Session) error {
	m.sessions[s.ID] = s
	return nil
}

func (m *mockSessionRepository) GetByID(_ context.Context, id uuid.UUID) (*domain.Session, error) {
	s, ok := m.sessions[id]
	if !ok {
		return nil, ErrInvalidToken
	}
	return s, nil
}

func (m *mockSessionRepository) Update(_ context.Context, s *domain.Session) error {
	m.sessions[s.ID] = s
	return nil
}

func (m *mockSessionRepository) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.sessions, id)
	return nil
}

func (m *mockSessionRepository) GetByUserID(_ context.Context, _ uuid.UUID) ([]*domain.Session, error) {
	return []*domain.Session{}, nil
}

func (m *mockSessionRepository) DeleteByUserID(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockSessionRepository) DeleteExpired(_ context.Context) (int64, error) {
	return 0, nil
}

type mockAPIKeyRepository struct {
	keys map[uuid.UUID]*domain.APIKey
}

func newMockAPIKeyRepository() *mockAPIKeyRepository {
	return &mockAPIKeyRepository{keys: make(map[uuid.UUID]*domain.APIKey)}
}

func (m *mockAPIKeyRepository) Create(_ context.Context, k *domain.APIKey) error {
	m.keys[k.ID] = k
	return nil
}

func (m *mockAPIKeyRepository) GetByID(_ context.Context, id uuid.UUID) (*domain.APIKey, error) {
	k, ok := m.keys[id]
	if !ok {
		return nil, ErrInvalidToken
	}
	return k, nil
}

func (m *mockAPIKeyRepository) GetByUserID(_ context.Context, _ uuid.UUID) ([]*domain.APIKey, error) {
	return []*domain.APIKey{}, nil
}

func (m *mockAPIKeyRepository) GetByPrefix(_ context.Context, _ string) ([]*domain.APIKey, error) {
	return []*domain.APIKey{}, nil
}

func (m *mockAPIKeyRepository) Update(_ context.Context, k *domain.APIKey) error {
	m.keys[k.ID] = k
	return nil
}

func (m *mockAPIKeyRepository) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.keys, id)
	return nil
}

func (m *mockAPIKeyRepository) DeleteByUserID(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockAPIKeyRepository) DeleteExpired(_ context.Context) (int64, error) {
	return 0, nil
}

type mockAuditLogRepository struct {
	logs []*domain.AuditLog
}

func newMockAuditLogRepository() *mockAuditLogRepository {
	return &mockAuditLogRepository{logs: make([]*domain.AuditLog, 0)}
}

func (m *mockAuditLogRepository) Create(_ context.Context, log *domain.AuditLog) error {
	m.logs = append(m.logs, log)
	return nil
}

func (m *mockAuditLogRepository) List(_ context.Context, _ ports.AuditLogFilter) ([]*domain.AuditLog, error) {
	return m.logs, nil
}

func (m *mockAuditLogRepository) GetByID(_ context.Context, id uuid.UUID) (*domain.AuditLog, error) {
	for _, log := range m.logs {
		if log.ID == id {
			return log, nil
		}
	}
	return nil, nil
}

func (m *mockAuditLogRepository) DeleteBefore(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}

// Tests
func TestDefaultAuthConfig(t *testing.T) {
	cfg := DefaultAuthConfig()

	if cfg.MaxLoginAttempts != 5 {
		t.Errorf("MaxLoginAttempts = %d, want 5", cfg.MaxLoginAttempts)
	}
	if cfg.LockDuration != 15*time.Minute {
		t.Errorf("LockDuration = %v, want 15m", cfg.LockDuration)
	}
	if cfg.SessionDuration != 24*time.Hour {
		t.Errorf("SessionDuration = %v, want 24h", cfg.SessionDuration)
	}
}

func TestNewAuthService(t *testing.T) {
	svc := NewAuthService(
		newMockUserRepository(),
		newMockSessionRepository(),
		newMockAPIKeyRepository(),
		newMockAuditLogRepository(),
		DefaultAuthConfig(),
		&mockLogger{},
	)

	if svc == nil {
		t.Fatal("NewAuthService returned nil")
	}
}

func TestAuthService_CreateUser(t *testing.T) {
	svc := NewAuthService(
		newMockUserRepository(),
		newMockSessionRepository(),
		newMockAPIKeyRepository(),
		newMockAuditLogRepository(),
		DefaultAuthConfig(),
		&mockLogger{},
	)

	user, err := svc.CreateUser(context.Background(), "testuser", "test@example.com", "password123", domain.RoleOperator)

	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}
	if user == nil {
		t.Fatal("User is nil")
	}
	if user.Username != "testuser" {
		t.Errorf("Username = %v, want testuser", user.Username)
	}
	if user.Role != domain.RoleOperator {
		t.Errorf("Role = %v, want operator", user.Role)
	}
}

func TestAuthService_CreateUser_Duplicate(t *testing.T) {
	userRepo := newMockUserRepository()
	svc := NewAuthService(
		userRepo,
		newMockSessionRepository(),
		newMockAPIKeyRepository(),
		newMockAuditLogRepository(),
		DefaultAuthConfig(),
		&mockLogger{},
	)

	// Create first user
	svc.CreateUser(context.Background(), "testuser", "test@example.com", "password123", domain.RoleOperator)

	// Try to create duplicate
	_, err := svc.CreateUser(context.Background(), "testuser", "other@example.com", "password123", domain.RoleOperator)

	if err != ErrUserExists {
		t.Errorf("Expected ErrUserExists, got %v", err)
	}
}

func TestAuthService_Login(t *testing.T) {
	userRepo := newMockUserRepository()
	sessionRepo := newMockSessionRepository()
	svc := NewAuthService(
		userRepo,
		sessionRepo,
		newMockAPIKeyRepository(),
		newMockAuditLogRepository(),
		DefaultAuthConfig(),
		&mockLogger{},
	)

	// Create user first
	svc.CreateUser(context.Background(), "testuser", "test@example.com", "password123", domain.RoleOperator)

	// Login
	session, token, err := svc.Login(context.Background(), "testuser", "password123", "127.0.0.1", "TestAgent")

	if err != nil {
		t.Fatalf("Login error: %v", err)
	}
	if session == nil {
		t.Fatal("Session is nil")
	}
	if token == "" {
		t.Error("Token is empty")
	}
}

func TestAuthService_Login_InvalidPassword(t *testing.T) {
	userRepo := newMockUserRepository()
	svc := NewAuthService(
		userRepo,
		newMockSessionRepository(),
		newMockAPIKeyRepository(),
		newMockAuditLogRepository(),
		DefaultAuthConfig(),
		&mockLogger{},
	)

	svc.CreateUser(context.Background(), "testuser", "test@example.com", "password123", domain.RoleOperator)

	_, _, err := svc.Login(context.Background(), "testuser", "wrongpassword", "127.0.0.1", "TestAgent")

	if err != ErrInvalidCredentials {
		t.Errorf("Expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthService_Logout(t *testing.T) {
	userRepo := newMockUserRepository()
	sessionRepo := newMockSessionRepository()
	svc := NewAuthService(
		userRepo,
		sessionRepo,
		newMockAPIKeyRepository(),
		newMockAuditLogRepository(),
		DefaultAuthConfig(),
		&mockLogger{},
	)

	svc.CreateUser(context.Background(), "testuser", "test@example.com", "password123", domain.RoleOperator)
	session, _, _ := svc.Login(context.Background(), "testuser", "password123", "127.0.0.1", "TestAgent")

	err := svc.Logout(context.Background(), session.ID)

	if err != nil {
		t.Fatalf("Logout error: %v", err)
	}
}

func TestAuthService_CreateAPIKey(t *testing.T) {
	userRepo := newMockUserRepository()
	svc := NewAuthService(
		userRepo,
		newMockSessionRepository(),
		newMockAPIKeyRepository(),
		newMockAuditLogRepository(),
		DefaultAuthConfig(),
		&mockLogger{},
	)

	user, _ := svc.CreateUser(context.Background(), "testuser", "test@example.com", "password123", domain.RoleOperator)

	apiKey, key, err := svc.CreateAPIKey(context.Background(), user.ID, "test-key", []string{"read", "write"}, nil)

	if err != nil {
		t.Fatalf("CreateAPIKey error: %v", err)
	}
	if apiKey == nil {
		t.Fatal("APIKey is nil")
	}
	if key == "" {
		t.Error("Key is empty")
	}
	if apiKey.Name != "test-key" {
		t.Errorf("Name = %v, want test-key", apiKey.Name)
	}
}

func TestAuthService_GetUser(t *testing.T) {
	userRepo := newMockUserRepository()
	svc := NewAuthService(
		userRepo,
		newMockSessionRepository(),
		newMockAPIKeyRepository(),
		newMockAuditLogRepository(),
		DefaultAuthConfig(),
		&mockLogger{},
	)

	created, _ := svc.CreateUser(context.Background(), "testuser", "test@example.com", "password123", domain.RoleOperator)

	user, err := svc.GetUser(context.Background(), created.ID)

	if err != nil {
		t.Fatalf("GetUser error: %v", err)
	}
	if user.ID != created.ID {
		t.Error("User ID mismatch")
	}
}

