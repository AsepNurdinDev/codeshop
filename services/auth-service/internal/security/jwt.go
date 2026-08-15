package security

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/asepnrdn/codeshop/services/auth-service/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTManager struct {
	secretKey     []byte
	issuer        string
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

func NewJWTManager(secretKey string, issuer string, accessExpiry, refreshExpiry time.Duration) *JWTManager {
	return &JWTManager{
		secretKey:     []byte(secretKey),
		issuer:        issuer,
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
	}
}

func (m *JWTManager) GenerateTokenPair(user *domain.User) (*domain.TokenPair, error) {
	now := time.Now()

	// 1. Generate Access Token
	accessJTI := uuid.New().String()
	accessClaims := &domain.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   user.ID.String(),
			ID:        accessJTI,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessExpiry)),
		},
		UserID:    user.ID.String(),
		Email:     user.Email,
		Role:      string(user.Role),
		TokenType: string(domain.TokenTypeAccess),
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(m.secretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// 2. Generate Refresh Token
	refreshJTI := uuid.New().String()
	refreshClaims := &domain.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   user.ID.String(),
			ID:        refreshJTI,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.refreshExpiry)),
		},
		UserID:    user.ID.String(),
		Email:     user.Email,
		Role:      string(user.Role),
		TokenType: string(domain.TokenTypeRefresh),
	}

	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(m.secretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(m.accessExpiry.Seconds()),
		TokenType:    "Bearer",
		UserID:       user.ID,
	}, nil
}

func (m *JWTManager) ValidateToken(tokenStr string, expectedType domain.TokenType) (*domain.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &domain.Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, domain.ErrExpiredToken
		}
		return nil, domain.ErrInvalidToken
	}

	claims, ok := token.Claims.(*domain.Claims)
	if !ok || !token.Valid {
		return nil, domain.ErrInvalidToken
	}

	if claims.Issuer != m.issuer {
		return nil, domain.ErrInvalidToken
	}

	if claims.TokenType != string(expectedType) {
		return nil, domain.ErrInvalidTokenType
	}

	return claims, nil
}

// HashToken generates a SHA256 hex string of the token for storage in DB/Redis
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
