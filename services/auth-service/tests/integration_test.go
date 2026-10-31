// integration_test.go exercises the Auth Service's REST API end-to-end
// (router -> middleware -> handler -> service) using httptest and the same
// in-memory fakes as auth_test.go, so it runs without a real Postgres,
// Redis, or RabbitMQ.
//
// This intentionally sits between the pure unit tests in auth_test.go and
// a true infrastructure integration test. A docker-compose-backed suite
// (real Postgres + Redis) is the natural next step once the service is
// deployed alongside its infrastructure (see README.md "Testing").
package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/codeshop-platform/auth-service/internal/config"
	"github.com/codeshop-platform/auth-service/internal/handler"
	appmiddleware "github.com/codeshop-platform/auth-service/internal/middleware"
	"github.com/codeshop-platform/auth-service/internal/service"
)

// newTestRouter builds the same route tree as cmd/server/main.go's
// buildHTTPServer, wired to in-memory fakes instead of real
// Postgres/Redis/RabbitMQ.
func newTestRouter(t *testing.T) (http.Handler, service.TokenService) {
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
	authSvc := service.NewAuthService(cfg, users, tokens, tokenSvc, nil, nil)

	authHandler := handler.NewAuthHandler(authSvc)
	userHandler := handler.NewUserHandler(authSvc)

	r := chi.NewRouter()
	r.Use(appmiddleware.RequestID)
	r.Use(appmiddleware.Recovery)

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.Refresh)
			r.Post("/logout", authHandler.Logout)
		})
		r.Route("/users", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(appmiddleware.RequireAuth(tokenSvc))
				r.Get("/me", userHandler.Me)
			})
		})
	})

	return r, tokenSvc
}

func doJSON(t *testing.T, router http.Handler, method, path string, body any, bearer string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode request body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestIntegration_RegisterLoginMeFlow(t *testing.T) {
	router, _ := newTestRouter(t)

	// 1. Register.
	rec := doJSON(t, router, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email":     "flow@example.com",
		"password":  "correcthorse1",
		"full_name": "Flow User",
	}, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("register status = %d, want %d, body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	// 2. Login.
	rec = doJSON(t, router, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email":    "flow@example.com",
		"password": "correcthorse1",
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var loginBody struct {
		Data struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int64  `json:"expires_in"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &loginBody); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if loginBody.Data.AccessToken == "" {
		t.Fatal("expected non-empty access_token")
	}

	// 3. GET /users/me with the issued access token.
	rec = doJSON(t, router, http.MethodGet, "/api/v1/users/me", nil, loginBody.Data.AccessToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("me status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var meBody struct {
		Data struct {
			Email string `json:"email"`
			Role  string `json:"role"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &meBody); err != nil {
		t.Fatalf("decode me response: %v", err)
	}
	if meBody.Data.Email != "flow@example.com" {
		t.Errorf("me.email = %q, want %q", meBody.Data.Email, "flow@example.com")
	}
	if meBody.Data.Role != "buyer" {
		t.Errorf("me.role = %q, want %q", meBody.Data.Role, "buyer")
	}

	// 4. Refresh.
	rec = doJSON(t, router, http.MethodPost, "/api/v1/auth/refresh", map[string]string{
		"refresh_token": loginBody.Data.RefreshToken,
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("refresh status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	// 5. Logout with the (now-rotated) refresh token from step 4.
	var refreshBody struct {
		Data struct {
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &refreshBody); err != nil {
		t.Fatalf("decode refresh response: %v", err)
	}
	rec = doJSON(t, router, http.MethodPost, "/api/v1/auth/logout", map[string]string{
		"refresh_token": refreshBody.Data.RefreshToken,
	}, "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestIntegration_MeWithoutAuth_Returns401(t *testing.T) {
	router, _ := newTestRouter(t)

	rec := doJSON(t, router, http.MethodGet, "/api/v1/users/me", nil, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestIntegration_MeWithBogusToken_Returns401(t *testing.T) {
	router, _ := newTestRouter(t)

	rec := doJSON(t, router, http.MethodGet, "/api/v1/users/me", nil, "not-a-real-token")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestIntegration_DuplicateRegister_Returns409(t *testing.T) {
	router, _ := newTestRouter(t)

	payload := map[string]string{
		"email":     "dup-http@example.com",
		"password":  "correcthorse1",
		"full_name": "Dup User",
	}
	rec := doJSON(t, router, http.MethodPost, "/api/v1/auth/register", payload, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("first register status = %d, want %d", rec.Code, http.StatusCreated)
	}

	rec = doJSON(t, router, http.MethodPost, "/api/v1/auth/register", payload, "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("second register status = %d, want %d, body=%s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestIntegration_WrongPasswordLogin_Returns401(t *testing.T) {
	router, _ := newTestRouter(t)

	doJSON(t, router, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email":     "wrongpw-http@example.com",
		"password":  "correcthorse1",
		"full_name": "User",
	}, "")

	rec := doJSON(t, router, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email":    "wrongpw-http@example.com",
		"password": "totallywrong1",
	}, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestIntegration_RequestIDIsEchoed(t *testing.T) {
	router, _ := newTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"email":"x","password":"y"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(appmiddleware.HeaderRequestID, "test-request-id-123")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	got := rec.Header().Get(appmiddleware.HeaderRequestID)
	if got != "test-request-id-123" {
		t.Errorf("X-Request-ID = %q, want %q (should echo caller-supplied ID)", got, "test-request-id-123")
	}
}
