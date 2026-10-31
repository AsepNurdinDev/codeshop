package service

import (
	"context"
	"fmt"
	"net/mail"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/codeshop-platform/auth-service/internal/config"
	"github.com/codeshop-platform/auth-service/internal/domain"
	"github.com/codeshop-platform/auth-service/internal/event"
	"github.com/codeshop-platform/auth-service/internal/repository"
)

// AuthService implements the core identity use cases owned by Auth
// Service: register, login, refresh, logout, and identity lookup
// (docs/architecture/SERVICE_CATALOG.md §2).
type AuthService interface {
	Register(ctx context.Context, in RegisterInput) (*domain.User, error)
	Login(ctx context.Context, in LoginInput) (*domain.User, *domain.TokenPair, error)
	Refresh(ctx context.Context, refreshTokenRaw string) (*domain.TokenPair, error)
	Logout(ctx context.Context, refreshTokenRaw string) error
	ValidateAccessToken(ctx context.Context, accessToken string) (*domain.AccessTokenClaims, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
}

type RegisterInput struct {
	Email    string
	Password string
	FullName string
}

type LoginInput struct {
	Email    string
	Password string
	IP       string // used for rate limiting & audit
}

type authService struct {
	cfg       *config.Config
	users     repository.UserRepository
	tokens    repository.TokenRepository
	tokenSvc  TokenService
	publisher event.Publisher
	redis     *redis.Client
}

// NewAuthService wires up the AuthService with its dependencies.
// publisher may be nil in environments (e.g. unit tests) where RabbitMQ is
// unavailable; in that case UserRegistered simply isn't published, and the
// registration itself still succeeds (event delivery must never block the
// primary write, consistent with the failure-mode philosophy in
// docs/architecture/ARCHITECTURE.md §12 for downstream event consumers).
func NewAuthService(
	cfg *config.Config,
	users repository.UserRepository,
	tokens repository.TokenRepository,
	tokenSvc TokenService,
	publisher event.Publisher,
	redisClient *redis.Client,
) AuthService {
	return &authService{
		cfg:       cfg,
		users:     users,
		tokens:    tokens,
		tokenSvc:  tokenSvc,
		publisher: publisher,
		redis:     redisClient,
	}
}

// Register creates a new buyer account.
// See docs/architecture/API_DESIGN.md §13 POST /api/v1/auth/register.
func (s *authService) Register(ctx context.Context, in RegisterInput) (*domain.User, error) {
	email := normalizeEmail(in.Email)
	if !isValidEmail(email) {
		return nil, domain.ErrInvalidEmail
	}
	if err := validatePasswordStrength(in.Password); err != nil {
		return nil, err
	}

	exists, err := s.users.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("check existing email: %w", err)
	}
	if exists {
		return nil, domain.ErrEmailAlreadyInUse
	}

	hash, err := s.tokenSvc.HashPassword(in.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := domain.NewUser(email, hash, in.FullName)
	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}

	// Publish UserRegistered (docs/architecture/EVENT_DESIGN.md §4).
	// Best-effort: a notification-email failure must never fail
	// registration itself.
	if s.publisher != nil {
		correlationID := uuid.NewString()
		if err := s.publisher.PublishUserRegistered(ctx, correlationID, event.UserRegisteredPayload{
			UserID:   user.ID.String(),
			Email:    user.Email,
			FullName: user.FullName,
		}); err != nil {
			// Log-and-continue in a real deployment (structured logger);
			// kept as a plain print here to avoid pulling a logging
			// dependency into this layer.
			fmt.Printf("warn: failed to publish UserRegistered event: %v\n", err)
		}
	}

	return user, nil
}

// Login authenticates a user and issues a new token pair.
// See docs/architecture/ARCHITECTURE.md §11 Authentication Flow.
func (s *authService) Login(ctx context.Context, in LoginInput) (*domain.User, *domain.TokenPair, error) {
	email := normalizeEmail(in.Email)

	if err := s.checkLoginRateLimit(ctx, email); err != nil {
		return nil, nil, err
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		s.recordLoginFailure(ctx, email)
		if err == domain.ErrUserNotFound {
			return nil, nil, domain.ErrInvalidCredential
		}
		return nil, nil, err
	}

	if !user.IsActive() {
		return nil, nil, domain.ErrAccountSuspended
	}

	ok, err := s.tokenSvc.VerifyPassword(in.Password, user.PasswordHash)
	if err != nil {
		return nil, nil, fmt.Errorf("verify password: %w", err)
	}
	if !ok {
		s.recordLoginFailure(ctx, email)
		return nil, nil, domain.ErrInvalidCredential
	}

	s.resetLoginRateLimit(ctx, email)

	pair, err := s.issueTokenPair(ctx, user)
	if err != nil {
		return nil, nil, err
	}
	return user, pair, nil
}

// Refresh redeems a refresh token for a new token pair, rotating the
// refresh token in the process (docs/architecture/SERVICE_CATALOG.md §2
// "refresh token rotation").
func (s *authService) Refresh(ctx context.Context, refreshTokenRaw string) (*domain.TokenPair, error) {
	tokenID, err := uuid.Parse(refreshTokenRaw)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	stored, err := s.tokens.Get(ctx, tokenID)
	if err != nil {
		return nil, err
	}
	if !stored.IsValid(time.Now().UTC()) {
		return nil, domain.ErrRefreshTokenExpired
	}

	user, err := s.users.GetByID(ctx, stored.UserID)
	if err != nil {
		return nil, err
	}
	if !user.IsActive() {
		return nil, domain.ErrAccountSuspended
	}

	newRefresh := s.tokenSvc.NewRefreshToken(user.ID)
	if err := s.tokens.Rotate(ctx, tokenID, newRefresh); err != nil {
		return nil, fmt.Errorf("rotate refresh token: %w", err)
	}

	accessToken, expiresIn, err := s.tokenSvc.IssueAccessToken(domain.AccessTokenClaims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
	})
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRefresh.TokenID.String(),
		ExpiresIn:    expiresIn,
	}, nil
}

// Logout revokes a refresh token so it can no longer be used to obtain new
// access tokens. Idempotent: logging out twice is not an error.
func (s *authService) Logout(ctx context.Context, refreshTokenRaw string) error {
	tokenID, err := uuid.Parse(refreshTokenRaw)
	if err != nil {
		return domain.ErrInvalidToken
	}
	return s.tokens.Revoke(ctx, tokenID)
}

// ValidateAccessToken performs stateless JWT validation. This backs the
// internal gRPC ValidateToken RPC (proto/auth.proto) used by other
// services and, later, the API Gateway.
func (s *authService) ValidateAccessToken(ctx context.Context, accessToken string) (*domain.AccessTokenClaims, error) {
	return s.tokenSvc.ParseAccessToken(accessToken)
}

// GetUserByID backs the internal gRPC GetUser RPC and the REST
// GET /api/v1/users/me endpoint.
func (s *authService) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return s.users.GetByID(ctx, id)
}

// --- helpers ---

func (s *authService) issueTokenPair(ctx context.Context, user *domain.User) (*domain.TokenPair, error) {
	refresh := s.tokenSvc.NewRefreshToken(user.ID)
	if err := s.tokens.Store(ctx, refresh); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	accessToken, expiresIn, err := s.tokenSvc.IssueAccessToken(domain.AccessTokenClaims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
	})
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refresh.TokenID.String(),
		ExpiresIn:    expiresIn,
	}, nil
}

// --- login rate limiting (brute-force protection) ---
//
// docs/architecture/SERVICE_CATALOG.md §2: "rate limiting login untuk
// mencegah brute force". Implemented as a simple fixed-window counter in
// Redis, keyed by email (protects a given account regardless of source
// IP) - per docs/decisions/ADR-007-redis.md.

func (s *authService) loginRateLimitKey(email string) string {
	return "auth:login_attempts:" + email
}

func (s *authService) checkLoginRateLimit(ctx context.Context, email string) error {
	if s.redis == nil {
		return nil
	}
	key := s.loginRateLimitKey(email)
	count, err := s.redis.Get(ctx, key).Int()
	if err != nil && err != redis.Nil {
		// Fail open: a Redis outage should not lock everyone out of login.
		return nil
	}
	if count >= s.cfg.LoginRateLimitAttempts {
		return fmt.Errorf("%w: too many failed login attempts, try again later", domain.ErrInvalidCredential)
	}
	return nil
}

func (s *authService) recordLoginFailure(ctx context.Context, email string) {
	if s.redis == nil {
		return
	}
	key := s.loginRateLimitKey(email)
	pipe := s.redis.TxPipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, s.cfg.LoginRateLimitWindow)
	_, _ = pipe.Exec(ctx)
}

func (s *authService) resetLoginRateLimit(ctx context.Context, email string) {
	if s.redis == nil {
		return
	}
	s.redis.Del(ctx, s.loginRateLimitKey(email))
}

// --- input validation ---

func normalizeEmail(email string) string {
	return email
}

func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

// validatePasswordStrength enforces a minimum bar: at least 8 characters,
// containing at least one letter and one digit. This is intentionally
// simple for the MVP; a dedicated password-policy library can replace it
// later without changing the AuthService interface.
func validatePasswordStrength(password string) error {
	if len(password) < 8 {
		return domain.ErrWeakPassword
	}
	var hasLetter, hasDigit bool
	for _, r := range password {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return domain.ErrWeakPassword
	}
	return nil
}
