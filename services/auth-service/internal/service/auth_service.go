package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/asepnrdn/codeshop/services/auth-service/internal/domain"
	"github.com/asepnrdn/codeshop/services/auth-service/internal/repository"
	"github.com/asepnrdn/codeshop/services/auth-service/internal/security"
	"github.com/google/uuid"
)

type AuthService interface {
	Register(ctx context.Context, email, password, fullName string) (*domain.User, error)
	Login(ctx context.Context, email, password string) (*domain.TokenPair, error)
	ValidateToken(ctx context.Context, accessToken string) (*domain.Claims, error)
	RefreshToken(ctx context.Context, refreshToken string) (*domain.TokenPair, error)
	Logout(ctx context.Context, refreshToken string, userID uuid.UUID) error
	GetUser(ctx context.Context, userID uuid.UUID) (*domain.User, error)
}

type authService struct {
	userRepo  repository.UserRepository
	tokenRepo repository.TokenRepository
	jwtMgr    *security.JWTManager
}

func NewAuthService(
	userRepo repository.UserRepository,
	tokenRepo repository.TokenRepository,
	jwtMgr *security.JWTManager,
) AuthService {
	return &authService{
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
		jwtMgr:    jwtMgr,
	}
}

func (s *authService) Register(ctx context.Context, email, password, fullName string) (*domain.User, error) {
	if err := domain.ValidateEmail(email); err != nil {
		return nil, err
	}
	if err := domain.ValidatePassword(password); err != nil {
		return nil, err
	}
	if err := domain.ValidateFullName(fullName); err != nil {
		return nil, err
	}

	hashedPassword, err := security.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now()
	user := &domain.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: hashedPassword,
		FullName:     fullName,
		Role:         domain.RoleBuyer,
		Status:       domain.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *authService) Login(ctx context.Context, email, password string) (*domain.TokenPair, error) {
	if err := domain.ValidateEmail(email); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, err
	}

	if user.Status == domain.UserStatusSuspended {
		return nil, domain.ErrAccountSuspended
	}

	match, err := security.ComparePassword(password, user.PasswordHash)
	if err != nil || !match {
		return nil, domain.ErrInvalidCredentials
	}

	tokens, err := s.jwtMgr.GenerateTokenPair(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token pair: %w", err)
	}

	// Persist refresh token session
	tokenHash := security.HashToken(tokens.RefreshToken)
	now := time.Now()
	session := &domain.RefreshTokenSession{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: now.Add(time.Duration(tokens.ExpiresIn) * time.Second * 28), // 28x access token duration (approx 7 days)
		Revoked:   false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.tokenRepo.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to save refresh token session: %w", err)
	}

	return tokens, nil
}

func (s *authService) ValidateToken(ctx context.Context, accessToken string) (*domain.Claims, error) {
	claims, err := s.jwtMgr.ValidateToken(accessToken, domain.TokenTypeAccess)
	if err != nil {
		return nil, err
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	if user.Status == domain.UserStatusSuspended {
		return nil, domain.ErrAccountSuspended
	}

	return claims, nil
}

func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*domain.TokenPair, error) {
	claims, err := s.jwtMgr.ValidateToken(refreshToken, domain.TokenTypeRefresh)
	if err != nil {
		return nil, err
	}

	tokenHash := security.HashToken(refreshToken)
	session, err := s.tokenRepo.GetSessionByHash(ctx, tokenHash)
	if err != nil {
		return nil, err
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil || session.UserID != userID {
		return nil, domain.ErrInvalidToken
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user.Status == domain.UserStatusSuspended {
		return nil, domain.ErrAccountSuspended
	}

	// Revoke old refresh token (Token Rotation)
	_ = s.tokenRepo.RevokeSession(ctx, tokenHash)

	// Issue new token pair
	newTokens, err := s.jwtMgr.GenerateTokenPair(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new tokens: %w", err)
	}

	newTokenHash := security.HashToken(newTokens.RefreshToken)
	now := time.Now()
	newSession := &domain.RefreshTokenSession{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: newTokenHash,
		ExpiresAt: now.Add(time.Duration(newTokens.ExpiresIn) * time.Second * 28),
		Revoked:   false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.tokenRepo.CreateSession(ctx, newSession); err != nil {
		return nil, fmt.Errorf("failed to save new refresh token session: %w", err)
	}

	return newTokens, nil
}

func (s *authService) Logout(ctx context.Context, refreshToken string, userID uuid.UUID) error {
	if refreshToken != "" {
		tokenHash := security.HashToken(refreshToken)
		_ = s.tokenRepo.RevokeSession(ctx, tokenHash)
	}

	if userID != uuid.Nil {
		_ = s.tokenRepo.RevokeAllUserSessions(ctx, userID)
	}

	return nil
}

func (s *authService) GetUser(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	return s.userRepo.GetByID(ctx, userID)
}
