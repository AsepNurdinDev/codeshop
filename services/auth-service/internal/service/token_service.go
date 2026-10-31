package service

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"

	"github.com/codeshop-platform/auth-service/internal/config"
	"github.com/codeshop-platform/auth-service/internal/domain"
)

// TokenService owns password hashing/verification and JWT access-token
// issuance/validation. It has no dependency on storage: refresh-token
// persistence lives in TokenRepository, orchestrated by AuthService.
type TokenService interface {
	HashPassword(plain string) (string, error)
	VerifyPassword(plain, encodedHash string) (bool, error)

	IssueAccessToken(claims domain.AccessTokenClaims) (string, int64, error)
	ParseAccessToken(tokenString string) (*domain.AccessTokenClaims, error)

	NewRefreshToken(userID uuid.UUID) *domain.RefreshToken
}

type tokenService struct {
	cfg *config.Config
}

// NewTokenService returns the default TokenService implementation.
func NewTokenService(cfg *config.Config) TokenService {
	return &tokenService{cfg: cfg}
}

// --- Password hashing (Argon2id) ---
//
// Per docs/architecture/DATABASE_DESIGN.md §2 and
// docs/architecture/SERVICE_CATALOG.md §2, passwords are hashed with
// Argon2id. The encoded hash embeds the parameters + salt so that
// verification works even if DefaultArgon2Params() changes later
// (standard PHC-string-like format).

const argon2idPrefix = "argon2id"

func (s *tokenService) HashPassword(plain string) (string, error) {
	p := s.cfg.PasswordHashParams

	salt := make([]byte, p.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(plain), salt, p.Iterations, p.Memory, p.Parallelism, p.KeyLength)

	encoded := fmt.Sprintf(
		"%s$v=19$m=%d,t=%d,p=%d$%s$%s",
		argon2idPrefix, p.Memory, p.Iterations, p.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)
	return encoded, nil
}

func (s *tokenService) VerifyPassword(plain, encodedHash string) (bool, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 5 || parts[0] != argon2idPrefix {
		return false, fmt.Errorf("unrecognized password hash format")
	}

	var memory, iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[2], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return false, fmt.Errorf("parse hash params: %w", err)
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false, fmt.Errorf("decode salt: %w", err)
	}
	wantHash, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("decode hash: %w", err)
	}

	gotHash := argon2.IDKey([]byte(plain), salt, iterations, memory, parallelism, uint32(len(wantHash)))

	// Constant-time comparison to avoid timing side-channels.
	return subtle.ConstantTimeCompare(gotHash, wantHash) == 1, nil
}

// --- JWT access tokens ---
//
// Access tokens are short-lived and signature-based so that other
// services (or a future API Gateway) can validate them statelessly,
// without a round-trip to Auth Service, per
// docs/architecture/API_DESIGN.md §2 and ARCHITECTURE.md §12
// ("token JWT tetap dapat divalidasi secara stateless").

type jwtClaims struct {
	Email string `json:"email"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

func (s *tokenService) IssueAccessToken(claims domain.AccessTokenClaims) (string, int64, error) {
	now := time.Now().UTC()
	ttl := s.cfg.AccessTokenTTL
	expiresAt := now.Add(ttl)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims{
		Email: claims.Email,
		Role:  string(claims.Role),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   claims.UserID.String(),
			Issuer:    "auth-service",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        uuid.NewString(),
		},
	})

	signed, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", 0, fmt.Errorf("sign access token: %w", err)
	}
	return signed, int64(ttl.Seconds()), nil
}

func (s *tokenService) ParseAccessToken(tokenString string) (*domain.AccessTokenClaims, error) {
	var claims jwtClaims
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil {
		if strings.Contains(err.Error(), "token is expired") {
			return nil, domain.ErrTokenExpired
		}
		return nil, domain.ErrInvalidToken
	}
	if !token.Valid {
		return nil, domain.ErrInvalidToken
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	return &domain.AccessTokenClaims{
		UserID: userID,
		Email:  claims.Email,
		Role:   domain.Role(claims.Role),
	}, nil
}

// --- Refresh tokens ---

func (s *tokenService) NewRefreshToken(userID uuid.UUID) *domain.RefreshToken {
	now := time.Now().UTC()
	return &domain.RefreshToken{
		TokenID:   uuid.New(),
		UserID:    userID,
		IssuedAt:  now,
		ExpiresAt: now.Add(s.cfg.RefreshTokenTTL),
		Revoked:   false,
	}
}
