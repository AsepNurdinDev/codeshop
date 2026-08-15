package tests

import (
	"context"
	"testing"
	"time"

	"github.com/asepnrdn/codeshop/services/auth-service/internal/domain"
	"github.com/asepnrdn/codeshop/services/auth-service/internal/handler"
	"github.com/asepnrdn/codeshop/services/auth-service/internal/security"
	"github.com/asepnrdn/codeshop/services/auth-service/internal/service"
	pb "github.com/asepnrdn/codeshop/services/auth-service/proto/v1"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

func TestGRPCHandlerRegisterAndLogin(t *testing.T) {
	userRepo := newMockUserRepo()
	tokenRepo := newMockTokenRepo()
	jwtMgr := security.NewJWTManager("secret", "issuer", 15*time.Minute, 1*time.Hour)

	authSvc := service.NewAuthService(userRepo, tokenRepo, jwtMgr)
	grpcHandler := handler.NewGRPCHandler(authSvc)

	ctx := context.Background()

	// 1. Register RPC
	regResp, err := grpcHandler.Register(ctx, &pb.RegisterRequest{
		Email:    "test@codeshop.dev",
		Password: "SecurePassword123!",
		FullName: "CodeShop User",
	})

	assert.NoError(t, err)
	assert.NotEmpty(t, regResp.GetUserId())
	assert.Equal(t, "test@codeshop.dev", regResp.GetEmail())
	assert.Equal(t, "buyer", regResp.GetRole())

	// 2. Duplicate Register RPC (Should return AlreadyExists code)
	_, err = grpcHandler.Register(ctx, &pb.RegisterRequest{
		Email:    "test@codeshop.dev",
		Password: "SecurePassword123!",
		FullName: "CodeShop User",
	})

	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.AlreadyExists, st.Code())

	// 3. Login RPC
	loginResp, err := grpcHandler.Login(ctx, &pb.LoginRequest{
		Email:    "test@codeshop.dev",
		Password: "SecurePassword123!",
	})

	assert.NoError(t, err)
	assert.NotEmpty(t, loginResp.GetAccessToken())
	assert.NotEmpty(t, loginResp.GetRefreshToken())

	// 4. ValidateToken RPC
	valResp, err := grpcHandler.ValidateToken(ctx, &pb.ValidateTokenRequest{
		AccessToken: loginResp.GetAccessToken(),
	})

	assert.NoError(t, err)
	assert.True(t, valResp.GetValid())
	assert.Equal(t, "test@codeshop.dev", valResp.GetEmail())

	// 5. GetUser RPC
	userResp, err := grpcHandler.GetUser(ctx, &pb.GetUserRequest{
		UserId: regResp.GetUserId(),
	})

	assert.NoError(t, err)
	assert.Equal(t, "test@codeshop.dev", userResp.GetEmail())
	assert.Equal(t, "active", userResp.GetStatus())
}
