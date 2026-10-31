package handler

import (
	"errors"
	"net/http"

	"github.com/codeshop-platform/auth-service/internal/domain"
	"github.com/codeshop-platform/auth-service/internal/service"
	"github.com/codeshop-platform/auth-service/internal/webutil"
)

// UserHandler implements the /api/v1/users/* endpoints
// (docs/architecture/API_DESIGN.md §13).
type UserHandler struct {
	auth service.AuthService
}

func NewUserHandler(auth service.AuthService) *UserHandler {
	return &UserHandler{auth: auth}
}

type meResponse struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

// Me handles GET /api/v1/users/me. Requires authentication
// (see middleware.RequireAuth); only ever returns the caller's own data
// (docs/architecture/API_DESIGN.md §13: "hanya data milik diri sendiri").
func (h *UserHandler) Me(w http.ResponseWriter, r *http.Request) {
	requestID := webutil.RequestIDFromContext(r.Context())

	claims := webutil.ClaimsFromContext(r.Context())
	if claims == nil {
		webutil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required", requestID)
		return
	}

	user, err := h.auth.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			webutil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "user no longer exists", requestID)
			return
		}
		webutil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to load user", requestID)
		return
	}

	webutil.WriteJSON(w, http.StatusOK, meResponse{
		UserID:   user.ID.String(),
		Email:    user.Email,
		FullName: user.FullName,
		Role:     string(user.Role),
		Status:   string(user.Status),
	})
}
