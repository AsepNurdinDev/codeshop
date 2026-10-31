package middleware

import (
	"net/http"
	"strings"

	"github.com/codeshop-platform/auth-service/internal/domain"
	"github.com/codeshop-platform/auth-service/internal/service"
	"github.com/codeshop-platform/auth-service/internal/webutil"
)

// RequireAuth validates the `Authorization: Bearer <token>` header
// (docs/architecture/API_DESIGN.md §2) and stores the resulting claims in
// the request context. It rejects the request with 401 UNAUTHORIZED if the
// token is missing, malformed, or expired.
func RequireAuth(tokenSvc service.TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := webutil.RequestIDFromContext(r.Context())

			raw := r.Header.Get("Authorization")
			if raw == "" || !strings.HasPrefix(raw, "Bearer ") {
				webutil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing or malformed Authorization header", requestID)
				return
			}
			tokenString := strings.TrimPrefix(raw, "Bearer ")

			claims, err := tokenSvc.ParseAccessToken(tokenString)
			if err != nil {
				code := "UNAUTHORIZED"
				if err == domain.ErrTokenExpired {
					code = "TOKEN_EXPIRED"
				}
				webutil.WriteError(w, http.StatusUnauthorized, code, "invalid or expired access token", requestID)
				return
			}

			ctx := webutil.WithClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole further restricts access to callers holding one of the given
// roles (docs/architecture/API_DESIGN.md §3: role-based authorization for
// administrative operations). Must run after RequireAuth.
func RequireRole(roles ...domain.Role) func(http.Handler) http.Handler {
	allowed := make(map[domain.Role]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := webutil.RequestIDFromContext(r.Context())
			claims := webutil.ClaimsFromContext(r.Context())
			if claims == nil {
				webutil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required", requestID)
				return
			}
			if _, ok := allowed[claims.Role]; !ok {
				webutil.WriteError(w, http.StatusForbidden, "FORBIDDEN", "you do not have permission to perform this action", requestID)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
