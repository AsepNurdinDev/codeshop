package security

import (
	"testing"
	"time"

	"github.com/asepnrdn/codeshop/services/auth-service/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGenerateAndValidateTokenPair(t *testing.T) {
	secret := "test-secret-key-12345"
	issuer := "test-issuer"
	mgr := NewJWTManager(secret, issuer, 15*time.Minute, 1*time.Hour)

	user := &domain.User{
		ID:    uuid.New(),
		Email: "user@example.com",
		Role:  domain.RoleBuyer,
	}

	// 1. Generate Token Pair
	pair, err := mgr.GenerateTokenPair(user)
	assert.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	assert.Equal(t, int64(900), pair.ExpiresIn)
	assert.Equal(t, "Bearer", pair.TokenType)

	// 2. Validate Access Token
	claims, err := mgr.ValidateToken(pair.AccessToken, domain.TokenTypeAccess)
	assert.NoError(t, err)
	assert.Equal(t, user.ID.String(), claims.UserID)
	assert.Equal(t, user.Email, claims.Email)
	assert.Equal(t, string(domain.RoleBuyer), claims.Role)
	assert.Equal(t, string(domain.TokenTypeAccess), claims.TokenType)

	// 3. Validate Refresh Token
	refreshClaims, err := mgr.ValidateToken(pair.RefreshToken, domain.TokenTypeRefresh)
	assert.NoError(t, err)
	assert.Equal(t, user.ID.String(), refreshClaims.UserID)
	assert.Equal(t, string(domain.TokenTypeRefresh), refreshClaims.TokenType)

	// 4. Validate Access Token as Refresh Token (should fail type check)
	_, err = mgr.ValidateToken(pair.AccessToken, domain.TokenTypeRefresh)
	assert.ErrorIs(t, err, domain.ErrInvalidTokenType)
}

func TestValidateExpiredToken(t *testing.T) {
	secret := "test-secret-key"
	issuer := "test-issuer"
	mgr := NewJWTManager(secret, issuer, -1*time.Second, 1*time.Hour)

	user := &domain.User{
		ID:    uuid.New(),
		Email: "user@example.com",
		Role:  domain.RoleBuyer,
	}

	pair, err := mgr.GenerateTokenPair(user)
	assert.NoError(t, err)

	_, err = mgr.ValidateToken(pair.AccessToken, domain.TokenTypeAccess)
	assert.ErrorIs(t, err, domain.ErrExpiredToken)
}

func TestValidateInvalidSignatureToken(t *testing.T) {
	mgr1 := NewJWTManager("secret-1", "issuer", 15*time.Minute, 1*time.Hour)
	mgr2 := NewJWTManager("secret-2", "issuer", 15*time.Minute, 1*time.Hour)

	user := &domain.User{
		ID:    uuid.New(),
		Email: "user@example.com",
		Role:  domain.RoleBuyer,
	}

	pair, err := mgr1.GenerateTokenPair(user)
	assert.NoError(t, err)

	_, err = mgr2.ValidateToken(pair.AccessToken, domain.TokenTypeAccess)
	assert.ErrorIs(t, err, domain.ErrInvalidToken)
}
