package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/codeshop-platform/auth-service/internal/webutil"
)

// Recovery converts a panic in any downstream handler into a clean 500
// error envelope (docs/architecture/API_DESIGN.md §6) instead of crashing
// the process or leaking a Go stack trace to the client.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				requestID := webutil.RequestIDFromContext(r.Context())
				log.Printf("panic recovered: %v request_id=%s\n%s", rec, requestID, debug.Stack())
				webutil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred", requestID)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
