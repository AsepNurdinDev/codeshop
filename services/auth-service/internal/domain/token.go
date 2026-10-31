package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
	ErrRefreshTokenExpired  = errors.New("refresh token expired")
	ErrRefreshTokenRevoked  = errors.New("refresh token has been revoked")
	ErrInvalidToken         = errors.New("invalid or malformed token")
	ErrTokenExpired         = errors.New("token expired")
)

// AccessTokenClaims is the payload encoded into the JWT access token.
// Signature-based validation (see docs/architecture/API_DESIGN.md §2 and
// ARCHITECTURE.md §12) lets other services validate a token statelessly,
// without calling back into Auth Service, as long as they trust the
// signing key.
type AccessTokenClaims struct {
	UserID uuid.UUID
	Email  string
	Role   Role
}

// TokenPair is what is returned to the client on login/refresh.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64 // access token TTL in seconds, per API_DESIGN.md §13 (POST /auth/login response)
}

// RefreshToken represents a refresh-token session persisted in Redis
// (docs/decisions/ADR-007-redis.md). It is opaque to the client; the
// client only ever sees the raw token string, never this struct.
//
// Refresh tokens are rotated on every use (docs/architecture/SERVICE_CATALOG.md
// §2 "refresh token rotation") so TokenID identifies a single-use session
// record, not the family of tokens issued to a user.
type RefreshToken struct {
	TokenID   uuid.UUID // used as the Redis key suffix
	UserID    uuid.UUID
	IssuedAt  time.Time
	ExpiresAt time.Time
	Revoked   bool
}

// IsValid reports whether the refresh token can still be redeemed.
func (t *RefreshToken) IsValid(now time.Time) bool {
	if t.Revoked {
		return false
	}
	return now.Before(t.ExpiresAt)
}
