package tests

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/codeshop-platform/auth-service/internal/config"
	"github.com/codeshop-platform/auth-service/internal/domain"
	"github.com/codeshop-platform/auth-service/internal/repository"
	"github.com/codeshop-platform/auth-service/internal/service"
)

// --- in-memory fakes ---
//
// These fakes let AuthService's business logic (validation, password
// hashing, token issuance/rotation, error mapping) be tested without a
// real PostgreSQL/Redis, keeping unit tests fast and hermetic. Full
// wiring against real Postgres/Redis/RabbitMQ is exercised separately in
// integration_test.go.

type fakeUserRepo struct {
	mu    sync.Mutex
	byID  map[uuid.UUID]*domain.User
	byEml map[string]uuid.UUID
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		byID:  make(map[uuid.UUID]*domain.User),
		byEml: make(map[string]uuid.UUID),
	}
}

func (f *fakeUserRepo) Create(_ context.Context, u *domain.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, exists := f.byEml[u.Email]; exists {
		return domain.ErrEmailAlreadyInUse
	}
	cp := *u
	f.byID[u.ID] = &cp
	f.byEml[u.Email] = u.ID
	return nil
}

func (f *fakeUserRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	cp := *u
	return &cp, nil
}

func (f *fakeUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.byEml[email]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	cp := *f.byID[id]
	return &cp, nil
}

func (f *fakeUserRepo) ExistsByEmail(_ context.Context, email string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.byEml[email]
	return ok, nil
}

func (f *fakeUserRepo) Update(_ context.Context, u *domain.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.byID[u.ID]; !ok {
		return domain.ErrUserNotFound
	}
	cp := *u
	f.byID[u.ID] = &cp
	return nil
}

var _ repository.UserRepository = (*fakeUserRepo)(nil)

type fakeTokenRepo struct {
	mu     sync.Mutex
	tokens map[uuid.UUID]*domain.RefreshToken
}

func newFakeTokenRepo() *fakeTokenRepo {
	return &fakeTokenRepo{tokens: make(map[uuid.UUID]*domain.RefreshToken)}
}

func (f *fakeTokenRepo) Store(_ context.Context, t *domain.RefreshToken) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := *t
	f.tokens[t.TokenID] = &cp
	return nil
}

func (f *fakeTokenRepo) Get(_ context.Context, tokenID uuid.UUID) (*domain.RefreshToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.tokens[tokenID]
	if !ok {
		return nil, domain.ErrRefreshTokenNotFound
	}
	cp := *t
	return &cp, nil
}

func (f *fakeTokenRepo) Rotate(_ context.Context, oldTokenID uuid.UUID, newToken *domain.RefreshToken) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.tokens, oldTokenID)
	cp := *newToken
	f.tokens[newToken.TokenID] = &cp
	return nil
}

func (f *fakeTokenRepo) Revoke(_ context.Context, tokenID uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.tokens, tokenID)
	return nil
}

func (f *fakeTokenRepo) RevokeAllForUser(_ context.Context, userID uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for id, t := range f.tokens {
		if t.UserID == userID {
			delete(f.tokens, id)
		}
	}
	return nil
}

var _ repository.TokenRepository = (*fakeTokenRepo)(nil)

// --- test setup helper ---

func newTestAuthService(t *testing.T) service.AuthService {
	t.Helper()
	cfg := &config.Config{
		JWTSecret:              "test-secret",
		AccessTokenTTL:         15 * time.Minute,
		RefreshTokenTTL:        7 * 24 * time.Hour,
		LoginRateLimitAttempts: 5,
		LoginRateLimitWindow:   15 * time.Minute,
		PasswordHashParams:     config.DefaultArgon2Params(),
	}
	users := newFakeUserRepo()
	tokens := newFakeTokenRepo()
	tokenSvc := service.NewTokenService(cfg)
	// publisher and redis are both nil: Register/Login must still work
	// without RabbitMQ or Redis available (best-effort event publish,
	// fail-open rate limiting - see auth_service.go).
	return service.NewAuthService(cfg, users, tokens, tokenSvc, nil, nil)
}

// --- tests ---

func TestRegister_Success(t *testing.T) {
	svc := newTestAuthService(t)
	ctx := context.Background()

	user, err := svc.Register(ctx, service.RegisterInput{
		Email:    "buyer@example.com",
		Password: "correcthorse1",
		FullName: "Buyer One",
	})
	if err != nil {
		t.Fatalf("Register() error = %v, want nil", err)
	}
	if user.Email != "buyer@example.com" {
		t.Errorf("Email = %q, want %q", user.Email, "buyer@example.com")
	}
	if user.Role != domain.RoleBuyer {
		t.Errorf("Role = %q, want %q", user.Role, domain.RoleBuyer)
	}
	if user.PasswordHash == "correcthorse1" {
		t.Error("PasswordHash must not equal the plaintext password")
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	svc := newTestAuthService(t)
	ctx := context.Background()
	in := service.RegisterInput{Email: "dup@example.com", Password: "correcthorse1", FullName: "First"}

	if _, err := svc.Register(ctx, in); err != nil {
		t.Fatalf("first Register() error = %v, want nil", err)
	}
	_, err := svc.Register(ctx, in)
	if err != domain.ErrEmailAlreadyInUse {
		t.Errorf("second Register() error = %v, want %v", err, domain.ErrEmailAlreadyInUse)
	}
}

func TestRegister_WeakPassword(t *testing.T) {
	svc := newTestAuthService(t)
	ctx := context.Background()

	_, err := svc.Register(ctx, service.RegisterInput{
		Email: "weak@example.com", Password: "short", FullName: "Weak",
	})
	if err != domain.ErrWeakPassword {
		t.Errorf("Register() error = %v, want %v", err, domain.ErrWeakPassword)
	}
}

func TestRegister_InvalidEmail(t *testing.T) {
	svc := newTestAuthService(t)
	ctx := context.Background()

	_, err := svc.Register(ctx, service.RegisterInput{
		Email: "not-an-email", Password: "correcthorse1", FullName: "Bad",
	})
	if err != domain.ErrInvalidEmail {
		t.Errorf("Register() error = %v, want %v", err, domain.ErrInvalidEmail)
	}
}

func TestLogin_Success(t *testing.T) {
	svc := newTestAuthService(t)
	ctx := context.Background()

	_, err := svc.Register(ctx, service.RegisterInput{
		Email: "login@example.com", Password: "correcthorse1", FullName: "Login User",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	user, pair, err := svc.Login(ctx, service.LoginInput{Email: "login@example.com", Password: "correcthorse1"})
	if err != nil {
		t.Fatalf("Login() error = %v, want nil", err)
	}
	if user.Email != "login@example.com" {
		t.Errorf("Email = %q, want %q", user.Email, "login@example.com")
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Error("expected non-empty access and refresh tokens")
	}
	if pair.ExpiresIn <= 0 {
		t.Errorf("ExpiresIn = %d, want > 0", pair.ExpiresIn)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	svc := newTestAuthService(t)
	ctx := context.Background()

	_, _ = svc.Register(ctx, service.RegisterInput{
		Email: "wrongpw@example.com", Password: "correcthorse1", FullName: "User",
	})

	_, _, err := svc.Login(ctx, service.LoginInput{Email: "wrongpw@example.com", Password: "totallywrong1"})
	if err != domain.ErrInvalidCredential {
		t.Errorf("Login() error = %v, want %v", err, domain.ErrInvalidCredential)
	}
}

func TestLogin_UnknownEmail(t *testing.T) {
	svc := newTestAuthService(t)
	ctx := context.Background()

	_, _, err := svc.Login(ctx, service.LoginInput{Email: "nobody@example.com", Password: "correcthorse1"})
	if err != domain.ErrInvalidCredential {
		t.Errorf("Login() error = %v, want %v (must not leak user-existence)", err, domain.ErrInvalidCredential)
	}
}

func TestRefresh_RotatesToken(t *testing.T) {
	svc := newTestAuthService(t)
	ctx := context.Background()

	_, _ = svc.Register(ctx, service.RegisterInput{
		Email: "refresh@example.com", Password: "correcthorse1", FullName: "User",
	})
	_, pair, err := svc.Login(ctx, service.LoginInput{Email: "refresh@example.com", Password: "correcthorse1"})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	newPair, err := svc.Refresh(ctx, pair.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh() error = %v, want nil", err)
	}
	if newPair.RefreshToken == pair.RefreshToken {
		t.Error("expected refresh token to rotate (new token differs from old)")
	}

	// Old refresh token must no longer be usable (rotation invalidates it).
	if _, err := svc.Refresh(ctx, pair.RefreshToken); err == nil {
		t.Error("expected old, rotated refresh token to be rejected")
	}
}

func TestLogout_RevokesToken(t *testing.T) {
	svc := newTestAuthService(t)
	ctx := context.Background()

	_, _ = svc.Register(ctx, service.RegisterInput{
		Email: "logout@example.com", Password: "correcthorse1", FullName: "User",
	})
	_, pair, err := svc.Login(ctx, service.LoginInput{Email: "logout@example.com", Password: "correcthorse1"})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if err := svc.Logout(ctx, pair.RefreshToken); err != nil {
		t.Fatalf("Logout() error = %v, want nil", err)
	}
	if _, err := svc.Refresh(ctx, pair.RefreshToken); err == nil {
		t.Error("expected refresh with a logged-out token to fail")
	}

	// Logout must be idempotent.
	if err := svc.Logout(ctx, pair.RefreshToken); err != nil {
		t.Errorf("second Logout() error = %v, want nil (idempotent)", err)
	}
}

func TestValidateAccessToken_RoundTrip(t *testing.T) {
	svc := newTestAuthService(t)
	ctx := context.Background()

	user, err := svc.Register(ctx, service.RegisterInput{
		Email: "validate@example.com", Password: "correcthorse1", FullName: "User",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	_, pair, err := svc.Login(ctx, service.LoginInput{Email: "validate@example.com", Password: "correcthorse1"})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	claims, err := svc.ValidateAccessToken(ctx, pair.AccessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken() error = %v, want nil", err)
	}
	if claims.UserID != user.ID {
		t.Errorf("UserID = %v, want %v", claims.UserID, user.ID)
	}
	if claims.Role != domain.RoleBuyer {
		t.Errorf("Role = %v, want %v", claims.Role, domain.RoleBuyer)
	}
}

func TestValidateAccessToken_RejectsGarbage(t *testing.T) {
	svc := newTestAuthService(t)
	ctx := context.Background()

	if _, err := svc.ValidateAccessToken(ctx, "not-a-jwt"); err == nil {
		t.Error("expected ValidateAccessToken to reject a malformed token")
	}
}
