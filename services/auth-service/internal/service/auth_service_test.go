package service

import (
	"context"
	"testing"
	"time"

	"github.com/asepnrdn/codeshop/services/auth-service/internal/domain"
	"github.com/asepnrdn/codeshop/services/auth-service/internal/security"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// Mock Repositories
type mockUserRepo struct {
	users map[string]*domain.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*domain.User)}
}

func (m *mockUserRepo) Create(ctx context.Context, u *domain.User) error {
	if _, exists := m.users[u.Email]; exists {
		return domain.ErrEmailAlreadyExists
	}
	m.users[u.Email] = u
	m.users[u.ID.String()] = u
	return nil
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, ok := m.users[email]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u, ok := m.users[id.String()]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

func (m *mockUserRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.UserStatus) error {
	u, ok := m.users[id.String()]
	if !ok {
		return domain.ErrUserNotFound
	}
	u.Status = status
	return nil
}

type mockTokenRepo struct {
	sessions map[string]*domain.RefreshTokenSession
}

func newMockTokenRepo() *mockTokenRepo {
	return &mockTokenRepo{sessions: make(map[string]*domain.RefreshTokenSession)}
}

func (m *mockTokenRepo) CreateSession(ctx context.Context, s *domain.RefreshTokenSession) error {
	m.sessions[s.TokenHash] = s
	return nil
}

func (m *mockTokenRepo) GetSessionByHash(ctx context.Context, hash string) (*domain.RefreshTokenSession, error) {
	s, ok := m.sessions[hash]
	if !ok {
		return nil, domain.ErrInvalidToken
	}
	if s.Revoked {
		return nil, domain.ErrTokenRevoked
	}
	return s, nil
}

func (m *mockTokenRepo) RevokeSession(ctx context.Context, hash string) error {
	if s, ok := m.sessions[hash]; ok {
		s.Revoked = true
	}
	return nil
}

func (m *mockTokenRepo) RevokeAllUserSessions(ctx context.Context, userID uuid.UUID) error {
	for _, s := range m.sessions {
		if s.UserID == userID {
			s.Revoked = true
		}
	}
	return nil
}

func TestRegisterAndLoginFlow(t *testing.T) {
	userRepo := newMockUserRepo()
	tokenRepo := newMockTokenRepo()
	jwtMgr := security.NewJWTManager("secret", "issuer", 15*time.Minute, 1*time.Hour)

	authSvc := NewAuthService(userRepo, tokenRepo, jwtMgr)
	ctx := context.Background()

	email := "testuser@example.com"
	password := "Password123!"
	fullName := "Test User"

	// 1. Register User
	user, err := authSvc.Register(ctx, email, password, fullName)
	assert.NoError(t, err)
	assert.Equal(t, email, user.Email)
	assert.Equal(t, domain.RoleBuyer, user.Role)

	// 2. Duplicate Registration (Should fail)
	_, err = authSvc.Register(ctx, email, password, fullName)
	assert.ErrorIs(t, err, domain.ErrEmailAlreadyExists)

	// 3. Login Success
	tokens, err := authSvc.Login(ctx, email, password)
	assert.NoError(t, err)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)

	// 4. Login Wrong Password
	_, err = authSvc.Login(ctx, email, "WrongPassword!")
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)

	// 5. Validate Access Token
	claims, err := authSvc.ValidateToken(ctx, tokens.AccessToken)
	assert.NoError(t, err)
	assert.Equal(t, email, claims.Email)

	// 6. Refresh Tokens
	newTokens, err := authSvc.RefreshToken(ctx, tokens.RefreshToken)
	assert.NoError(t, err)
	assert.NotEmpty(t, newTokens.AccessToken)

	// 7. Old Refresh Token reused (Should fail - token rotation)
	_, err = authSvc.RefreshToken(ctx, tokens.RefreshToken)
	assert.ErrorIs(t, err, domain.ErrTokenRevoked)

	// 8. Logout
	err = authSvc.Logout(ctx, newTokens.RefreshToken, user.ID)
	assert.NoError(t, err)
}
