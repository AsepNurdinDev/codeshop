package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/codeshop-platform/auth-service/internal/domain"
	"github.com/codeshop-platform/auth-service/internal/service"
	"github.com/codeshop-platform/auth-service/internal/webutil"
)

// AuthHandler implements the /api/v1/auth/* endpoints
// (docs/architecture/API_DESIGN.md §13).
type AuthHandler struct {
	auth service.AuthService
}

func NewAuthHandler(auth service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

// --- request/response DTOs ---

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

type registerResponse struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Register handles POST /api/v1/auth/register.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	requestID := webutil.RequestIDFromContext(r.Context())

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		webutil.WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "malformed JSON body", requestID)
		return
	}

	user, err := h.auth.Register(r.Context(), service.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
	})
	if err != nil {
		writeRegisterError(w, err, requestID)
		return
	}

	webutil.WriteJSON(w, http.StatusCreated, registerResponse{
		UserID: user.ID.String(),
		Email:  user.Email,
	})
}

// Login handles POST /api/v1/auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	requestID := webutil.RequestIDFromContext(r.Context())

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		webutil.WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "malformed JSON body", requestID)
		return
	}

	_, pair, err := h.auth.Login(r.Context(), service.LoginInput{
		Email:    req.Email,
		Password: req.Password,
		IP:       clientIP(r),
	})
	if err != nil {
		writeLoginError(w, err, requestID)
		return
	}

	webutil.WriteJSON(w, http.StatusOK, loginResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresIn:    pair.ExpiresIn,
	})
}

// Refresh handles POST /api/v1/auth/refresh.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	requestID := webutil.RequestIDFromContext(r.Context())

	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		webutil.WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "malformed JSON body", requestID)
		return
	}

	pair, err := h.auth.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		status, code := mapTokenError(err)
		webutil.WriteError(w, status, code, err.Error(), requestID)
		return
	}

	webutil.WriteJSON(w, http.StatusOK, loginResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresIn:    pair.ExpiresIn,
	})
}

// Logout handles POST /api/v1/auth/logout.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	requestID := webutil.RequestIDFromContext(r.Context())

	var req logoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		webutil.WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "malformed JSON body", requestID)
		return
	}

	// Logout is intentionally idempotent/best-effort: an already-revoked
	// or unknown token still results in 204, since the caller's desired
	// end-state (not logged in) is achieved either way.
	_ = h.auth.Logout(r.Context(), req.RefreshToken)
	w.WriteHeader(http.StatusNoContent)
}

// --- error mapping ---

func writeRegisterError(w http.ResponseWriter, err error, requestID string) {
	switch {
	case errors.Is(err, domain.ErrEmailAlreadyInUse):
		webutil.WriteError(w, http.StatusConflict, "EMAIL_ALREADY_EXISTS", err.Error(), requestID)
	case errors.Is(err, domain.ErrInvalidEmail), errors.Is(err, domain.ErrWeakPassword):
		webutil.WriteError(w, http.StatusBadRequest, "INVALID_INPUT", err.Error(), requestID)
	default:
		webutil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to register user", requestID)
	}
}

func writeLoginError(w http.ResponseWriter, err error, requestID string) {
	switch {
	case errors.Is(err, domain.ErrAccountSuspended):
		webutil.WriteError(w, http.StatusForbidden, "ACCOUNT_SUSPENDED", err.Error(), requestID)
	case errors.Is(err, domain.ErrInvalidCredential):
		webutil.WriteError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", err.Error(), requestID)
	default:
		webutil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to login", requestID)
	}
}

func mapTokenError(err error) (int, string) {
	switch {
	case errors.Is(err, domain.ErrRefreshTokenNotFound):
		return http.StatusUnauthorized, "REFRESH_TOKEN_NOT_FOUND"
	case errors.Is(err, domain.ErrRefreshTokenExpired):
		return http.StatusUnauthorized, "REFRESH_TOKEN_EXPIRED"
	case errors.Is(err, domain.ErrRefreshTokenRevoked):
		return http.StatusUnauthorized, "REFRESH_TOKEN_REVOKED"
	case errors.Is(err, domain.ErrAccountSuspended):
		return http.StatusForbidden, "ACCOUNT_SUSPENDED"
	case errors.Is(err, domain.ErrInvalidToken):
		return http.StatusUnauthorized, "INVALID_TOKEN"
	case errors.Is(err, domain.ErrUserNotFound):
		return http.StatusUnauthorized, "INVALID_TOKEN"
	default:
		return http.StatusInternalServerError, "INTERNAL_ERROR"
	}
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return fwd
	}
	return r.RemoteAddr
}
