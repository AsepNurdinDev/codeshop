// Package webutil holds small, dependency-free helpers (request-scoped
// context values, the JSON response envelope) shared by both the
// middleware and handler packages. It exists specifically so that
// middleware and handler never need to import each other directly:
// middleware sets values defined here, handler reads them, both depend
// only on webutil.
package webutil

import (
	"context"

	"github.com/codeshop-platform/auth-service/internal/domain"
)

type ctxKey string

const (
	requestIDKey ctxKey = "request_id"
	claimsKey    ctxKey = "auth_claims"
)

// WithRequestID stores the request ID for this request in ctx.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestIDFromContext retrieves the request ID set by the RequestID
// middleware, or "" if none is present (e.g. a handler invoked directly
// in a unit test).
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

// WithClaims stores the authenticated caller's claims in ctx.
func WithClaims(ctx context.Context, claims *domain.AccessTokenClaims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// ClaimsFromContext retrieves the authenticated caller's claims, set by
// the RequireAuth middleware. Returns nil if the request was not
// authenticated.
func ClaimsFromContext(ctx context.Context) *domain.AccessTokenClaims {
	claims, _ := ctx.Value(claimsKey).(*domain.AccessTokenClaims)
	return claims
}
